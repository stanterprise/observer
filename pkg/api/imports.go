package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	pgRepo "github.com/stanterprise/observer/internal/repository/postgres"
	attachmentimport "github.com/stanterprise/observer/pkg/attachments"
	"github.com/stanterprise/observer/pkg/importer"
)

type ImportHandler struct {
	registry    *importer.Registry
	repo        *pgRepo.PostgresRepository
	attachments *attachmentimport.ImportService
	logger      *slog.Logger
	tempDir     string
	semaphore   chan struct{}
}

func NewImportHandler(registry *importer.Registry, repo *pgRepo.PostgresRepository, attachments *attachmentimport.ImportService, logger *slog.Logger, tempDir string, concurrency int) *ImportHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if concurrency <= 0 {
		concurrency = 2
	}
	return &ImportHandler{registry: registry, repo: repo, attachments: attachments, logger: logger, tempDir: tempDir, semaphore: make(chan struct{}, concurrency)}
}

func (h *ImportHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/imports", h.handleCapabilities)
	r.Post("/api/v1/imports/{producer}/{format}/{reportVersion}", h.handleImport)
}

func (h *ImportHandler) handleCapabilities(w http.ResponseWriter, _ *http.Request) {
	writeImportJSON(w, http.StatusOK, map[string]interface{}{"data": h.registry.Capabilities()})
}

func (h *ImportHandler) handleImport(w http.ResponseWriter, r *http.Request) {
	reportType := importer.ReportType{Producer: chi.URLParam(r, "producer"), Format: chi.URLParam(r, "format"), Version: chi.URLParam(r, "reportVersion")}
	adapter, ok := h.registry.Lookup(reportType)
	if !ok {
		versions := h.registry.SupportedVersions(reportType.Producer, reportType.Format)
		if len(versions) > 0 {
			writeImportError(w, http.StatusUnprocessableEntity, &importer.Error{Code: "unsupported_report_version", Message: "the requested report version is not supported", Details: map[string]interface{}{"supportedVersions": versions}})
		} else {
			writeImportError(w, http.StatusNotFound, &importer.Error{Code: "unsupported_report_type", Message: "the requested report type is not supported"})
		}
		return
	}
	select {
	case h.semaphore <- struct{}{}:
		defer func() { <-h.semaphore }()
	default:
		writeImportError(w, http.StatusTooManyRequests, &importer.Error{Code: "import_capacity_exceeded", Message: "too many imports are currently running"})
		return
	}

	files, name, cleanup, err := h.receiveMultipart(w, r, adapter.Limits())
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		h.writeError(w, err)
		return
	}
	bundle, err := adapter.Parse(r.Context(), files)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if bundle.Cleanup != nil {
		defer bundle.Cleanup()
	}
	if name != "" {
		bundle.Run.Name = name
	}

	exists, existingName, err := h.repo.ImportedRunExists(r.Context(), bundle.Run.ID, bundle.SourceDigest)
	if err != nil {
		h.logger.Error("failed to check import idempotency", "error", err)
		writeImportError(w, http.StatusInternalServerError, &importer.Error{Code: "import_failed", Message: "failed to check existing imports"})
		return
	}
	if exists {
		bundle.Run.Name = existingName
		writeImportJSON(w, http.StatusOK, importResponse(bundle, false))
		return
	}

	uploaded, err := h.attachments.Materialize(r.Context(), bundle)
	if err != nil {
		h.cleanupUploads(uploaded)
		h.writeError(w, importer.NewError("attachment_storage_failed", err.Error(), err))
		return
	}
	outcome, err := h.repo.ImportRun(r.Context(), bundle)
	if err != nil {
		h.cleanupUploads(uploaded)
		h.logger.Error("failed to persist imported run", "run_id", bundle.Run.ID, "error", err)
		writeImportError(w, http.StatusInternalServerError, &importer.Error{Code: "import_failed", Message: "failed to persist imported test run"})
		return
	}
	if !outcome.Created {
		h.cleanupUploads(uploaded)
	}
	status := http.StatusCreated
	if !outcome.Created {
		status = http.StatusOK
	}
	writeImportJSON(w, status, importResponse(bundle, outcome.Created))
}

