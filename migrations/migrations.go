package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	postgresdriver "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed *.sql
var files embed.FS

const initialMigrationVersion = 1

func Up(dsn string) error {
	if err := baselineLegacySchemaIfNeeded(dsn); err != nil {
		return err
	}

	return run(dsn, func(m *migrate.Migrate) error {
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("apply migrations: %w", err)
		}
		return nil
	})
}

func Down(dsn string) error {
	return run(dsn, func(m *migrate.Migrate) error {
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("rollback migrations: %w", err)
		}
		return nil
	})
}

func Force(dsn string, version int) error {
	return run(dsn, func(m *migrate.Migrate) error {
		if err := m.Force(version); err != nil {
			return fmt.Errorf("force migration version %d: %w", version, err)
		}
		return nil
	})
}

func run(dsn string, apply func(*migrate.Migrate) error) (err error) {
	m, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil || dbErr != nil {
			err = errors.Join(err, sourceErr, dbErr)
		}
	}()

	return apply(m)
}

func newMigrator(dsn string) (*migrate.Migrate, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}

	sourceDriver, err := iofs.New(files, ".")
	if err != nil {
		return nil, fmt.Errorf("open embedded migrations: %w", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres sql db: %w", err)
	}

	// Use the default postgres migrate driver configuration for the public schema.
	targetDriver, err := postgresdriver.WithInstance(db, &postgresdriver.Config{})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create postgres migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", targetDriver)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return m, nil
}

func baselineLegacySchemaIfNeeded(dsn string) (err error) {
	if dsn == "" {
		return fmt.Errorf("postgres dsn is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres sql db for legacy schema baseline: %w", err)
	}
	defer func() {
		err = errors.Join(err, db.Close())
	}()

	hasMigrationsTable, err := hasTable(db, "schema_migrations")
	if err != nil {
		return err
	}
	if hasMigrationsTable {
		return nil
	}

	hasLegacySchema, err := hasAllTables(db,
		"runs",
		"run_executions",
		"run_shards",
		"suites",
		"tests",
		"test_attempts",
		"attachments",
	)
	if err != nil {
		return err
	}
	if !hasLegacySchema {
		return nil
	}

	if err := Force(dsn, initialMigrationVersion); err != nil {
		return fmt.Errorf("baseline existing postgres schema at version %d: %w", initialMigrationVersion, err)
	}

	return nil
}

func hasTable(db *sql.DB, tableName string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = $1
		)
	`, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check for table %q: %w", tableName, err)
	}

	return exists, nil
}

func hasAllTables(db *sql.DB, tableNames ...string) (bool, error) {
	if len(tableNames) == 0 {
		return false, nil
	}

	placeholders := make([]string, 0, len(tableNames))
	args := make([]any, 0, len(tableNames))
	for index, tableName := range tableNames {
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
		args = append(args, tableName)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = current_schema()
		  AND table_name IN (%s)
	`, strings.Join(placeholders, ", "))

	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("check for legacy postgres schema tables: %w", err)
	}

	return count == len(tableNames), nil
}
