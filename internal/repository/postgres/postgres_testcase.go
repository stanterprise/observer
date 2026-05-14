package postgres

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)

func upsertRelationalTest(tx *gorm.DB, test *m.Test, now time.Time) error {
	var stored m.Test
	err := tx.Where("run_id = ? AND id = ?", test.RunID, test.ID).First(&stored).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load relational test: %w", err)
		}
		if test.SuiteID == nil {
			return fmt.Errorf("suite id is required for new relational test")
		}
		if err := ensureRelationalSuiteExists(tx, test.RunID, *test.SuiteID, now); err != nil {
			return err
		}
		create := m.Test{
			ID:             test.ID,
			RunID:          test.RunID,
			ExternalTestID: test.ExternalTestID,
			SuiteID:        test.SuiteID,
			Name:           test.Name,
			Title:          test.Title,
			Description:    test.Description,
			Status:         test.Status,
			StartTime:      test.StartTime,
			EndTime:        test.EndTime,
			Duration:       test.Duration,
			Metadata:       test.Metadata,
			Tags:           test.Tags,
			Location:       test.Location,
			RetryCount:     test.RetryCount,
			RetryIndex:     test.RetryIndex,
			Timeout:        test.Timeout,
			UpdatedAt:      now,
			CreatedAt:      now,
		}
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create relational test: %w", err)
		}
		return nil
	}

	if test.RunID != "" {
		stored.RunID = test.RunID
	}
	if test.ExternalTestID != "" {
		stored.ExternalTestID = test.ExternalTestID
	}
	if test.SuiteID != nil {
		stored.SuiteID = test.SuiteID
	}
	if test.Name != "" {
		stored.Name = test.Name
	}
	if test.Title != "" {
		stored.Title = test.Title
	}
	if test.Description != "" {
		stored.Description = test.Description
	}
	if test.Status != "" {
		stored.Status = test.Status
	}
	if test.StartTime != nil {
		stored.StartTime = test.StartTime
	}
	if test.EndTime != nil {
		stored.EndTime = test.EndTime
	}
	if test.Duration != nil {
		stored.Duration = test.Duration
	}
	if len(test.Metadata) > 0 {
		stored.Metadata = test.Metadata
	}
	if len(test.Tags) > 0 {
		stored.Tags = test.Tags
	}
	if test.Location != "" {
		stored.Location = test.Location
	}
	if test.RetryCount != nil {
		stored.RetryCount = test.RetryCount
	}
	if test.RetryIndex != nil {
		stored.RetryIndex = test.RetryIndex
	}
	if test.Timeout != nil {
		stored.Timeout = test.Timeout
	}
	stored.UpdatedAt = now

	if err := tx.Save(&stored).Error; err != nil {
		return fmt.Errorf("update relational test: %w", err)
	}
	return nil
}

func ensureRelationalSuiteExists(tx *gorm.DB, runID, suiteID string, now time.Time) error {
	var stored m.Suite
	err := tx.Where("run_id = ? AND id = ?", runID, suiteID).First(&stored).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("load relational suite: %w", err)
	}

	placeholder := m.Suite{
		ID:              suiteID,
		RunID:           runID,
		ExternalSuiteID: suiteID,
		Status:          "RUNNING",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&placeholder).Error; err != nil {
		return fmt.Errorf("create placeholder suite: %w", err)
	}

	return nil
}

