package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
)

// FinalizeRunEnd applies terminal state from TestRun to the existing run row.
func (r *PostgresRepository) FinalizeRunEnd(ctx context.Context, fields m.TestRun) error {
	if err := repository.ValidateRunID(fields.ID); err != nil {
		return err
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      fields.Status,
		"updated_at":  now,
		"finished_at": now,
	}

	if fields.StartTime != nil {
		updates["started_at"] = *fields.StartTime
	}
	if fields.EndTime != nil {
		updates["finished_at"] = *fields.EndTime
	}
	if fields.Duration != nil {
		updates["duration"] = *fields.Duration
	}

	result := r.db.WithContext(ctx).
		Model(&m.TestRun{}).
		Where("id = ?", fields.ID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("finalize run end: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("test run not found: %s", fields.ID)
	}

	r.logger.Info("test run finalized", "run_id", fields.ID, "status", fields.Status)
	return nil
}
