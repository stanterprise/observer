package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	m "github.com/stanterprise/observer/internal/models"
	pgRepo "github.com/stanterprise/observer/internal/repository/postgres"
)

type PostgresHandler struct {
	repo        *pgRepo.PostgresRepository
	liveRunRepo liveRunRepository
	logger      *slog.Logger
}

type liveRunRepository interface {
	GetTestRun(ctx context.Context, id string) (*m.TestRunDocument, error)
}

func NewPostgresHandler(repo *pgRepo.PostgresRepository, logger *slog.Logger) *PostgresHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostgresHandler{repo: repo, logger: logger}
}

func (h *PostgresHandler) SetLiveRunRepo(r liveRunRepository) {
	h.liveRunRepo = r
}

func (h *PostgresHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/tests", h.handleTests)
	r.Get("/api/tests/{testId}/trends", h.handleTestTrends)
	r.Get("/api/runs", h.handleRuns)
	r.Get("/api/runs/stats", h.handleRunsStats)
	r.Get("/api/runs/{runId}", h.handleRunDetail)
	r.Get("/api/runs/{runId}/tests/{testId}", h.handleTestDetailByRunAndTest)
	r.Delete("/api/runs/delete", h.handleDeleteRuns)
	r.Put("/api/runs/marker", h.handleUpdateMarker)
	r.Get("/api/markers", h.handleMarkers)
	r.Get("/api/marker/{markerValue}/stats", h.handleMarkerStats)
}

func (h *PostgresHandler) handleTests(w http.ResponseWriter, r *http.Request) {
	filter := pgRepo.ListRunsFilter{
		RunID:       r.URL.Query().Get("runId"),
		Status:      r.URL.Query().Get("status"),
		ProjectName: r.URL.Query().Get("project"),
	}
	limit, offset := parseLimitOffset(r, 20)
	docs, total, err := h.repo.GetRunDocuments(r.Context(), filter, limit, offset)
	if err != nil {
		h.logger.Error("failed to fetch test runs from postgres", "error", err)
		h.internalError(w)
		return
	}

	tests := make([]*m.TestDocument, 0)
	for _, doc := range docs {
		tests = append(tests, doc.Tests...)
		for _, suite := range doc.Suites {
			collectSuiteTests(suite, &tests)
		}
	}

	h.writeJSON(w, map[string]interface{}{
		"data": tests,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (h *PostgresHandler) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	limit, offset := parseLimitOffset(r, 50)
	docs, _, err := h.repo.GetRunDocuments(r.Context(), pgRepo.ListRunsFilter{}, limit, offset)
	if err != nil {
		h.logger.Error("failed to fetch runs from postgres", "error", err)
		h.internalError(w)
		return
	}

	runs := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		allTests := flattenRunTests(doc)
		runs = append(runs, map[string]interface{}{
			"id":         doc.ID,
			"name":       doc.Name,
			"updatedAt":  doc.UpdatedAt,
			"totalTests": len(allTests),
			"status":     doc.Status,
			"metadata":   doc.Metadata,
			"statistics": buildTestStatistics(allTests),
		})
	}

	h.writeJSON(w, map[string]interface{}{"runs": runs})
}

func (h *PostgresHandler) handleRunsStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	limit, offset := parseLimitOffset(r, 50)
	docs, _, err := h.repo.GetRunDocuments(r.Context(), pgRepo.ListRunsFilter{}, limit, offset)
	if err != nil {
		h.logger.Error("failed to fetch run stats from postgres", "error", err)
		h.internalError(w)
		return
	}

	runStats := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		allTests := flattenRunTests(doc)
		stats := buildTestStatistics(allTests)
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
		lastUpdated := latestTestUpdate(allTests)
		if !lastUpdated.IsZero() {
			runStat["lastUpdated"] = lastUpdated.Format(time.RFC3339)
		}
		runStats = append(runStats, runStat)
	}

	h.writeJSON(w, map[string]interface{}{"runs": runStats})
}

