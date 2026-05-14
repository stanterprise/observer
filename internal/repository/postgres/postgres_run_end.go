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

func (r *PostgresRepository) HandleRunEnd(ctx context.Context, event *events.TestRunEndEventRequest) error {
	if event == nil {
		return fmt.Errorf("run end request is nil")
	}

	runFields := m.RunEndEventToTestRun(event)
	execution := m.RunEndEventToRunExecution(event)
	if execution == nil {
		return fmt.Errorf("run end mapping produced nil execution")
	}
	if err := repository.ValidateRunID(runFields.ID); err != nil {
		return err
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	now := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertRunExecutionEnd(tx, execution, now); err != nil {
			return err
		}
		if err := refreshLogicalRunAggregate(tx, runFields.ID, now); err != nil {
			return err
		}

		statsRun := runFields
		var storedRun m.TestRun
		if err := tx.Where("id = ?", runFields.ID).First(&storedRun).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("load run after run end %s: %w", runFields.ID, err)
			}
		} else {
			statsRun = storedRun
		}

		if err := ensureRunStats(tx, &statsRun, now); err != nil {
			return err
		}
		if _, err := r.collectRunStats(ctx, tx, runFields.ID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	r.logger.Info("run end persisted", "run_id", execution.RunID, "execution_id", execution.ID, "status", execution.Status)
	return nil
}
