package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stanterprise/observer/pkg/server"
)

func main() {
	var (
		port = flag.String("port", envOr("PORT", "50051"), "TCP port to listen on")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	addr := ":" + *port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("listen failed", "error", err, "addr", addr)
		os.Exit(1)
	}

	grpcServer := server.NewGRPCServer(logger)
	// Ingestion service does not use database directly - it publishes to NATS
	// For now, we run without DB to maintain stateless ingestion pattern
	server.RegisterServices(grpcServer, logger, nil)
	logger.Info("ingestion server starting", "addr", lis.Addr().String())

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

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
