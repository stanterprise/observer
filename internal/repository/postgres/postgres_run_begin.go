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
			if err := upsertRunStartSuite(tx, suite, now); err != nil {
				return err
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
			if err := upsertRunStartTest(tx, test, now); err != nil {
				return err
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

	shard.ID = m.BuildRunShardID(shard.RunID, shard.ExecutionID, shard.ShardIndex)

	if shard.StartTime == nil {
		shard.StartTime = &now
	}
	shard.CreatedAt = now
	shard.UpdatedAt = now

	result := r.db.WithContext(ctx).
		Where(m.RunShard{RunID: shard.RunID, ExecutionID: shard.ExecutionID, ShardIndex: shard.ShardIndex}).
		Assign(m.RunShard{
			ID:                 shard.ID,
			ExecutionID:        shard.ExecutionID,
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

	r.logger.Info("run shard upserted", "run_id", shard.RunID, "execution_id", shard.ExecutionID, "shard_index", *shard.ShardIndex)
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

func upsertRunStartSuite(tx *gorm.DB, suite *m.Suite, now time.Time) error {
	var stored m.Suite
	err := tx.Where("run_id = ? AND id = ?", suite.RunID, suite.ID).First(&stored).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("load run start suite %s: %w", suite.ID, err)
		}

		create := *suite
		create.CreatedAt = now
		create.UpdatedAt = now
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create run start suite %s: %w", suite.ID, err)
		}
		return nil
	}

	if suite.RunID != "" {
		stored.RunID = suite.RunID
	}
	if suite.ExternalSuiteID != "" {
		stored.ExternalSuiteID = suite.ExternalSuiteID
	}
	if suite.ParentSuiteID != nil {
		stored.ParentSuiteID = suite.ParentSuiteID
	}
	if suite.Name != "" {
		stored.Name = suite.Name
	}
	if suite.Description != "" {
		stored.Description = suite.Description
	}
	stored.Status = mergeRunStartEntityStatus(stored.Status, suite.Status)
	if len(suite.Metadata) > 0 {
		stored.Metadata = mergeRunStartMetadata(stored.Metadata, suite.Metadata)
	}
	if suite.Duration != nil {
		stored.Duration = cloneInt64Ptr(suite.Duration)
	}
	if suite.Location != "" {
		stored.Location = suite.Location
	}
	if suite.Type != "" {
		stored.Type = suite.Type
	}
	if suite.TestSuiteSpecID != "" {
		stored.TestSuiteSpecID = suite.TestSuiteSpecID
	}
	if suite.InitiatedBy != "" {
		stored.InitiatedBy = suite.InitiatedBy
	}
	if suite.ProjectName != "" {
		stored.ProjectName = suite.ProjectName
	}
	if suite.Author != "" {
		stored.Author = suite.Author
	}
	if suite.Owner != "" {
		stored.Owner = suite.Owner
	}
	if len(suite.TestCaseIDs) > 0 {
		stored.TestCaseIDs = append([]string(nil), suite.TestCaseIDs...)
	}
	if len(suite.SubSuiteIDs) > 0 {
		stored.SubSuiteIDs = append([]string(nil), suite.SubSuiteIDs...)
	}
	if len(suite.Tags) > 0 {
		stored.Tags = append([]string(nil), suite.Tags...)
	}
	if suite.StartTime != nil && (stored.StartTime == nil || suite.StartTime.Before(*stored.StartTime)) {
		stored.StartTime = cloneTimePtr(suite.StartTime)
	}
	if suite.EndTime != nil && (stored.EndTime == nil || suite.EndTime.After(*stored.EndTime)) {
		stored.EndTime = cloneTimePtr(suite.EndTime)
	}
	stored.UpdatedAt = now

	if err := tx.Save(&stored).Error; err != nil {
		return fmt.Errorf("update run start suite %s: %w", suite.ID, err)
	}
	return nil
}

func upsertRunStartTest(tx *gorm.DB, test *m.Test, now time.Time) error {
	var stored m.Test
	err := tx.Where("run_id = ? AND id = ?", test.RunID, test.ID).First(&stored).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("load run start test %s: %w", test.ID, err)
		}

		create := *test
		create.CreatedAt = now
		create.UpdatedAt = now
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create run start test %s: %w", test.ID, err)
		}
		return nil
	}

	if test.RunID != "" {
		stored.RunID = test.RunID
	}
	if test.ExternalTestID != "" {
		stored.ExternalTestID = test.ExternalTestID
	}
	if test.SuiteID != nil {
		stored.SuiteID = test.SuiteID
	}
	if test.Name != "" {
		stored.Name = test.Name
	}
	if test.Title != "" {
		stored.Title = test.Title
	}
	if test.Description != "" {
		stored.Description = test.Description
	}
	stored.Status = mergeRunStartEntityStatus(stored.Status, test.Status)
	if test.StartTime != nil && (stored.StartTime == nil || test.StartTime.Before(*stored.StartTime)) {
		stored.StartTime = cloneTimePtr(test.StartTime)
	}
	if test.EndTime != nil && (stored.EndTime == nil || test.EndTime.After(*stored.EndTime)) {
		stored.EndTime = cloneTimePtr(test.EndTime)
	}
	if test.Duration != nil {
		stored.Duration = cloneInt64Ptr(test.Duration)
	}
	if len(test.Metadata) > 0 {
		stored.Metadata = mergeRunStartMetadata(stored.Metadata, test.Metadata)
	}
	if len(test.Tags) > 0 {
		stored.Tags = append([]string(nil), test.Tags...)
	}
	if test.Location != "" {
		stored.Location = test.Location
	}
	if test.RetryCount != nil {
		stored.RetryCount = test.RetryCount
	}
	if test.RetryIndex != nil {
		stored.RetryIndex = test.RetryIndex
	}
	if test.Timeout != nil {
		stored.Timeout = test.Timeout
	}
	stored.UpdatedAt = now

	if err := tx.Save(&stored).Error; err != nil {
		return fmt.Errorf("update run start test %s: %w", test.ID, err)
	}
	return nil
}

func mergeRunStartEntityStatus(existing, incoming string) string {
	if incoming == "" {
		return existing
	}
	if isRunStartPlaceholderStatus(incoming) && !isRunStartPlaceholderStatus(existing) {
		return existing
	}
	return incoming
}

func isRunStartPlaceholderStatus(status string) bool {
	switch status {
	case "", "NOT_RUN", "UNKNOWN":
		return true
	default:
		return false
	}
}
