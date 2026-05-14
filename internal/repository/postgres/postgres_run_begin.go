package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/proto-go/testsystem/v1/events"
	"gorm.io/gorm"
)

// HandleRunStart handles the ReportRunStartEventRequest by creating or updating the TestRun and associated RunExecution, Suites, and Tests.
// It ensures that sharded run metadata is merged correctly and that existing records are updated rather than duplicated.
func (r *PostgresRepository) HandleRunStart(ctx context.Context, req *events.ReportRunStartEventRequest) error {
	if req == nil {
		return fmt.Errorf("run start request is nil")
	}

	testRun, runExecution, testSuites, testCases := m.RunStartEventToAllEntities(req)
	if testRun == nil {
		return fmt.Errorf("run start mapping produced nil run")
	}
	if runExecution == nil {
		return fmt.Errorf("run start mapping produced nil execution")
	}
	if err := repository.ValidateRunID(testRun.ID); err != nil {
		return err
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertRunStart(tx, testRun, now); err != nil {
			return err
		}
		if err := upsertRunExecutionStart(tx, runExecution, now); err != nil {
			return err
		}
		for _, suite := range testSuites {
			if suite == nil {
				continue
			}
			if err := upsertRunStartSuite(tx, suite, now); err != nil {
				return err
			}
		}
		for _, test := range testCases {
			if test == nil {
				continue
			}
			if err := upsertRelationalTest(tx, test, now); err != nil {
				return err
			}
		}
		if err := ensureRunStats(tx, testRun, now); err != nil {
			return err
		}
		if _, err := r.collectRunStats(ctx, tx, testRun.ID); err != nil {
			return err
		}
		if err := refreshLogicalRunAggregate(tx, testRun.ID, now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("run start persisted", "run_id", testRun.ID, "execution_id", runExecution.ID)
	return nil
}

func upsertRunStart(tx *gorm.DB, run *m.TestRun, now time.Time) error {
	var existing m.TestRun
	err := tx.Where("id = ?", run.ID).First(&existing).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("load existing run start %s: %w", run.ID, err)
		}

		create := *run
		if create.Status == "" {
			create.Status = "RUNNING"
		}
		create.CreatedAt = now
		create.UpdatedAt = now
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create run start %s: %w", run.ID, err)
		}
		*run = create
		return nil
	}

	merged := mergeRuns(existing, *run)
	if merged.Status == "" {
		merged.Status = "RUNNING"
	}
	merged.UpdatedAt = now
	if err := tx.Save(&merged).Error; err != nil {
		return fmt.Errorf("update run start %s: %w", run.ID, err)
	}
	*run = merged
	return nil
}

func ensureRunStats(tx *gorm.DB, run *m.TestRun, now time.Time) error {
	var stats m.RunStat
	err := tx.Where("run_id = ?", run.ID).First(&stats).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("load run stats for run start %s: %w", run.ID, err)
		}

		createdAt := now
		if run.StartTime != nil {
			createdAt = *run.StartTime
		}
		name := run.Name
		if name == "" {
			name = run.ID
		}

		create := m.RunStat{
			RunID:     run.ID,
			Name:      name,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create run stats for run start %s: %w", run.ID, err)
		}
		return nil
	}

	shouldSave := false
	if stats.Name == "" {
		if run.Name != "" {
			stats.Name = run.Name
		} else {
			stats.Name = run.ID
		}
		shouldSave = true
	}
	if stats.CreatedAt.IsZero() {
		if run.StartTime != nil {
			stats.CreatedAt = *run.StartTime
		} else {
			stats.CreatedAt = now
		}
		shouldSave = true
	}
	if shouldSave {
		stats.UpdatedAt = now
		if err := tx.Save(&stats).Error; err != nil {
			return fmt.Errorf("update run stats for run start %s: %w", run.ID, err)
		}
	}

	return nil
}

func mergeRuns(existing, incoming m.TestRun) m.TestRun {
	merged := existing

	if incoming.Name != "" {
		merged.Name = incoming.Name
	}
	if incoming.Status != "" {
		merged.Status = incoming.Status
	}
	if len(incoming.Metadata) > 0 {
		merged.Metadata = mergeRunStartMetadata(existing.Metadata, incoming.Metadata)
	}
	if incoming.StartTime != nil && (existing.StartTime == nil || incoming.StartTime.Before(*existing.StartTime)) {
		merged.StartTime = cloneTimePtr(incoming.StartTime)
	}
	if incoming.EndTime != nil && (existing.EndTime == nil || incoming.EndTime.After(*existing.EndTime)) {
		merged.EndTime = cloneTimePtr(incoming.EndTime)
	}
	if incoming.Duration != nil && (existing.Duration == nil || *incoming.Duration > *existing.Duration) {
		merged.Duration = cloneInt64Ptr(incoming.Duration)
	}

	return merged
}
