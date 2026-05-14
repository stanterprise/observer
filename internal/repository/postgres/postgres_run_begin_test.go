package postgres

import (
	"context"
	"fmt"
	"testing"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/proto-go/testsystem/v1/events"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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

func TestUpsertRunStart_ShardedMergesMetadataAndTotals(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()

	first := &events.ReportRunStartEventRequest{
		RunId:       "run-123",
		Name:        "Shard 1",
		ExecutionId: "execution-123",
		Metadata: map[string]string{
			"shard.total":   "2",
			"shard.current": "1",
			"marker":        "first",
		},
	}
	second := &events.ReportRunStartEventRequest{
		RunId:       "run-123",
		Name:        "Shard 2",
		ExecutionId: "execution-456",
		Metadata: map[string]string{
			"shard.total":   "2",
			"shard.current": "2",
			"marker":        "second",
			"extra":         "yes",
		},
	}

	if err := repo.HandleRunStart(ctx, first); err != nil {
		t.Fatalf("HandleRunStart(first) failed: %v", err)
	}
	if err := repo.HandleRunStart(ctx, second); err != nil {
		t.Fatalf("HandleRunStart(second) failed: %v", err)
	}

	var stored m.TestRun
	if err := repo.db.WithContext(ctx).First(&stored, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}
	if stored.Metadata["extra"] != "yes" {
		t.Fatalf("stored.Metadata[extra] = %v, want yes", stored.Metadata["extra"])
	}

	var firstExecution m.RunExecution
	if err := repo.db.WithContext(ctx).First(&firstExecution, "run_id = ? AND id = ?", "run-123", "execution-123").Error; err != nil {
		t.Fatalf("load first execution: %v", err)
	}
	if !firstExecution.IsShard {
		t.Fatal("expected first execution to be marked as shard")
	}
	if firstExecution.ShardIndex == nil || *firstExecution.ShardIndex != 1 {
		t.Fatalf("firstExecution.ShardIndex = %v, want 1", firstExecution.ShardIndex)
	}
	if firstExecution.ShardCountExpected == nil || *firstExecution.ShardCountExpected != 2 {
		t.Fatalf("firstExecution.ShardCountExpected = %v, want 2", firstExecution.ShardCountExpected)
	}
}

func TestUpsertRunStart_CreatesRunStatsWhenStartTimeMissing(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()

	run := &events.ReportRunStartEventRequest{
		RunId: "run-456",
		Name:  "No Start Time",
		Metadata: map[string]string{
			"marker": "smoke",
		},
	}

	if err := repo.HandleRunStart(ctx, run); err != nil {
		t.Fatalf("HandleRunStart failed: %v", err)
	}

	var storedRun m.TestRun
	if err := repo.db.WithContext(ctx).First(&storedRun, "id = ?", run.RunId).Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}

	var stats m.RunStat
	if err := repo.db.WithContext(ctx).First(&stats, "run_id = ?", run.RunId).Error; err != nil {
		t.Fatalf("load run stats: %v", err)
	}
	if stats.Name != run.Name {
		t.Fatalf("run stats name = %q, want %q", stats.Name, run.Name)
	}
}

func TestHandleRunStartAggregatesLogicalRun(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()

	if err := repo.HandleRunStart(ctx, &events.ReportRunStartEventRequest{RunId: "run-123", Name: "Logical Aggregate", ExecutionId: "exec-a"}); err != nil {
		t.Fatalf("HandleRunStart(exec-a) failed: %v", err)
	}
	if err := repo.HandleRunStart(ctx, &events.ReportRunStartEventRequest{RunId: "run-123", Name: "Logical Aggregate", ExecutionId: "exec-b"}); err != nil {
		t.Fatalf("HandleRunStart(exec-b) failed: %v", err)
	}

	var storedRun m.TestRun
	if err := repo.db.WithContext(ctx).First(&storedRun, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load stored run: %v", err)
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

func TestHandleRunStart_ShardedExecutionsShareLogicalRun(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	ctx := context.Background()

	first := &events.ReportRunStartEventRequest{
		RunId:       "run-123",
		Name:        "Shard 1",
		ExecutionId: "exec-a",
		Metadata: map[string]string{
			"shard.total":   "2",
			"shard.current": "1",
		},
	}
	second := &events.ReportRunStartEventRequest{
		RunId:       "run-123",
		Name:        "Shard 2",
		ExecutionId: "exec-b",
		Metadata: map[string]string{
			"shard.total":   "2",
			"shard.current": "2",
		},
	}

	if err := repo.HandleRunStart(ctx, first); err != nil {
		t.Fatalf("HandleRunStart(first) failed: %v", err)
	}
	if err := repo.HandleRunStart(ctx, second); err != nil {
		t.Fatalf("HandleRunStart(second) failed: %v", err)
	}

	var storedRun m.TestRun
	if err := repo.db.WithContext(ctx).First(&storedRun, "id = ?", "run-123").Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}
	if storedRun.Status != "RUNNING" {
		t.Fatalf("storedRun.Status = %q, want RUNNING", storedRun.Status)
	}

	var runCount int64
	if err := repo.db.WithContext(ctx).Model(&m.TestRun{}).Where("id = ?", "run-123").Count(&runCount).Error; err != nil {
		t.Fatalf("count logical runs: %v", err)
	}
	if runCount != 1 {
		t.Fatalf("runCount = %d, want 1", runCount)
	}

	var shardExecutionCount int64
	if err := repo.db.WithContext(ctx).Model(&m.RunExecution{}).Where("run_id = ? AND is_shard = ?", "run-123", true).Count(&shardExecutionCount).Error; err != nil {
		t.Fatalf("count sharded executions: %v", err)
	}
	if shardExecutionCount != 2 {
		t.Fatalf("shardExecutionCount = %d, want 2", shardExecutionCount)
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