func upsertRelationalTestAttempt(tx *gorm.DB, attempt *m.TestAttempt, now time.Time) error {
	var stored m.TestAttempt
	err := tx.Where("run_id = ? AND test_id = ? AND execution_id = ? AND attempt_index = ?", attempt.RunID, attempt.TestID, attempt.ExecutionID, attempt.AttemptIndex).First(&stored).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load relational test attempt: %w", err)
		}
		create := m.TestAttempt{
			ID:           attempt.ID,
			RunID:        attempt.RunID,
			ExecutionID:  attempt.ExecutionID,
			TestID:       attempt.TestID,
			AttemptIndex: attempt.AttemptIndex,
			Status:       attempt.Status,
			StartTime:    attempt.StartTime,
			EndTime:      attempt.EndTime,
			Duration:     attempt.Duration,
			Attachments:  attempt.Attachments,
			ErrorMessage: attempt.ErrorMessage,
			StackTrace:   attempt.StackTrace,
			ErrorList:    attempt.ErrorList,
			UpdatedAt:    now,
			CreatedAt:    now,
		}
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create relational test attempt: %w", err)
		}
		return nil
	}

	if attempt.ID != "" {
		stored.ID = attempt.ID
	}
	if attempt.RunID != "" {
		stored.RunID = attempt.RunID
	}
	stored.ExecutionID = attempt.ExecutionID
	if attempt.Status != "" {
		stored.Status = attempt.Status
	}
	if attempt.StartTime != nil {
		stored.StartTime = attempt.StartTime
	}
	if attempt.EndTime != nil {
		stored.EndTime = attempt.EndTime
	}
	if attempt.Duration != nil {
		stored.Duration = attempt.Duration
	}
	if attempt.Attachments != nil {
		stored.Attachments = attempt.Attachments
	}
	if attempt.ErrorMessage != "" {
		stored.ErrorMessage = attempt.ErrorMessage
	}
	if attempt.StackTrace != "" {
		stored.StackTrace = attempt.StackTrace
	}
	if attempt.ErrorList != nil {
		stored.ErrorList = attempt.ErrorList
	}
	stored.UpdatedAt = now

	if err := tx.Save(&stored).Error; err != nil {
		return fmt.Errorf("update relational test attempt: %w", err)
	}
	return nil
}

func aggregateTestAttemptStatuses(attempts []m.TestAttempt) string {
	if len(attempts) == 0 {
		return "NOT_RUN"
	} else if len(attempts) == 1 {
		return attempts[0].Status
	}

	// Sorted in descending order of creation time, so that the latest attempt is first.
	sort.Slice(attempts, func(i, j int) bool {
		return attempts[i].CreatedAt.After(attempts[j].CreatedAt)
	})

	latest := attempts[0].Status

	if latest == "PASSED" {
		isFlaky := false
		for _, attempt := range attempts[1:] {
			if attempt.Status == "FAILED" ||
				attempt.Status == "BROKEN" ||
				attempt.Status == "TIMEDOUT" ||
				attempt.Status == "INTERRUPTED" {
				isFlaky = true
				break
			}
		}

		if isFlaky {
			return "FLAKY"
		}
		return "PASSED"
	}

	allSame := true
	for _, attempt := range attempts[1:] {
		if attempt.Status != latest {
			allSame = false
			break
		}
	}
	if allSame {
		return latest
	}

	if len(attempts) > 1 && (latest == "UNKNOWN" || latest == "" || latest == "NOT_RUN" || latest == "SKIPPED") {
		return "UNKNOWN"
	}

	return latest
}

func latestExecutionAttemptSet(attempts []m.TestAttempt) ([]m.TestAttempt, string) {
	if len(attempts) == 0 {
		return nil, ""
	}

	type executionWindow struct {
		executionID string
		attempts    []m.TestAttempt
		latestAt    time.Time
	}

	windows := make(map[string]*executionWindow, len(attempts))
	order := make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		executionID := attempt.ExecutionID
		window, ok := windows[executionID]
		if !ok {
			window = &executionWindow{executionID: executionID}
			windows[executionID] = window
			order = append(order, executionID)
		}
		window.attempts = append(window.attempts, attempt)

		candidate := latestAttemptTimestamp(attempt)
		if candidate.After(window.latestAt) {
			window.latestAt = candidate
		}
	}

	selected := windows[order[0]]
	for _, executionID := range order[1:] {
		window := windows[executionID]
		if window.latestAt.After(selected.latestAt) {
			selected = window
		}
	}

	return selected.attempts, selected.executionID
}

