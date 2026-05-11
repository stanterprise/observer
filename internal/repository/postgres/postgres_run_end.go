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

	shard.ID = m.BuildRunShardID(shard.RunID, shard.ExecutionID, shard.ShardIndex)

	if shard.EndTime == nil {
		shard.EndTime = &now
	}
	shard.CreatedAt = now
	shard.UpdatedAt = now

	var parentRunFinalized bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where(m.RunShard{RunID: shard.RunID, ExecutionID: shard.ExecutionID, ShardIndex: shard.ShardIndex}).
			Assign(m.RunShard{
				ID:                 shard.ID,
				ExecutionID:        shard.ExecutionID,
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
			Where("run_id = ? AND execution_id = ?", shard.RunID, shard.ExecutionID).
			Find(&shards).Error; err != nil {
			return fmt.Errorf("load run shards: %w", err)
		}

		if !allRunShardsFinished(shards, *shard.ShardCountExpected) {
			return nil
		}

		execution, ok := buildAggregatedExecutionFromShards(shard.RunID, shard.ExecutionID, shards, now)
		if !ok {
			return nil
		}

		if err := upsertRunExecutionEnd(tx, execution, now); err != nil {
			return fmt.Errorf("finalize run execution from shards: %w", err)
		}
		if err := refreshLogicalRunAggregate(tx, shard.RunID, now); err != nil {
			return err
		}

		if _, err := r.collectRunStats(ctx, tx, shard.RunID); err != nil {
			r.logger.Error("collect run stats: %w", err)
		}

		parentRunFinalized = true
		return nil
	})
	if err != nil {
		return false, err
	}

	r.logger.Info("run shard finalized", "run_id", shard.RunID, "execution_id", shard.ExecutionID, "shard_index", *shard.ShardIndex, "status", shard.Status, "parent_run_finalized", parentRunFinalized)
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

func allPersistedRunShardsFinished(shards []m.RunShard) bool {
	if len(shards) == 0 {
		return false
	}

	for _, shard := range shards {
		if shard.EndTime == nil {
			return false
		}
	}

	return true
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

func buildAggregatedExecutionFromShards(runID, executionID string, shards []m.RunShard, now time.Time) (*m.RunExecution, bool) {
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

	return &m.RunExecution{
		ID:        executionID,
		RunID:     runID,
		Status:    status,
		StartTime: startedAt,
		EndTime:   finishedAt,
		Duration:  duration,
		CreatedAt: now,
		UpdatedAt: now,
	}, true
}

func aggregateRunShardStatuses(shards []m.RunShard) string {
	if len(shards) == 0 {
		return "UNKNOWN"
	}

	statusCounts := make(map[string]int)
	for _, shard := range shards {
		statusCounts[shard.Status]++
	}

	switch {
	case statusCounts["PASSED"] == len(shards):
		return "PASSED"
	case statusCounts["RUNNING"] > 0:
		return "RUNNING"
	case statusCounts["NOT_RUN"] > 0:
		return "RUNNING"
	case statusCounts["FAILED"] > 0:
		return "FAILED"
	case statusCounts["TIMEDOUT"] > 0:
		return "TIMEDOUT"
	case statusCounts["INTERRUPTED"] > 0:
		return "INTERRUPTED"
	default:
		return "UNKNOWN"
	}
}
