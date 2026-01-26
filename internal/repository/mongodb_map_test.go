package repository

import (
	"context"
	"testing"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMapSuites_NonSharded(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "test-run-001"

	// First call with some tests
	metadata1 := map[string]interface{}{
		"browser": "chrome",
		"env":     "staging",
	}
	err := repo.MapSuites(ctx, runID, "Test Run 1", metadata1, 10, []m.SuiteDocument{})
	if err != nil {
		t.Fatalf("MapSuites failed: %v", err)
	}

	// Second call should replace metadata and total_tests
	metadata2 := map[string]interface{}{
		"browser": "firefox",
	}
	err = repo.MapSuites(ctx, runID, "Test Run 1 Updated", metadata2, 15, []m.SuiteDocument{})
	if err != nil {
		t.Fatalf("MapSuites failed: %v", err)
	}

	// Verify document
	var doc bson.M
	err = repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc)
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	// Check total_tests was replaced
	if totalTests, ok := doc["total_tests"].(int32); !ok || totalTests != 15 {
		t.Errorf("Expected total_tests=15, got %v", doc["total_tests"])
	}

	// Check metadata was replaced (should only have browser=firefox, not env)
	metadata, ok := doc["metadata"].(bson.M)
	if !ok {
		t.Fatalf("metadata is not bson.M: %T", doc["metadata"])
	}
	if browser, ok := metadata["browser"].(string); !ok || browser != "firefox" {
		t.Errorf("Expected browser=firefox, got %v", metadata["browser"])
	}
	if _, hasEnv := metadata["env"]; hasEnv {
		t.Errorf("Expected env to be removed in non-sharded mode, but it exists")
	}
}

func TestMapSuites_Sharded(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "test-run-sharded-001"

	// Shard 1 of 3 reports 10 tests
	metadata1 := map[string]interface{}{
		"shard.total":   3,
		"shard.current": 1,
		"browser":       "chrome",
		"env":           "staging",
	}
	err := repo.MapSuites(ctx, runID, "Sharded Test Run", metadata1, 10, []m.SuiteDocument{})
	if err != nil {
		t.Fatalf("MapSuites shard 1 failed: %v", err)
	}

	// Verify after shard 1
	var doc bson.M
	err = repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc)
	if err != nil {
		t.Fatalf("FindOne after shard 1 failed: %v", err)
	}
	if totalTests, ok := doc["total_tests"].(int32); !ok || totalTests != 10 {
		t.Errorf("After shard 1: expected total_tests=10, got %v", doc["total_tests"])
	}

	// Shard 2 of 3 reports 12 tests
	metadata2 := map[string]interface{}{
		"shard.total":   3,
		"shard.current": 2,
		"browser":       "chrome",
		"env":           "staging",
	}
	err = repo.MapSuites(ctx, runID, "Sharded Test Run", metadata2, 12, []m.SuiteDocument{})
	if err != nil {
		t.Fatalf("MapSuites shard 2 failed: %v", err)
	}

	// Verify after shard 2 - total_tests should be accumulated
	err = repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc)
	if err != nil {
		t.Fatalf("FindOne after shard 2 failed: %v", err)
	}
	if totalTests, ok := doc["total_tests"].(int32); !ok || totalTests != 22 {
		t.Errorf("After shard 2: expected total_tests=22 (10+12), got %v", doc["total_tests"])
	}

	// Shard 3 of 3 reports 8 tests
	metadata3 := map[string]interface{}{
		"shard.total":   3,
		"shard.current": 3,
		"browser":       "chrome",
		"env":           "staging",
	}
	err = repo.MapSuites(ctx, runID, "Sharded Test Run", metadata3, 8, []m.SuiteDocument{})
	if err != nil {
		t.Fatalf("MapSuites shard 3 failed: %v", err)
	}

	// Final verification - total_tests should be sum of all shards
	err = repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc)
	if err != nil {
		t.Fatalf("FindOne after shard 3 failed: %v", err)
	}
	if totalTests, ok := doc["total_tests"].(int32); !ok || totalTests != 30 {
		t.Errorf("After shard 3: expected total_tests=30 (10+12+8), got %v", doc["total_tests"])
	}

	// Verify metadata is preserved (not replaced)
	metadata, ok := doc["metadata"].(bson.M)
	if !ok {
		t.Fatalf("metadata is not bson.M: %T", doc["metadata"])
	}
	if shardTotal, ok := metadata["shard.total"].(int32); !ok || shardTotal != 3 {
		t.Errorf("Expected shard.total=3, got %v", metadata["shard.total"])
	}
	if shardCurrent, ok := metadata["shard.current"].(int32); !ok || shardCurrent != 3 {
		t.Errorf("Expected shard.current=3 (last shard), got %v", metadata["shard.current"])
	}
	if browser, ok := metadata["browser"].(string); !ok || browser != "chrome" {
		t.Errorf("Expected browser=chrome, got %v", metadata["browser"])
	}
	if env, ok := metadata["env"].(string); !ok || env != "staging" {
		t.Errorf("Expected env=staging, got %v", metadata["env"])
	}
}

func TestMapSuites_ShardedWithSuites(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "test-run-sharded-002"

	// Shard 1 of 2 with some suites
	metadata1 := map[string]interface{}{
		"shard.total":   2,
		"shard.current": 1,
	}
	suites1 := []m.SuiteDocument{
		{
			ID:     "suite-1",
			RunID:  runID,
			Name:   "Suite from Shard 1",
			Status: "passed",
		},
	}
	err := repo.MapSuites(ctx, runID, "Sharded with Suites", metadata1, 5, suites1)
	if err != nil {
		t.Fatalf("MapSuites shard 1 failed: %v", err)
	}

	// Shard 2 of 2 with different suites
	metadata2 := map[string]interface{}{
		"shard.total":   2,
		"shard.current": 2,
	}
	suites2 := []m.SuiteDocument{
		{
			ID:     "suite-2",
			RunID:  runID,
			Name:   "Suite from Shard 2",
			Status: "passed",
		},
	}
	err = repo.MapSuites(ctx, runID, "Sharded with Suites", metadata2, 7, suites2)
	if err != nil {
		t.Fatalf("MapSuites shard 2 failed: %v", err)
	}

	// Verify both suites were appended
	var doc bson.M
	err = repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc)
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	suites, ok := doc["suites"].(bson.A)
	if !ok {
		t.Fatalf("suites is not bson.A: %T", doc["suites"])
	}
	if len(suites) != 2 {
		t.Errorf("Expected 2 suites, got %d", len(suites))
	}

	// Verify total_tests accumulated
	if totalTests, ok := doc["total_tests"].(int32); !ok || totalTests != 12 {
		t.Errorf("Expected total_tests=12 (5+7), got %v", doc["total_tests"])
	}
}
