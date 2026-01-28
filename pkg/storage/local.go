package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LocalDriver implements storage.Driver for local filesystem storage
type LocalDriver struct {
	basePath string
	baseURL  string
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewLocalDriver creates a new local filesystem storage driver
func NewLocalDriver(cfg Config, logger *slog.Logger) (*LocalDriver, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	if cfg.LocalBasePath == "" {
		return nil, fmt.Errorf("local base path is required")
	}

	if cfg.LocalBaseURL == "" {
		return nil, fmt.Errorf("local base URL is required")
	}

	// Ensure base path exists
	if err := os.MkdirAll(cfg.LocalBasePath, 0755); err != nil {
		return nil, fmt.Errorf("create base path: %w", err)
	}

	logger.Info("initialized local storage driver",
		"base_path", cfg.LocalBasePath,
		"base_url", cfg.LocalBaseURL)

	return &LocalDriver{
		basePath: cfg.LocalBasePath,
		baseURL:  strings.TrimRight(cfg.LocalBaseURL, "/"),
		logger:   logger,
	}, nil
}

// Upload stores an attachment to the local filesystem
func (d *LocalDriver) Upload(ctx context.Context, name, mimeType string, content io.Reader) (*AttachmentMetadata, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Generate unique ID
	id := uuid.New().String()

	// Extract file extension from name
	ext := filepath.Ext(name)
	if ext == "" {
		ext = inferExtensionFromMimeType(mimeType)
	}

	// Storage key format: {id}{ext}
	storageKey := id + ext

	// Full file path
	filePath := filepath.Join(d.basePath, storageKey)

	// Create temporary file for atomic write
	tmpPath := filePath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	// Copy content to temp file
	size, err := io.Copy(tmpFile, content)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("write content: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("rename file: %w", err)
	}

	uploadedAt := time.Now()
	storageURI := fmt.Sprintf("%s/%s", d.baseURL, storageKey)

	d.logger.Info("attachment uploaded",
		"id", id,
		"name", name,
		"size", size,
		"storage_key", storageKey)

	return &AttachmentMetadata{
		ID:         id,
		Name:       name,
		MimeType:   mimeType,
		Size:       size,
		StorageKey: storageKey,
		StorageURI: storageURI,
		UploadedAt: uploadedAt,
	}, nil
}

// Download retrieves an attachment from the local filesystem
func (d *LocalDriver) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	filePath := filepath.Join(d.basePath, storageKey)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("attachment not found: %s", storageKey)
		}
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}

// GetMetadata retrieves metadata for an attachment
func (d *LocalDriver) GetMetadata(ctx context.Context, storageKey string) (*AttachmentMetadata, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	filePath := filepath.Join(d.basePath, storageKey)

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("attachment not found: %s", storageKey)
		}
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Extract ID from storage key (filename without extension)
	id := strings.TrimSuffix(storageKey, filepath.Ext(storageKey))
	storageURI := fmt.Sprintf("%s/%s", d.baseURL, storageKey)

	return &AttachmentMetadata{
		ID:         id,
		Name:       filepath.Base(storageKey),
		MimeType:   inferMimeTypeFromExtension(filepath.Ext(storageKey)),
		Size:       info.Size(),
		StorageKey: storageKey,
		StorageURI: storageURI,
		UploadedAt: info.ModTime(),
	}, nil
}

// Delete removes an attachment from the local filesystem
func (d *LocalDriver) Delete(ctx context.Context, storageKey string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	filePath := filepath.Join(d.basePath, storageKey)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("attachment not found: %s", storageKey)
		}
		return fmt.Errorf("remove file: %w", err)
	}

	d.logger.Info("attachment deleted", "storage_key", storageKey)

	return nil
}

// GetSignedURL is not supported for local storage (returns the public URL)
func (d *LocalDriver) GetSignedURL(ctx context.Context, storageKey string, expiration time.Duration) (string, error) {
	return fmt.Sprintf("%s/%s", d.baseURL, storageKey), nil
}

// Name returns the driver name
func (d *LocalDriver) Name() string {
	return "local"
}

// Close cleans up resources (no-op for local driver)
func (d *LocalDriver) Close() error {
	return nil
}

// inferExtensionFromMimeType returns a file extension based on MIME type
func inferExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	case "application/json":
		return ".json"
	case "application/pdf":
		return ".pdf"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	default:
		return ""
	}
}

// inferMimeTypeFromExtension returns a MIME type based on file extension
func inferMimeTypeFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".txt":
		return "text/plain"
	case ".html":
		return "text/html"
	case ".json":
		return "application/json"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}