func latestAttemptTimestamp(attempt m.TestAttempt) time.Time {
	if !attempt.UpdatedAt.IsZero() {
		return attempt.UpdatedAt
	}
	if attempt.EndTime != nil {
		return *attempt.EndTime
	}
	if attempt.StartTime != nil {
		return *attempt.StartTime
	}
	if !attempt.CreatedAt.IsZero() {
		return attempt.CreatedAt
	}
	return time.Time{}
}

// AppendTestFailure adds a failure document to the specified test attempt.
func (r *PostgresRepository) AppendTestFailure(ctx context.Context, runID, executionID, testID string, attemptIndex int32, failure *m.TestFailureDocument) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("test id is required")
	}
	if failure == nil {
		return fmt.Errorf("failure is nil")
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		internalTestID, err := resolveInternalTestID(tx, runID, testID)
		if err != nil {
			return err
		}

		var attempt m.TestAttempt
		if err := tx.Where("run_id = ? AND execution_id = ? AND test_id = ? AND attempt_index = ?", runID, executionID, internalTestID, attemptIndex).First(&attempt).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("test attempt not found: runID=%s, executionID=%s, testID=%s, attemptIndex=%d", runID, executionID, testID, attemptIndex)
			}
			return fmt.Errorf("load test attempt for failure: %w", err)
		}

		attempt.Failures = append(attempt.Failures, failure)
		attempt.UpdatedAt = now
		if err := tx.Model(&attempt).Select("Failures", "UpdatedAt").Updates(attempt).Error; err != nil {
			return fmt.Errorf("append relational test failure: %w", err)
		}

		if err := tx.Model(&m.Test{}).Where("run_id = ? AND id = ?", runID, internalTestID).Update("updated_at", now).Error; err != nil {
			return fmt.Errorf("touch relational test after failure: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("test failure appended", "run_id", runID, "execution_id", executionID, "test_id", testID, "attempt_index", attemptIndex)
	return nil
}

// AppendTestError adds an error document to the specified test attempt.
func (r *PostgresRepository) AppendTestError(ctx context.Context, runID, executionID, testID string, attemptIndex int32, errorDoc *m.TestErrorDocument) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("test id is required")
	}
	if errorDoc == nil {
		return fmt.Errorf("error document is nil")
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		internalTestID, err := resolveInternalTestID(tx, runID, testID)
		if err != nil {
			return err
		}

		var attempt m.TestAttempt
		if err := tx.Where("run_id = ? AND execution_id = ? AND test_id = ? AND attempt_index = ?", runID, executionID, internalTestID, attemptIndex).First(&attempt).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("test attempt not found: runID=%s, executionID=%s, testID=%s, attemptIndex=%d", runID, executionID, testID, attemptIndex)
			}
			return fmt.Errorf("load test attempt for error: %w", err)
		}

		attempt.Errors = append(attempt.Errors, errorDoc)
		attempt.UpdatedAt = now
		if err := tx.Model(&attempt).Select("Errors", "UpdatedAt").Updates(attempt).Error; err != nil {
			return fmt.Errorf("append relational test error: %w", err)
		}

		if err := tx.Model(&m.Test{}).Where("run_id = ? AND id = ?", runID, internalTestID).Update("updated_at", now).Error; err != nil {
			return fmt.Errorf("touch relational test after error: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("test error appended", "run_id", runID, "execution_id", executionID, "test_id", testID, "attempt_index", attemptIndex)
	return nil
}

func resolveInternalTestID(tx *gorm.DB, runID, externalTestID string) (string, error) {
	var test m.Test
	err := tx.Where("run_id = ? AND (external_test_id = ? OR id = ?)", runID, externalTestID, externalTestID).First(&test).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("test not found: runID=%s, testID=%s", runID, externalTestID)
		}
		return "", fmt.Errorf("load relational test: %w", err)
	}
	return test.ID, nil
}
