package postgres

import (
	"context"
	"fmt"

	m "github.com/stanterprise/observer/internal/models"
)

// GetRunMetadataByIDs returns run metadata keyed by run ID.
// It intentionally reads from runs to keep run_stats queries focused on stats only.
func (r *PostgresRepository) GetRunMetadataByIDs(ctx context.Context, runIDs []string) (map[string]map[string]interface{}, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	if len(runIDs) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	var runs []m.TestRun
	if err := r.db.WithContext(ctx).
		Model(&m.TestRun{}).
		Select("id", "metadata").
		Where("id IN ?", runIDs).
		Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("load run metadata: %w", err)
	}

	byRunID := make(map[string]map[string]interface{}, len(runs))
	for _, run := range runs {
		byRunID[run.ID] = run.Metadata
	}

	return byRunID, nil
}
