package attachments

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/pkg/importer"
	"github.com/stanterprise/observer/pkg/storage"
)

type ImportService struct {
	driver          storage.Driver
	inlineThreshold int64
}

func NewImportService(driver storage.Driver, inlineThreshold int64) *ImportService {
	return &ImportService{driver: driver, inlineThreshold: inlineThreshold}
}

// Materialize stores normalized attachment sources and annotates their attempts.
// It returns external storage keys so a caller can compensate if persistence fails.
func (s *ImportService) Materialize(ctx context.Context, bundle *importer.RunImportBundle) ([]string, error) {
	attempts := make(map[string]*m.TestAttempt, len(bundle.Attempts))
	for _, attempt := range bundle.Attempts {
		attempts[attempt.ID] = attempt
	}
	uploaded := make([]string, 0)
	for index, source := range bundle.Attachments {
		attempt := attempts[source.AttemptID]
		if attempt == nil {
			return uploaded, fmt.Errorf("attachment references unknown attempt %q", source.AttemptID)
		}
		entry := map[string]interface{}{
			"id":   hashID(bundle.Run.ID, source.AttemptID, fmt.Sprintf("%d", index), source.Name),
			"name": source.Name, "mime_type": source.MimeType, "size": source.Size,
			"available": source.Available,
		}
		if source.StepID != "" {
			entry["step_id"] = source.StepID
		}
		if !source.Available || source.Open == nil {
			entry["storage"] = "missing"
			attempt.Attachments = append(attempt.Attachments, entry)
			continue
		}

		reader, err := source.Open(ctx)
		if err != nil {
			return uploaded, fmt.Errorf("open attachment %q: %w", source.Name, err)
		}
		if source.Size <= s.inlineThreshold {
			content, readErr := io.ReadAll(io.LimitReader(reader, s.inlineThreshold+1))
			closeErr := reader.Close()
			if readErr != nil {
				return uploaded, fmt.Errorf("read attachment %q: %w", source.Name, readErr)
			}
			if closeErr != nil {
				return uploaded, fmt.Errorf("close attachment %q: %w", source.Name, closeErr)
			}
			if int64(len(content)) > s.inlineThreshold {
				return uploaded, fmt.Errorf("attachment %q exceeded its declared inline size", source.Name)
			}
			digest := sha256.Sum256(content)
			entry["storage"] = "inline"
			entry["content"] = base64.StdEncoding.EncodeToString(content)
			entry["content_encoding"] = "base64"
			entry["checksum"] = hex.EncodeToString(digest[:])
			entry["size"] = len(content)
			attempt.Attachments = append(attempt.Attachments, entry)
			continue
		}
		if s.driver == nil {
			_ = reader.Close()
			return uploaded, fmt.Errorf("attachment %q requires external storage, but no storage driver is configured", source.Name)
		}
		hasher := sha256.New()
		metadata, uploadErr := s.driver.Upload(ctx, filepath.Base(source.Name), source.MimeType, io.TeeReader(reader, hasher))
		closeErr := reader.Close()
		if uploadErr != nil {
			return uploaded, fmt.Errorf("upload attachment %q: %w", source.Name, uploadErr)
		}
		if closeErr != nil {
			_ = s.driver.Delete(ctx, metadata.StorageKey)
			return uploaded, fmt.Errorf("close attachment %q: %w", source.Name, closeErr)
		}
		uploaded = append(uploaded, metadata.StorageKey)
		checksum := hex.EncodeToString(hasher.Sum(nil))
		entry["storage"] = s.driver.Name()
		entry["storage_key"] = metadata.StorageKey
		entry["uri"] = metadata.StorageURI
		entry["checksum"] = checksum
		entry["size"] = metadata.Size
		attempt.Attachments = append(attempt.Attachments, entry)
		stepID, _ := entry["step_id"].(string)
		var stepIDPtr *string
		if stepID != "" {
			stepIDPtr = &stepID
		}
		bundle.AttachmentRows = append(bundle.AttachmentRows, &m.Attachment{
			ID: entry["id"].(string), RunID: bundle.Run.ID, TestID: attempt.TestID, TestAttemptID: attempt.ID,
			StepID: stepIDPtr, Kind: "test", Name: source.Name, ContentType: source.MimeType,
			SizeBytes: metadata.Size, StorageKey: metadata.StorageKey, Checksum: checksum,
		})
	}
	return uploaded, nil
}

func (s *ImportService) Cleanup(ctx context.Context, keys []string) {
	if s == nil || s.driver == nil {
		return
	}
	for _, key := range keys {
		_ = s.driver.Delete(ctx, key)
	}
}

func hashID(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = io.WriteString(h, part)
		_, _ = io.WriteString(h, "\x00")
	}
	return "att-" + hex.EncodeToString(h.Sum(nil))[:32]
}
