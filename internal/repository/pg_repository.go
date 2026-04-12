package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	m "github.com/stanterprise/observer/internal/models"
)

// noopWriter implements io.Writer but drops output.
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// PgRepository implements all PostgreSQL repository interfaces.
type PgRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPgRepository creates a new PgRepository backed by the given connection pool.
func NewPgRepository(pool *pgxpool.Pool, logger *slog.Logger) *PgRepository {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &PgRepository{pool: pool, logger: logger}
}

// ─── Runs ───────────────────────────────────────────────────────────────────

// UpsertRun inserts or updates a run using ON CONFLICT for idempotent redelivery.
func (r *PgRepository) UpsertRun(ctx context.Context, run *m.Run) error {
	now := time.Now().UTC()
	md := ensureJSON(run.Metadata)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO runs (id, logical_run_key, source, project, pipeline, branch, commit_sha, status, started_at, finished_at, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
		ON CONFLICT (id) DO UPDATE SET
			logical_run_key = EXCLUDED.logical_run_key,
			source          = EXCLUDED.source,
			project         = EXCLUDED.project,
			pipeline        = EXCLUDED.pipeline,
			branch          = EXCLUDED.branch,
			commit_sha      = EXCLUDED.commit_sha,
			status          = EXCLUDED.status,
			started_at      = EXCLUDED.started_at,
			finished_at     = EXCLUDED.finished_at,
			metadata        = EXCLUDED.metadata,
			updated_at      = $12
	`, run.ID, run.LogicalRunKey, run.Source, run.Project, run.Pipeline, run.Branch, run.CommitSHA, run.Status, run.StartedAt, run.FinishedAt, md, now)
	if err != nil {
		return fmt.Errorf("upsert run %s: %w", run.ID, err)
	}
	return nil
}

// GetRun retrieves a run by its primary key.
func (r *PgRepository) GetRun(ctx context.Context, id string) (*m.Run, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, logical_run_key, source, project, pipeline, branch, commit_sha, status, started_at, finished_at, metadata, created_at, updated_at
		FROM runs WHERE id = $1
	`, id)
	return scanRun(row)
}

// GetRunByLogicalKey retrieves a run by its unique logical run key.
func (r *PgRepository) GetRunByLogicalKey(ctx context.Context, logicalRunKey string) (*m.Run, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, logical_run_key, source, project, pipeline, branch, commit_sha, status, started_at, finished_at, metadata, created_at, updated_at
		FROM runs WHERE logical_run_key = $1
	`, logicalRunKey)
	return scanRun(row)
}

// ListRuns returns runs filtered by the provided options.
func (r *PgRepository) ListRuns(ctx context.Context, opts ListRunsOpts) ([]*m.Run, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	var rows pgx.Rows
	var err error

	if opts.Status != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, logical_run_key, source, project, pipeline, branch, commit_sha, status, started_at, finished_at, metadata, created_at, updated_at
			FROM runs WHERE status = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3
		`, opts.Status, limit, offset)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, logical_run_key, source, project, pipeline, branch, commit_sha, status, started_at, finished_at, metadata, created_at, updated_at
			FROM runs ORDER BY started_at DESC LIMIT $1 OFFSET $2
		`, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	var runs []*m.Run
	for rows.Next() {
		run, err := scanRunFromRows(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// UpdateRunStatus sets the status (and finished_at for terminal statuses) on a run.
func (r *PgRepository) UpdateRunStatus(ctx context.Context, id, status string) error {
	now := time.Now().UTC()
	var err error
	if m.IsTerminalStatus(status) {
		_, err = r.pool.Exec(ctx, `
			UPDATE runs SET status = $1, finished_at = $2, updated_at = $2 WHERE id = $3
		`, status, now, id)
	} else {
		_, err = r.pool.Exec(ctx, `
			UPDATE runs SET status = $1, updated_at = $2 WHERE id = $3
		`, status, now, id)
	}
	if err != nil {
		return fmt.Errorf("update run status %s: %w", id, err)
	}
	return nil
}

// ─── Run Shards ─────────────────────────────────────────────────────────────

// UpsertRunShard inserts or updates a run shard.
func (r *PgRepository) UpsertRunShard(ctx context.Context, shard *m.RunShard) error {
	now := time.Now().UTC()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO run_shards (id, run_id, shard_key, shard_index, shard_count_expected, status, started_at, finished_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		ON CONFLICT (id) DO UPDATE SET
			shard_key            = EXCLUDED.shard_key,
			shard_index          = EXCLUDED.shard_index,
			shard_count_expected = EXCLUDED.shard_count_expected,
			status               = EXCLUDED.status,
			started_at           = EXCLUDED.started_at,
			finished_at          = EXCLUDED.finished_at,
			updated_at           = $9
	`, shard.ID, shard.RunID, shard.ShardKey, shard.ShardIndex, shard.ShardCountExpected, shard.Status, shard.StartedAt, shard.FinishedAt, now)
	if err != nil {
		return fmt.Errorf("upsert run shard %s: %w", shard.ID, err)
	}
	return nil
}

