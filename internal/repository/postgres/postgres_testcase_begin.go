package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
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
	if test.RunID == "" {
		return fmt.Errorf("test run id is required")
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
	attempt.ID = m.BuildTestAttemptID(test.ID, attempt.ExecutionID, attempt.AttemptIndex)

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
