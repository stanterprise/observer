package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	postgresdriver "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed *.sql
var files embed.FS

func Up(dsn string) error {
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
