package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)

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
	attempt.ID = m.BuildTestAttemptID(test.RunID, test.ID, attempt.ExecutionID, attempt.AttemptIndex)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertRelationalTest(tx, test, now); err != nil {
			return err
		}
		if err := upsertRelationalTestAttempt(tx, attempt, now); err != nil {
			return err
		}
		if attempt.Steps != nil {
			if err := tx.Model(&m.TestAttempt{}).
				Where("run_id = ? AND test_id = ? AND execution_id = ? AND attempt_index = ?", attempt.RunID, attempt.TestID, attempt.ExecutionID, attempt.AttemptIndex).
				Updates(map[string]interface{}{
					"steps":       attempt.Steps,
					"steps_count": attempt.StepsCount,
					"updated_at":  now,
				}).Error; err != nil {
				return fmt.Errorf("persist relational test attempt steps: %w", err)
			}
		}

		var attempts []m.TestAttempt
		if err := tx.Where("run_id = ? AND test_id = ? AND execution_id = ?", test.RunID, test.ID, attempt.ExecutionID).Find(&attempts).Error; err != nil {
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

		if err := tx.Model(&m.Test{}).Where("run_id = ? AND id = ?", test.RunID, test.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("finalize relational test: %w", err)
		}

		if _, err := r.collectRunStats(ctx, tx, test.RunID); err != nil {
			r.logger.Error("collect run stats: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("test end finalized", "run_id", test.RunID, "test_id", test.ID, "attempt_index", attempt.AttemptIndex, "status", attempt.Status)
	return nil
}
