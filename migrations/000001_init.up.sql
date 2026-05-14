CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    name TEXT,
    description TEXT,
    status TEXT,
    metadata JSONB,
    duration BIGINT,
    initiated_by TEXT,
    project_name TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runs_status_started_at ON runs (status, started_at);
CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs (started_at);

CREATE TABLE IF NOT EXISTS run_executions (
    id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    name TEXT,
    status TEXT,
    metadata JSONB,
    total_tests INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT run_executions_pkey PRIMARY KEY (id, run_id),
    CONSTRAINT fk_runs_executions FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_executions_run_status ON run_executions (run_id, status);
CREATE INDEX IF NOT EXISTS idx_run_executions_started_at ON run_executions (started_at);

CREATE TABLE IF NOT EXISTS suites (
    id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    external_suite_id TEXT,
    parent_suite_id TEXT,
    name TEXT,
    description TEXT,
    status TEXT,
    metadata JSONB,
    duration BIGINT,
    location TEXT,
    type TEXT,
    test_suite_spec_id TEXT,
    initiated_by TEXT,
    project_name TEXT,
    author TEXT,
    owner TEXT,
    test_case_ids JSONB,
    sub_suite_ids JSONB,
    tags JSONB,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT suites_pkey PRIMARY KEY (id, run_id),
    CONSTRAINT fk_runs_suites FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_suites_suites FOREIGN KEY (parent_suite_id, run_id) REFERENCES suites(id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_suites_run_id ON suites (run_id);
CREATE INDEX IF NOT EXISTS idx_suites_run_status ON suites (run_id, status);
CREATE INDEX IF NOT EXISTS idx_suites_run_external_suite_id ON suites (run_id, external_suite_id);

CREATE TABLE IF NOT EXISTS tests (
    id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    external_test_id TEXT,
    suite_id TEXT NOT NULL,
    name TEXT,
    title TEXT,
    description TEXT,
    status TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration BIGINT,
    metadata JSONB,
    tags JSONB,
    location TEXT,
    retry_count INTEGER,
    retry_index INTEGER,
    timeout INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT tests_pkey PRIMARY KEY (id, run_id),
    CONSTRAINT fk_runs_tests FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_suites_tests FOREIGN KEY (suite_id, run_id) REFERENCES suites(id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_tests_run_id ON tests (run_id);
CREATE INDEX IF NOT EXISTS idx_tests_suite_external_test_id ON tests (suite_id, external_test_id);
CREATE INDEX IF NOT EXISTS idx_tests_run_external_test_id ON tests (run_id, external_test_id);
CREATE INDEX IF NOT EXISTS idx_tests_suite_status ON tests (suite_id, status);

CREATE TABLE IF NOT EXISTS test_attempts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    execution_id TEXT NOT NULL DEFAULT '',
    test_id TEXT NOT NULL,
    attempt_index INTEGER NOT NULL,
    status TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration BIGINT,
    steps JSONB,
    steps_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB,
    attachments JSONB,
    error_message TEXT,
    stack_trace TEXT,
    error_list JSONB,
    failures JSONB,
    errors JSONB,
    stdout JSONB,
    stderr JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_runs_attempts FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_tests_attempts FOREIGN KEY (test_id, run_id) REFERENCES tests(id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_attempts_run_id ON test_attempts (run_id);
CREATE INDEX IF NOT EXISTS idx_attempts_execution_id ON test_attempts (execution_id);
CREATE INDEX IF NOT EXISTS idx_attempts_test_attempt ON test_attempts (test_id, attempt_index);
CREATE INDEX IF NOT EXISTS idx_attempts_status_finished_at ON test_attempts (status, finished_at);
CREATE UNIQUE INDEX IF NOT EXISTS ux_attempts_run_test_execution_attempt_index ON test_attempts (run_id, test_id, execution_id, attempt_index);

CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    test_id TEXT NOT NULL,
    test_attempt_id TEXT NOT NULL,
    step_id TEXT,
    kind TEXT,
    name TEXT,
    content_type TEXT,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_key TEXT,
    checksum TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_runs_attachments FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE,
    CONSTRAINT fk_tests_attachments FOREIGN KEY (test_id, run_id) REFERENCES tests(id, run_id) ON DELETE CASCADE,
    CONSTRAINT fk_attempts_attachments FOREIGN KEY (test_attempt_id) REFERENCES test_attempts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_stats (
    run_id TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    total INTEGER NOT NULL DEFAULT 0,
    passed INTEGER NOT NULL DEFAULT 0,
    failed INTEGER NOT NULL DEFAULT 0,
    skipped INTEGER NOT NULL DEFAULT 0,
    flaky INTEGER NOT NULL DEFAULT 0,
    broken INTEGER NOT NULL DEFAULT 0,
    timedout INTEGER NOT NULL DEFAULT 0,
    interrupted INTEGER NOT NULL DEFAULT 0,
    unknown INTEGER NOT NULL DEFAULT 0,
    not_run INTEGER NOT NULL DEFAULT 0,
    running INTEGER NOT NULL DEFAULT 0,
    duration BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_attachments_run ON attachments (run_id);
CREATE INDEX IF NOT EXISTS idx_attachments_test ON attachments (test_id);
CREATE INDEX IF NOT EXISTS idx_attachments_attempt ON attachments (test_attempt_id);
CREATE INDEX IF NOT EXISTS idx_attachments_created_at ON attachments (created_at);
