package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
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

// FinalizeRunShardEnd upserts the terminal state of a run shard.
// When all expected shards have finished, it also upserts the parent run row.
func (r *PostgresRepository) FinalizeRunShardEnd(ctx context.Context, shard *m.RunShard) (bool, error) {
	if shard == nil {
		return false, fmt.Errorf("run shard is nil")
	}
	if err := repository.ValidateRunID(shard.RunID); err != nil {
		return false, err
	}
	if shard.ShardIndex == nil {
		return false, fmt.Errorf("shardIndex is required")
	}
	if err := r.ensureDB(); err != nil {
		return false, err
	}
	if shard.ShardCountExpected == nil || *shard.ShardCountExpected <= 0 {
		return false, fmt.Errorf("shardCountExpected is required")
	}

	now := time.Now()
	if shard.ID == "" {
		shard.ID = fmt.Sprintf("%s:%d", shard.RunID, *shard.ShardIndex)
	}
	if shard.EndTime == nil {
		shard.EndTime = &now
	}
	shard.CreatedAt = now
	shard.UpdatedAt = now

	var parentRunFinalized bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where(m.RunShard{RunID: shard.RunID, ShardIndex: shard.ShardIndex}).
			Assign(m.RunShard{
				ID:                 shard.ID,
				ShardIndex:         shard.ShardIndex,
				ShardCountExpected: shard.ShardCountExpected,
				Status:             shard.Status,
				StartTime:          shard.StartTime,
				EndTime:            shard.EndTime,
				UpdatedAt:          now,
				CreatedAt:          now,
			}).
			FirstOrCreate(shard)

		if result.Error != nil {
			return fmt.Errorf("finalize run shard end: %w", result.Error)
		}

		var shards []m.RunShard
		if err := tx.
			Where("run_id = ?", shard.RunID).
			Find(&shards).Error; err != nil {
			return fmt.Errorf("load run shards: %w", err)
		}

		if !allRunShardsFinished(shards, *shard.ShardCountExpected) {
			return nil
		}

		parentRun, ok := buildAggregatedRunFromShards(shard.RunID, shards, now)
		if !ok {
			return nil
		}

		result = tx.
			Where(m.TestRun{ID: parentRun.ID}).
			Assign(m.TestRun{
				Status:    parentRun.Status,
				StartTime: parentRun.StartTime,
				EndTime:   parentRun.EndTime,
				Duration:  parentRun.Duration,
				UpdatedAt: now,
				CreatedAt: now,
			}).
			FirstOrCreate(parentRun)
		if result.Error != nil {
			return fmt.Errorf("finalize parent run from shards: %w", result.Error)
		}

		parentRunFinalized = true
		return nil
	})
	if err != nil {
		return false, err
	}

	r.logger.Info("run shard finalized", "run_id", shard.RunID, "shard_index", *shard.ShardIndex, "status", shard.Status, "parent_run_finalized", parentRunFinalized)
	return parentRunFinalized, nil
}

func allRunShardsFinished(shards []m.RunShard, expectedCount int32) bool {
	if expectedCount <= 0 || int32(len(shards)) < expectedCount {
		return false
	}

	finishedCount := int32(0)
	for _, shard := range shards {
		if shard.EndTime != nil {
			finishedCount++
		}
	}

	return finishedCount >= expectedCount
}

func buildAggregatedRunFromShards(runID string, shards []m.RunShard, now time.Time) (*m.TestRun, bool) {
	if len(shards) == 0 {
		return nil, false
	}

	status := aggregateRunShardStatuses(shards)
	if status == "" {
		return nil, false
	}

	var startedAt *time.Time
	var finishedAt *time.Time
	for _, shard := range shards {
		if shard.StartTime != nil && (startedAt == nil || shard.StartTime.Before(*startedAt)) {
			t := *shard.StartTime
			startedAt = &t
		}
		if shard.EndTime != nil && (finishedAt == nil || shard.EndTime.After(*finishedAt)) {
			t := *shard.EndTime
			finishedAt = &t
		}
	}

	var duration *int64
	if startedAt != nil && finishedAt != nil {
		d := finishedAt.Sub(*startedAt).Nanoseconds()
		duration = &d
	}

	return &m.TestRun{
		ID:        runID,
		Status:    status,
		StartTime: startedAt,
		EndTime:   finishedAt,
		Duration:  duration,
		CreatedAt: now,
		UpdatedAt: now,
	}, true
}

func aggregateRunShardStatuses(shards []m.RunShard) string {
	statusCounts := make(map[string]int)
	for _, shard := range shards {
		statusCounts[shard.Status]++
	}

	switch {
	case statusCounts["FAILED"] > 0:
		return "FAILED"
	case statusCounts["BROKEN"] > 0:
		return "BROKEN"
	case statusCounts["TIMEDOUT"] > 0:
		return "TIMEDOUT"
	case statusCounts["INTERRUPTED"] > 0:
		return "INTERRUPTED"
	case statusCounts["RUNNING"] > 0:
		return "RUNNING"
	case statusCounts["PASSED"] > 0:
		return "PASSED"
	case statusCounts["SKIPPED"] > 0:
		return "SKIPPED"
	case statusCounts["NOT_RUN"] > 0:
		return "NOT_RUN"
	default:
		return "UNKNOWN"
	}
}
