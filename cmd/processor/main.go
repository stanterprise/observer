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

	"gorm.io/gorm"

	"github.com/stanterprise/observer/internal/database"
	"github.com/stanterprise/observer/internal/repository/mongodb"
	"github.com/stanterprise/observer/internal/repository/postgres"
	"github.com/stanterprise/observer/internal/telemetry"
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
		retainMessages   = flag.Bool("retain-messages", envOr("RETAIN_MESSAGES", "") == "true", "Deprecated no-op retained for compatibility; raw MongoDB message retention has been removed")
		metricsPort      = flag.String("metrics-port", envOr("METRICS_PORT", "9090"), "HTTP metrics port")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialise OTel telemetry (Prometheus exporter).
	shutdownTelemetry, err := telemetry.Setup(context.Background(), "observer-processor", logger)
	if err != nil {
		logger.Warn("telemetry setup failed – metrics disabled", "error", err)
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdownTelemetry(ctx); err != nil {
				logger.Warn("telemetry shutdown error", "error", err)
			}
		}()
	}

	// Start metrics HTTP server (Prometheus scrape endpoint).
	stopMetrics := telemetry.StartMetricsServer(*metricsPort, logger)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := stopMetrics(ctx); err != nil {
			logger.Warn("metrics server shutdown error", "error", err)
		}
	}()

	// Connect to MongoDB for bufferized step management only.
	mongoDB, err := database.ConnectMongoDBFromEnv(logger)
	if err != nil {
		logger.Error("mongodb connect failed", "error", err)
		os.Exit(1)
	}
	if mongoDB == nil {
		logger.Error("MONGODB_URI or MONGO_URI not set (required for bufferized step management)")
		os.Exit(1)
	}
	defer mongoDB.Close(context.Background())

	// Connect to PostgreSQL (optional — processor continues without it but PG writes will no-op).
	pgDB, err := database.ConnectPostgresFromEnv(logger)
	if err != nil {
		logger.Error("postgres connect failed", "error", err)
		os.Exit(1)
	}
	var pgGormDB *gorm.DB
	if pgDB != nil {
		defer func() {
			if closeErr := pgDB.Close(); closeErr != nil {
				logger.Warn("failed to close postgres connection", "error", closeErr)
			}
		}()
		pgGormDB = pgDB.DB
	}
	pgRepo := postgres.NewPostgresRepository(pgGormDB, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Instantiate MongoRepository for the standalone live step buffer collection.
	bufferRepo := mongodb.NewMongoRepository(mongoDB.LiveStepBuffersCollection(), logger)

	cfg := consumer.NATSConsumerConfig{
		URL:          *natsURL,
		StreamName:   *streamName,
		ConsumerName: *consumerName,
		BatchSize:    *batchSize,
		MaxWait:      *maxWait,
		MaxDeliver:   *maxDeliver,
		AckWait:      *ackWait,
		DLQSubject:   *dlqSubject,
	}

	natsConsumer, err := consumer.NewNATSConsumer(cfg, logger, bufferRepo, pgRepo)
	if err != nil {
		logger.Error("failed to create NATS consumer", "error", err)
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
