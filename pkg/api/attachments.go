package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/observer/pkg/storage"
)

// AttachmentHandler handles attachment retrieval endpoints
type AttachmentHandler struct {
	repo          *repository.MongoRepository
	storageDriver storage.Driver
	logger        *slog.Logger
}

// NewAttachmentHandler creates a new attachment handler
func NewAttachmentHandler(repo *repository.MongoRepository, storageDriver storage.Driver, logger *slog.Logger) *AttachmentHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AttachmentHandler{
		repo:          repo,
		storageDriver: storageDriver,
		logger:        logger,
	}
}

// RegisterRoutes registers attachment-related routes
func (h *AttachmentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/attachments/", h.handleAttachment)
}

// handleAttachment handles GET /api/attachments/{storageKey}
// This endpoint retrieves attachments by their storage key.
// It supports multiple storage backends:
// - inline: Returns content directly from MongoDB
// - local: Retrieves from local filesystem
// - s3: Retrieves from S3 or redirects to signed URL
// - external: Redirects to external URI
func (h *AttachmentHandler) handleAttachment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract storage key from path: /api/attachments/{storageKey}
	path := strings.TrimPrefix(r.URL.Path, "/api/attachments/")
	if path == "" {
		http.Error(w, "Storage key is required", http.StatusBadRequest)
		return
	}

	storageKey := path
	ctx := r.Context()

	// Find the attachment in MongoDB to get its storage metadata
	attachment, err := h.findAttachmentByStorageKey(ctx, storageKey)
	if err != nil {
		h.logger.Error("failed to find attachment", "storage_key", storageKey, "error", err)
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	// Handle different storage types
	storageType, _ := attachment["storage"].(string)

	switch storageType {
	case "inline":
		// Return inline content directly
		h.handleInlineAttachment(w, r, attachment)

	case "local", "s3":
		// Use storage driver to retrieve attachment
		if h.storageDriver == nil {
			http.Error(w, "Storage driver not configured", http.StatusInternalServerError)
			return
		}

		// For S3, prefer signed URL redirect
		if storageType == "s3" {
			if signedURL, err := h.storageDriver.GetSignedURL(ctx, storageKey, 15*time.Minute); err == nil {
				// Redirect to signed URL
				http.Redirect(w, r, signedURL, http.StatusTemporaryRedirect)
				return
			}
			// Fall through to proxy mode if signed URL fails
		}

		// Proxy the attachment through the API
		h.handleProxyAttachment(w, r, storageKey, attachment)

	case "external":
		// Redirect to external URI
		if uri, ok := attachment["uri"].(string); ok && uri != "" {
			http.Redirect(w, r, uri, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "External URI not found", http.StatusInternalServerError)

	default:
		http.Error(w, "Unknown storage type", http.StatusInternalServerError)
	}
}

// handleInlineAttachment serves inline attachment content
func (h *AttachmentHandler) handleInlineAttachment(w http.ResponseWriter, r *http.Request, attachment map[string]interface{}) {
	content, ok := attachment["content"].(string)
	if !ok || content == "" {
		http.Error(w, "Inline content not found", http.StatusInternalServerError)
		return
	}

	contentBytes := []byte(content)
	if encoding, ok := attachment["content_encoding"].(string); ok && encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			h.logger.Error("failed to decode base64 attachment", "error", err)
		} else {
			contentBytes = decoded
		}
	}

	// Set content type
	if mimeType, ok := attachment["mime_type"].(string); ok && mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Set content disposition with filename
	if name, ok := attachment["name"].(string); ok && name != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, name))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(contentBytes)
}

// handleProxyAttachment proxies attachment content through the API
func (h *AttachmentHandler) handleProxyAttachment(w http.ResponseWriter, r *http.Request, storageKey string, attachment map[string]interface{}) {
	ctx := r.Context()

	// Download from storage
	reader, err := h.storageDriver.Download(ctx, storageKey)
	if err != nil {
		h.logger.Error("failed to download attachment", "storage_key", storageKey, "error", err)
		http.Error(w, "Failed to retrieve attachment", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Set content type
	if mimeType, ok := attachment["mime_type"].(string); ok && mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Set content length if available
	if rawSize, ok := attachment["size"]; ok {
		var size int64
		switch v := rawSize.(type) {
		case int64:
			size = v
		case int32:
			size = int64(v)
		case int:
			size = int64(v)
		case float64:
			size = int64(v)
		}
		if size > 0 {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
		}
	}

	// Set content disposition with filename
	if name, ok := attachment["name"].(string); ok && name != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, name))
	}

	// Stream the content
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, reader); err != nil {
		h.logger.Error("failed to stream attachment", "storage_key", storageKey, "error", err)
	}
}

