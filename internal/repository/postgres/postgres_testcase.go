package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)

// UpsertTestBegin creates or updates the relational test row and its current attempt.
func (r *PostgresRepository) UpsertTestBegin(ctx context.Context, test *m.Test, attempt *m.TestAttempt) error {
	if test == nil {
		return fmt.Errorf("test is nil")
	}
	if attempt == nil {
		return fmt.Errorf("test attempt is nil")
	}
	if err := repository.ValidateRunID(test.RunID); err != nil {
		return err
	}
	if test.ID == "" {
		return fmt.Errorf("test id is required")
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	test.CreatedAt = now
	test.UpdatedAt = now
	attempt.CreatedAt = now
	attempt.UpdatedAt = now
	if attempt.ID == "" {
		attempt.ID = fmt.Sprintf("%s:%d", test.ID, attempt.AttemptIndex)
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertRelationalTest(tx, test, now); err != nil {
			return err
		}
		if err := upsertRelationalTestAttempt(tx, attempt, now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("test begin upserted", "run_id", test.RunID, "test_id", test.ID, "attempt_index", attempt.AttemptIndex)
	return nil
}

// FinalizeTestEnd applies terminal state to the relational test row and current attempt.
func (r *PostgresRepository) FinalizeTestEnd(ctx context.Context, test *m.Test, attempt *m.TestAttempt) error {
	if test == nil {
		return fmt.Errorf("test is nil")
	}
	if attempt == nil {
		return fmt.Errorf("test attempt is nil")
	}
	if err := repository.ValidateRunID(test.RunID); err != nil {
		return err
	}
	if test.ID == "" {
		return fmt.Errorf("test id is required")
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	test.CreatedAt = now
	test.UpdatedAt = now
	attempt.CreatedAt = now
	attempt.UpdatedAt = now
	if attempt.ID == "" {
		attempt.ID = fmt.Sprintf("%s:%d", test.ID, attempt.AttemptIndex)
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertRelationalTest(tx, test, now); err != nil {
			return err
		}
		if err := upsertRelationalTestAttempt(tx, attempt, now); err != nil {
			return err
		}

		var attempts []m.TestAttempt
		if err := tx.Where("test_id = ?", test.ID).Find(&attempts).Error; err != nil {
			return fmt.Errorf("load test attempts: %w", err)
		}

		overallStatus := aggregateTestAttemptStatuses(attempts, attempt.Status)
		updates := map[string]interface{}{
			"status":      overallStatus,
			"updated_at":  now,
			"retry_index": test.RetryIndex,
			"retry_count": test.RetryCount,
		}
		if test.StartTime != nil {
			updates["started_at"] = *test.StartTime
		}
		if test.EndTime != nil {
			updates["finished_at"] = *test.EndTime
		}
		if test.Duration != nil {
			updates["duration"] = *test.Duration
		}

		if err := tx.Model(&m.Test{}).Where("id = ?", test.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("finalize relational test: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("test end finalized", "run_id", test.RunID, "test_id", test.ID, "attempt_index", attempt.AttemptIndex, "status", attempt.Status)
	return nil
}

func upsertRelationalTest(tx *gorm.DB, test *m.Test, now time.Time) error {
	assignment := m.Test{
		RunID:       test.RunID,
		SuiteID:     test.SuiteID,
		Name:        test.Name,
		Title:       test.Title,
		Description: test.Description,
		Status:      test.Status,
		StartTime:   test.StartTime,
		EndTime:     test.EndTime,
		Duration:    test.Duration,
		Metadata:    test.Metadata,
		Tags:        test.Tags,
		Location:    test.Location,
		RetryCount:  test.RetryCount,
		RetryIndex:  test.RetryIndex,
		Timeout:     test.Timeout,
		UpdatedAt:   now,
		CreatedAt:   now,
	}

	stored := m.Test{ID: test.ID}
	result := tx.Where(&stored).Assign(assignment).FirstOrCreate(&stored)
	if result.Error != nil {
		return fmt.Errorf("upsert relational test: %w", result.Error)
	}
	return nil
}

func upsertRelationalTestAttempt(tx *gorm.DB, attempt *m.TestAttempt, now time.Time) error {
	assignment := m.TestAttempt{
		ID:           attempt.ID,
		RunID:        attempt.RunID,
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

	stored := m.TestAttempt{TestID: attempt.TestID, AttemptIndex: attempt.AttemptIndex}
	result := tx.Where(&stored).Assign(assignment).FirstOrCreate(&stored)
	if result.Error != nil {
		return fmt.Errorf("upsert relational test attempt: %w", result.Error)
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