// GetRunShard retrieves a run shard by ID.
func (r *PgRepository) GetRunShard(ctx context.Context, id string) (*m.RunShard, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, run_id, shard_key, shard_index, shard_count_expected, status, started_at, finished_at, created_at, updated_at
		FROM run_shards WHERE id = $1
	`, id)
	return scanRunShard(row)
}

// ListRunShardsByRunID returns all shards belonging to a run.
func (r *PgRepository) ListRunShardsByRunID(ctx context.Context, runID string) ([]*m.RunShard, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, run_id, shard_key, shard_index, shard_count_expected, status, started_at, finished_at, created_at, updated_at
		FROM run_shards WHERE run_id = $1 ORDER BY shard_index
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("list run shards for run %s: %w", runID, err)
	}
	defer rows.Close()

	var shards []*m.RunShard
	for rows.Next() {
		s, err := scanRunShardFromRows(rows)
		if err != nil {
			return nil, err
		}
		shards = append(shards, s)
	}
	return shards, rows.Err()
}

// UpdateRunShardStatus updates the status of a run shard.
func (r *PgRepository) UpdateRunShardStatus(ctx context.Context, id, status string) error {
	now := time.Now().UTC()
	var err error
	if m.IsTerminalStatus(status) {
		_, err = r.pool.Exec(ctx, `
			UPDATE run_shards SET status = $1, finished_at = $2, updated_at = $2 WHERE id = $3
		`, status, now, id)
	} else {
		_, err = r.pool.Exec(ctx, `
			UPDATE run_shards SET status = $1, updated_at = $2 WHERE id = $3
		`, status, now, id)
	}
	if err != nil {
		return fmt.Errorf("update run shard status %s: %w", id, err)
	}
	return nil
}

// ─── Suites ─────────────────────────────────────────────────────────────────

// UpsertSuite inserts or updates a suite.
func (r *PgRepository) UpsertSuite(ctx context.Context, suite *m.Suite) error {
	now := time.Now().UTC()
	md := ensureJSON(suite.Metadata)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO suites (id, run_id, external_suite_id, name, status, started_at, finished_at, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		ON CONFLICT (id) DO UPDATE SET
			run_id            = EXCLUDED.run_id,
			external_suite_id = EXCLUDED.external_suite_id,
			name              = EXCLUDED.name,
			status            = EXCLUDED.status,
			started_at        = EXCLUDED.started_at,
			finished_at       = EXCLUDED.finished_at,
			metadata          = EXCLUDED.metadata,
			updated_at        = $9
	`, suite.ID, suite.RunID, suite.ExternalSuiteID, suite.Name, suite.Status, suite.StartedAt, suite.FinishedAt, md, now)
	if err != nil {
		return fmt.Errorf("upsert suite %s: %w", suite.ID, err)
	}
	return nil
}

// GetSuite retrieves a suite by ID.
func (r *PgRepository) GetSuite(ctx context.Context, id string) (*m.Suite, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, run_id, external_suite_id, name, status, started_at, finished_at, metadata, created_at, updated_at
		FROM suites WHERE id = $1
	`, id)
	return scanSuite(row)
}

// ListSuitesByRunID returns all suites belonging to a run.
func (r *PgRepository) ListSuitesByRunID(ctx context.Context, runID string) ([]*m.Suite, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, run_id, external_suite_id, name, status, started_at, finished_at, metadata, created_at, updated_at
		FROM suites WHERE run_id = $1 ORDER BY started_at
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("list suites for run %s: %w", runID, err)
	}
	defer rows.Close()

	var suites []*m.Suite
	for rows.Next() {
		s, err := scanSuiteFromRows(rows)
		if err != nil {
			return nil, err
		}
		suites = append(suites, s)
	}
	return suites, rows.Err()
}

// UpdateSuiteStatus updates the status of a suite.
func (r *PgRepository) UpdateSuiteStatus(ctx context.Context, id, status string) error {
	now := time.Now().UTC()
	var err error
	if m.IsTerminalStatus(status) {
		_, err = r.pool.Exec(ctx, `
			UPDATE suites SET status = $1, finished_at = $2, updated_at = $2 WHERE id = $3
		`, status, now, id)
	} else {
		_, err = r.pool.Exec(ctx, `
			UPDATE suites SET status = $1, updated_at = $2 WHERE id = $3
		`, status, now, id)
	}
	if err != nil {
		return fmt.Errorf("update suite status %s: %w", id, err)
	}
	return nil
}

// ─── Tests ──────────────────────────────────────────────────────────────────

// UpsertTest inserts or updates a test.
func (r *PgRepository) UpsertTest(ctx context.Context, test *m.Test) error {
	now := time.Now().UTC()
	md := ensureJSON(test.Metadata)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO tests (id, suite_id, external_test_id, name, status, attempt_count, started_at, finished_at, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		ON CONFLICT (id) DO UPDATE SET
			suite_id         = EXCLUDED.suite_id,
			external_test_id = EXCLUDED.external_test_id,
			name             = EXCLUDED.name,
			status           = EXCLUDED.status,
			attempt_count    = EXCLUDED.attempt_count,
			started_at       = EXCLUDED.started_at,
			finished_at      = EXCLUDED.finished_at,
			metadata         = EXCLUDED.metadata,
			updated_at       = $10
	`, test.ID, test.SuiteID, test.ExternalTestID, test.Name, test.Status, test.AttemptCount, test.StartedAt, test.FinishedAt, md, now)
	if err != nil {
		return fmt.Errorf("upsert test %s: %w", test.ID, err)
	}
	return nil
}

