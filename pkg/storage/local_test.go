package storage

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalDriver_Upload(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "observer-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		LocalBasePath: tmpDir,
		LocalBaseURL:  "http://localhost:8080/attachments",
	}

	driver, err := NewLocalDriver(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	tests := []struct {
		name     string
		filename string
		mimeType string
		content  []byte
		wantErr  bool
	}{
		{
			name:     "upload text file",
			filename: "test.txt",
			mimeType: "text/plain",
			content:  []byte("hello world"),
			wantErr:  false,
		},
		{
			name:     "upload image",
			filename: "screenshot.png",
			mimeType: "image/png",
			content:  []byte{0x89, 0x50, 0x4E, 0x47}, // PNG header
			wantErr:  false,
		},
		{
			name:     "upload without extension",
			filename: "noext",
			mimeType: "application/json",
			content:  []byte(`{"test": true}`),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			reader := bytes.NewReader(tt.content)

			metadata, err := driver.Upload(ctx, tt.filename, tt.mimeType, reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("Upload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Verify metadata
			if metadata.Name != tt.filename {
				t.Errorf("Name = %v, want %v", metadata.Name, tt.filename)
			}
			if metadata.MimeType != tt.mimeType {
				t.Errorf("MimeType = %v, want %v", metadata.MimeType, tt.mimeType)
			}
			if metadata.Size != int64(len(tt.content)) {
				t.Errorf("Size = %v, want %v", metadata.Size, len(tt.content))
			}
			if metadata.ID == "" {
				t.Error("ID is empty")
			}
			if metadata.StorageKey == "" {
				t.Error("StorageKey is empty")
			}
			if !strings.HasPrefix(metadata.StorageURI, cfg.LocalBaseURL) {
				t.Errorf("StorageURI = %v, should start with %v", metadata.StorageURI, cfg.LocalBaseURL)
			}

			// Verify file exists
			filePath := filepath.Join(tmpDir, metadata.StorageKey)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("uploaded file does not exist: %v", filePath)
			}

			// Verify content
			readContent, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("failed to read uploaded file: %v", err)
			}
			if !bytes.Equal(readContent, tt.content) {
				t.Error("uploaded content does not match")
			}
		})
	}
}

func TestLocalDriver_Download(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "observer-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		LocalBasePath: tmpDir,
		LocalBaseURL:  "http://localhost:8080/attachments",
	}

	driver, err := NewLocalDriver(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	ctx := context.Background()

	// Upload a file first
	testContent := []byte("test content for download")
	uploadReader := bytes.NewReader(testContent)
	metadata, err := driver.Upload(ctx, "test.txt", "text/plain", uploadReader)
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Test successful download
	t.Run("successful download", func(t *testing.T) {
		reader, err := driver.Download(ctx, metadata.StorageKey)
		if err != nil {
			t.Fatalf("Download() error = %v", err)
		}
		defer reader.Close()

		downloadedContent, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read downloaded content: %v", err)
		}

		if !bytes.Equal(downloadedContent, testContent) {
			t.Error("downloaded content does not match original")
		}
	})

	// Test download non-existent file
	t.Run("download non-existent file", func(t *testing.T) {
		_, err := driver.Download(ctx, "nonexistent.txt")
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})
}

func TestLocalDriver_GetMetadata(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "observer-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		LocalBasePath: tmpDir,
		LocalBaseURL:  "http://localhost:8080/attachments",
	}

	driver, err := NewLocalDriver(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	ctx := context.Background()

	// Upload a file first
	testContent := []byte("test content for metadata")
	uploadReader := bytes.NewReader(testContent)
	uploadMetadata, err := driver.Upload(ctx, "test.txt", "text/plain", uploadReader)
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Test getting metadata
	t.Run("get metadata", func(t *testing.T) {
		metadata, err := driver.GetMetadata(ctx, uploadMetadata.StorageKey)
		if err != nil {
			t.Fatalf("GetMetadata() error = %v", err)
		}

		if metadata.StorageKey != uploadMetadata.StorageKey {
			t.Errorf("StorageKey = %v, want %v", metadata.StorageKey, uploadMetadata.StorageKey)
		}
		if metadata.Size != int64(len(testContent)) {
			t.Errorf("Size = %v, want %v", metadata.Size, len(testContent))
		}
	})

	// Test metadata for non-existent file
	t.Run("metadata for non-existent file", func(t *testing.T) {
		_, err := driver.GetMetadata(ctx, "nonexistent.txt")
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})
}

func TestLocalDriver_Delete(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "observer-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		LocalBasePath: tmpDir,
		LocalBaseURL:  "http://localhost:8080/attachments",
	}

	driver, err := NewLocalDriver(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	ctx := context.Background()

	// Upload a file first
	testContent := []byte("test content for deletion")
	uploadReader := bytes.NewReader(testContent)
	metadata, err := driver.Upload(ctx, "test.txt", "text/plain", uploadReader)
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tmpDir, metadata.StorageKey)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("uploaded file does not exist before deletion")
	}

	// Test deletion
	t.Run("delete file", func(t *testing.T) {
		err := driver.Delete(ctx, metadata.StorageKey)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify file is deleted
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("file still exists after deletion")
		}
	})

	// Test deleting non-existent file
	t.Run("delete non-existent file", func(t *testing.T) {
		err := driver.Delete(ctx, "nonexistent.txt")
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})
}

func TestLocalDriver_GetSignedURL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "observer-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := Config{
		LocalBasePath: tmpDir,
		LocalBaseURL:  "http://localhost:8080/attachments",
	}

	driver, err := NewLocalDriver(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	ctx := context.Background()

	// For local driver, GetSignedURL should return the public URL
	storageKey := "test.txt"
	url, err := driver.GetSignedURL(ctx, storageKey, 0)
	if err != nil {
		t.Fatalf("GetSignedURL() error = %v", err)
	}

	expectedURL := "http://localhost:8080/attachments/test.txt"
	if url != expectedURL {
		t.Errorf("GetSignedURL() = %v, want %v", url, expectedURL)
	}
}

func TestNewDriverFromEnv(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with no STORAGE_DRIVER set
	t.Run("no driver configured", func(t *testing.T) {
		os.Unsetenv("STORAGE_DRIVER")

		driver, err := NewDriverFromEnv(logger)
		if err != nil {
			t.Errorf("NewDriverFromEnv() error = %v, want nil", err)
		}
		if driver != nil {
			t.Error("expected nil driver when STORAGE_DRIVER not set")
			driver.Close()
		}
	})

	// Test with local driver
	t.Run("local driver from env", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "observer-storage-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		os.Setenv("STORAGE_DRIVER", "local")
		os.Setenv("STORAGE_LOCAL_BASE_PATH", tmpDir)
		os.Setenv("STORAGE_LOCAL_BASE_URL", "http://localhost:8080/attachments")
		defer func() {
			os.Unsetenv("STORAGE_DRIVER")
			os.Unsetenv("STORAGE_LOCAL_BASE_PATH")
			os.Unsetenv("STORAGE_LOCAL_BASE_URL")
		}()

		driver, err := NewDriverFromEnv(logger)
		if err != nil {
			t.Fatalf("NewDriverFromEnv() error = %v", err)
		}
		if driver == nil {
			t.Fatal("expected driver, got nil")
		}
		defer driver.Close()

		if driver.Name() != "local" {
			t.Errorf("driver name = %v, want local", driver.Name())
		}
	})
}
