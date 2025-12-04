package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/stanterprise/observer/internal/models"
	"gorm.io/gorm"
)

// Handler provides REST API endpoints for the Observer service
type Handler struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewHandler creates a new REST API handler
func NewHandler(db *gorm.DB, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		db:     db,
		logger: logger,
	}
}

// RegisterRoutes registers all REST API routes on the provided mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/tests", h.handleTests)
	mux.HandleFunc("/api/tests/", h.handleTestDetail)
	mux.HandleFunc("/api/runs", h.handleRuns)
	mux.HandleFunc("/api/runs/", h.handleRunDetail)
}

// DB returns the underlying GORM database connection
func (h *Handler) DB() *gorm.DB {
	return h.db
}

// handleTests handles GET /api/tests - list all test cases with optional filtering
func (h *Handler) handleTests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := h.db.Model(&models.TestCaseRun{})

	// Apply filters from query parameters
	if runID := r.URL.Query().Get("runId"); runID != "" {
		query = query.Where("run_id = ?", runID)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if search := r.URL.Query().Get("search"); search != "" {
		query = query.Where("LOWER(title) LIKE LOWER(?)", "%"+search+"%")
	}

	// Pagination
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.logger.Error("failed to count tests", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Fetch test cases
	var tests []models.TestCaseRun
	if err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&tests).Error; err != nil {
		h.logger.Error("failed to fetch tests", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": tests,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTestDetail handles GET /api/tests/{id} - get a specific test case with steps
func (h *Handler) handleTestDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/tests/")
	if id == "" {
		http.Error(w, "Test ID required", http.StatusBadRequest)
		return
	}

	// Fetch test case
	var test models.TestCaseRun
	if err := h.db.Where("id = ?", id).First(&test).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Test not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to fetch test", "id", id, "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Fetch associated steps
	var steps []models.StepRun
	if err := h.db.Where("test_case_run_id = ?", id).Order("created_at ASC").Find(&steps).Error; err != nil {
		h.logger.Error("failed to fetch steps", "test_id", id, "error", err)
		// Continue even if steps fail - just return empty array
	}

	response := map[string]interface{}{
		"test":  test,
		"steps": steps,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRuns handles GET /api/runs - list all unique run IDs
func (h *Handler) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	var runIDs []string
	if err := h.db.Model(&models.TestCaseRun{}).
		Select("DISTINCT run_id").
		Limit(limit).Offset(offset).
		Order("run_id DESC").
		Pluck("run_id", &runIDs).Error; err != nil {
		h.logger.Error("failed to fetch run IDs", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"runs": runIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRunDetail handles GET /api/runs/{runId} - get statistics and tests for a specific run
func (h *Handler) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract run ID from URL path
	runID := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	if runID == "" {
		http.Error(w, "Run ID required", http.StatusBadRequest)
		return
	}

	// Fetch tests for this run
	var tests []models.TestCaseRun
	if err := h.db.Where("run_id = ?", runID).Order("created_at ASC").Find(&tests).Error; err != nil {
		h.logger.Error("failed to fetch tests for run", "run_id", runID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(tests) == 0 {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	// Calculate statistics
	stats := map[string]int{
		"total":   len(tests),
		"passed":  0,
		"failed":  0,
		"skipped": 0,
	}

	for _, test := range tests {
		switch test.Status {
		case "PASSED":
			stats["passed"]++
		case "FAILED":
			stats["failed"]++
		case "SKIPPED":
			stats["skipped"]++
		}
	}

	// Get step count
	var stepCount int64
	h.db.Model(&models.StepRun{}).Where("run_id = ?", runID).Count(&stepCount)

	response := map[string]interface{}{
		"runId":      runID,
		"tests":      tests,
		"statistics": stats,
		"totalSteps": stepCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
