package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
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

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		assignment := m.TestRun{
			Name:        run.Name,
			Status:      "RUNNING",
			InitiatedBy: run.InitiatedBy,
			ProjectName: run.ProjectName,
			Metadata:    run.Metadata,
			StartTime:   run.StartTime,
			UpdatedAt:   now,
			CreatedAt:   now,
			Description: run.Description,
		}

		if isShardedRunStart(run.Metadata) {
			var existing m.TestRun
			err := tx.Where("id = ?", run.ID).First(&existing).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return fmt.Errorf("load existing run start: %w", err)
			}
			assignment.Metadata = mergeRunStartMetadata(existing.Metadata, run.Metadata)
			assignment.TotalTests = mergeRunStartTotalTests(existing.TotalTests, run.TotalTests, true)
		} else {
			assignment.TotalTests = mergeRunStartTotalTests(0, run.TotalTests, false)
		}

		result := tx.
			Where(m.TestRun{ID: run.ID}).
			Assign(assignment).
			FirstOrCreate(run)

		if result.Error != nil {
			return fmt.Errorf("upsert run start: %w", result.Error)
		}

		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("test run upserted", "run_id", run.ID)
	return nil
}

// UpsertRunStartSuites upserts suite rows emitted in the run-start payload.
func (r *PostgresRepository) UpsertRunStartSuites(ctx context.Context, suites []*m.Suite) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		for _, suite := range suites {
			if suite == nil {
				continue
			}
			if err := repository.ValidateRunID(suite.RunID); err != nil {
				return err
			}
			suite.CreatedAt = now
			suite.UpdatedAt = now

			result := tx.
				Where(m.Suite{ID: suite.ID}).
				Assign(m.Suite{
					RunID:           suite.RunID,
					ExternalSuiteID: suite.ExternalSuiteID,
					ParentSuiteID:   suite.ParentSuiteID,
					Name:            suite.Name,
					Description:     suite.Description,
					Status:          suite.Status,
					Metadata:        suite.Metadata,
					Duration:        suite.Duration,
					Location:        suite.Location,
					Type:            suite.Type,
					TestSuiteSpecID: suite.TestSuiteSpecID,
					InitiatedBy:     suite.InitiatedBy,
					ProjectName:     suite.ProjectName,
					Author:          suite.Author,
					Owner:           suite.Owner,
					TestCaseIDs:     suite.TestCaseIDs,
					SubSuiteIDs:     suite.SubSuiteIDs,
					Tags:            suite.Tags,
					StartTime:       suite.StartTime,
					EndTime:         suite.EndTime,
					UpdatedAt:       now,
					CreatedAt:       now,
				}).
				FirstOrCreate(suite)
			if result.Error != nil {
				return fmt.Errorf("upsert run start suite %s: %w", suite.ID, result.Error)
			}
		}

		return nil
	})
}

// UpsertRunStartTests upserts test rows emitted in the run-start payload.
func (r *PostgresRepository) UpsertRunStartTests(ctx context.Context, tests []*m.Test) error {
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		for _, test := range tests {
			if test == nil {
				continue
			}
			if err := repository.ValidateRunID(test.RunID); err != nil {
				return err
			}
			test.CreatedAt = now
			test.UpdatedAt = now

			result := tx.
				Where(m.Test{ID: test.ID}).
				Assign(m.Test{
					RunID:          test.RunID,
					ExternalTestID: test.ExternalTestID,
					SuiteID:        test.SuiteID,
					Name:           test.Name,
					Title:          test.Title,
					Description:    test.Description,
					Status:         test.Status,
					StartTime:      test.StartTime,
					EndTime:        test.EndTime,
					Duration:       test.Duration,
					Metadata:       test.Metadata,
					Tags:           test.Tags,
					Location:       test.Location,
					RetryCount:     test.RetryCount,
					RetryIndex:     test.RetryIndex,
					Timeout:        test.Timeout,
					UpdatedAt:      now,
					CreatedAt:      now,
				}).
				FirstOrCreate(test)
			if result.Error != nil {
				return fmt.Errorf("upsert run start test %s: %w", test.ID, result.Error)
			}
		}

		return nil
	})
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
		shard.ID = fmt.Sprintf("%s:%s:%d", shard.RunID, normalizeRepositoryExecutionID(shard.ExecutionID), *shard.ShardIndex)
	}
	if shard.StartTime == nil {
		shard.StartTime = &now
	}
	shard.CreatedAt = now
	shard.UpdatedAt = now

	result := r.db.WithContext(ctx).
		Where(m.RunShard{RunID: shard.RunID, ExecutionID: normalizeRepositoryExecutionID(shard.ExecutionID), ShardIndex: shard.ShardIndex}).
		Assign(m.RunShard{
			ID:                 shard.ID,
			ExecutionID:        normalizeRepositoryExecutionID(shard.ExecutionID),
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

	r.logger.Info("run shard upserted", "run_id", shard.RunID, "execution_id", normalizeRepositoryExecutionID(shard.ExecutionID), "shard_index", *shard.ShardIndex)
	return nil
}

func isShardedRunStart(metadata map[string]interface{}) bool {
	if metadata == nil {
		return false
	}
	_, hasTotal := metadata["shard.total"]
	_, hasCurrent := metadata["shard.current"]
	return hasTotal && hasCurrent
}

func mergeRunStartMetadata(existing, incoming map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(existing)+len(incoming))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range incoming {
		merged[key] = value
	}
	return merged
}

func mergeRunStartTotalTests(existing, incoming int32, sharded bool) int32 {
	if !sharded {
		return incoming
	}
	if incoming <= 0 {
		return existing
	}
	return existing + incoming
}
