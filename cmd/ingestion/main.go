package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/stanterprise/observer/pkg/healthhttp"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/observer/pkg/server"
	"github.com/stanterprise/observer/pkg/version"
	"google.golang.org/grpc"
)

func main() {
	var (
		port       = flag.String("port", envOr("PORT", "50051"), "TCP port to listen on")
		healthPort = flag.String("health-port", envOr("HEALTH_PORT", "8081"), "HTTP health check port")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mode := envOr("OBSERVER_MODE", "distributed")
	grpcServer, err := newIngestionGRPCServer(logger, mode, os.Getenv("INGESTION_AUTH_KEY"))
	if err != nil {
		logger.Error("invalid ingestion authentication configuration", "error", err)
		os.Exit(1)
	}

	reflectionEnabled, err := strconv.ParseBool(envOr("GRPC_REFLECTION", "false"))
	if err != nil {
		logger.Error("invalid GRPC_REFLECTION value", "error", err)
		os.Exit(1)
	}
	if reflectionEnabled && mode != "local" {
		logger.Error("GRPC_REFLECTION can only be enabled in local mode")
		os.Exit(1)
	}

	addr := ":" + *port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("listen failed", "error", err, "addr", addr)
		os.Exit(1)
	}

	// Initialize NATS publisher if NATS_URL is configured
	var pub *publisher.NATSPublisher
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		cfg := publisher.NATSConfig{
			URL:           natsURL,
			StreamName:    envOr("NATS_STREAM", publisher.DefaultStreamName),
			SubjectPrefix: envOr("NATS_SUBJECT_PREFIX", publisher.DefaultSubjectPrefix),
		}
		pub, err = publisher.NewNATSPublisher(cfg, logger)
		if err != nil {
			logger.Error("failed to initialize NATS publisher", "error", err)
			os.Exit(1)
		}
		defer pub.Close()
		logger.Info("NATS publisher enabled", "url", natsURL)
	} else {
		logger.Info("NATS publisher disabled - NATS_URL not set")
	}

	// Start HTTP health check server for GCE load balancer compatibility
	healthServer := healthhttp.NewServer(*healthPort, logger)
	go func() {
		if err := healthServer.Start(); err != nil {
			logger.Error("HTTP health server failed", "error", err)
		}
	}()

	// Ingestion service does not use database directly - it publishes to NATS
	// For now, we run without DB to maintain stateless ingestion pattern
	if pub != nil {
		server.RegisterServicesWithPublisher(grpcServer, logger, nil, pub)
	} else {
		server.RegisterServices(grpcServer, logger, nil)
	}
	if reflectionEnabled {
		logger.Warn("gRPC reflection is enabled; use only in local development")
		server.RegisterReflection(grpcServer)
	}
	logger.Info("ingestion server starting", "version", version.Version, "commit", version.Commit, "addr", lis.Addr().String())

	// Run server in separate goroutine and capture fatal serve errors.
	errChan := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("serve failed: %w", err)
		}
		close(errChan)
	}()

	// Graceful shutdown handling or fatal serve error.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		logger.Info("shutdown signal received", "signal", sig)
	case err := <-errChan:
		if err != nil {
			logger.Error("server serve error", "error", err)
		}
	}

	// Allow up to 5s for graceful stop.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown HTTP health server
	if err := healthServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP health server shutdown error", "error", err)
	}

	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-ctx.Done():
		logger.Warn("graceful stop timeout, forcing stop")
		grpcServer.Stop()
	case <-done:
		logger.Info("ingestion server stopped gracefully")
	}

	// If Serve returned an error earlier, exit non-zero.
	select {
	case err := <-errChan:
		if err != nil {
			os.Exit(1)
		}
	default:
	}
}

func newIngestionGRPCServer(logger *slog.Logger, mode, authKey string) (*grpc.Server, error) {
	switch mode {
	case "local":
		if authKey != "" {
			return nil, fmt.Errorf("INGESTION_AUTH_KEY must not be set in local mode")
		}
		return server.NewGRPCServer(logger), nil
	case "distributed":
		if authKey == "" {
			return nil, fmt.Errorf("INGESTION_AUTH_KEY is required in distributed mode")
		}
		return server.NewAuthenticatedGRPCServer(logger, authKey)
	default:
		return nil, fmt.Errorf("OBSERVER_MODE must be either local or distributed")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
