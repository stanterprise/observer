package database

import (
	"context"
	"fmt"
)

// schemaSQL contains the idempotent DDL for all PostgreSQL tables.
// Tables are created with IF NOT EXISTS so this can safely run on every startup.
// No migration of existing MongoDB data occurs — this is for NEW runs only.
const schemaSQL = `
-- runs: top-level logical test run
CREATE TABLE IF NOT EXISTS runs (
    id              TEXT PRIMARY KEY,
    logical_run_key TEXT NOT NULL,
    source          TEXT NOT NULL DEFAULT '',
    project         TEXT NOT NULL DEFAULT '',
    pipeline        TEXT NOT NULL DEFAULT '',
    branch          TEXT NOT NULL DEFAULT '',
    commit_sha      TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'RUNNING',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at     TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_runs_logical_run_key ON runs (logical_run_key);
CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_status_started_at ON runs (status, started_at DESC);

-- run_shards: individual shard within a logical run
CREATE TABLE IF NOT EXISTS run_shards (
    id                   TEXT PRIMARY KEY,
    run_id               TEXT NOT NULL REFERENCES runs(id),
    shard_key            TEXT NOT NULL DEFAULT '',
    shard_index          INT NOT NULL DEFAULT 0,
    shard_count_expected INT,
    status               TEXT NOT NULL DEFAULT 'RUNNING',
    started_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_run_shards_run_shard ON run_shards (run_id, shard_key);
CREATE INDEX IF NOT EXISTS idx_run_shards_run_status ON run_shards (run_id, status);

-- suites: test suites belonging to a run
CREATE TABLE IF NOT EXISTS suites (
    id                TEXT PRIMARY KEY,
    run_id            TEXT NOT NULL REFERENCES runs(id),
    external_suite_id TEXT NOT NULL DEFAULT '',
    name              TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'RUNNING',
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at       TIMESTAMPTZ,
    metadata          JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_suites_run_id ON suites (run_id);
CREATE INDEX IF NOT EXISTS idx_suites_run_status ON suites (run_id, status);

-- tests: individual test cases belonging to a suite
CREATE TABLE IF NOT EXISTS tests (
    id               TEXT PRIMARY KEY,
    suite_id         TEXT NOT NULL REFERENCES suites(id),
    external_test_id TEXT NOT NULL DEFAULT '',
    name             TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'RUNNING',
    attempt_count    INT NOT NULL DEFAULT 0,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ,
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tests_suite_external ON tests (suite_id, external_test_id);
CREATE INDEX IF NOT EXISTS idx_tests_suite_status ON tests (suite_id, status);

-- test_attempts: individual attempts/retries of a test
CREATE TABLE IF NOT EXISTS test_attempts (
    id              TEXT PRIMARY KEY,
    test_id         TEXT NOT NULL REFERENCES tests(id),
    attempt_index   INT NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'RUNNING',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at     TIMESTAMPTZ,
    steps           JSONB,
    steps_ref       TEXT,
    step_count      INT NOT NULL DEFAULT 0,
    duration_ms     BIGINT NOT NULL DEFAULT 0,
    failure_reason  TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_test_attempts_test_attempt UNIQUE (test_id, attempt_index),
    CONSTRAINT chk_steps_exclusive CHECK (
        NOT (steps IS NOT NULL AND steps_ref IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_test_attempts_test_attempt ON test_attempts (test_id, attempt_index);
CREATE INDEX IF NOT EXISTS idx_test_attempts_status_finished ON test_attempts (status, finished_at DESC);
`

// InitSchema runs the idempotent schema creation DDL against the given connection.
// Safe to call on every application startup.
func (p *PostgresConnection) InitSchema(ctx context.Context) error {
	if _, err := p.Pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("init postgres schema: %w", err)
	}
	p.logger.Info("postgres schema initialized")
	return nil
}
