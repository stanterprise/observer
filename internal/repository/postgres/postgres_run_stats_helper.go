package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type runStatusCount struct {
	Status string
	Count  int64
}

func mapStatusToRunStatsColumn(status string) string {
	switch status {
	case "PASSED":
		return "passed"
	case "FAILED":
		return "failed"
	case "SKIPPED":
		return "skipped"
	case "FLAKY":
		return "flaky"
	case "BROKEN":
		return "broken"
	case "TIMEDOUT":
		return "timedout"
	case "INTERRUPTED":
		return "interrupted"
	case "NOT_RUN":
		return "not_run"
	case "RUNNING", "":
		return "running"
	default:
		return "unknown"
	}
}

func (r *PostgresRepository) collectRunStats(ctx context.Context, tx *gorm.DB, runID string) (map[string]interface{}, error) {
	var grouped []runStatusCount
	if err := tx.WithContext(ctx).
		Table("tests").
		Select("status, count(*) as count").
		Where("run_id = ?", runID).
		Group("status").
		Scan(&grouped).Error; err != nil {
		return nil, err
	}

	// Always set a complete stats payload so old counts are cleared.
	stats := map[string]int64{
		"total":       0,
		"passed":      0,
		"failed":      0,
		"skipped":     0,
		"flaky":       0,
		"broken":      0,
		"timedout":    0,
		"interrupted": 0,
		"unknown":     0,
		"not_run":     0,
		"running":     0,
	}

	for _, entry := range grouped {
		column := mapStatusToRunStatsColumn(entry.Status)
		stats[column] += entry.Count
		stats["total"] += entry.Count
	}

	// Build update payload with stats and timestamp.
	updatePayload := map[string]interface{}{
		"updated_at": time.Now(),
	}
	for k, v := range stats {
		updatePayload[k] = v
	}

	update := tx.WithContext(ctx).
		Table("public.run_stats").
		Where("run_id = ?", runID).
		Updates(updatePayload)
	if update.Error != nil {
		return nil, update.Error
	}
	if update.RowsAffected == 0 {
		return nil, fmt.Errorf("run_stats row not found for run_id %q", runID)
	}

	// Return stats for caller reference.
	result := make(map[string]interface{})
	for k, v := range stats {
		result[k] = v
	}
	result["updated_at"] = updatePayload["updated_at"]
	return result, nil
}