func (h *PostgresHandler) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	runID := routeParamOrPath(r, "runId", "/api/runs/", "")
	if runID == "" {
		h.badRequest(w, "Run ID required")
		return
	}

	doc, err := h.repo.GetRunDocument(r.Context(), runID)
	if err != nil {
		h.logger.Error("failed to fetch run from postgres", "run_id", runID, "error", err)
		h.internalError(w)
		return
	}
	if doc == nil {
		h.notFound(w, "Run not found")
		return
	}

	h.writeJSON(w, doc)
}

func (h *PostgresHandler) handleTestDetailByRunAndTest(w http.ResponseWriter, r *http.Request) {
	runID, testID := runAndTestParams(r)
	if runID == "" || testID == "" {
		h.badRequest(w, "Run ID and Test ID required")
		return
	}

	doc, err := h.repo.GetRunDocument(r.Context(), runID)
	if err != nil {
		h.logger.Error("failed to fetch run from postgres", "run_id", runID, "error", err)
		h.internalError(w)
		return
	}
	if doc == nil {
		h.notFound(w, "Run not found")
		return
	}

	foundTests := findTestsInRun(doc, testID)
	if len(foundTests) == 0 {
		h.notFound(w, "Test not found")
		return
	}

	liveTests, err := h.loadLiveRunningTestDetails(r.Context(), runID, foundTests)
	if err != nil {
		h.logger.Error("failed to load live running test details", "run_id", runID, "test_id", testID, "error", err)
		h.internalError(w)
		return
	}
	if liveTests != nil {
		foundTests = liveTests
	}

	h.writeJSON(w, map[string]interface{}{
		"runId": runID,
		"tests": foundTests,
	})
}

func (h *PostgresHandler) handleTestTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	testID := routeParamOrPath(r, "testId", "/api/tests/", "/trends")
	if testID == "" {
		h.badRequest(w, "Test ID required")
		return
	}

	limit := int64(50)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	trends, err := h.repo.GetTestTrends(r.Context(), testID, limit)
	if err != nil {
		h.logger.Error("failed to fetch test trends from postgres", "test_id", testID, "error", err)
		h.internalError(w)
		return
	}
	if trends == nil {
		trends = []*pgRepo.TestTrendItem{}
	}

	h.writeJSON(w, map[string]interface{}{
		"testId": testID,
		"trends": trends,
		"count":  len(trends),
	})
}

func (h *PostgresHandler) handleMarkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	markers, err := h.repo.GetUniqueMarkers(r.Context())
	if err != nil {
		h.logger.Error("failed to fetch markers from postgres", "error", err)
		h.internalError(w)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"markers": markers,
		"count":   len(markers),
	})
}

