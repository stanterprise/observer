package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoHandler provides REST API endpoints for the Observer service using MongoDB
type MongoHandler struct {
	repo   *repository.MongoRepository
	logger *slog.Logger
}

// NewMongoHandler creates a new MongoDB REST API handler
func NewMongoHandler(repo *repository.MongoRepository, logger *slog.Logger) *MongoHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &MongoHandler{
		repo:   repo,
		logger: logger,
	}
}

// RegisterRoutes registers all REST API routes on the provided mux
func (h *MongoHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/tests", h.handleTests)
	mux.HandleFunc("/api/tests/", h.handleTestDetail)
	mux.HandleFunc("/api/runs", h.handleRuns)
	mux.HandleFunc("/api/runs/", h.handleRunDetail)
}

// handleTests handles GET /api/tests - list all test cases with optional filtering
func (h *MongoHandler) handleTests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build filter from query parameters
	filter := bson.M{}
	if runID := r.URL.Query().Get("runId"); runID != "" {
		filter["_id"] = runID
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}
	if projectName := r.URL.Query().Get("project"); projectName != "" {
		filter["project_name"] = projectName
	}

	// Pagination
	limit := int64(20)
	offset := int64(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.ParseInt(l, 10, 64); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.ParseInt(o, 10, 64); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Fetch test runs from MongoDB
	docs, total, err := h.repo.ListTestRuns(r.Context(), filter, limit, offset)
	if err != nil {
		h.logger.Error("failed to fetch test runs", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Flatten all tests from all runs for the response
	var tests []*m.TestDocument
	for _, doc := range docs {
		tests = append(tests, doc.Tests...)
		// Also include tests from nested suites
		for _, suite := range doc.Suites {
			tests = append(tests, suite.Tests...)
		}
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
func (h *MongoHandler) handleTestDetail(w http.ResponseWriter, r *http.Request) {
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

	// Fetch test from MongoDB
	test, err := h.repo.GetTestFromRun(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to fetch test", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if test == nil {
		http.Error(w, "Test not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"test":  test,
		"steps": test.Steps,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRuns handles GET /api/runs - list all test runs
func (h *MongoHandler) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := int64(50)
	offset := int64(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.ParseInt(l, 10, 64); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.ParseInt(o, 10, 64); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Fetch all test runs
	docs, _, err := h.repo.ListTestRuns(r.Context(), bson.M{}, limit, offset)
	if err != nil {
		h.logger.Error("failed to fetch test runs", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Extract run IDs
	runIDs := make([]string, 0, len(docs))
	for _, doc := range docs {
		runIDs = append(runIDs, doc.ID)
	}

	response := map[string]interface{}{
		"runs": runIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRunDetail handles GET /api/runs/{runId} - get statistics and tests for a specific run
func (h *MongoHandler) handleRunDetail(w http.ResponseWriter, r *http.Request) {
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

	// Fetch the test run document
	doc, err := h.repo.GetTestRun(r.Context(), runID)
	if err != nil {
		h.logger.Error("failed to fetch test run", "run_id", runID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if doc == nil {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	// Collect all tests (from root and nested suites)
	var allTests []*m.TestDocument
	allTests = append(allTests, doc.Tests...)
	for _, suite := range doc.Suites {
		allTests = append(allTests, suite.Tests...)
	}

	// Calculate statistics
	stats := map[string]int{
		"total":   len(allTests),
		"passed":  0,
		"failed":  0,
		"skipped": 0,
	}

	totalSteps := 0
	for _, test := range allTests {
		switch test.Status {
		case "PASSED":
			stats["passed"]++
		case "FAILED":
			stats["failed"]++
		case "SKIPPED":
			stats["skipped"]++
		}
		totalSteps += len(test.Steps)
	}

	response := map[string]interface{}{
		"runId":      runID,
		"tests":      allTests,
		"statistics": stats,
		"totalSteps": totalSteps,
		"document":   doc, // Include full document for advanced clients
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
