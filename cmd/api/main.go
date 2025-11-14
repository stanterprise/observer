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

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/stanterprise/observer/internal/database"
	"github.com/stanterprise/observer/pkg/api/graph"
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
		logger.Warn("DATABASE_URL not set; GraphQL queries will fail without database")
	}

	// Create GraphQL resolver and handler
	resolver := graph.NewResolver(db, logger)
	gqlHandler := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	playgroundHandler := playground.Handler("GraphQL Playground", "/api/graphql")

	// Create HTTP server with endpoints
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK\n")
	})
	
	// Root information endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Observer API Service\n")
		fmt.Fprintf(w, "Available endpoints:\n")
		fmt.Fprintf(w, "  GET  /health          - Health check\n")
		fmt.Fprintf(w, "  POST /api/graphql     - GraphQL API endpoint\n")
		fmt.Fprintf(w, "  GET  /api/playground  - GraphQL Playground (interactive query tool)\n")
	})

	// GraphQL endpoints
	mux.Handle("/api/graphql", gqlHandler)
	mux.Handle("/api/playground", playgroundHandler)

	addr := ":" + *port
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	logger.Info("api server starting", "addr", addr, "graphql", "/api/graphql", "playground", "/api/playground")

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
