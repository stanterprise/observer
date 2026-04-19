package postgres

import (
	"context"
	"fmt"
	"testing"

	m "github.com/stanterprise/observer/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIsShardedRunStart(t *testing.T) {
	if !isShardedRunStart(map[string]interface{}{"shard.total": "4", "shard.current": "1"}) {
		t.Fatal("expected sharded run metadata to be detected")
	}
	if isShardedRunStart(map[string]interface{}{"shard.total": "4"}) {
		t.Fatal("expected incomplete shard metadata to be non-sharded")
	}
}

func TestMergeRunStartMetadata(t *testing.T) {
	existing := map[string]interface{}{"MARKER": "test", "existing": "value"}
	incoming := map[string]interface{}{"shard.total": "4", "MARKER": "updated"}

	merged := mergeRunStartMetadata(existing, incoming)
	if merged["existing"] != "value" {
		t.Fatalf("existing key lost: %+v", merged)
	}
	if merged["MARKER"] != "updated" {
		t.Fatalf("incoming metadata should win, got %+v", merged)
	}
	if merged["shard.total"] != "4" {
		t.Fatalf("missing sharded metadata, got %+v", merged)
	}
}

func TestMergeRunStartTotalTests(t *testing.T) {
	if got := mergeRunStartTotalTests(10, 5, true); got != 15 {
		t.Fatalf("mergeRunStartTotalTests(sharded) = %d, want 15", got)
	}
	if got := mergeRunStartTotalTests(10, 5, false); got != 5 {
		t.Fatalf("mergeRunStartTotalTests(non-sharded) = %d, want 5", got)
	}
	if got := mergeRunStartTotalTests(10, 0, true); got != 10 {
		t.Fatalf("mergeRunStartTotalTests(zero incoming) = %d, want 10", got)
	}
}

func TestUpsertRunStart_ShardedMergesMetadataAndTotals(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()

	first := &m.TestRun{
		ID:         "run-123",
		Name:       "Shard 1",
		Status:     "RUNNING",
		Metadata:   map[string]interface{}{"shard.total": "2", "shard.current": "1", "marker": "first"},
		TotalTests: 3,
	}
	second := &m.TestRun{
		ID:         "run-123",
		Name:       "Shard 2",
		Status:     "RUNNING",
		Metadata:   map[string]interface{}{"shard.total": "2", "shard.current": "2", "marker": "second", "extra": "yes"},
		TotalTests: 5,
	}

	if err := repo.UpsertRunStart(ctx, first); err != nil {
		t.Fatalf("UpsertRunStart(first) failed: %v", err)
	}
	if err := repo.UpsertRunStart(ctx, second); err != nil {
		t.Fatalf("UpsertRunStart(second) failed: %v", err)
	}

	var stored m.TestRun
	if err := repo.db.WithContext(ctx).First(&stored, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}
	if stored.TotalTests != 8 {
		t.Fatalf("stored.TotalTests = %d, want 8", stored.TotalTests)
	}
	if stored.Metadata["shard.current"] != "2" {
		t.Fatalf("stored.Metadata[shard.current] = %v, want 2", stored.Metadata["shard.current"])
	}
	if stored.Metadata["extra"] != "yes" {
		t.Fatalf("stored.Metadata[extra] = %v, want yes", stored.Metadata["extra"])
	}
}

func TestUpsertRunStartSuitesAndTests(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()

	suiteID := "suite-123"
	suites := []*m.Suite{{
		ID:          suiteID,
		RunID:       "run-123",
		Name:        "Root Suite",
		Metadata:    map[string]interface{}{"suite": "meta"},
		ProjectName: "chromium",
	}}
	tests := []*m.Test{{
		ID:         "test-123",
		RunID:      "run-123",
		SuiteID:    &suiteID,
		Name:       "Test Name",
		Title:      "Test Name",
		Metadata:   map[string]interface{}{"test": "meta"},
		RetryCount: int32Ptr(1),
		RetryIndex: int32Ptr(0),
	}}

	if err := repo.UpsertRunStartSuites(ctx, suites); err != nil {
		t.Fatalf("UpsertRunStartSuites failed: %v", err)
	}
	if err := repo.UpsertRunStartTests(ctx, tests); err != nil {
		t.Fatalf("UpsertRunStartTests failed: %v", err)
	}

	var storedSuite m.Suite
	if err := repo.db.WithContext(ctx).First(&storedSuite, "id = ?", suiteID).Error; err != nil {
		t.Fatalf("load stored suite: %v", err)
	}
	if storedSuite.Metadata["suite"] != "meta" {
		t.Fatalf("storedSuite.Metadata = %+v, want suite metadata", storedSuite.Metadata)
	}

	var storedTest m.Test
	if err := repo.db.WithContext(ctx).First(&storedTest, "id = ?", "test-123").Error; err != nil {
		t.Fatalf("load stored test: %v", err)
	}
	if storedTest.SuiteID == nil || *storedTest.SuiteID != suiteID {
		t.Fatalf("storedTest.SuiteID = %v, want %s", storedTest.SuiteID, suiteID)
	}
	if storedTest.Metadata["test"] != "meta" {
		t.Fatalf("storedTest.Metadata = %+v, want test metadata", storedTest.Metadata)
	}
}

func newSQLitePostgresRepository(t *testing.T) *PostgresRepository {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(modelsForSQLiteMigration()...); err != nil {
		t.Fatalf("auto migrate sqlite db: %v", err)
	}
	return NewPostgresRepository(db, nil)
}

func modelsForSQLiteMigration() []interface{} {
	return []interface{}{
		&m.TestRun{},
		&m.RunShard{},
		&m.Suite{},
		&m.Test{},
	}
}

func int32Ptr(value int32) *int32 {
	converted := value
	return &converted
}
