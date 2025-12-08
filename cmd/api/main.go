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
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/observer/pkg/api"
	"github.com/stanterprise/observer/pkg/websocket"
)

func main() {
	var (
		port = flag.String("port", envOr("PORT", "8080"), "HTTP port to listen on")
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
		logger.Error("MONGODB_URI or MONGO_URI not set; API requires MongoDB")
		os.Exit(1)
	}
	
	logger.Info("using MongoDB backend for API")
	
	// Ensure MongoDB connection cleanup
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		mongoDB.Close(ctx)
	}()
	
	// Create MongoDB repository and handler
	repo := repository.NewMongoRepository(mongoDB.TestRunsCollection(), logger)
	mongoHandler := api.NewMongoHandler(repo, logger)

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

	// Create HTTP server with endpoints
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK\n")
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Observer API Service\n")
		fmt.Fprintf(w, "Available endpoints:\n")
		fmt.Fprintf(w, "  GET /health - Health check\n")
		fmt.Fprintf(w, "  GET /ws - WebSocket endpoint for real-time events\n")
		fmt.Fprintf(w, "\nREST API:\n")
		fmt.Fprintf(w, "  GET  /api/tests           - List test cases (supports ?runId, ?status, ?search, ?limit, ?offset)\n")
		fmt.Fprintf(w, "  GET  /api/tests/{id}      - Get specific test case with steps\n")
		fmt.Fprintf(w, "  GET  /api/runs            - List all test runs\n")
		fmt.Fprintf(w, "  GET  /api/runs/{runId}    - Get run details with statistics\n")
	})

	// REST API endpoints
	mongoHandler.RegisterRoutes(mux)

	addr := ":" + *port

	// Wrap with CORS middleware for local development
	handler := corsMiddleware(mux, logger)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	logger.Info("api server starting",
		"addr", addr,
		"rest_api", "/api/tests, /api/runs")

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

// corsMiddleware adds CORS headers to support local web development
// where the web UI runs on a different port (e.g., Vite dev server on :3000)
func corsMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	allowedOrigins := envOr("CORS_ALLOWED_ORIGINS", "*")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Set CORS headers
		if allowedOrigins == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			// TODO: Check origin against allowedOrigins list
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
