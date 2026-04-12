package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/stanterprise/observer/internal/database"
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/observer/pkg/consumer"
)

func main() {
	var (
		natsURL          = flag.String("nats-url", envOr("NATS_URL", "nats://localhost:4222"), "NATS server URL")
		streamName       = flag.String("stream", envOr("NATS_STREAM", "tests_events"), "NATS stream name")
		consumerName     = flag.String("consumer", envOr("NATS_CONSUMER", "processor"), "NATS consumer name")
		batchSize        = flag.Int("batch-size", 10, "Number of messages to fetch per batch")
		maxWait          = flag.Duration("max-wait", 5*time.Second, "Maximum wait time for messages")
		maxDeliver       = flag.Int("max-deliver", envOrInt("NATS_MAX_DELIVER", 5), "Maximum delivery attempts before DLQ")
		ackWait          = flag.Duration("ack-wait", envOrDuration("NATS_ACK_WAIT", 30*time.Second), "JetStream ack wait timeout")
		dlqSubject       = flag.String("dlq-subject", envOr("NATS_DLQ_SUBJECT", "tests.events.v1.dlq"), "JetStream subject for dead-letter messages")
		deferMaxAttempts = flag.Int("defer-max-attempts", envOrInt("DEFER_QUEUE_MAX_ATTEMPTS", 5), "Maximum replay attempts for deferred orphan step events")
		deferTTL         = flag.Duration("defer-ttl", envOrDuration("DEFER_QUEUE_TTL", 5*time.Minute), "TTL for deferred orphan step events")
		retainMessages   = flag.Bool("retain-messages", envOr("RETAIN_MESSAGES", "") == "true", "Retain all raw NATS messages grouped by run_id in MongoDB (overrides RETAIN_MESSAGES env var)")
		rawMsgCollection = flag.String("raw-messages-collection", envOr("RAW_MESSAGES_COLLECTION", "raw_messages"), "MongoDB collection for retained raw messages (overrides RAW_MESSAGES_COLLECTION env var)")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Connect to MongoDB
	mongoDB, err := database.ConnectMongoDBFromEnv(logger)
	if err != nil {
		logger.Error("mongodb connect failed", "error", err)
		os.Exit(1)
	}
	if mongoDB == nil {
		logger.Error("MONGODB_URI or MONGO_URI not set; processor requires MongoDB")
		os.Exit(1)
	}

	logger.Info("using MongoDB backend")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer mongoDB.Close(ctx)

	// Optionally connect to PostgreSQL (relational execution store).
	pgConn, err := database.ConnectPostgresFromEnv(logger)
	if err != nil {
		logger.Error("postgres connect failed", "error", err)
		os.Exit(1)
	}
	if pgConn != nil {
		defer pgConn.Close()
		if err := pgConn.InitSchema(ctx); err != nil {
			logger.Error("postgres schema init failed", "error", err)
			os.Exit(1)
		}
		logger.Info("postgres available for relational execution",
			"database", pgConn.DatabaseName())
	}

	repo := repository.NewMongoRepository(mongoDB.TestRunsCollection(), logger)

	// Optionally create the raw message repository when retention is enabled.
	var rawMsgRepo *repository.RawMessageRepository
	if *retainMessages {
		rawMsgRepo = repository.NewRawMessageRepository(mongoDB.Collection(*rawMsgCollection), logger)
		logger.Info("raw message retention enabled",
			"database", mongoDB.DatabaseName(),
			"collection", rawMsgRepo.CollectionName())
	}

	cfg := consumer.MongoNATSConsumerConfig{
		URL:                   *natsURL,
		StreamName:            *streamName,
		ConsumerName:          *consumerName,
		BatchSize:             *batchSize,
		MaxWait:               *maxWait,
		MaxDeliver:            *maxDeliver,
		AckWait:               *ackWait,
		DLQSubject:            *dlqSubject,
		DeferQueueMaxAttempts: *deferMaxAttempts,
		DeferQueueTTL:         *deferTTL,
		RetainMessages:        *retainMessages,
	}

	natsConsumer, err := consumer.NewMongoNATSConsumer(cfg, logger, repo, rawMsgRepo)
	if err != nil {
		logger.Error("failed to create MongoDB NATS consumer", "error", err)
		os.Exit(1)
	}
	defer natsConsumer.Close()

	logger.Info("processor service starting",
		"nats_url", *natsURL,
		"stream", *streamName,
		"consumer", *consumerName,
		"max_deliver", *maxDeliver,
		"ack_wait", *ackWait,
		"dlq_subject", *dlqSubject,
		"defer_max_attempts", *deferMaxAttempts,
		"defer_ttl", *deferTTL,
		"retain_messages", *retainMessages)

	errChan := make(chan error, 1)
	go func() {
		if err := natsConsumer.Start(ctx, cfg); err != nil && err != context.Canceled {
			errChan <- err
		}
		close(errChan)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("shutdown signal received", "signal", sig)
	case err := <-errChan:
		if err != nil {
			logger.Error("consumer error", "error", err)
			os.Exit(1)
		}
	}

	cancel()

	// Allow up to 5s for graceful stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		// Wait for consumer to finish
		<-errChan
		close(done)
	}()

	select {
	case <-shutdownCtx.Done():
		logger.Warn("graceful stop timeout")
	case <-done:
		logger.Info("processor service stopped gracefully")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envOrDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
