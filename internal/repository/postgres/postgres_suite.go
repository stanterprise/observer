package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)

func (r *PostgresRepository) UpsertSuite(ctx context.Context, suite *m.Suite) error {
	if suite == nil {
		return fmt.Errorf("suite is nil")
	}
	if err := repository.ValidateRunID(suite.RunID); err != nil {
		return err
	}
	if suite.ID == "" {
		return fmt.Errorf("suite id is required")
	}
	if err := r.ensureDB(); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return upsertRunStartSuite(tx, suite, time.Now())
	})
}

func upsertRunStartSuite(tx *gorm.DB, suite *m.Suite, now time.Time) error {
	var stored m.Suite
	err := tx.Where("run_id = ? AND id = ?", suite.RunID, suite.ID).First(&stored).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("load run start suite %s: %w", suite.ID, err)
		}

		create := *suite
		create.CreatedAt = now
		create.UpdatedAt = now
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create run start suite %s: %w", suite.ID, err)
		}
		return nil
	}

	if suite.RunID != "" {
		stored.RunID = suite.RunID
	}
	if suite.ExternalSuiteID != "" {
		stored.ExternalSuiteID = suite.ExternalSuiteID
	}
	if suite.ParentSuiteID != nil {
		stored.ParentSuiteID = suite.ParentSuiteID
	}
	if suite.Name != "" {
		stored.Name = suite.Name
	}
	if suite.Description != "" {
		stored.Description = suite.Description
	}
	stored.Status = mergeRunStartEntityStatus(stored.Status, suite.Status)
	if len(suite.Metadata) > 0 {
		stored.Metadata = mergeRunStartMetadata(stored.Metadata, suite.Metadata)
	}
	if suite.Duration != nil {
		stored.Duration = cloneInt64Ptr(suite.Duration)
	}
	if suite.Location != "" {
		stored.Location = suite.Location
	}
	if suite.Type != "" {
		stored.Type = suite.Type
	}
	if suite.TestSuiteSpecID != "" {
		stored.TestSuiteSpecID = suite.TestSuiteSpecID
	}
	if suite.InitiatedBy != "" {
		stored.InitiatedBy = suite.InitiatedBy
	}
	if suite.ProjectName != "" {
		stored.ProjectName = suite.ProjectName
	}
	if suite.Author != "" {
		stored.Author = suite.Author
	}
	if suite.Owner != "" {
		stored.Owner = suite.Owner
	}
	if len(suite.TestCaseIDs) > 0 {
		stored.TestCaseIDs = append([]string(nil), suite.TestCaseIDs...)
	}
	if len(suite.SubSuiteIDs) > 0 {
		stored.SubSuiteIDs = append([]string(nil), suite.SubSuiteIDs...)
	}
	if len(suite.Tags) > 0 {
		stored.Tags = append([]string(nil), suite.Tags...)
	}
	if suite.StartTime != nil && (stored.StartTime == nil || suite.StartTime.Before(*stored.StartTime)) {
		stored.StartTime = cloneTimePtr(suite.StartTime)
	}
	if suite.EndTime != nil && (stored.EndTime == nil || suite.EndTime.After(*stored.EndTime)) {
		stored.EndTime = cloneTimePtr(suite.EndTime)
	}
	stored.UpdatedAt = now

	if err := tx.Save(&stored).Error; err != nil {
		return fmt.Errorf("update run start suite %s: %w", suite.ID, err)
	}
	return nil
}

func mergeRunStartEntityStatus(existing, incoming string) string {
	if incoming == "" {
		return existing
	}
	if isRunStartPlaceholderStatus(incoming) && !isRunStartPlaceholderStatus(existing) {
		return existing
	}
	return incoming
}

func isRunStartPlaceholderStatus(status string) bool {
	switch status {
	case "", "NOT_RUN", "UNKNOWN":
		return true
	default:
		return false
	}
}

func mergeRunStartMetadata(existing, incoming map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(existing)+len(incoming))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range incoming {
		merged[key] = value
	}
	return merged
}

func mergeRunStartTotalTests(existing, incoming int32, sharded bool) int32 {
	if !sharded {
		return incoming
	}
	if incoming <= 0 {
		return existing
	}
	return existing + incoming
}
