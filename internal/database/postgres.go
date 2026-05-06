package database

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/stanterprise/observer/internal/models"
)

// PostgresConnection wraps a GORM DB connection to PostgreSQL.
type PostgresConnection struct {
	DB     *gorm.DB
	logger *slog.Logger
}

// ConnectPostgres opens a GORM connection to PostgreSQL using the provided DSN,
// verifies the connection, and runs AutoMigrate to ensure the schema is up to date.
func ConnectPostgres(dsn string, logger *slog.Logger) (*PostgresConnection, error) {
	if logger == nil {
		logger = slog.Default()
	}

	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	// Configure the underlying sql.DB connection pool.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	// Verify connectivity.
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := reconcileLegacyExecutionIDColumns(db); err != nil {
		return nil, fmt.Errorf("postgres legacy execution_id backfill: %w", err)
	}
	if err := reconcileLegacyAttemptIndexes(db); err != nil {
		return nil, fmt.Errorf("postgres legacy attempt index reconciliation: %w", err)
	}

	// Apply schema migrations so all tables exist in the correct state.
	if err := db.AutoMigrate(models.ModelsForMigration()...); err != nil {
		return nil, fmt.Errorf("postgres auto-migrate: %w", err)
	}

	logger.Info("connected to postgres and schema is up to date")

	return &PostgresConnection{
		DB:     db,
		logger: logger,
	}, nil
}

// ConnectPostgresFromEnv reads POSTGRES_DSN (or DATABASE_URL as a fallback)
// from the environment and opens a connection. Returns (nil, nil) if neither
// variable is set so callers can treat Postgres as optional.
func ConnectPostgresFromEnv(logger *slog.Logger) (*PostgresConnection, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		return nil, nil
	}

	return ConnectPostgres(dsn, logger)
}

// Close releases the underlying database connection pool.
func (p *PostgresConnection) Close() error {
	if p == nil || p.DB == nil {
		return nil
	}
	sqlDB, err := p.DB.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB for close: %w", err)
	}
	return sqlDB.Close()
}

func reconcileLegacyExecutionIDColumns(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("gorm db is nil")
	}

	targets := []struct {
		table  string
		column string
	}{
		{table: "run_shards", column: "execution_id"},
		{table: "test_attempts", column: "execution_id"},
	}

	migrator := db.Migrator()
	for _, target := range targets {
		if !migrator.HasTable(target.table) || !migrator.HasColumn(target.table, target.column) {
			continue
		}

		query := fmt.Sprintf(`UPDATE "%s" SET "%s" = '' WHERE "%s" IS NULL`, target.table, target.column, target.column)
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("backfill %s.%s: %w", target.table, target.column, err)
		}
	}

	return nil
}

func reconcileLegacyAttemptIndexes(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("gorm db is nil")
	}

	migrator := db.Migrator()
	if !migrator.HasTable("test_attempts") {
		return nil
	}
	if !migrator.HasIndex(&models.TestAttempt{}, "ux_attempts_test_execution_attempt_index") {
		return nil
	}
	if err := migrator.DropIndex(&models.TestAttempt{}, "ux_attempts_test_execution_attempt_index"); err != nil {
		return fmt.Errorf("drop legacy test_attempts unique index: %w", err)
	}

	return nil
}
