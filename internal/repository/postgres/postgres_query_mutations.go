package postgres

import (
	"context"
	"fmt"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)

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
