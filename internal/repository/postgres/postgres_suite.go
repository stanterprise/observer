package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)

func (r *PostgresRepository) UpsertSuite(ctx context.Context, suite *m.Suite) error {
	if suite == nil {
		return fmt.Errorf("suite is nil")
	}
	if err := repository.ValidateRunID(suite.RunID); err != nil {
		return err
	}
	if suite.ID == "" {
		return fmt.Errorf("suite id is required")
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return upsertRunStartSuite(tx, suite, time.Now())
	})
}
