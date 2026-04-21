package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)




func upsertRelationalTest(tx *gorm.DB, test *m.Test, now time.Time) error {
	var stored m.Test
	err := tx.Where("id = ?", test.ID).First(&stored).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load relational test: %w", err)
		}
		if test.SuiteID == nil {
			return fmt.Errorf("suite id is required for new relational test")
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

func upsertRelationalTestAttempt(tx *gorm.DB, attempt *m.TestAttempt, now time.Time) error {
	var stored m.TestAttempt
	err := tx.Where("test_id = ? AND attempt_index = ?", attempt.TestID, attempt.AttemptIndex).First(&stored).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load relational test attempt: %w", err)
		}
		create := m.TestAttempt{
			ID:           attempt.ID,
			RunID:        attempt.RunID,
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

func aggregateTestAttemptStatuses(attempts []m.TestAttempt, fallback string) string {
	for _, attempt := range attempts {
		if attempt.Status == "PASSED" {
			return "PASSED"
		}
	}
	if fallback != "" {
		return fallback
	}
	if len(attempts) == 0 {
		return ""
	}
	latest := attempts[0]
	for _, attempt := range attempts[1:] {
		if attempt.AttemptIndex >= latest.AttemptIndex {
			latest = attempt
		}
	}
	return latest.Status
}

// AppendTestFailure adds a failure document to the specified test attempt.
func (r *PostgresRepository) AppendTestFailure(ctx context.Context, runID, testID string, attemptIndex int32, failure *m.TestFailureDocument) error {
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
		if err := tx.Where("run_id = ? AND test_id = ? AND attempt_index = ?", runID, internalTestID, attemptIndex).First(&attempt).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("test attempt not found: runID=%s, testID=%s, attemptIndex=%d", runID, testID, attemptIndex)
			}
			return fmt.Errorf("load test attempt for failure: %w", err)
		}

		attempt.Failures = append(attempt.Failures, failure)
		attempt.UpdatedAt = now
		if err := tx.Model(&attempt).Select("Failures", "UpdatedAt").Updates(attempt).Error; err != nil {
			return fmt.Errorf("append relational test failure: %w", err)
		}

		if err := tx.Model(&m.Test{}).Where("id = ?", internalTestID).Update("updated_at", now).Error; err != nil {
			return fmt.Errorf("touch relational test after failure: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("test failure appended", "run_id", runID, "test_id", testID, "attempt_index", attemptIndex)
	return nil
}

// AppendTestError adds an error document to the specified test attempt.
func (r *PostgresRepository) AppendTestError(ctx context.Context, runID, testID string, attemptIndex int32, errorDoc *m.TestErrorDocument) error {
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
		if err := tx.Where("run_id = ? AND test_id = ? AND attempt_index = ?", runID, internalTestID, attemptIndex).First(&attempt).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("test attempt not found: runID=%s, testID=%s, attemptIndex=%d", runID, testID, attemptIndex)
			}
			return fmt.Errorf("load test attempt for error: %w", err)
		}

		attempt.Errors = append(attempt.Errors, errorDoc)
		attempt.UpdatedAt = now
		if err := tx.Model(&attempt).Select("Errors", "UpdatedAt").Updates(attempt).Error; err != nil {
			return fmt.Errorf("append relational test error: %w", err)
		}

		if err := tx.Model(&m.Test{}).Where("id = ?", internalTestID).Update("updated_at", now).Error; err != nil {
			return fmt.Errorf("touch relational test after error: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("test error appended", "run_id", runID, "test_id", testID, "attempt_index", attemptIndex)
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
