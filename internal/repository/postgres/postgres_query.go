package postgres

import (
	"context"
	"encoding/json"
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

func (r *PostgresRepository) GetRunDocument(ctx context.Context, runID string) (*m.TestRun, error) {
	if err := repository.ValidateRunID(runID); err != nil {
		return nil, err
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	runDocs, err := r.buildRuns(ctx, []string{runID})
	if err != nil {
		return nil, err
	}
	if len(runDocs) == 0 {
		return nil, nil
	}

	return runDocs[0], nil
}

func (r *PostgresRepository) GetRunDocuments(ctx context.Context, filter ListRunsFilter, limit, offset int64) ([]*m.TestRun, int64, error) {
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

	docs, err := r.buildRuns(ctx, runIDs)
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

func (r *PostgresRepository) buildRuns(ctx context.Context, runIDs []string) ([]*m.TestRun, error) {
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

	// var suites []m.Suite
	// if err := r.db.WithContext(ctx).Where("run_id IN ?", runIDs).Order("created_at asc").Find(&suites).Error; err != nil {
	// 	return nil, fmt.Errorf("load suites: %w", err)
	// }

	// var tests []m.Test
	// if err := r.db.WithContext(ctx).Where("run_id IN ?", runIDs).Order("created_at asc").Find(&tests).Error; err != nil {
	// 	return nil, fmt.Errorf("load tests: %w", err)
	// }

	// testIDs := make([]string, 0, len(tests))
	// for _, test := range tests {
	// 	testIDs = append(testIDs, test.ID)
	// }

	// attemptsByTest := map[string][]m.TestAttempt{}
	// if len(testIDs) > 0 {
	// 	var attempts []m.TestAttempt
	// 	if err := r.db.WithContext(ctx).Where("test_id IN ?", testIDs).Order("attempt_index asc").Find(&attempts).Error; err != nil {
	// 		return nil, fmt.Errorf("load test attempts: %w", err)
	// 	}
	// 	for _, attempt := range attempts {
	// 		attempt.Steps = nil // Steps can be large, we will decode them only for the current attempt of each test
	// 		attemptsByTest[attempt.TestID] = append(attemptsByTest[attempt.TestID], attempt)
	// 	}
	// }

	// suiteDocsByRun := make(map[string][]*m.Suite)
	// rootSuiteDocsByRun := make(map[string][]*m.Suite)
	// suiteDocByID := make(map[string]*m.Suite, len(suites))
	// for _, suite := range suites {
	// 	doc := buildSuite(suite)
	// 	suiteDocByID[suite.ID] = doc
	// 	suiteDocsByRun[suite.RunID] = append(suiteDocsByRun[suite.RunID], doc)
	// }
	// for _, suite := range suites {
	// 	doc := suiteDocByID[suite.ID]
	// 	if suite.ParentSuiteID != nil && *suite.ParentSuiteID != "" {
	// 		if parent := suiteDocByID[*suite.ParentSuiteID]; parent != nil {
	// 			parent.Suites = append(parent.Suites, doc)
	// 			continue
	// 		}
	// 	}
	// 	rootSuiteDocsByRun[suite.RunID] = append(rootSuiteDocsByRun[suite.RunID], doc)
	// }

	// rootTestsByRun := make(map[string][]*m.Test)
	// for _, test := range tests {
	// 	doc := buildTest(test, attemptsByTest[test.ID])
	// 	if test.SuiteID != nil && *test.SuiteID != "" {
	// 		if suiteDoc := suiteDocByID[*test.SuiteID]; suiteDoc != nil {
	// 			suiteDoc.Tests = append(suiteDoc.Tests, doc)
	// 			continue
	// 		}
	// 	}
	// 	rootTestsByRun[test.RunID] = append(rootTestsByRun[test.RunID], doc)
	// }

	// docByRunID := make(map[string]*m.TestRun, len(runs))
	// for _, run := range runs {
	// 	docByRunID[run.ID] = buildRun(run)
	// }
	// for runID, doc := range docByRunID {
	// 	doc.Suites = rootSuiteDocsByRun[runID]
	// 	doc.Tests = rootTestsByRun[runID]
	// }

	// ordered := make([]*m.TestRun, 0, len(runIDs))
	// for _, runID := range runIDs {
	// 	if doc, ok := docByRunID[runID]; ok {
	// 		ordered = append(ordered, doc)
	// 	}
	// }

	return runs, nil
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

// func buildRun(run m.TestRun) *m.TestRun {
// 	return &m.TestRun{
// 		ID:          run.ID,
// 		Name:        run.Name,
// 		Description: run.Description,
// 		Status:      run.Status,
// 		Metadata:    cloneMetadata(run.Metadata),
// 		Duration:    run.Duration,
// 		TotalTests:  run.TotalTests,
// 		InitiatedBy: run.InitiatedBy,
// 		ProjectName: run.ProjectName,
// 		StartTime:   run.StartTime,
// 		EndTime:     run.EndTime,
// 		CreatedAt:   run.CreatedAt,
// 		UpdatedAt:   run.UpdatedAt,
// 		Suites:      []*m.Suite{},
// 		Tests:       []*m.Test{},
// 	}
// }

// func buildSuite(suite m.Suite) *m.Suite {
// 	parentSuiteID := ""
// 	if suite.ParentSuiteID != nil {
// 		parentSuiteID = *suite.ParentSuiteID
// 	}
// 	return &m.Suite{
// 		ID:              suiteExternalID(suite),
// 		RunID:           suite.RunID,
// 		ParentSuiteID:   suiteParentExternalID(suite, parentSuiteID),
// 		Name:            suite.Name,
// 		Description:     suite.Description,
// 		Status:          suite.Status,
// 		Metadata:        cloneMetadata(suite.Metadata),
// 		Duration:        suite.Duration,
// 		Location:        suite.Location,
// 		Type:            suite.Type,
// 		TestSuiteSpecID: suite.TestSuiteSpecID,
// 		InitiatedBy:     suite.InitiatedBy,
// 		ProjectName:     suite.ProjectName,
// 		Author:          suite.Author,
// 		Owner:           suite.Owner,
// 		TestCaseIDs:     append([]string(nil), suite.TestCaseIDs...),
// 		SubSuiteIDs:     append([]string(nil), suite.SubSuiteIDs...),
// 		Tags:            append([]string(nil), suite.Tags...),
// 		StartTime:       suite.StartTime,
// 		EndTime:         suite.EndTime,
// 		CreatedAt:       suite.CreatedAt,
// 		UpdatedAt:       suite.UpdatedAt,
// 		Suites:          []*m.Suite{},
// 		Tests:           []*m.Test{},
// 	}
// }

// func buildTest(test m.Test, attempts []m.TestAttempt) *m.Test {
// 	attemptDocs := make([]*m.TestAttempt, 0, len(attempts))
// 	for _, attempt := range attempts {
// 		attemptDocs = append(attemptDocs, buildAttemptDocument(attempt))
// 	}
// 	currentAttempt := selectCurrentAttemptDoc(attemptDocs, test.RetryIndex)

// 	testDoc := &m.Test{
// 		ID:          testExternalID(test),
// 		Name:        test.Name,
// 		Title:       test.Title,
// 		Description: test.Description,
// 		RunID:       test.RunID,
// 		Status:      test.Status,
// 		Metadata:    cloneMetadata(test.Metadata),
// 		Tags:        append([]string(nil), test.Tags...),
// 		Location:    test.Location,
// 		RetryCount:  test.RetryCount,
// 		RetryIndex:  test.RetryIndex,
// 		Timeout:     test.Timeout,
// 		Attempts:    attemptDocs,
// 		CreatedAt:   test.CreatedAt,
// 		UpdatedAt:   test.UpdatedAt,
// 		StartTime:   test.StartTime,
// 		EndTime:     test.EndTime,
// 		Duration:    test.Duration,
// 		Attachments: []map[string]interface{}{},
// 		Failures:    []*m.TestFailureDocument{},
// 		Errors:      []*m.TestErrorDocument{},
// 		ErrorList:   []string{},
// 		StdOut:      []*m.OutputDocument{},
// 		StdErr:      []*m.OutputDocument{},
// 		Steps:       []*m.StepDocument{},
// 	}
// 	if test.SuiteID != nil {
// 		testDoc.SuiteID = suiteIDToExternal(*test.SuiteID)
// 	}
// 	if currentAttempt != nil {
// 		testDoc.ErrorMessage = currentAttempt.ErrorMessage
// 		testDoc.StackTrace = currentAttempt.StackTrace
// 		testDoc.Attachments = cloneAttachmentMaps(currentAttempt.Attachments)
// 		testDoc.Failures = cloneFailures(currentAttempt.Failures)
// 		testDoc.Errors = cloneErrors(currentAttempt.Errors)
// 		testDoc.ErrorList = append([]string(nil), currentAttempt.ErrorList...)
// 		testDoc.StdOut = cloneOutputs(currentAttempt.StdOut)
// 		testDoc.StdErr = cloneOutputs(currentAttempt.StdErr)
// 		testDoc.Steps = cloneSteps(currentAttempt.Steps)
// 	}
// 	return testDoc
// }

// func buildAttempt(attempt m.TestAttempt) *m.TestAttempt {
// 	steps := decodeAttemptSteps(attempt.Steps)
// 	if attempt.StepsCount == 0 {
// 		attempt.StepsCount = int32(len(steps))
// 	}

// 	return &m.TestAttempt{
// 		RetryIndex:   attempt.AttemptIndex,
// 		Steps:        steps,
// 		StepsCount:   &attempt.StepsCount,
// 		Status:       attempt.Status,
// 		StartTime:    attempt.StartTime,
// 		EndTime:      attempt.EndTime,
// 		Duration:     attempt.Duration,
// 		Attachments:  cloneAttachmentMaps(attempt.Attachments),
// 		ErrorMessage: attempt.ErrorMessage,
// 		StackTrace:   attempt.StackTrace,
// 		ErrorList:    append([]string(nil), attempt.ErrorList...),
// 		Failures:     cloneFailures(attempt.Failures),
// 		Errors:       cloneErrors(attempt.Errors),
// 		StdOut:       cloneOutputs(attempt.StdOut),
// 		StdErr:       cloneOutputs(attempt.StdErr),
// 		CreatedAt:    attempt.CreatedAt,
// 		UpdatedAt:    attempt.UpdatedAt,
// 	}
// }

func decodeAttemptSteps(raw *json.RawMessage) []*m.StepDocument {
	if raw == nil || len(*raw) == 0 {
		return []*m.StepDocument{}
	}
	var steps []*m.StepDocument
	if err := json.Unmarshal(*raw, &steps); err != nil {
		return []*m.StepDocument{}
	}
	return steps
}

func selectCurrentAttemptDoc(attempts []*m.TestAttempt, retryIndex *int32) *m.TestAttempt {
	if len(attempts) == 0 {
		return nil
	}
	if retryIndex != nil {
		for _, attempt := range attempts {
			if attempt.AttemptIndex == *retryIndex {
				return attempt
			}
		}
	}
	latest := attempts[0]
	for _, attempt := range attempts[1:] {
		if attempt.AttemptIndex >= latest.AttemptIndex {
			latest = attempt
		}
	}
	return latest
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
