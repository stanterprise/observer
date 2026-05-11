package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	if got := mergeRunStartTotalTests(10, 5, true); got != 10 {
		t.Fatalf("mergeRunStartTotalTests(sharded) = %d, want 10", got)
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
	if stored.TotalTests != 5 {
		t.Fatalf("stored.TotalTests = %d, want 5", stored.TotalTests)
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

func TestUpsertRunStart_CreatesRunStatsWhenStartTimeMissing(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()

	run := &m.TestRun{
		ID:       "run-456",
		Name:     "No Start Time",
		Status:   "RUNNING",
		Metadata: map[string]interface{}{"marker": "smoke"},
	}

	if err := repo.UpsertRunStart(ctx, run); err != nil {
		t.Fatalf("UpsertRunStart failed: %v", err)
	}

	var storedRun m.TestRun
	if err := repo.db.WithContext(ctx).First(&storedRun, "id = ?", run.ID).Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}

	var stats m.RunStat
	if err := repo.db.WithContext(ctx).First(&stats, "run_id = ?", run.ID).Error; err != nil {
		t.Fatalf("load run stats: %v", err)
	}
	if stats.Name != run.Name {
		t.Fatalf("run stats name = %q, want %q", stats.Name, run.Name)
	}
}

func TestUpsertRunStartTests_PreservesTerminalStateAndAddsUniqueTests(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	start := time.Date(2026, 4, 29, 3, 35, 1, 0, time.UTC)
	end := start.Add(2 * time.Second)
	suiteID := "run-123:suite:suite-123"

	existing := m.Test{
		ID:             "run-123:test:test-a",
		RunID:          "run-123",
		ExternalTestID: "test-a",
		SuiteID:        &suiteID,
		Name:           "Test A",
		Title:          "Test A",
		Status:         "PASSED",
		StartTime:      &start,
		EndTime:        &end,
		Duration:       int64Ptr(int64((2 * time.Second).Nanoseconds())),
		CreatedAt:      start,
		UpdatedAt:      end,
	}
	if err := repo.db.WithContext(ctx).Create(&existing).Error; err != nil {
		t.Fatalf("seed existing test: %v", err)
	}

	if err := repo.UpsertRunStartTests(ctx, []*m.Test{
		{
			ID:             existing.ID,
			RunID:          existing.RunID,
			ExternalTestID: existing.ExternalTestID,
			SuiteID:        &suiteID,
			Name:           existing.Name,
			Title:          existing.Title,
			Status:         "NOT_RUN",
		},
		{
			ID:             "run-123:test:test-b",
			RunID:          "run-123",
			ExternalTestID: "test-b",
			SuiteID:        &suiteID,
			Name:           "Test B",
			Title:          "Test B",
			Status:         "NOT_RUN",
		},
	}); err != nil {
		t.Fatalf("UpsertRunStartTests failed: %v", err)
	}

	var storedExisting m.Test
	if err := repo.db.WithContext(ctx).First(&storedExisting, "id = ?", existing.ID).Error; err != nil {
		t.Fatalf("load existing test: %v", err)
	}
	if storedExisting.Status != "PASSED" {
		t.Fatalf("storedExisting.Status = %q, want PASSED", storedExisting.Status)
	}
	if storedExisting.EndTime == nil || !storedExisting.EndTime.Equal(end) {
		t.Fatalf("storedExisting.EndTime = %v, want %v", storedExisting.EndTime, end)
	}

	var count int64
	if err := repo.db.WithContext(ctx).Model(&m.Test{}).Where("run_id = ?", "run-123").Count(&count).Error; err != nil {
		t.Fatalf("count run tests: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestUpsertRunExecutionStartAggregatesLogicalRun(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()

	if err := repo.UpsertRunExecutionStart(ctx, &m.RunExecution{RunID: "run-123", ID: "exec-a", Name: "A", Status: "RUNNING", TotalTests: 3}); err != nil {
		t.Fatalf("UpsertRunExecutionStart(exec-a) failed: %v", err)
	}
	if err := repo.UpsertRunExecutionStart(ctx, &m.RunExecution{RunID: "run-123", ID: "exec-b", Name: "B", Status: "RUNNING", TotalTests: 5}); err != nil {
		t.Fatalf("UpsertRunExecutionStart(exec-b) failed: %v", err)
	}

	var storedRun m.TestRun
	if err := repo.db.WithContext(ctx).First(&storedRun, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}
	if storedRun.TotalTests != 8 {
		t.Fatalf("storedRun.TotalTests = %d, want 8", storedRun.TotalTests)
	}
	if storedRun.Status != "RUNNING" {
		t.Fatalf("storedRun.Status = %q, want RUNNING", storedRun.Status)
	}

	var executionCount int64
	if err := repo.db.WithContext(ctx).Model(&m.RunExecution{}).Where("run_id = ?", "run-123").Count(&executionCount).Error; err != nil {
		t.Fatalf("count run executions: %v", err)
	}
	if executionCount != 2 {
		t.Fatalf("executionCount = %d, want 2", executionCount)
	}
}

func TestUpsertRunExecutionStart_SharedShardedTotalsUseLogicalCount(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()
	mdA := map[string]interface{}{"shard.total": "2", "shard.current": "1"}
	mdB := map[string]interface{}{"shard.total": "2", "shard.current": "2"}

	if err := repo.UpsertRunExecutionStart(ctx, &m.RunExecution{RunID: "run-123", ID: "exec-a", Name: "A", Status: "RUNNING", TotalTests: 5, Metadata: mdA}); err != nil {
		t.Fatalf("UpsertRunExecutionStart(exec-a) failed: %v", err)
	}
	if err := repo.UpsertRunExecutionStart(ctx, &m.RunExecution{RunID: "run-123", ID: "exec-b", Name: "B", Status: "RUNNING", TotalTests: 5, Metadata: mdB}); err != nil {
		t.Fatalf("UpsertRunExecutionStart(exec-b) failed: %v", err)
	}

	var storedRun m.TestRun
	if err := repo.db.WithContext(ctx).First(&storedRun, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}
	if storedRun.TotalTests != 5 {
		t.Fatalf("storedRun.TotalTests = %d, want 5", storedRun.TotalTests)
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
		&m.RunExecution{},
		&m.RunShard{},
		&m.Suite{},
		&m.Test{},
		&m.TestAttempt{},
		&m.Attachment{},
		&m.RunStat{},
	}
}

func int32Ptr(value int32) *int32 {
	converted := value
	return &converted
}
