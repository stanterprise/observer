package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// noopWriter implements io.Writer but drops logs when no logger provided.
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// AttachmentMetadata contains metadata about a stored attachment
type AttachmentMetadata struct {
	ID         string    // Unique identifier (UUID)
	Name       string    // Original filename
	MimeType   string    // MIME type
	Size       int64     // Size in bytes
	StorageKey string    // Storage backend key
	StorageURI string    // Full URI for retrieval
	UploadedAt time.Time // Upload timestamp
}

// Driver defines the interface for attachment storage backends
type Driver interface {
	// Upload stores an attachment and returns metadata
	Upload(ctx context.Context, name, mimeType string, content io.Reader) (*AttachmentMetadata, error)

	// Download retrieves an attachment by storage key
	Download(ctx context.Context, storageKey string) (io.ReadCloser, error)

	// GetMetadata retrieves metadata for an attachment
	GetMetadata(ctx context.Context, storageKey string) (*AttachmentMetadata, error)

	// Delete removes an attachment (for cleanup)
	Delete(ctx context.Context, storageKey string) error

	// GetSignedURL generates a time-limited direct access URL (for cloud storage)
	GetSignedURL(ctx context.Context, storageKey string, expiration time.Duration) (string, error)

	// Name returns the driver name (e.g., "local", "s3")
	Name() string

	// Close cleans up resources
	Close() error
}

// Config holds storage driver configuration
type Config struct {
	Driver string // "local", "s3", "gcs", "minio"

	// Local filesystem config
	LocalBasePath string // e.g., "/data/artifacts"
	LocalBaseURL  string // e.g., "http://localhost:8080/attachments"

	// S3-compatible config
	S3Endpoint        string // AWS endpoint or MinIO URL
	S3Region          string // AWS region
	S3Bucket          string // Bucket name
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UsePathStyle    bool   // For MinIO compatibility
	S3PublicURL       string // Base URL for public access (optional)
}

// NewDriver creates a storage driver based on configuration
func NewDriver(cfg Config, logger *slog.Logger) (Driver, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	switch cfg.Driver {
	case "local":
		return NewLocalDriver(cfg, logger)
	case "s3", "minio":
		return NewS3Driver(cfg, logger)
	case "":
		return nil, fmt.Errorf("storage driver not specified")
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", cfg.Driver)
	}
}

// NewDriverFromEnv creates a driver from environment variables
func NewDriverFromEnv(logger *slog.Logger) (Driver, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	cfg := Config{
		Driver: os.Getenv("STORAGE_DRIVER"),

		// Local config
		LocalBasePath: os.Getenv("STORAGE_LOCAL_BASE_PATH"),
		LocalBaseURL:  os.Getenv("STORAGE_LOCAL_BASE_URL"),

		// S3 config
		S3Endpoint:        os.Getenv("STORAGE_S3_ENDPOINT"),
		S3Region:          os.Getenv("STORAGE_S3_REGION"),
		S3Bucket:          os.Getenv("STORAGE_S3_BUCKET"),
		S3AccessKeyID:     os.Getenv("STORAGE_S3_ACCESS_KEY_ID"),
		S3SecretAccessKey: os.Getenv("STORAGE_S3_SECRET_ACCESS_KEY"),
		S3UsePathStyle:    os.Getenv("STORAGE_S3_USE_PATH_STYLE") == "true",
		S3PublicURL:       os.Getenv("STORAGE_S3_PUBLIC_URL"),
	}

	if cfg.Driver == "" {
		logger.Info("STORAGE_DRIVER not set; storage driver not configured")
		return nil, nil
	}

	return NewDriver(cfg, logger)
}
