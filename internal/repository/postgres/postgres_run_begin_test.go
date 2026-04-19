package postgres

import "testing"

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