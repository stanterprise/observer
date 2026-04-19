package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
)

// UpsertRunStart upserts a TestRun row from a mapped run start event.
// On conflict the mutable fields (name, status, metadata, timing) are updated.
func (r *PostgresRepository) UpsertRunStart(ctx context.Context, run *m.TestRun) error {
	if run == nil {
		return fmt.Errorf("run is nil")
	}
	if err := repository.ValidateRunID(run.ID); err != nil {
		return err
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	run.CreatedAt = now
	run.UpdatedAt = now

	result := r.db.WithContext(ctx).
		Where(m.TestRun{ID: run.ID}).
		Assign(m.TestRun{
			Name:        run.Name,
			Status:      run.Status,
			TotalTests:  run.TotalTests,
			InitiatedBy: run.InitiatedBy,
			ProjectName: run.ProjectName,
			Metadata:    run.Metadata,
			StartTime:   run.StartTime,
			UpdatedAt:   now,
			CreatedAt:   now,
			Description: run.Description,
		}).
		FirstOrCreate(run)

	if result.Error != nil {
		return fmt.Errorf("upsert run start: %w", result.Error)
	}

	r.logger.Info("test run upserted", "run_id", run.ID)
	return nil
}

// UpsertRunShardStart upserts a run shard row derived from run-level shard metadata.
func (r *PostgresRepository) UpsertRunShardStart(ctx context.Context, shard *m.RunShard) error {
	if shard == nil {
		return fmt.Errorf("run shard is nil")
	}
	if err := repository.ValidateRunID(shard.RunID); err != nil {
		return err
	}
	if shard.ShardIndex == nil {
		return fmt.Errorf("shardIndex is required")
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	if shard.ID == "" {
		shard.ID = fmt.Sprintf("%s:%d", shard.RunID, *shard.ShardIndex)
	}
	if shard.StartTime == nil {
		shard.StartTime = &now
	}
	shard.CreatedAt = now
	shard.UpdatedAt = now

	result := r.db.WithContext(ctx).
		Where(m.RunShard{RunID: shard.RunID, ShardIndex: shard.ShardIndex}).
		Assign(m.RunShard{
			ID:                 shard.ID,
			ShardIndex:         shard.ShardIndex,
			ShardCountExpected: shard.ShardCountExpected,
			Status:             shard.Status,
			StartTime:          shard.StartTime,
			UpdatedAt:          now,
			CreatedAt:          now,
		}).
		FirstOrCreate(shard)

	if result.Error != nil {
		return fmt.Errorf("upsert run shard start: %w", result.Error)
	}

	r.logger.Info("run shard upserted", "run_id", shard.RunID, "shard_index", *shard.ShardIndex)
	return nil
}
