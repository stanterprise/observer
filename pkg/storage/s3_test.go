package storage

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestS3Driver_Integration tests S3 driver with MinIO testcontainer
func TestS3Driver_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Start MinIO container
	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "minio/minio:latest",
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     "minioadmin",
				"MINIO_ROOT_PASSWORD": "minioadmin",
			},
			Cmd:        []string{"server", "/data"},
			WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("failed to start MinIO container: %v", err)
	}
	defer minioContainer.Terminate(ctx)

	// Get MinIO endpoint
	endpoint, err := minioContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("failed to get MinIO endpoint: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Configure S3 driver with MinIO
	cfg := Config{
		Driver:            "s3",
		S3Endpoint:        "http://" + endpoint,
		S3Region:          "us-east-1",
		S3Bucket:          "test-bucket",
		S3AccessKeyID:     "minioadmin",
		S3SecretAccessKey: "minioadmin",
		S3UsePathStyle:    true,
	}

	driver, err := NewS3Driver(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create S3 driver: %v", err)
	}
	defer driver.Close()

	// Create bucket
	// Note: In real scenario, bucket should exist. For test, we create it manually
	if err := createMinIOBucket(ctx, driver, cfg.S3Bucket); err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}

	// Test Upload
	t.Run("upload", func(t *testing.T) {
		testContent := []byte("test content for S3")
		reader := bytes.NewReader(testContent)

		metadata, err := driver.Upload(ctx, "test.txt", "text/plain", reader)
		if err != nil {
			t.Fatalf("Upload() error = %v", err)
		}

		if metadata.Name != "test.txt" {
			t.Errorf("Name = %v, want test.txt", metadata.Name)
		}
		if metadata.Size != int64(len(testContent)) {
			t.Errorf("Size = %v, want %v", metadata.Size, len(testContent))
		}
		if metadata.StorageKey == "" {
			t.Error("StorageKey is empty")
		}
		if !strings.HasPrefix(metadata.StorageKey, "attachments/") {
			t.Errorf("StorageKey should start with 'attachments/', got %v", metadata.StorageKey)
		}

		// Test Download
		t.Run("download", func(t *testing.T) {
			downloadReader, err := driver.Download(ctx, metadata.StorageKey)
			if err != nil {
				t.Fatalf("Download() error = %v", err)
			}
			defer downloadReader.Close()

			downloadedContent, err := io.ReadAll(downloadReader)
			if err != nil {
				t.Fatalf("failed to read downloaded content: %v", err)
			}

			if !bytes.Equal(downloadedContent, testContent) {
				t.Error("downloaded content does not match original")
			}
		})

		// Test GetMetadata
		t.Run("get metadata", func(t *testing.T) {
			meta, err := driver.GetMetadata(ctx, metadata.StorageKey)
			if err != nil {
				t.Fatalf("GetMetadata() error = %v", err)
			}

			if meta.StorageKey != metadata.StorageKey {
				t.Errorf("StorageKey = %v, want %v", meta.StorageKey, metadata.StorageKey)
			}
			if meta.Size != int64(len(testContent)) {
				t.Errorf("Size = %v, want %v", meta.Size, len(testContent))
			}
		})

		// Test GetSignedURL
		t.Run("get signed URL", func(t *testing.T) {
			url, err := driver.GetSignedURL(ctx, metadata.StorageKey, 15*time.Minute)
			if err != nil {
				t.Fatalf("GetSignedURL() error = %v", err)
			}

			if url == "" {
				t.Error("signed URL is empty")
			}
			// URL should contain the storage key or be a valid presigned URL
			if !strings.Contains(url, "s3://") && !strings.Contains(url, endpoint) {
				t.Errorf("unexpected URL format: %v", url)
			}
		})

		// Test Delete
		t.Run("delete", func(t *testing.T) {
			err := driver.Delete(ctx, metadata.StorageKey)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}

			// Verify file is deleted
			_, err = driver.Download(ctx, metadata.StorageKey)
			if err == nil {
				t.Error("expected error when downloading deleted file")
			}
		})
	})
}

// createMinIOBucket is a helper to create a bucket in MinIO for testing
func createMinIOBucket(ctx context.Context, driver *S3Driver, bucketName string) error {
	// Use the S3 client to create the bucket
	_, err := driver.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucketName,
	})
	return err
}