// findAttachmentByStorageKey searches for an attachment by its storage key in all test runs.
// This implementation scans test documents for the storage key using paginated queries.
// In production, you might want to maintain a separate index or collection for attachments.
func (h *AttachmentHandler) findAttachmentByStorageKey(ctx context.Context, storageKey string) (map[string]interface{}, error) {
	const pageSize int64 = 100
	var offset int64 = 0

	for {
		// Get a page of test runs and search through their attachments
		runs, _, err := h.repo.ListTestRuns(ctx, nil, pageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("list runs failed: %w", err)
		}

		// No more runs to search
		if len(runs) == 0 {
			break
		}

		// Search through runs for the attachment
		for _, run := range runs {
			if attachment := h.findAttachmentInTestRun(run, storageKey); attachment != nil {
				return attachment, nil
			}
		}

		offset += pageSize
	}

	return nil, fmt.Errorf("attachment not found")
}

// findAttachmentInTestRun searches for an attachment within a test run
func (h *AttachmentHandler) findAttachmentInTestRun(run interface{}, storageKey string) map[string]interface{} {
	// Convert run to document structure for search
	// ListTestRuns returns *models.TestRunDocument structs
	doc, ok := run.(*models.TestRunDocument)
	if !ok {
		return nil
	}
	return h.searchAttachmentsInTestRun(doc, storageKey)
}

// searchAttachmentsInTestRun searches for an attachment within a TestRunDocument
func (h *AttachmentHandler) searchAttachmentsInTestRun(doc *models.TestRunDocument, storageKey string) map[string]interface{} {
	// Search in top-level tests
	for _, test := range doc.Tests {
		if attachment := h.searchAttachmentsInTest(test, storageKey); attachment != nil {
			return attachment
		}
	}

	// Search in suites
	for _, suite := range doc.Suites {
		if attachment := h.searchAttachmentsInSuite(suite, storageKey); attachment != nil {
			return attachment
		}
	}

	return nil
}

// searchAttachmentsInSuite searches for an attachment within a SuiteDocument
func (h *AttachmentHandler) searchAttachmentsInSuite(suite *models.SuiteDocument, storageKey string) map[string]interface{} {
	// Search in suite's tests
	for _, test := range suite.Tests {
		if attachment := h.searchAttachmentsInTest(test, storageKey); attachment != nil {
			return attachment
		}
	}

	// Search in nested suites
	for _, nestedSuite := range suite.Suites {
		if attachment := h.searchAttachmentsInSuite(nestedSuite, storageKey); attachment != nil {
			return attachment
		}
	}

	return nil
}

// searchAttachmentsInTest searches for an attachment within a TestDocument
func (h *AttachmentHandler) searchAttachmentsInTest(test *models.TestDocument, storageKey string) map[string]interface{} {
	// Search in attempts
	for _, attempt := range test.Attempts {
		for _, att := range attempt.Attachments {
			if key, ok := att["storage_key"].(string); ok && key == storageKey {
				return att
			}
		}

		// Search in failures
		for _, failure := range attempt.Failures {
			for _, att := range failure.Attachments {
				if key, ok := att["storage_key"].(string); ok && key == storageKey {
					return att
				}
			}
		}

		// Search in errors
		for _, error := range attempt.Errors {
			for _, att := range error.Attachments {
				if key, ok := att["storage_key"].(string); ok && key == storageKey {
					return att
				}
			}
		}

		// Search in steps
		for _, step := range attempt.Steps {
			if attachment := h.searchAttachmentsInStep(step, storageKey); attachment != nil {
				return attachment
			}
		}
	}

	// Also search legacy attachments field for backward compatibility
	for _, att := range test.Attachments {
		if key, ok := att["storage_key"].(string); ok && key == storageKey {
			return att
		}
	}

	return nil
}

// searchAttachmentsInStep searches for an attachment within a StepDocument (recursive for nested steps)
func (h *AttachmentHandler) searchAttachmentsInStep(step *models.StepDocument, storageKey string) map[string]interface{} {
	// Steps don't have attachments directly, only tests do
	// Search in nested steps
	for _, nestedStep := range step.Steps {
		if attachment := h.searchAttachmentsInStep(nestedStep, storageKey); attachment != nil {
			return attachment
		}
	}

	return nil
}

// findAttachmentInRun recursively searches for an attachment by storage key within a run document
// Deprecated: Use searchAttachmentsInTestRun instead
func (h *AttachmentHandler) findAttachmentInRun(data interface{}, storageKey string) map[string]interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this is an attachment with matching storage key
		if key, ok := v["storage_key"].(string); ok && key == storageKey {
			return v
		}
		// Recursively search nested maps
		for _, value := range v {
			if result := h.findAttachmentInRun(value, storageKey); result != nil {
				return result
			}
		}
	case []interface{}:
		// Recursively search arrays
		for _, item := range v {
			if result := h.findAttachmentInRun(item, storageKey); result != nil {
				return result
			}
		}
	}
	return nil
}
