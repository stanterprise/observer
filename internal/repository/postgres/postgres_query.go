package postgres

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)

type ListRunsFilter struct {
	RunID       string
	Status      string
	ProjectName string
	Marker      string
}

type TestTrendItem struct {
	TestID    string     `json:"testId"`
	RunID     string     `json:"runId"`
	Status    string     `json:"status"`
	Duration  *int64     `json:"duration,omitempty"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type MarkerInfo struct {
	Marker string `json:"marker"`
	Count  int64  `json:"count"`
}

func (r *PostgresRepository) ListRuns(ctx context.Context, filter ListRunsFilter, limit, offset int64) ([]*m.TestRun, int64, error) {
	if err := r.ensureDB(); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := r.db.WithContext(ctx).Model(&m.TestRun{})
	query, err := r.applyRunListFilter(ctx, query, filter)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count runs: %w", err)
	}

	var runs []*m.TestRun
	if err := query.Order("created_at desc").Offset(int(offset)).Limit(int(limit)).Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("list runs: %w", err)
	}

	return runs, total, nil
}

func (r *PostgresRepository) GetRun(ctx context.Context, runID string, includeSteps bool) (*m.TestRun, error) {
	if err := repository.ValidateRunID(runID); err != nil {
		return nil, err
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	runDocs, err := r.buildRuns(ctx, []string{runID}, includeSteps)
	if err != nil {
		return nil, err
	}
	if len(runDocs) == 0 {
		return nil, nil
	}

	return runDocs[0], nil
}

func (r *PostgresRepository) GetRuns(ctx context.Context, filter ListRunsFilter, limit, offset int64, includeSteps bool) ([]*m.TestRun, int64, error) {
	runs, total, err := r.ListRuns(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(runs) == 0 {
		return []*m.TestRun{}, total, nil
	}

	runIDs := make([]string, 0, len(runs))
	for _, run := range runs {
		runIDs = append(runIDs, run.ID)
	}

	docs, err := r.buildRuns(ctx, runIDs, includeSteps)
	if err != nil {
		return nil, 0, err
	}

	ordered := make([]*m.TestRun, 0, len(runIDs))
	byID := make(map[string]*m.TestRun, len(docs))
	for _, doc := range docs {
		byID[doc.ID] = doc
	}
	for _, runID := range runIDs {
		if doc, ok := byID[runID]; ok {
			ordered = append(ordered, doc)
		}
	}

	return ordered, total, nil
}

func (r *PostgresRepository) GetTestTrends(ctx context.Context, testID string, limit int64) ([]*TestTrendItem, error) {
	if testID == "" {
		return nil, fmt.Errorf("testID is required")
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}

	var tests []m.Test
	if err := r.db.WithContext(ctx).
		Where("external_test_id = ? OR id = ?", testID, testID).
		Order("created_at desc").
		Limit(int(limit)).
		Find(&tests).Error; err != nil {
		return nil, fmt.Errorf("list test trends: %w", err)
	}

	items := make([]*TestTrendItem, 0, len(tests))
	for _, test := range tests {
		item := &TestTrendItem{
			TestID:    testExternalID(test),
			RunID:     test.RunID,
			Status:    test.Status,
			Duration:  test.Duration,
			CreatedAt: test.CreatedAt,
		}
		if test.StartTime != nil {
			start := *test.StartTime
			item.StartTime = &start
		}
		if test.EndTime != nil {
			end := *test.EndTime
			item.EndTime = &end
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *PostgresRepository) GetUniqueMarkers(ctx context.Context) ([]*MarkerInfo, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	var runs []m.TestRun
	if err := r.db.WithContext(ctx).Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("list runs for markers: %w", err)
	}

	counts := map[string]int64{}
	for _, run := range runs {
		marker, ok := markerFromMetadata(run.Metadata)
		if !ok {
			continue
		}
		counts[marker]++
	}

	markers := make([]*MarkerInfo, 0, len(counts))
	for marker, count := range counts {
		markers = append(markers, &MarkerInfo{Marker: marker, Count: count})
	}

	sort.Slice(markers, func(i, j int) bool {
		if markers[i].Count == markers[j].Count {
			return markers[i].Marker < markers[j].Marker
		}
		return markers[i].Count > markers[j].Count
	})

	return markers, nil
}

func (r *PostgresRepository) DeleteRuns(ctx context.Context, runIDs []string) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	if len(runIDs) == 0 {
		return 0, nil
	}
	for _, runID := range runIDs {
		if err := repository.ValidateRunID(runID); err != nil {
			return 0, fmt.Errorf("invalid runID %s: %w", runID, err)
		}
	}

	var deletedRuns int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("run_id IN ?", runIDs).Delete(&m.Attachment{}).Error; err != nil {
			return fmt.Errorf("delete run attachments: %w", err)
		}
		if err := tx.Where("run_id IN ?", runIDs).Delete(&m.TestAttempt{}).Error; err != nil {
			return fmt.Errorf("delete run attempts: %w", err)
		}
		if err := tx.Where("run_id IN ?", runIDs).Delete(&m.Test{}).Error; err != nil {
			return fmt.Errorf("delete run tests: %w", err)
		}
		if err := tx.Where("run_id IN ?", runIDs).Delete(&m.Suite{}).Error; err != nil {
			return fmt.Errorf("delete run suites: %w", err)
		}
		if err := tx.Where("run_id IN ?", runIDs).Delete(&m.RunShard{}).Error; err != nil {
			return fmt.Errorf("delete run shards: %w", err)
		}
		if err := tx.Where("run_id IN ?", runIDs).Delete(&m.RunExecution{}).Error; err != nil {
			return fmt.Errorf("delete run executions: %w", err)
		}

		result := tx.Where("id IN ?", runIDs).Delete(&m.TestRun{})
		if result.Error != nil {
			return fmt.Errorf("delete runs: %w", result.Error)
		}
		deletedRuns = result.RowsAffected
		return nil
	})
	if err != nil {
		return 0, err
	}

	return deletedRuns, nil
}

func (r *PostgresRepository) UpdateRunsMarker(ctx context.Context, runIDs []string, marker string) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}
	if marker == "" {
		return 0, fmt.Errorf("marker value cannot be empty")
	}

	var runs []m.TestRun
	if err := r.db.WithContext(ctx).Where("id IN ?", runIDs).Find(&runs).Error; err != nil {
		return 0, fmt.Errorf("load runs for marker update: %w", err)
	}

	var modified int64
	for _, run := range runs {
		metadata := cloneMetadata(run.Metadata)
		if metadata == nil {
			metadata = map[string]interface{}{}
		}
		metadata["MARKER"] = marker
		if err := r.db.WithContext(ctx).Model(&m.TestRun{}).Where("id = ?", run.ID).Updates(map[string]interface{}{"metadata": metadata}).Error; err != nil {
			return modified, fmt.Errorf("update run marker %s: %w", run.ID, err)
		}
		modified++
	}

	return modified, nil
}

func (r *PostgresRepository) RemoveRunsMarker(ctx context.Context, runIDs []string) (int64, error) {
	if err := r.ensureDB(); err != nil {
		return 0, err
	}

	var runs []m.TestRun
	if err := r.db.WithContext(ctx).Where("id IN ?", runIDs).Find(&runs).Error; err != nil {
		return 0, fmt.Errorf("load runs for marker removal: %w", err)
	}

	var modified int64
	for _, run := range runs {
		metadata := cloneMetadata(run.Metadata)
		if metadata == nil {
			continue
		}
		if _, exists := metadata["MARKER"]; !exists {
			continue
		}
		delete(metadata, "MARKER")
		if err := r.db.WithContext(ctx).Model(&m.TestRun{}).Where("id = ?", run.ID).Updates(map[string]interface{}{"metadata": metadata}).Error; err != nil {
			return modified, fmt.Errorf("remove run marker %s: %w", run.ID, err)
		}
		modified++
	}

	return modified, nil
}

func (r *PostgresRepository) FindAttachmentByStorageKey(ctx context.Context, storageKey string) (map[string]interface{}, error) {
	if storageKey == "" {
		return nil, fmt.Errorf("storage key is required")
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	var attempts []m.TestAttempt
	if err := r.db.WithContext(ctx).Order("updated_at desc").Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("list attempts for attachment lookup: %w", err)
	}

	for _, attempt := range attempts {
		if attachment := findAttachmentInMaps(attempt.Attachments, storageKey); attachment != nil {
			return attachment, nil
		}
		for _, failure := range attempt.Failures {
			if attachment := findAttachmentInMaps(failure.Attachments, storageKey); attachment != nil {
				return attachment, nil
			}
		}
		for _, errDoc := range attempt.Errors {
			if attachment := findAttachmentInMaps(errDoc.Attachments, storageKey); attachment != nil {
				return attachment, nil
			}
		}
	}

	return nil, nil
}

func (r *PostgresRepository) buildRuns(ctx context.Context, runIDs []string, includeSteps bool) ([]*m.TestRun, error) {
	if len(runIDs) == 0 {
		return []*m.TestRun{}, nil
	}

	var runs []*m.TestRun
	if err := r.db.WithContext(ctx).Where("id IN ?", runIDs).Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("load runs: %w", err)
	}

	if len(runs) == 0 {
		return []*m.TestRun{}, nil
	}

	var suites []*m.Suite
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("created_at asc, id asc").
		Find(&suites).Error; err != nil {
		return nil, fmt.Errorf("load suites: %w", err)
	}

	var tests []*m.Test
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("created_at asc, id asc").
		Find(&tests).Error; err != nil {
		return nil, fmt.Errorf("load tests: %w", err)
	}

	var executions []*m.RunExecution
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("created_at asc, id asc").
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("load run executions: %w", err)
	}

	var shards []m.RunShard
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("created_at asc, id asc").
		Find(&shards).Error; err != nil {
		return nil, fmt.Errorf("load run shards: %w", err)
	}

	var attempts []m.TestAttempt
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("test_id asc, attempt_index asc").
		Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("load test attempts: %w", err)
	}

	if !includeSteps {
		for i := range attempts {
			attempts[i].Steps = nil
		}
	}

	attemptsByTestID := make(map[string][]m.TestAttempt, len(attempts))
	for _, attempt := range attempts {
		attemptsByTestID[attempt.TestID] = append(attemptsByTestID[attempt.TestID], attempt)
	}

	runByID := make(map[string]*m.TestRun, len(runs))
	for _, run := range runs {
		run.Executions = nil
		run.Suites = nil
		run.Tests = nil
		runByID[run.ID] = run
	}

	for _, execution := range executions {
		if run, ok := runByID[execution.RunID]; ok {
			run.Executions = append(run.Executions, execution)
		}
	}

	shardsByRunID := make(map[string][]m.RunShard, len(runIDs))
	for _, shard := range shards {
		shardsByRunID[shard.RunID] = append(shardsByRunID[shard.RunID], shard)
	}
	for runID, run := range runByID {
		if shardAggregate, ok := buildLogicalRunAggregateFromShards(runID, shardsByRunID[runID], run.TotalTests, run.UpdatedAt); ok {
			run.Status = shardAggregate.Status
			run.StartTime = shardAggregate.StartTime
			run.EndTime = shardAggregate.EndTime
			run.Duration = shardAggregate.Duration
		}
	}

	suiteByID := make(map[string]*m.Suite, len(suites))
	for _, suite := range suites {
		suite.Suites = nil
		suite.Tests = nil
		suiteByID[suite.ID] = suite
	}

	for _, suite := range suites {
		if suite.ParentSuiteID != nil {
			if parent, ok := suiteByID[*suite.ParentSuiteID]; ok {
				parent.Suites = append(parent.Suites, suite)
				continue
			}
		}
		if run, ok := runByID[suite.RunID]; ok {
			run.Suites = append(run.Suites, suite)
		}
	}

	for _, test := range tests {
		if attachedAttempts, ok := attemptsByTestID[test.ID]; ok {
			test.Attempts = attachedAttempts
		} else {
			test.Attempts = nil
		}

		if test.SuiteID != nil {
			if suite, ok := suiteByID[*test.SuiteID]; ok {
				suite.Tests = append(suite.Tests, test)
				continue
			}
		}
		if run, ok := runByID[test.RunID]; ok {
			run.Tests = append(run.Tests, test)
		}
	}

	for _, run := range runByID {
		if actualTotalTests := countRunTests(run); actualTotalTests > 0 {
			run.TotalTests = actualTotalTests
		}
	}

	return runs, nil
}

func countRunTests(run *m.TestRun) int32 {
	if run == nil {
		return 0
	}

	seen := make(map[string]struct{})
	for _, test := range run.Tests {
		if test == nil || test.ID == "" {
			continue
		}
		seen[test.ID] = struct{}{}
	}
	for _, suite := range run.Suites {
		collectSuiteTestIDs(suite, seen)
	}

	return int32(len(seen))
}

func collectSuiteTestIDs(suite *m.Suite, seen map[string]struct{}) {
	if suite == nil {
		return
	}

	for _, test := range suite.Tests {
		if test == nil || test.ID == "" {
			continue
		}
		seen[test.ID] = struct{}{}
	}
	for _, nested := range suite.Suites {
		collectSuiteTestIDs(nested, seen)
	}
}

func (r *PostgresRepository) applyRunListFilter(ctx context.Context, query *gorm.DB, filter ListRunsFilter) (*gorm.DB, error) {
	if filter.RunID != "" {
		query = query.Where("id = ?", filter.RunID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ProjectName != "" {
		query = query.Where("project_name = ?", filter.ProjectName)
	}
	if filter.Marker != "" {
		var runs []m.TestRun
		if err := query.Session(&gorm.Session{}).WithContext(ctx).Find(&runs).Error; err != nil {
			return nil, fmt.Errorf("list runs for marker filter: %w", err)
		}
		ids := make([]string, 0)
		for _, run := range runs {
			if marker, ok := markerFromMetadata(run.Metadata); ok && marker == filter.Marker {
				ids = append(ids, run.ID)
			}
		}
		if len(ids) == 0 {
			query = query.Where("1 = 0")
		} else {
			query = query.Where("id IN ?", ids)
		}
	}
	return query, nil
}

func decodeAttemptSteps(raw *m.Step) []*m.StepDocument {
	steps, err := m.StepDocumentsFromStep(raw)
	if err != nil {
		return []*m.StepDocument{}
	}
	return steps
}

func markerFromMetadata(metadata map[string]interface{}) (string, bool) {
	if metadata == nil {
		return "", false
	}
	value, ok := metadata["MARKER"]
	if !ok || value == nil {
		return "", false
	}
	marker := strings.TrimSpace(fmt.Sprint(value))
	if marker == "" || marker == "<nil>" {
		return "", false
	}
	return marker, true
}

func suiteExternalID(suite m.Suite) string {
	if suite.ExternalSuiteID != "" {
		return suite.ExternalSuiteID
	}
	return suiteIDToExternal(suite.ID)
}

func suiteParentExternalID(suite m.Suite, fallback string) string {
	if suite.ParentSuiteID == nil {
		return ""
	}
	if fallback != "" {
		return suiteIDToExternal(fallback)
	}
	return suiteIDToExternal(*suite.ParentSuiteID)
}

func suiteIDToExternal(value string) string {
	parts := strings.SplitN(value, ":suite:", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return value
}

func testExternalID(test m.Test) string {
	if test.ExternalTestID != "" {
		return test.ExternalTestID
	}
	parts := strings.SplitN(test.ID, ":test:", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return test.ID
}

func cloneMetadata(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]interface{}, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneAttachmentMaps(input []map[string]interface{}) []map[string]interface{} {
	if len(input) == 0 {
		return []map[string]interface{}{}
	}
	output := make([]map[string]interface{}, 0, len(input))
	for _, item := range input {
		output = append(output, cloneMetadata(item))
	}
	return output
}

func cloneFailures(input []*m.TestFailureDocument) []*m.TestFailureDocument {
	if len(input) == 0 {
		return []*m.TestFailureDocument{}
	}
	output := make([]*m.TestFailureDocument, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		copied := *item
		copied.Attachments = cloneAttachmentMaps(item.Attachments)
		output = append(output, &copied)
	}
	return output
}

func cloneErrors(input []*m.TestErrorDocument) []*m.TestErrorDocument {
	if len(input) == 0 {
		return []*m.TestErrorDocument{}
	}
	output := make([]*m.TestErrorDocument, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		copied := *item
		copied.Attachments = cloneAttachmentMaps(item.Attachments)
		output = append(output, &copied)
	}
	return output
}

func cloneOutputs(input []*m.OutputDocument) []*m.OutputDocument {
	if len(input) == 0 {
		return []*m.OutputDocument{}
	}
	output := make([]*m.OutputDocument, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		copied := *item
		output = append(output, &copied)
	}
	return output
}

func cloneSteps(input []*m.StepDocument) []*m.StepDocument {
	if len(input) == 0 {
		return []*m.StepDocument{}
	}
	output := make([]*m.StepDocument, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		copied := *item
		copied.Metadata = cloneMetadata(item.Metadata)
		copied.Tags = append([]string(nil), item.Tags...)
		copied.Errors = append([]string(nil), item.Errors...)
		copied.Steps = cloneSteps(item.Steps)
		output = append(output, &copied)
	}
	return output
}

func findAttachmentInMaps(attachments []map[string]interface{}, storageKey string) map[string]interface{} {
	for _, attachment := range attachments {
		if key, ok := attachment["storage_key"].(string); ok && key == storageKey {
			return cloneMetadata(attachment)
		}
	}
	return nil
}
