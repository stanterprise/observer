package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	mux.HandleFunc("/api/runs", h.handleRuns)
	mux.HandleFunc("/api/runs/stats", h.handleRunsStats)
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

// handleTestDetailByRunAndTest handles GET /api/runs/{runId}/tests/{testId} - get a specific test case with steps
func (h *MongoHandler) handleTestDetailByRunAndTest(w http.ResponseWriter, r *http.Request, path string) {
	// path is already trimmed of "/api/runs/" prefix
	// Extract runId and testId from path: {runId}/tests/{testId}
	parts := strings.Split(path, "/tests/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "Run ID and Test ID required", http.StatusBadRequest)
		return
	}

	runID := parts[0]
	testID := parts[1]

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

	// Search for the test in root tests and nested suites
	var foundTests []*m.TestDocument = make([]*m.TestDocument, 0)
	for _, test := range doc.Tests {
		if test.ID == testID {
			foundTests = append(foundTests, test)
			break
		}
	}

	if len(foundTests) == 0 {
		for _, suite := range doc.Suites {
			for _, test := range suite.Tests {
				if test.ID == testID {
					foundTests = append(foundTests, test)
					break
				}
			}
			if len(foundTests) > 0 {
				break
			}
		}
	}

	if len(foundTests) == 0 {
		http.Error(w, "Test not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"runId": runID,
		"tests": foundTests,
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

	// Extract run Data
	runData := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		var totalTests int = 0
		for _, suite := range doc.Suites {
			totalTests += len(suite.TestCaseIds)
		}
		runData = append(runData, map[string]interface{}{
			"id":         doc.ID,
			"name":       doc.Name,
			"updatedAt":  doc.UpdatedAt,
			"totalTests": totalTests,
			"status":     doc.Status,
			"metadata":   doc.Metadata,
			"statistics": map[string]interface{}{
				"total":       totalTests,
				"passed":      len(FilterTestsByStatus(doc.Tests, "PASSED")),
				"failed":      len(FilterTestsByStatus(doc.Tests, "FAILED")),
				"skipped":     len(FilterTestsByStatus(doc.Tests, "SKIPPED")),
				"running":     len(FilterTestsByStatus(doc.Tests, "RUNNING")),
				"broken":      len(FilterTestsByStatus(doc.Tests, "BROKEN")),
				"timedout":    len(FilterTestsByStatus(doc.Tests, "TIMEDOUT")),
				"interrupted": len(FilterTestsByStatus(doc.Tests, "INTERRUPTED")),
				"unknown":     len(FilterTestsByStatus(doc.Tests, "UNKNOWN")),
			},
		})
	}
	response := map[string]interface{}{
		"runs": runData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRunsStats handles GET /api/runs/stats - get statistics for all test runs in one request
func (h *MongoHandler) handleRunsStats(w http.ResponseWriter, r *http.Request) {
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

	// Calculate statistics for each run
	runStats := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		// Collect all tests (from root and nested suites)
		var allTests []*m.TestDocument
		allTests = append(allTests, doc.Tests...)
		for _, suite := range doc.Suites {
			allTests = append(allTests, suite.Tests...)
		}

		// Calculate statistics
		stats := map[string]int{
			"total":       len(allTests),
			"passed":      0,
			"failed":      0,
			"skipped":     0,
			"running":     0,
			"broken":      0,
			"timedout":    0,
			"interrupted": 0,
			"unknown":     0,
		}

		var lastUpdated time.Time
		for _, test := range allTests {
			switch test.Status {
			case "PASSED":
				stats["passed"]++
			case "FAILED":
				stats["failed"]++
			case "SKIPPED":
				stats["skipped"]++
			case "RUNNING":
				stats["running"]++
			case "BROKEN":
				stats["broken"]++
			case "TIMEDOUT":
				stats["timedout"]++
			case "INTERRUPTED":
				stats["interrupted"]++
			case "UNKNOWN":
				stats["unknown"]++
			case "":
				stats["running"]++
			default:
				stats["unknown"]++
			}

			// Track last updated time
			if !test.UpdatedAt.IsZero() && (lastUpdated.IsZero() || test.UpdatedAt.After(lastUpdated)) {
				lastUpdated = test.UpdatedAt
			}
		}

		runStat := map[string]interface{}{
			"runName":     doc.Name,
			"runId":       doc.ID,
			"total":       stats["total"],
			"passed":      stats["passed"],
			"failed":      stats["failed"],
			"skipped":     stats["skipped"],
			"running":     stats["running"],
			"broken":      stats["broken"],
			"timedout":    stats["timedout"],
			"interrupted": stats["interrupted"],
			"unknown":     stats["unknown"],
		}

		if !lastUpdated.IsZero() {
			runStat["lastUpdated"] = lastUpdated.Format(time.RFC3339)
		}

		runStats = append(runStats, runStat)
	}

	response := map[string]interface{}{
		"runs": runStats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRunDetail handles GET /api/runs/{runId} - get statistics and tests for a specific run
// and also handles GET /api/runs/{runId}/tests/{testId} - get specific test detail
func (h *MongoHandler) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract run ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/runs/")

	// Check if this is a test detail request: /api/runs/{runId}/tests/{testId}
	if strings.Contains(path, "/tests/") {
		h.handleTestDetailByRunAndTest(w, r, path)
		return
	}

	runID := path
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
		"total":       len(allTests),
		"passed":      0,
		"failed":      0,
		"skipped":     0,
		"running":     0,
		"broken":      0,
		"timedout":    0,
		"interrupted": 0,
		"unknown":     0,
	}

	totalSteps := 0
	for _, test := range allTests {
		switch test.Status {
		case "PASSED":
			stats["passed"]++
			if stats["running"] > 0 {
				stats["running"]--
			}
		case "FAILED":
			stats["failed"]++
			if stats["running"] > 0 {
				stats["running"]--
			}
		case "SKIPPED":
			stats["skipped"]++
		case "RUNNING":
			stats["running"]++
		case "BROKEN":
			stats["broken"]++
			if stats["running"] > 0 {
				stats["running"]--
			}
		case "TIMEDOUT":
			stats["timedout"]++
			if stats["running"] > 0 {
				stats["running"]--
			}
		case "INTERRUPTED":
			stats["interrupted"]++
			if stats["running"] > 0 {
				stats["running"]--
			}
		case "UNKNOWN":
			stats["unknown"]++
		case "":
			// Empty status - treat as running (test started but status not set)
			stats["running"]++
		default:
			// Unknown status value - count as unknown
			stats["unknown"]++
		}
		totalSteps += len(test.Steps)
	}

	// response := map[string]interface{}{
	// 	"runId":      runID,
	// 	"tests":      allTests,
	// 	"statistics": stats,
	// 	"totalSteps": totalSteps,
	// 	"document":   doc, // Include full document for advanced clients
	// }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}
