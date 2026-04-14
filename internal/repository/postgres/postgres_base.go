package postgres

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// PostgresRepository handles PostgreSQL operations for relational execution data.
type PostgresRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(db *gorm.DB, logger *slog.Logger) *PostgresRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &PostgresRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PostgresRepository) ensureDB() error {
	if r == nil || r.db == nil {
		return fmt.Errorf("postgres database is not configured")
	}
	return nil
}
