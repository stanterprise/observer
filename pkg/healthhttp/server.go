package healthhttp

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Server provides a simple HTTP health check endpoint for GCE load balancers.
// This runs alongside the gRPC server to provide HTTP-based health checks
// which are more reliable with GKE NEG configurations than HTTP/2 gRPC health checks.
type Server struct {
	server *http.Server
	logger *slog.Logger
}

// NewServer creates a new HTTP health check server.
// It listens on the specified port and responds to all requests with 200 OK.
func NewServer(port string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Also respond to root path for flexibility
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return &Server{
		server: &http.Server{
			Addr:         ":" + port,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

// Start starts the HTTP health server. Blocks until server stops.
func (s *Server) Start() error {
	s.logger.Info("HTTP health server starting", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("HTTP health server shutting down")
	return s.server.Shutdown(ctx)
}

// noopWriter implements io.Writer but drops logs when no logger provided.
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }
