package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/pkg/importer"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ImportOutcome struct {
	Created bool
	RunID   string
}

func (r *PostgresRepository) ImportedRunExists(ctx context.Context, runID, sourceDigest string) (bool, string, error) {
	if err := r.ensureDB(); err != nil {
		return false, "", err
	}
	var run m.TestRun
	err := r.db.WithContext(ctx).Select("id", "name", "metadata").Where("id = ?", runID).First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("find imported run: %w", err)
	}
	if digestFromMetadata(run.Metadata) != sourceDigest {
		return false, "", fmt.Errorf("run id %q already exists with different import provenance", runID)
	}
	return true, run.Name, nil
}

func (r *PostgresRepository) ImportRun(ctx context.Context, bundle *importer.RunImportBundle) (ImportOutcome, error) {
	if err := r.ensureDB(); err != nil {
		return ImportOutcome{}, err
	}
	if bundle == nil || bundle.Run == nil || bundle.Run.ID == "" {
		return ImportOutcome{}, fmt.Errorf("import bundle must contain a run")
	}
	outcome := ImportOutcome{RunID: bundle.Run.ID}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", bundle.Run.ID).Error; err != nil {
				return fmt.Errorf("lock imported run: %w", err)
			}
		}
		var existing m.TestRun
		err := tx.Select("id", "metadata").Where("id = ?", bundle.Run.ID).First(&existing).Error
		if err == nil {
			if digestFromMetadata(existing.Metadata) != bundle.SourceDigest {
				return fmt.Errorf("run id %q already exists with different import provenance", bundle.Run.ID)
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check imported run: %w", err)
		}

		now := time.Now().UTC()
		applyImportTimestamps(bundle, now)
		if err := tx.Omit("Executions", "Suites", "Tests").Create(bundle.Run).Error; err != nil {
			return fmt.Errorf("create imported run: %w", err)
		}
		if err := createInBatches(tx, bundle.Executions); err != nil {
			return fmt.Errorf("create imported executions: %w", err)
		}
		if err := createInBatches(tx, bundle.Suites); err != nil {
			return fmt.Errorf("create imported suites: %w", err)
		}
		if err := createInBatches(tx, bundle.Tests); err != nil {
			return fmt.Errorf("create imported tests: %w", err)
		}
		if err := createInBatches(tx, bundle.Attempts); err != nil {
			return fmt.Errorf("create imported attempts: %w", err)
		}
		if err := createInBatches(tx, bundle.AttachmentRows); err != nil {
			return fmt.Errorf("create imported attachment metadata: %w", err)
		}
		stat := runStatForBundle(bundle, now)
		if err := tx.Create(stat).Error; err != nil {
			return fmt.Errorf("create imported run statistics: %w", err)
		}
		outcome.Created = true
		return nil
	})
	return outcome, err
}

func createInBatches[T any](tx *gorm.DB, records []*T) error {
	if len(records) == 0 {
		return nil
	}
	return tx.Omit(clause.Associations).CreateInBatches(records, 250).Error
}

func digestFromMetadata(metadata map[string]interface{}) string {
	source, _ := metadata["source"].(map[string]interface{})
	digest, _ := source["digest"].(string)
	return digest
}

func applyImportTimestamps(bundle *importer.RunImportBundle, now time.Time) {
	bundle.Run.CreatedAt, bundle.Run.UpdatedAt = now, now
	for _, record := range bundle.Executions {
		record.CreatedAt, record.UpdatedAt = now, now
	}
	for _, record := range bundle.Suites {
		record.CreatedAt, record.UpdatedAt = now, now
	}
	for _, record := range bundle.Tests {
		record.CreatedAt, record.UpdatedAt = now, now
	}
	for _, record := range bundle.Attempts {
		record.CreatedAt, record.UpdatedAt = now, now
	}
	for _, record := range bundle.AttachmentRows {
		record.CreatedAt = now
	}
}

func runStatForBundle(bundle *importer.RunImportBundle, now time.Time) *m.RunStat {
	stat := &m.RunStat{RunID: bundle.Run.ID, Name: bundle.Run.Name, Total: int32(len(bundle.Tests)), CreatedAt: now, UpdatedAt: now}
	if bundle.Run.StartTime != nil {
		stat.CreatedAt = *bundle.Run.StartTime
	}
	if bundle.Run.Duration != nil {
		stat.Duration = *bundle.Run.Duration / int64(time.Millisecond)
	}
	for _, test := range bundle.Tests {
		switch mapStatusToRunStatsColumn(test.Status) {
		case "passed":
			stat.Passed++
		case "failed":
			stat.Failed++
		case "flaky":
			stat.Flaky++
		case "skipped":
			stat.Skipped++
		case "broken":
			stat.Broken++
		case "timedout":
			stat.TimedOut++
		case "interrupted":
			stat.Interrupted++
		case "not_run":
			stat.NotRun++
		case "running":
			stat.Running++
		default:
			stat.Unknown++
		}
	}
	return stat
}