// GetTest retrieves a test by ID.
func (r *PgRepository) GetTest(ctx context.Context, id string) (*m.Test, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, suite_id, external_test_id, name, status, attempt_count, started_at, finished_at, metadata, created_at, updated_at
		FROM tests WHERE id = $1
	`, id)
	return scanTest(row)
}

// ListTestsBySuiteID returns all tests belonging to a suite.
func (r *PgRepository) ListTestsBySuiteID(ctx context.Context, suiteID string) ([]*m.Test, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, suite_id, external_test_id, name, status, attempt_count, started_at, finished_at, metadata, created_at, updated_at
		FROM tests WHERE suite_id = $1 ORDER BY started_at
	`, suiteID)
	if err != nil {
		return nil, fmt.Errorf("list tests for suite %s: %w", suiteID, err)
	}
	defer rows.Close()

	var tests []*m.Test
	for rows.Next() {
		t, err := scanTestFromRows(rows)
		if err != nil {
			return nil, err
		}
		tests = append(tests, t)
	}
	return tests, rows.Err()
}

// UpdateTestStatus updates the status of a test.
func (r *PgRepository) UpdateTestStatus(ctx context.Context, id, status string) error {
	now := time.Now().UTC()
	var err error
	if m.IsTerminalStatus(status) {
		_, err = r.pool.Exec(ctx, `
			UPDATE tests SET status = $1, finished_at = $2, updated_at = $2 WHERE id = $3
		`, status, now, id)
	} else {
		_, err = r.pool.Exec(ctx, `
			UPDATE tests SET status = $1, updated_at = $2 WHERE id = $3
		`, status, now, id)
	}
	if err != nil {
		return fmt.Errorf("update test status %s: %w", id, err)
	}
	return nil
}

// ─── Test Attempts ──────────────────────────────────────────────────────────

// UpsertTestAttempt inserts or updates a test attempt.
func (r *PgRepository) UpsertTestAttempt(ctx context.Context, attempt *m.TestAttempt) error {
	now := time.Now().UTC()
	md := ensureJSON(attempt.Metadata)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO test_attempts (id, test_id, attempt_index, status, started_at, finished_at, steps, steps_ref, step_count, duration_ms, failure_reason, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)
		ON CONFLICT (id) DO UPDATE SET
			test_id        = EXCLUDED.test_id,
			attempt_index  = EXCLUDED.attempt_index,
			status         = EXCLUDED.status,
			started_at     = EXCLUDED.started_at,
			finished_at    = EXCLUDED.finished_at,
			steps          = EXCLUDED.steps,
			steps_ref      = EXCLUDED.steps_ref,
			step_count     = EXCLUDED.step_count,
			duration_ms    = EXCLUDED.duration_ms,
			failure_reason = EXCLUDED.failure_reason,
			metadata       = EXCLUDED.metadata,
			updated_at     = $13
	`, attempt.ID, attempt.TestID, attempt.AttemptIndex, attempt.Status, attempt.StartedAt, attempt.FinishedAt, attempt.Steps, attempt.StepsRef, attempt.StepCount, attempt.DurationMs, attempt.FailureReason, md, now)
	if err != nil {
		return fmt.Errorf("upsert test attempt %s: %w", attempt.ID, err)
	}
	return nil
}

// GetTestAttempt retrieves a test attempt by ID.
func (r *PgRepository) GetTestAttempt(ctx context.Context, id string) (*m.TestAttempt, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, test_id, attempt_index, status, started_at, finished_at, steps, steps_ref, step_count, duration_ms, failure_reason, metadata, created_at, updated_at
		FROM test_attempts WHERE id = $1
	`, id)
	return scanTestAttempt(row)
}

