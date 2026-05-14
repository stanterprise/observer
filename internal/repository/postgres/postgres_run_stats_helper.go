package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
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

func generateRunStatWithStatusCounts(runId string, grouped []runStatusCount) *m.RunStat {
	record := &m.RunStat{
		RunID:       runId,
		Total:       0,
		Passed:      0,
		Failed:      0,
		Skipped:     0,
		Flaky:       0,
		Broken:      0,
		TimedOut:    0,
		Interrupted: 0,
		Unknown:     0,
		NotRun:      0,
		Running:     0,
		Duration:    0,
	}

	for _, entry := range grouped {
		column := mapStatusToRunStatsColumn(entry.Status)
		switch column {
		case "passed":
			record.Passed += int32(entry.Count)
		case "failed":
			record.Failed += int32(entry.Count)
		case "skipped":
			record.Skipped += int32(entry.Count)
		case "flaky":
			record.Flaky += int32(entry.Count)
		case "broken":
			record.Broken += int32(entry.Count)
		case "timedout":
			record.TimedOut += int32(entry.Count)
		case "interrupted":
			record.Interrupted += int32(entry.Count)
		case "not_run":
			record.NotRun += int32(entry.Count)
		case "running":
			record.Running += int32(entry.Count)
		default:
			record.Unknown += int32(entry.Count)
		}
		record.Total += int32(entry.Count)
	}

	return record
}

func (r *PostgresRepository) collectRunStats(ctx context.Context, tx *gorm.DB, runID string) (*m.RunStat, error) {
	var grouped []runStatusCount
	if err := tx.WithContext(ctx).
		Table("tests").
		Select("status, count(*) as count").
		Where("run_id = ?", runID).
		Group("status").
		Scan(&grouped).Error; err != nil {
		return nil, err
	}

	record := generateRunStatWithStatusCounts(runID, grouped)

	if err := tx.WithContext(ctx).
		Table("run_stats").
		Where("run_id = ?", runID).
		Scan(&record).Error; err != nil {
		return nil, fmt.Errorf("fetching run_stats created_at for run_id %q: %w", runID, err)
	}

	if !record.CreatedAt.IsZero() {
		now := time.Now()
		record.Duration = now.Sub(record.CreatedAt).Milliseconds()
	}

	update := tx.WithContext(ctx).Save(record)
	if update.Error != nil {
		return nil, update.Error
	}
	if update.RowsAffected == 0 {
		return nil, fmt.Errorf("run_stats row not found for run_id %q", runID)
	}

	return record, nil
}
