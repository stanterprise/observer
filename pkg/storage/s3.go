package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Driver implements storage.Driver for S3-compatible storage
type S3Driver struct {
	client    *s3.Client
	bucket    string
	publicURL string
	logger    *slog.Logger
}

// NewS3Driver creates a new S3 storage driver
func NewS3Driver(cfg Config, logger *slog.Logger) (*S3Driver, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}

	ctx := context.Background()

	var awsCfg aws.Config
	var err error

	// If endpoint is provided (MinIO or custom S3), use static credentials
	if cfg.S3Endpoint != "" {
		if cfg.S3AccessKeyID == "" || cfg.S3SecretAccessKey == "" {
			return nil, fmt.Errorf("S3 access key and secret key are required")
		}

		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.S3Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.S3AccessKeyID,
				cfg.S3SecretAccessKey,
				"",
			)),
		)
		if err != nil {
			return nil, fmt.Errorf("load AWS config: %w", err)
		}
	} else {
		// Use default AWS credential chain
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.S3Region),
		)
		if err != nil {
			return nil, fmt.Errorf("load AWS config: %w", err)
		}
	}

	// Create S3 client with custom options
	clientOpts := []func(*s3.Options){
		func(o *s3.Options) {
			if cfg.S3Endpoint != "" {
				o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			}
			if cfg.S3UsePathStyle {
				o.UsePathStyle = true
			}
		},
	}

	client := s3.NewFromConfig(awsCfg, clientOpts...)

	logger.Info("initialized S3 storage driver",
		"bucket", cfg.S3Bucket,
		"region", cfg.S3Region,
		"endpoint", cfg.S3Endpoint,
		"path_style", cfg.S3UsePathStyle)

	return &S3Driver{
		client:    client,
		bucket:    cfg.S3Bucket,
		publicURL: strings.TrimRight(cfg.S3PublicURL, "/"),
		logger:    logger,
	}, nil
}

// Upload stores an attachment to S3
func (d *S3Driver) Upload(ctx context.Context, name, mimeType string, content io.Reader) (*AttachmentMetadata, error) {
	// Generate unique ID
	id := uuid.New().String()

	// Extract file extension from name
	ext := path.Ext(name)
	if ext == "" {
		ext = inferExtensionFromMimeType(mimeType)
	}

	// Storage key format: attachments/{id}{ext}
	storageKey := fmt.Sprintf("attachments/%s%s", id, ext)

	// Wrap reader to count bytes as they're uploaded (stream directly without buffering)
	var size int64
	countingReader := &sizeCountingReader{
		reader: content,
		size:   &size,
	}

	// Upload to S3 - stream directly without buffering entire content in memory
	uploadedAt := time.Now()
	_, err := d.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(d.bucket),
		Key:         aws.String(storageKey),
		Body:        countingReader,
		ContentType: aws.String(mimeType),
		Metadata: map[string]string{
			"original-name": name,
			"uploaded-at":   uploadedAt.Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("S3 upload failed: %w", err)
	}

	// Generate storage URI
	var storageURI string
	if d.publicURL != "" {
		storageURI = fmt.Sprintf("%s/%s", d.publicURL, storageKey)
	} else {
		// Will use presigned URLs on download
		storageURI = fmt.Sprintf("s3://%s/%s", d.bucket, storageKey)
	}

	d.logger.Info("attachment uploaded to S3",
		"id", id,
		"name", name,
		"size", size,
		"storage_key", storageKey,
		"bucket", d.bucket)

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

// sizeCountingReader wraps an io.Reader and counts bytes read
type sizeCountingReader struct {
	reader io.Reader
	size   *int64
}

func (r *sizeCountingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	*r.size += int64(n)
	return n, err
}

// Download retrieves an attachment from S3
func (d *S3Driver) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	output, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return nil, fmt.Errorf("S3 download failed: %w", err)
	}

	return output.Body, nil
}

// GetMetadata retrieves metadata for an attachment from S3
func (d *S3Driver) GetMetadata(ctx context.Context, storageKey string) (*AttachmentMetadata, error) {
	output, err := d.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return nil, fmt.Errorf("S3 head object failed: %w", err)
	}

	// Extract ID from storage key
	parts := strings.Split(storageKey, "/")
	filename := parts[len(parts)-1]
	id := strings.TrimSuffix(filename, path.Ext(filename))

	// Get original name from metadata, fallback to filename
	name := filename
	if originalName, ok := output.Metadata["original-name"]; ok {
		name = originalName
	}

	// Get uploaded timestamp from metadata
	uploadedAt := time.Now()
	if uploadedAtStr, ok := output.Metadata["uploaded-at"]; ok {
		if t, err := time.Parse(time.RFC3339, uploadedAtStr); err == nil {
			uploadedAt = t
		}
	} else if output.LastModified != nil {
		uploadedAt = *output.LastModified
	}

	// Generate storage URI
	var storageURI string
	if d.publicURL != "" {
		storageURI = fmt.Sprintf("%s/%s", d.publicURL, storageKey)
	} else {
		storageURI = fmt.Sprintf("s3://%s/%s", d.bucket, storageKey)
	}

	mimeType := "application/octet-stream"
	if output.ContentType != nil {
		mimeType = *output.ContentType
	}

	return &AttachmentMetadata{
		ID:         id,
		Name:       name,
		MimeType:   mimeType,
		Size:       *output.ContentLength,
		StorageKey: storageKey,
		StorageURI: storageURI,
		UploadedAt: uploadedAt,
	}, nil
}

// Delete removes an attachment from S3
func (d *S3Driver) Delete(ctx context.Context, storageKey string) error {
	_, err := d.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return fmt.Errorf("S3 delete failed: %w", err)
	}

	d.logger.Info("attachment deleted from S3",
		"storage_key", storageKey,
		"bucket", d.bucket)

	return nil
}

// GetSignedURL generates a time-limited presigned URL for S3 object access
func (d *S3Driver) GetSignedURL(ctx context.Context, storageKey string, expiration time.Duration) (string, error) {
	// If public URL is configured, return it directly
	if d.publicURL != "" {
		return fmt.Sprintf("%s/%s", d.publicURL, storageKey), nil
	}

	// Generate presigned URL
	presignClient := s3.NewPresignClient(d.client)

	presignResult, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(storageKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("generate presigned URL: %w", err)
	}

	return presignResult.URL, nil
}

// Name returns the driver name
func (d *S3Driver) Name() string {
	return "s3"
}

// Close cleans up resources (no-op for S3 driver)
func (d *S3Driver) Close() error {
	return nil
}
