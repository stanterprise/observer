package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stanterprise/observer/pkg/database"
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
	// Optional: connect DB if DATABASE_URL is provided.
	db, err := database.ConnectFromEnv(logger)
	if err != nil {
		logger.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	if db != nil {
		logger.Info("database connected")
		if err := database.AutoMigrateSchema(db, logger); err != nil {
			logger.Error("automigrate failed", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Info("DATABASE_URL not set; running without DB")
	}
	server.RegisterServices(grpcServer, logger, db)
	logger.Info("server starting", "addr", lis.Addr().String())

	// Run server in separate goroutine.
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("serve failed", "error", err)
		}
	}()

	// Graceful shutdown handling.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("shutdown signal received")

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
		logger.Info("server stopped gracefully")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }
	return def
}
