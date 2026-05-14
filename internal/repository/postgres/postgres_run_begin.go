package postgres

import (
	"context"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/proto-go/testsystem/v1/events"
	"gorm.io/gorm"
)

// HandleRunStart handles the ReportRunStartEventRequest by creating or updating the TestRun and associated RunExecution, Suites, and Tests.
// It ensures that sharded run metadata is merged correctly and that existing records are updated rather than duplicated.
func (r *PostgresRepository) HandleRunStart(ctx context.Context, req *events.ReportRunStartEventRequest) error {
	testRun, runExecution, testSuites, testCases := m.RunStartEventToAllEntities(req) // map event to all entities (test run, execution, suites, tests)

	now := time.Now()

	// check if run exists
	var existing m.TestRun
	err := r.db.Where("id = ?", testRun.ID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		r.logger.Error("load existing run start", testRun.ID, err)
	}
	if err == gorm.ErrRecordNotFound {
		testRun.CreatedAt = time.Now()
		testRun.UpdatedAt = testRun.CreatedAt
		if err := r.db.Create(&testRun).Error; err != nil {
			r.logger.Error("create run start", testRun.ID, err)
		}
	} else {
		existing.UpdatedAt = now
		if testRun != nil {
			merged := mergeRuns(existing, *testRun) // merge existing and incoming run data, with incoming taking precedence
			testRun = &merged
		} else {
			testRun = &existing
		}
		if err := r.db.Save(&testRun).Error; err != nil {
			r.logger.Error("update run start", testRun.ID, err)
		}
	}

	r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&runExecution).Error; err != nil {
			tx.Logger.Error(ctx, "create run execution for run start", testRun.ID, runExecution.ID, err)
		}

		// create suites and tests if they don't exist
		for _, suite := range testSuites {
			var existingSuite m.Suite
			err := tx.Where("run_id = ? AND id = ?", suite.RunID, suite.ID).First(&existingSuite).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					tx.Logger.Error(ctx, "load existing run start suite", suite.ID, err)
					continue
				}
				suite.CreatedAt = now
				suite.UpdatedAt = now
				if err := tx.Create(&suite).Error; err != nil {
					tx.Logger.Error(ctx, "create run start suite", suite.ID, err)
				}
			}
		}

		for _, test := range testCases {
			var existingTest m.Test
			err := tx.Where("run_id = ? AND id = ?", test.RunID, test.ID).First(&existingTest).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					tx.Logger.Error(ctx, "load existing run start test", test.ID, err)
					continue
				}
				test.CreatedAt = now
				test.UpdatedAt = now
				if err := tx.Create(&test).Error; err != nil {
					tx.Logger.Error(ctx, "create run start test", test.ID, err)
				}
			}
		}

		// create run stats record
		stats, err := r.collectRunStats(ctx, tx, testRun.ID)
		if err != nil {
			tx.Logger.Error(ctx, "create run stats for run start", testRun.ID, err)
		}
		if err := tx.Create(&stats).Error; err != nil {
			tx.Logger.Error(ctx, "create run stats for run start", testRun.ID, err)
		}
		return nil
	})

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
