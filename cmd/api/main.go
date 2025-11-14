package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stanterprise/observer/internal/database"
	"github.com/stanterprise/observer/pkg/websocket"
)

func main() {
	var (
		port = flag.String("port", envOr("PORT", "8080"), "HTTP port to listen on")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// API service connects to database for read queries
	db, err := database.ConnectFromEnv(logger)
	if err != nil {
		logger.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	if db != nil {
		logger.Info("database connected")
	} else {
		logger.Info("DATABASE_URL not set; running without DB")
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub(logger)
	
	// Configure NATS for WebSocket if NATS_URL is provided
	natsURL := os.Getenv("NATS_URL")
	if natsURL != "" {
		wsConfig := websocket.NATSConfig{
			URL:          natsURL,
			StreamName:   envOr("NATS_STREAM", "tests_events"),
			ConsumerName: envOr("NATS_WS_CONSUMER", "websocket"),
			BatchSize:    10,
			MaxWait:      5 * time.Second,
		}
		
		if err := hub.InitNATS(wsConfig); err != nil {
			logger.Error("failed to initialize NATS for WebSocket", "error", err)
			os.Exit(1)
		}
	}
	
	// Start WebSocket hub in background
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()
	
	go hub.Run(hubCtx, websocket.NATSConfig{
		BatchSize: 10,
		MaxWait:   5 * time.Second,
	})

	// Create HTTP server with basic health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK\n")
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Observer API Service\n")
		fmt.Fprintf(w, "Available endpoints:\n")
		fmt.Fprintf(w, "  GET /health - Health check\n")
		fmt.Fprintf(w, "  GET /ws - WebSocket endpoint for real-time events\n")
		fmt.Fprintf(w, "  GET /api/graphql - GraphQL endpoint (coming soon)\n")
	})

	addr := ":" + *port
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	logger.Info("api server starting", "addr", addr)

	// Run server in separate goroutine and capture fatal serve errors.
	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	// Stop WebSocket hub
	hubCancel()

	// Allow up to 5s for graceful stop.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	logger.Info("api server stopped gracefully")

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