func (h *ImportHandler) receiveMultipart(w http.ResponseWriter, r *http.Request, limits importer.Limits) ([]importer.SourceFile, string, func(), error) {
	r.Body = http.MaxBytesReader(w, r.Body, limits.MaxRequestBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", nil, importer.NewError("invalid_multipart", "request must use multipart/form-data", err)
	}
	files := make([]importer.SourceFile, 0)
	paths := make([]string, 0)
	cleanup := func() {
		for _, path := range paths {
			_ = os.Remove(path)
		}
	}
	name := ""
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			cleanup()
			return nil, "", nil, multipartError(nextErr)
		}
		field, filename := part.FormName(), part.FileName()
		if filename == "" {
			if field == "name" {
				value, readErr := io.ReadAll(io.LimitReader(part, 257))
				if readErr != nil {
					_ = part.Close()
					cleanup()
					return nil, "", nil, multipartError(readErr)
				}
				if len(value) > 256 {
					_ = part.Close()
					cleanup()
					return nil, "", nil, importer.NewError("invalid_name", "import name must be at most 256 bytes", nil)
				}
				name = strings.TrimSpace(string(value))
			}
			_ = part.Close()
			continue
		}
		if field != "reports" && field != "reports[]" {
			_ = part.Close()
			cleanup()
			return nil, "", nil, importer.NewError("invalid_multipart", "file parts must use the reports field", nil)
		}
		if len(files) >= limits.MaxFiles {
			_ = part.Close()
			cleanup()
			return nil, "", nil, importer.NewError("upload_too_large", "too many report files", nil)
		}
		if !extensionAllowed(filename, limits.FileExtensions) {
			_ = part.Close()
			cleanup()
			return nil, "", nil, importer.NewError("invalid_file_type", "report file extension is not supported", nil)
		}
		tmp, createErr := os.CreateTemp(h.tempDir, "observer-import-*.zip")
		if createErr != nil {
			_ = part.Close()
			cleanup()
			return nil, "", nil, importer.NewError("temporary_storage_failed", "failed to create temporary upload", createErr)
		}
		paths = append(paths, tmp.Name())
		hasher := sha256.New()
		size, copyErr := io.Copy(io.MultiWriter(tmp, hasher), part)
		closeErr, partCloseErr := tmp.Close(), part.Close()
		if copyErr != nil {
			cleanup()
			return nil, "", nil, multipartError(copyErr)
		}
		if closeErr != nil || partCloseErr != nil {
			cleanup()
			return nil, "", nil, importer.NewError("temporary_storage_failed", "failed to finish temporary upload", errors.Join(closeErr, partCloseErr))
		}
		files = append(files, importer.SourceFile{Name: filepath.Base(filename), Path: tmp.Name(), Size: size, SHA256: hex.EncodeToString(hasher.Sum(nil))})
	}
	if len(files) == 0 {
		cleanup()
		return nil, "", nil, importer.NewError("missing_files", "at least one report file is required", nil)
	}
	return files, name, cleanup, nil
}

func (h *ImportHandler) cleanupUploads(keys []string) {
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	h.attachments.Cleanup(ctx, keys)
}

func (h *ImportHandler) writeError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeImportError(w, http.StatusRequestEntityTooLarge, &importer.Error{Code: "upload_too_large", Message: "upload exceeds the request size limit"})
		return
	}
	var typed *importer.Error
	if !errors.As(err, &typed) {
		h.logger.Error("test run import failed", "error", err)
		writeImportError(w, http.StatusInternalServerError, &importer.Error{Code: "import_failed", Message: "test run import failed"})
		return
	}
	status := http.StatusUnprocessableEntity
	switch typed.Code {
	case "upload_too_large":
		status = http.StatusRequestEntityTooLarge
	case "temporary_storage_failed", "attachment_storage_failed":
		status = http.StatusInternalServerError
	case "missing_files", "invalid_multipart", "invalid_name", "invalid_file_type":
		status = http.StatusBadRequest
	}
	writeImportError(w, status, typed)
}

func multipartError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return err
	}
	if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
		return &http.MaxBytesError{}
	}
	return importer.NewError("invalid_multipart", "failed to read multipart upload", err)
}

func extensionAllowed(name string, allowed []string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	for _, value := range allowed {
		if ext == strings.ToLower(value) {
			return true
		}
	}
	return false
}

func importResponse(bundle *importer.RunImportBundle, created bool) map[string]interface{} {
	return map[string]interface{}{"data": map[string]interface{}{
		"runId": bundle.Run.ID, "created": created, "status": bundle.Run.Status, "name": bundle.Run.Name,
		"summary":  map[string]interface{}{"executions": len(bundle.Executions), "suites": len(bundle.Suites), "tests": len(bundle.Tests), "attempts": len(bundle.Attempts), "attachments": len(bundle.Attachments)},
		"warnings": bundle.Warnings,
	}}
}

func writeImportError(w http.ResponseWriter, status int, err *importer.Error) {
	payload := map[string]interface{}{"code": err.Code, "message": err.Message}
	if err.Details != nil {
		payload["details"] = err.Details
	}
	writeImportJSON(w, status, map[string]interface{}{"error": payload})
}
func writeImportJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
