package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stanterprise/observer/internal/database"
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/observer/pkg/consumer"
)

func main() {
	var (
		natsURL        = flag.String("nats-url", envOr("NATS_URL", "nats://localhost:4222"), "NATS server URL")
		streamName     = flag.String("stream", envOr("NATS_STREAM", "tests_events"), "NATS stream name")
		consumerName   = flag.String("consumer", envOr("NATS_CONSUMER", "processor"), "NATS consumer name")
		batchSize      = flag.Int("batch-size", 10, "Number of messages to fetch per batch")
		maxWait        = flag.Duration("max-wait", 5*time.Second, "Maximum wait time for messages")
		retainMessages = flag.Bool("retain-messages", envOr("RETAIN_MESSAGES", "") == "true", "Retain all raw messages in MongoDB (overrides RETAIN_MESSAGES env var)")
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

	repo := repository.NewMongoRepository(mongoDB.TestRunsCollection(), logger)

	// Optionally create the raw message repository when retention is enabled.
	var rawMsgRepo *repository.RawMessageRepository
	if *retainMessages {
		rawMsgRepo = repository.NewRawMessageRepository(mongoDB.RawMessagesCollection(), logger)
		logger.Info("raw message retention enabled", "collection", "raw_messages")
	}

	cfg := consumer.MongoNATSConsumerConfig{
		URL:            *natsURL,
		StreamName:     *streamName,
		ConsumerName:   *consumerName,
		BatchSize:      *batchSize,
		MaxWait:        *maxWait,
		RetainMessages: *retainMessages,
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