func (h *PostgresHandler) handleMarkerStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	markerValue := routeParamOrPath(r, "markerValue", "/api/marker/", "/stats")
	if markerValue == "" {
		h.badRequest(w, "Marker value required")
		return
	}

	limit := int64(100)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	docs, total, err := h.repo.GetRunDocuments(r.Context(), pgRepo.ListRunsFilter{Marker: markerValue}, limit, 0)
	if err != nil {
		h.logger.Error("failed to fetch marker stats from postgres", "marker", markerValue, "error", err)
		h.internalError(w)
		return
	}

	runs := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		stats := buildTestStatistics(flattenRunTests(doc))
		runs = append(runs, map[string]interface{}{
			"runId":       doc.ID,
			"runName":     doc.Name,
			"status":      doc.Status,
			"metadata":    doc.Metadata,
			"startTime":   doc.StartTime,
			"endTime":     doc.EndTime,
			"duration":    doc.Duration,
			"createdAt":   doc.CreatedAt,
			"updatedAt":   doc.UpdatedAt,
			"total":       stats["total"],
			"passed":      stats["passed"],
			"failed":      stats["failed"],
			"skipped":     stats["skipped"],
			"running":     stats["running"],
			"broken":      stats["broken"],
			"timedout":    stats["timedout"],
			"interrupted": stats["interrupted"],
			"unknown":     stats["unknown"],
		})
	}

	h.writeJSON(w, map[string]interface{}{
		"marker": markerValue,
		"runs":   runs,
		"total":  total,
		"count":  len(runs),
	})
}
func (h *PostgresHandler) handleDeleteRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		h.methodNotAllowed(w)
		return
	}

	var req struct {
		RunIDs []string `json:"runIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode delete request", "error", err)
		h.badRequest(w, "Invalid request body")
		return
	}
	if len(req.RunIDs) == 0 {
		h.badRequest(w, "No run IDs provided")
		return
	}

	deleted, err := h.repo.DeleteRuns(r.Context(), req.RunIDs)
	if err != nil {
		h.logger.Error("failed to delete runs from postgres", "runIds", req.RunIDs, "error", err)
		h.internalError(w)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"deleted":   deleted,
		"requested": len(req.RunIDs),
	})
}

func (h *PostgresHandler) handleUpdateMarker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		h.methodNotAllowed(w)
		return
	}

	var req struct {
		RunIDs []string `json:"runIds"`
		Marker *string  `json:"marker"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode marker update request", "error", err)
		h.badRequest(w, "Invalid request body")
		return
	}
	if len(req.RunIDs) == 0 {
		h.badRequest(w, "No run IDs provided")
		return
	}

	var modified int64
	var err error
	if req.Marker == nil || *req.Marker == "" {
		modified, err = h.repo.RemoveRunsMarker(r.Context(), req.RunIDs)
	} else {
		modified, err = h.repo.UpdateRunsMarker(r.Context(), req.RunIDs, *req.Marker)
	}
	if err != nil {
		h.logger.Error("failed to update markers in postgres", "runIds", req.RunIDs, "error", err)
		h.internalError(w)
		return
	}

	response := map[string]interface{}{
		"modified":  modified,
		"requested": len(req.RunIDs),
	}
	if req.Marker != nil && *req.Marker != "" {
		response["marker"] = *req.Marker
	} else {
		response["action"] = "removed"
	}

	h.writeJSON(w, response)
}
func (h *PostgresHandler) writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (h *PostgresHandler) methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *PostgresHandler) badRequest(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusBadRequest)
}

func (h *PostgresHandler) notFound(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusNotFound)
}

