package storage_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stanterprise/observer/pkg/storage"
)

// TestStorageIntegration demonstrates the complete storage flow
func TestStorageIntegration(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "observer-storage-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test local driver
	t.Run("local driver complete workflow", func(t *testing.T) {
		cfg := storage.Config{
			Driver:        "local",
			LocalBasePath: tmpDir,
			LocalBaseURL:  "http://localhost:8080/api/attachments",
		}

		driver, err := storage.NewDriver(cfg, logger)
		if err != nil {
			t.Fatalf("failed to create driver: %v", err)
		}
		defer driver.Close()

		ctx := context.Background()

		// 1. Upload a small attachment (should be inline in production)
		smallContent := []byte("This is a small attachment")
		smallReader := bytes.NewReader(smallContent)
		smallMeta, err := driver.Upload(ctx, "small.txt", "text/plain", smallReader)
		if err != nil {
			t.Fatalf("failed to upload small attachment: %v", err)
		}

		t.Logf("Small attachment uploaded: ID=%s, Size=%d, URI=%s",
			smallMeta.ID, smallMeta.Size, smallMeta.StorageURI)

		// 2. Upload a large attachment (would use external storage in production)
		largeContent := make([]byte, 150*1024) // 150KB
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}
		largeReader := bytes.NewReader(largeContent)
		largeMeta, err := driver.Upload(ctx, "screenshot.png", "image/png", largeReader)
		if err != nil {
			t.Fatalf("failed to upload large attachment: %v", err)
		}

		t.Logf("Large attachment uploaded: ID=%s, Size=%d, URI=%s",
			largeMeta.ID, largeMeta.Size, largeMeta.StorageURI)

		// 3. Download the attachments
		t.Run("download small attachment", func(t *testing.T) {
			reader, err := driver.Download(ctx, smallMeta.StorageKey)
			if err != nil {
				t.Fatalf("failed to download: %v", err)
			}
			defer reader.Close()

			downloaded, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read content: %v", err)
			}

			if !bytes.Equal(downloaded, smallContent) {
				t.Error("downloaded content does not match original")
			}
		})

		t.Run("download large attachment", func(t *testing.T) {
			reader, err := driver.Download(ctx, largeMeta.StorageKey)
			if err != nil {
				t.Fatalf("failed to download: %v", err)
			}
			defer reader.Close()

			downloaded, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read content: %v", err)
			}

			if !bytes.Equal(downloaded, largeContent) {
				t.Error("downloaded content does not match original")
			}
		})

		// 4. Get metadata
		t.Run("get metadata", func(t *testing.T) {
			meta, err := driver.GetMetadata(ctx, largeMeta.StorageKey)
			if err != nil {
				t.Fatalf("failed to get metadata: %v", err)
			}

			if meta.Size != largeMeta.Size {
				t.Errorf("size mismatch: got %d, want %d", meta.Size, largeMeta.Size)
			}
		})

		// 5. Get signed URL (for local, returns public URL)
		t.Run("get signed URL", func(t *testing.T) {
			url, err := driver.GetSignedURL(ctx, largeMeta.StorageKey, 0)
			if err != nil {
				t.Fatalf("failed to get signed URL: %v", err)
			}

			if url == "" {
				t.Error("signed URL is empty")
			}
			t.Logf("Signed URL: %s", url)
		})

		// 6. Delete attachments
		t.Run("delete attachments", func(t *testing.T) {
			if err := driver.Delete(ctx, smallMeta.StorageKey); err != nil {
				t.Fatalf("failed to delete small attachment: %v", err)
			}

			if err := driver.Delete(ctx, largeMeta.StorageKey); err != nil {
				t.Fatalf("failed to delete large attachment: %v", err)
			}

			// Verify deletion
			_, err := driver.Download(ctx, smallMeta.StorageKey)
			if err == nil {
				t.Error("expected error when downloading deleted attachment")
			}
		})
	})
}

// TestDriverFactory tests the driver factory function
func TestDriverFactory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name      string
		cfg       storage.Config
		wantName  string
		wantError bool
	}{
		{
			name: "local driver",
			cfg: storage.Config{
				Driver:        "local",
				LocalBasePath: t.TempDir(),
				LocalBaseURL:  "http://localhost:8080/api/attachments",
			},
			wantName:  "local",
			wantError: false,
		},
		{
			name: "missing driver",
			cfg: storage.Config{
				Driver: "",
			},
			wantError: true,
		},
		{
			name: "unknown driver",
			cfg: storage.Config{
				Driver: "unknown",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, err := storage.NewDriver(tt.cfg, logger)
			if (err != nil) != tt.wantError {
				t.Errorf("NewDriver() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil {
				defer driver.Close()
				if driver.Name() != tt.wantName {
					t.Errorf("driver.Name() = %v, want %v", driver.Name(), tt.wantName)
				}
			}
		})
	}
}