// GetTestAttemptByIndex retrieves a test attempt by test ID and attempt index.
func (r *PgRepository) GetTestAttemptByIndex(ctx context.Context, testID string, attemptIndex int) (*m.TestAttempt, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, test_id, attempt_index, status, started_at, finished_at, steps, steps_ref, step_count, duration_ms, failure_reason, metadata, created_at, updated_at
		FROM test_attempts WHERE test_id = $1 AND attempt_index = $2
	`, testID, attemptIndex)
	return scanTestAttempt(row)
}

// ListAttemptsByTestID returns all attempts for a given test.
func (r *PgRepository) ListAttemptsByTestID(ctx context.Context, testID string) ([]*m.TestAttempt, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, test_id, attempt_index, status, started_at, finished_at, steps, steps_ref, step_count, duration_ms, failure_reason, metadata, created_at, updated_at
		FROM test_attempts WHERE test_id = $1 ORDER BY attempt_index
	`, testID)
	if err != nil {
		return nil, fmt.Errorf("list attempts for test %s: %w", testID, err)
	}
	defer rows.Close()

	var attempts []*m.TestAttempt
	for rows.Next() {
		a, err := scanTestAttemptFromRows(rows)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	return attempts, rows.Err()
}

// FinalizeAttempt atomically sets the terminal state on a test attempt.
// Only one of steps or stepsRef should be non-nil (enforced by DB constraint).
func (r *PgRepository) FinalizeAttempt(ctx context.Context, id string, status string, steps []byte, stepsRef *string, stepCount int, durationMs int64, failureReason *string) error {
	now := time.Now().UTC()

	_, err := r.pool.Exec(ctx, `
		UPDATE test_attempts SET
			status         = $1,
			finished_at    = $2,
			steps          = $3,
			steps_ref      = $4,
			step_count     = $5,
			duration_ms    = $6,
			failure_reason = $7,
			updated_at     = $2
		WHERE id = $8
	`, status, now, steps, stepsRef, stepCount, durationMs, failureReason, id)
	if err != nil {
		return fmt.Errorf("finalize attempt %s: %w", id, err)
	}
	return nil
}

// ─── Scan helpers ───────────────────────────────────────────────────────────

// scannable abstracts pgx.Row and pgx.Rows for scan helpers.
type scannable interface {
	Scan(dest ...any) error
}

func scanRun(row scannable) (*m.Run, error) {
	var run m.Run
	err := row.Scan(&run.ID, &run.LogicalRunKey, &run.Source, &run.Project, &run.Pipeline, &run.Branch, &run.CommitSHA, &run.Status, &run.StartedAt, &run.FinishedAt, &run.Metadata, &run.CreatedAt, &run.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan run: %w", err)
	}
	return &run, nil
}

func scanRunFromRows(rows pgx.Rows) (*m.Run, error) { return scanRun(rows) }

func scanRunShard(row scannable) (*m.RunShard, error) {
	var s m.RunShard
	err := row.Scan(&s.ID, &s.RunID, &s.ShardKey, &s.ShardIndex, &s.ShardCountExpected, &s.Status, &s.StartedAt, &s.FinishedAt, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan run shard: %w", err)
	}
	return &s, nil
}

func scanRunShardFromRows(rows pgx.Rows) (*m.RunShard, error) { return scanRunShard(rows) }

func scanSuite(row scannable) (*m.Suite, error) {
	var s m.Suite
	err := row.Scan(&s.ID, &s.RunID, &s.ExternalSuiteID, &s.Name, &s.Status, &s.StartedAt, &s.FinishedAt, &s.Metadata, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan suite: %w", err)
	}
	return &s, nil
}

func scanSuiteFromRows(rows pgx.Rows) (*m.Suite, error) { return scanSuite(rows) }

func scanTest(row scannable) (*m.Test, error) {
	var t m.Test
	err := row.Scan(&t.ID, &t.SuiteID, &t.ExternalTestID, &t.Name, &t.Status, &t.AttemptCount, &t.StartedAt, &t.FinishedAt, &t.Metadata, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan test: %w", err)
	}
	return &t, nil
}

func scanTestFromRows(rows pgx.Rows) (*m.Test, error) { return scanTest(rows) }

func scanTestAttempt(row scannable) (*m.TestAttempt, error) {
	var a m.TestAttempt
	err := row.Scan(&a.ID, &a.TestID, &a.AttemptIndex, &a.Status, &a.StartedAt, &a.FinishedAt, &a.Steps, &a.StepsRef, &a.StepCount, &a.DurationMs, &a.FailureReason, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan test attempt: %w", err)
	}
	return &a, nil
}

func scanTestAttemptFromRows(rows pgx.Rows) (*m.TestAttempt, error) { return scanTestAttempt(rows) }

// ensureJSON returns the input if non-nil, or a default empty JSON object.
func ensureJSON(data []byte) []byte {
	if len(data) == 0 {
		return []byte("{}")
	}
	return data
}
