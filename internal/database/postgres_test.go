package database

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	embeddedmigrations "github.com/stanterprise/observer/migrations"
)

func TestValidatePostgresConfigRejectsSharedDBAutoMigrate(t *testing.T) {
	err := validatePostgresConfig(PostgresConfig{
		Env:               "production",
		EnableAutoMigrate: true,
	})
	if err == nil {
		t.Fatal("expected shared database auto-migrate validation to fail")
	}
}

func TestConnectPostgresWithMigratedSchemaKeepsCompositeSuiteAndTestRelations(t *testing.T) {
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "observer",
				"POSTGRES_USER":     "observer",
				"POSTGRES_PASSWORD": "password",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("postgres container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("postgres container mapped port: %v", err)
	}

	dsn := fmt.Sprintf("postgres://observer:password@%s:%s/observer?sslmode=disable", host, port.Port())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := embeddedmigrations.Up(dsn); err != nil {
		t.Fatalf("apply embedded migrations: %v", err)
	}

	connection, err := ConnectPostgresWithConfig(PostgresConfig{DSN: dsn}, logger)
	if err != nil {
		t.Fatalf("ConnectPostgresWithConfig failed: %v", err)
	}
	defer connection.Close()

	type columnRow struct {
		ColumnName string
	}
	var runExecutionColumns []columnRow
	if err := connection.DB.Raw(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'run_executions'
		ORDER BY ordinal_position
	`).Scan(&runExecutionColumns).Error; err != nil {
		t.Fatalf("load run_executions columns: %v", err)
	}
	for _, column := range runExecutionColumns {
		if column.ColumnName == "execution_id" {
			t.Fatal("run_executions should not include execution_id after migration")
		}
	}

	type constraintRow struct {
		TableName      string
		ConstraintName string
		ConstraintDef  string
	}
	var constraints []constraintRow
	if err := connection.DB.Raw(`
		SELECT t.relname AS table_name, c.conname AS constraint_name, pg_get_constraintdef(c.oid) AS constraint_def
		FROM pg_constraint c
		JOIN pg_class t ON c.conrelid = t.oid
		JOIN pg_namespace n ON t.relnamespace = n.oid
		WHERE n.nspname = 'public'
		  AND t.relname IN ('suites', 'tests', 'test_attempts')
		  AND c.conname IN ('suites_pkey', 'tests_pkey', 'fk_suites_suites', 'fk_suites_tests', 'fk_tests_attempts')
		ORDER BY t.relname, c.conname
	`).Scan(&constraints).Error; err != nil {
		t.Fatalf("load constraint defs: %v", err)
	}

	wantByName := map[string]string{
		"suites_pkey":       "PRIMARY KEY (id, run_id)",
		"tests_pkey":        "PRIMARY KEY (id, run_id)",
		"fk_suites_suites":  "FOREIGN KEY (parent_suite_id, run_id) REFERENCES suites(id, run_id)",
		"fk_suites_tests":   "FOREIGN KEY (suite_id, run_id) REFERENCES suites(id, run_id)",
		"fk_tests_attempts": "FOREIGN KEY (test_id, run_id) REFERENCES tests(id, run_id)",
	}

	if len(constraints) != len(wantByName) {
		t.Fatalf("constraint count = %d, want %d (%+v)", len(constraints), len(wantByName), constraints)
	}

	for _, constraint := range constraints {
		want, ok := wantByName[constraint.ConstraintName]
		if !ok {
			t.Fatalf("unexpected constraint %s on %s: %s", constraint.ConstraintName, constraint.TableName, constraint.ConstraintDef)
		}
		if constraint.ConstraintDef != want {
			t.Fatalf("constraint %s = %q, want %q", constraint.ConstraintName, constraint.ConstraintDef, want)
		}
		delete(wantByName, constraint.ConstraintName)
	}

	if len(wantByName) != 0 {
		t.Fatalf("missing constraints after migration: %+v", wantByName)
	}
}
