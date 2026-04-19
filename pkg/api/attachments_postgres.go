package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	pgRepo "github.com/stanterprise/observer/internal/repository/postgres"
	"github.com/stanterprise/observer/pkg/storage"
)

type PostgresAttachmentHandler struct {
	repo          *pgRepo.PostgresRepository
	storageDriver storage.Driver
	logger        *slog.Logger
}

func NewPostgresAttachmentHandler(repo *pgRepo.PostgresRepository, storageDriver storage.Driver, logger *slog.Logger) *PostgresAttachmentHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostgresAttachmentHandler{repo: repo, storageDriver: storageDriver, logger: logger}
}

func (h *PostgresAttachmentHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/attachments/*", h.handleAttachment)
}

func (h *PostgresAttachmentHandler) handleAttachment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	storageKey := routeParamOrPath(r, "*", "/api/attachments/", "")
	if storageKey == "" {
		http.Error(w, "Storage key is required", http.StatusBadRequest)
		return
	}

	attachment, err := h.repo.FindAttachmentByStorageKey(r.Context(), storageKey)
	if err != nil {
		h.logger.Error("failed to find postgres attachment", "storage_key", storageKey, "error", err)
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}
	if attachment == nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	storageType, _ := attachment["storage"].(string)
	switch storageType {
	case "inline":
		handleInlineAttachment(w, h.logger, attachment)
	case "local", "s3":
		if h.storageDriver == nil {
			http.Error(w, "Storage driver not configured", http.StatusInternalServerError)
			return
		}
		if storageType == "s3" {
			if signedURL, err := h.storageDriver.GetSignedURL(r.Context(), storageKey, 15*time.Minute); err == nil {
				http.Redirect(w, r, signedURL, http.StatusTemporaryRedirect)
				return
			}
		}
		handleProxyAttachment(w, r, h.storageDriver, h.logger, storageKey, attachment)
	case "external":
		if uri, ok := attachment["uri"].(string); ok && uri != "" {
			http.Redirect(w, r, uri, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "External URI not found", http.StatusInternalServerError)
	default:
		http.Error(w, fmt.Sprintf("Unknown storage type: %s", storageType), http.StatusInternalServerError)
	}
}
