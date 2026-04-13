package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
)

// UpdateTestRunEnd updates run terminal status and timing fields.
func (r *PostgresRepository) UpdateTestRunEnd(ctx context.Context, runID string, status string, startTime *time.Time, duration *int64) error {
	if err := ValidateRunID(runID); err != nil {
		return err
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      status,
		"updated_at":  now,
		"finished_at": now,
	}

	if startTime != nil {
		updates["started_at"] = *startTime
	}
	if duration != nil {
		updates["duration"] = *duration
	}

	result := r.db.WithContext(ctx).
		Model(&m.TestRun{}).
		Where("id = ?", runID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("update test run end: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("test run not found: %s", runID)
	}

	r.logger.Info("test run end updated", "run_id", runID, "status", status)
	return nil
}