func (h *PostgresHandler) internalError(w http.ResponseWriter) {
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func parseLimitOffset(r *http.Request, defaultLimit int64) (int64, int64) {
	limit := defaultLimit
	offset := int64(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.ParseInt(o, 10, 64); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}

func flattenRunTests(doc *m.TestRunDocument) []*m.TestDocument {
	tests := make([]*m.TestDocument, 0, len(doc.Tests))
	tests = append(tests, doc.Tests...)
	for _, suite := range doc.Suites {
		collectSuiteTests(suite, &tests)
	}
	return tests
}

func collectSuiteTests(suite *m.SuiteDocument, target *[]*m.TestDocument) {
	*target = append(*target, suite.Tests...)
	for _, nested := range suite.Suites {
		collectSuiteTests(nested, target)
	}
}

func findTestsInRun(doc *m.TestRunDocument, testID string) []*m.TestDocument {
	found := make([]*m.TestDocument, 0, 1)
	for _, test := range doc.Tests {
		if test.ID == testID {
			found = append(found, test)
			return found
		}
	}
	for _, suite := range doc.Suites {
		findTestsInSuite(suite, testID, &found)
		if len(found) > 0 {
			break
		}
	}
	return found
}

func findTestsInSuite(suite *m.SuiteDocument, testID string, found *[]*m.TestDocument) {
	for _, test := range suite.Tests {
		if test.ID == testID {
			*found = append(*found, test)
			return
		}
	}
	for _, nested := range suite.Suites {
		findTestsInSuite(nested, testID, found)
		if len(*found) > 0 {
			return
		}
	}
}

func (h *PostgresHandler) loadLiveRunningTestDetails(ctx context.Context, runID string, tests []*m.TestDocument) ([]*m.TestDocument, error) {
	if h.liveRunRepo == nil || !needsLiveRunningDetails(tests) {
		return nil, nil
	}

	liveDoc, err := h.liveRunRepo.GetTestRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if liveDoc == nil {
		return nil, nil
	}

	liveTests := make([]*m.TestDocument, 0, len(tests))
	for _, test := range tests {
		liveMatches := findTestsInRun(liveDoc, test.ID)
		if len(liveMatches) == 0 {
			liveTests = append(liveTests, test)
			continue
		}
		liveTests = append(liveTests, mergeLiveRunningTestDetails(test, liveMatches[0]))
	}

	return liveTests, nil
}

func needsLiveRunningDetails(tests []*m.TestDocument) bool {
	for _, test := range tests {
		if test != nil && (test.Status == "RUNNING" || test.Status == "") {
			return true
		}
	}
	return false
}

func mergeLiveRunningTestDetails(base, live *m.TestDocument) *m.TestDocument {
	if base == nil {
		return live
	}
	if live == nil {
		return base
	}

	merged := *base
	if live.Status != "" {
		merged.Status = live.Status
	}
	if live.StartTime != nil {
		merged.StartTime = live.StartTime
	}
	if live.EndTime != nil {
		merged.EndTime = live.EndTime
	}
	if live.Duration != nil {
		merged.Duration = live.Duration
	}
	if !live.UpdatedAt.IsZero() {
		merged.UpdatedAt = live.UpdatedAt
	}
	if live.RetryIndex != nil {
		merged.RetryIndex = live.RetryIndex
	}
	if len(live.Attempts) > 0 {
		merged.Attempts = live.Attempts
	}
	if len(live.Steps) > 0 {
		merged.Steps = live.Steps
	}
	if len(live.Attachments) > 0 {
		merged.Attachments = live.Attachments
	}
	if len(live.Failures) > 0 {
		merged.Failures = live.Failures
	}
	if len(live.Errors) > 0 {
		merged.Errors = live.Errors
	}
	if len(live.ErrorList) > 0 {
		merged.ErrorList = live.ErrorList
	}
	if len(live.StdOut) > 0 {
		merged.StdOut = live.StdOut
	}
	if len(live.StdErr) > 0 {
		merged.StdErr = live.StdErr
	}
	if live.ErrorMessage != "" {
		merged.ErrorMessage = live.ErrorMessage
	}
	if live.StackTrace != "" {
		merged.StackTrace = live.StackTrace
	}

	return &merged
}

func buildTestStatistics(tests []*m.TestDocument) map[string]int {
	stats := map[string]int{
		"total":       len(tests),
		"passed":      0,
		"failed":      0,
		"skipped":     0,
		"running":     0,
		"broken":      0,
		"timedout":    0,
		"interrupted": 0,
		"unknown":     0,
	}
	for _, test := range tests {
		switch test.Status {
		case "PASSED":
			stats["passed"]++
		case "FAILED":
			stats["failed"]++
		case "SKIPPED":
			stats["skipped"]++
		case "RUNNING", "":
			stats["running"]++
		case "BROKEN":
			stats["broken"]++
		case "TIMEDOUT":
			stats["timedout"]++
		case "INTERRUPTED":
			stats["interrupted"]++
		default:
			stats["unknown"]++
		}
	}
	return stats
}

func latestTestUpdate(tests []*m.TestDocument) time.Time {
	var latest time.Time
	for _, test := range tests {
		if test != nil && test.UpdatedAt.After(latest) {
			latest = test.UpdatedAt
		}
	}
	return latest
}
