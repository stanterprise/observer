package database

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/stanterprise/observer/internal/models"
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

func TestRelationalModelsAutoMigrateCreatesExpectedIndexes(t *testing.T) {
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

	connection, err := ConnectPostgresWithConfig(PostgresConfig{DSN: dsn}, logger)
	if err != nil {
		t.Fatalf("ConnectPostgresWithConfig failed: %v", err)
	}
	defer connection.Close()

	if err := connection.DB.AutoMigrate(models.ModelsForMigration()...); err != nil {
		t.Fatalf("auto-migrate models: %v", err)
	}

	type indexRow struct {
		TableName string
		IndexName string
		IsUnique  bool
		Columns   string
	}

	var indexes []indexRow
	if err := connection.DB.Raw(`
		SELECT
			t.relname AS table_name,
			i.relname AS index_name,
			ix.indisunique AS is_unique,
			string_agg(a.attname, ',' ORDER BY x.ordinality) AS columns
		FROM pg_class t
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_index ix ON ix.indrelid = t.oid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS x(attnum, ordinality) ON TRUE
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = x.attnum
		WHERE n.nspname = 'public'
		  AND t.relname IN ('runs', 'run_executions', 'suites', 'tests', 'test_attempts', 'attachments', 'run_stats')
		  AND i.relname NOT LIKE '%_pkey'
		GROUP BY t.relname, i.relname, ix.indisunique
		ORDER BY t.relname, i.relname
	`).Scan(&indexes).Error; err != nil {
		t.Fatalf("load auto-migrated indexes: %v", err)
	}

	gotByName := make(map[string]indexRow, len(indexes))
	for _, index := range indexes {
		gotByName[index.IndexName] = index
	}

	wantByName := map[string]indexRow{
		"idx_runs_status_started_at":                       {TableName: "runs", IndexName: "idx_runs_status_started_at", Columns: "status,started_at"},
		"idx_runs_started_at":                              {TableName: "runs", IndexName: "idx_runs_started_at", Columns: "started_at"},
		"idx_run_executions_run_status":                    {TableName: "run_executions", IndexName: "idx_run_executions_run_status", Columns: "run_id,status"},
		"idx_run_executions_started_at":                    {TableName: "run_executions", IndexName: "idx_run_executions_started_at", Columns: "started_at"},
		"idx_suites_run_id":                                {TableName: "suites", IndexName: "idx_suites_run_id", Columns: "run_id"},
		"idx_suites_run_status":                            {TableName: "suites", IndexName: "idx_suites_run_status", Columns: "run_id,status"},
		"idx_suites_run_external_suite_id":                 {TableName: "suites", IndexName: "idx_suites_run_external_suite_id", Columns: "run_id,external_suite_id"},
		"idx_tests_run_id":                                 {TableName: "tests", IndexName: "idx_tests_run_id", Columns: "run_id"},
		"idx_tests_suite_external_test_id":                 {TableName: "tests", IndexName: "idx_tests_suite_external_test_id", Columns: "suite_id,external_test_id"},
		"idx_tests_run_external_test_id":                   {TableName: "tests", IndexName: "idx_tests_run_external_test_id", Columns: "run_id,external_test_id"},
		"idx_tests_suite_status":                           {TableName: "tests", IndexName: "idx_tests_suite_status", Columns: "suite_id,status"},
		"idx_attempts_run_id":                              {TableName: "test_attempts", IndexName: "idx_attempts_run_id", Columns: "run_id"},
		"idx_attempts_execution_id":                        {TableName: "test_attempts", IndexName: "idx_attempts_execution_id", Columns: "execution_id"},
		"idx_attempts_test_attempt":                        {TableName: "test_attempts", IndexName: "idx_attempts_test_attempt", Columns: "test_id,attempt_index"},
		"idx_attempts_status_finished_at":                  {TableName: "test_attempts", IndexName: "idx_attempts_status_finished_at", Columns: "status,finished_at"},
		"ux_attempts_run_test_execution_attempt_index":     {TableName: "test_attempts", IndexName: "ux_attempts_run_test_execution_attempt_index", IsUnique: true, Columns: "run_id,test_id,execution_id,attempt_index"},
		"idx_attachments_run":                              {TableName: "attachments", IndexName: "idx_attachments_run", Columns: "run_id"},
		"idx_attachments_test":                             {TableName: "attachments", IndexName: "idx_attachments_test", Columns: "test_id"},
		"idx_attachments_attempt":                          {TableName: "attachments", IndexName: "idx_attachments_attempt", Columns: "test_attempt_id"},
		"idx_attachments_created_at":                       {TableName: "attachments", IndexName: "idx_attachments_created_at", Columns: "created_at"},
	}

	if len(gotByName) != len(wantByName) {
		gotNames := make([]string, 0, len(gotByName))
		for name := range gotByName {
			gotNames = append(gotNames, name)
		}
		sort.Strings(gotNames)

		wantNames := make([]string, 0, len(wantByName))
		for name := range wantByName {
			wantNames = append(wantNames, name)
		}
		sort.Strings(wantNames)

		t.Fatalf("auto-migrated index count = %d, want %d\ngot: %v\nwant: %v", len(gotByName), len(wantByName), gotNames, wantNames)
	}

	for name, want := range wantByName {
		got, ok := gotByName[name]
		if !ok {
			t.Fatalf("missing auto-migrated index %s", name)
		}
		if got.TableName != want.TableName || got.IsUnique != want.IsUnique || got.Columns != want.Columns {
			t.Fatalf("index %s = %+v, want %+v", name, got, want)
		}
		delete(gotByName, name)
	}

	if len(gotByName) != 0 {
		extraNames := make([]string, 0, len(gotByName))
		for name := range gotByName {
			extraNames = append(extraNames, name)
		}
		sort.Strings(extraNames)
		t.Fatalf("unexpected auto-migrated indexes: %v", extraNames)
	}
}
