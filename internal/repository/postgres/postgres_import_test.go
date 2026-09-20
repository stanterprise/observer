package postgres

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/pkg/importer"
)

func TestImportRunPersistsGraphAtomicallyAndIsIdempotent(t *testing.T) {
	repo := newSQLitePostgresRepository(t)
	start := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Second)
	duration := end.Sub(start).Nanoseconds()
	suiteID := "suite-1"
	bundle := &importer.RunImportBundle{
		SourceDigest: "digest-1",
		Run:          &m.TestRun{ID: "import-1", Name: "Imported", Status: "PASSED", StartTime: &start, EndTime: &end, Duration: &duration, Metadata: map[string]interface{}{"source": map[string]interface{}{"digest": "digest-1"}}},
		Executions:   []*m.RunExecution{{ID: "execution-1", RunID: "import-1", Status: "PASSED", StartTime: &start, EndTime: &end, Duration: &duration}},
		Suites:       []*m.Suite{{ID: suiteID, RunID: "import-1", Name: "suite", Status: "PASSED"}},
		Tests:        []*m.Test{{ID: "test-1", RunID: "import-1", SuiteID: &suiteID, Name: "test", Status: "PASSED"}},
		Attempts:     []*m.TestAttempt{{ID: "attempt-1", RunID: "import-1", ExecutionID: "execution-1", TestID: "test-1", Status: "PASSED"}},
	}

	outcome, err := repo.ImportRun(context.Background(), bundle)
	if err != nil {
		t.Fatalf("ImportRun() error = %v", err)
	}
	if !outcome.Created {
		t.Fatal("first import was not created")
	}
	for model, want := range map[interface{}]int64{&m.TestRun{}: 1, &m.RunExecution{}: 1, &m.Suite{}: 1, &m.Test{}: 1, &m.TestAttempt{}: 1, &m.RunStat{}: 1} {
		var count int64
		if err := repo.db.Model(model).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("count %T = %d, want %d", model, count, want)
		}
	}
	var stat m.RunStat
	if err := repo.db.First(&stat, "run_id = ?", bundle.Run.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stat.Passed != 1 || stat.Duration != 2000 {
		t.Fatalf("stat = %#v", stat)
	}

	outcome, err = repo.ImportRun(context.Background(), bundle)
	if err != nil {
		t.Fatalf("idempotent ImportRun() error = %v", err)
	}
	if outcome.Created {
		t.Fatal("second import was reported as created")
	}
}
