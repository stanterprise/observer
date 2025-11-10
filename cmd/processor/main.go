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
	"github.com/stanterprise/observer/pkg/consumer"
)

func main() {
	var (
		natsURL      = flag.String("nats-url", envOr("NATS_URL", "nats://localhost:4222"), "NATS server URL")
		streamName   = flag.String("stream", envOr("NATS_STREAM", "tests_events"), "NATS stream name")
		consumerName = flag.String("consumer", envOr("NATS_CONSUMER", "processor"), "NATS consumer name")
		batchSize    = flag.Int("batch-size", 10, "Number of messages to fetch per batch")
		maxWait      = flag.Duration("max-wait", 5*time.Second, "Maximum wait time for messages")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Processor service requires database connection
	db, err := database.ConnectFromEnv(logger)
	if err != nil {
		logger.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	if db == nil {
		logger.Error("DATABASE_URL not set; processor requires database")
		os.Exit(1)
	}
	logger.Info("database connected")

	if err := database.AutoMigrateSchema(db, logger); err != nil {
		logger.Error("automigrate failed", "error", err)
		os.Exit(1)
	}

	// Create NATS consumer configuration
	cfg := consumer.NATSConsumerConfig{
		URL:          *natsURL,
		StreamName:   *streamName,
		ConsumerName: *consumerName,
		BatchSize:    *batchSize,
		MaxWait:      *maxWait,
	}

	// Initialize NATS consumer
	natsConsumer, err := consumer.NewNATSConsumer(cfg, logger, db)
	if err != nil {
		logger.Error("failed to create NATS consumer", "error", err)
		os.Exit(1)
	}
	defer natsConsumer.Close()

	logger.Info("processor service starting",
		"nats_url", *natsURL,
		"stream", *streamName,
		"consumer", *consumerName)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run consumer in separate goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := natsConsumer.Start(ctx, cfg); err != nil && err != context.Canceled {
			errChan <- err
		}
		close(errChan)
	}()

	// Wait for shutdown signal
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

	// Cancel context to stop consumer
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
