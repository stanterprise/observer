package database

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReconcileLegacyExecutionIDColumnsBackfillsNulls(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	statements := []string{
		`CREATE TABLE run_shards (id text primary key, run_id text not null, shard_index integer, execution_id text)`,
		`CREATE TABLE test_attempts (id text primary key, run_id text not null, test_id text not null, attempt_index integer not null, execution_id text)`,
		`INSERT INTO run_shards (id, run_id, shard_index, execution_id) VALUES ('run-1:shard:1', 'run-1', 1, NULL)`,
		`INSERT INTO test_attempts (id, run_id, test_id, attempt_index, execution_id) VALUES ('attempt-1', 'run-1', 'test-1', 0, NULL)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("exec %q: %v", statement, err)
		}
	}

	if err := reconcileLegacyExecutionIDColumns(db); err != nil {
		t.Fatalf("reconcileLegacyExecutionIDColumns failed: %v", err)
	}

	var shardExecutionID string
	if err := db.Raw(`SELECT COALESCE(execution_id, '__NULL__') FROM run_shards WHERE id = ?`, "run-1:shard:1").Scan(&shardExecutionID).Error; err != nil {
		t.Fatalf("load shard execution_id: %v", err)
	}
	if shardExecutionID != "" {
		t.Fatalf("run_shards.execution_id = %q, want empty string", shardExecutionID)
	}

	var attemptExecutionID string
	if err := db.Raw(`SELECT COALESCE(execution_id, '__NULL__') FROM test_attempts WHERE id = ?`, "attempt-1").Scan(&attemptExecutionID).Error; err != nil {
		t.Fatalf("load attempt execution_id: %v", err)
	}
	if attemptExecutionID != "" {
		t.Fatalf("test_attempts.execution_id = %q, want empty string", attemptExecutionID)
	}
}
