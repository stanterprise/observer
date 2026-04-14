package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
)

// MarkRunStarts updates an existing run to RUNNING and stamps start/update times.
func (r *PostgresRepository) MarkRunStarts(ctx context.Context, runID string) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":     "RUNNING",
		"started_at": now,
		"updated_at": now,
	}

	result := r.db.WithContext(ctx).
		Model(&m.TestRun{}).
		Where("id = ?", runID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("mark run starts: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("test run not found: %s", runID)
	}

	r.logger.Info("test run started", "run_id", runID)
	return nil
}
