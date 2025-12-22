package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertSuiteBegin creates or updates a suite within the document identified by runID.
// - runID: Required. Identifies the document (_id).
// - suite: The suite to create/update (suite.ID identifies the suite).
// - parentSuiteID: Empty for root-level suite, or ID of parent suite for nested suites.
// Returns error if runID is empty or document not found.
func (r *MongoRepository) UpsertSuiteBegin(ctx context.Context, runID string, suite *m.SuiteDocument, parentSuiteID string) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	suite.CreatedAt = now
	suite.UpdatedAt = now
	suite.ParentSuiteID = parentSuiteID

	// Initialize child arrays
	if suite.Tests == nil {
		suite.Tests = []*m.TestDocument{}
	}
	if suite.Suites == nil {
		suite.Suites = []*m.SuiteDocument{}
	}

	// Ensure document exists
	if err := r.ensureDocumentExists(ctx, runID); err != nil {
		return fmt.Errorf("ensure document exists: %w", err)
	}

	if parentSuiteID == "" {
		// Root-level suite: upsert into document's suites array
		return r.upsertRootSuite(ctx, runID, suite, now)
	}

	// Nested suite: upsert into parent suite's suites array
	return r.upsertNestedSuite(ctx, runID, suite, parentSuiteID, now)
}

// upsertRootSuite handles root-level suites
func (r *MongoRepository) upsertRootSuite(ctx context.Context, runID string, suite *m.SuiteDocument, now time.Time) error {
	// Try to update existing suite
	filter := bson.M{
		"_id":       runID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$.name":               suite.Name,
			"suites.$.description":        suite.Description,
			"suites.$.status":             suite.Status,
			"suites.$.metadata":           suite.Metadata,
			"suites.$.test_suite_spec_id": suite.TestSuiteSpecID,
			"suites.$.initiated_by":       suite.InitiatedBy,
			"suites.$.project_name":       suite.ProjectName,
			"suites.$.start_time":         suite.StartTime,
			"suites.$.updated_at":         now,
			"updated_at":                  now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update root suite: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("suite begin (root, updated)", "runID", runID, "suiteID", suite.ID)
		return nil
	}

	// Suite doesn't exist, append it
	filter = bson.M{"_id": runID}
	update = bson.M{
		"$push": bson.M{"suites": suite},
		"$set":  bson.M{"updated_at": now},
	}

	_, err = r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("append root suite: %w", err)
	}

	r.logger.Info("suite begin (root, inserted)", "runID", runID, "suiteID", suite.ID)
	return nil
}

// upsertNestedSuite handles nested suites using array filters
func (r *MongoRepository) upsertNestedSuite(ctx context.Context, runID string, suite *m.SuiteDocument, parentSuiteID string, now time.Time) error {
	// Try to update existing nested suite
	filter := bson.M{
		"_id":              runID,
		"suites.id":        parentSuiteID,
		"suites.suites.id": suite.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$[parent].suites.$[suite].name":               suite.Name,
			"suites.$[parent].suites.$[suite].description":        suite.Description,
			"suites.$[parent].suites.$[suite].status":             suite.Status,
			"suites.$[parent].suites.$[suite].metadata":           suite.Metadata,
			"suites.$[parent].suites.$[suite].test_suite_spec_id": suite.TestSuiteSpecID,
			"suites.$[parent].suites.$[suite].initiated_by":       suite.InitiatedBy,
			"suites.$[parent].suites.$[suite].project_name":       suite.ProjectName,
			"suites.$[parent].suites.$[suite].start_time":         suite.StartTime,
			"suites.$[parent].suites.$[suite].updated_at":         now,
			"updated_at": now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"parent.id": parentSuiteID},
			bson.M{"suite.id": suite.ID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested suite: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("suite begin (nested, updated)", "runID", runID, "suiteID", suite.ID, "parentID", parentSuiteID)
		return nil
	}

	// Suite doesn't exist, append it to parent's suites array
	filter = bson.M{
		"_id":       runID,
		"suites.id": parentSuiteID,
	}
	update = bson.M{
		"$push": bson.M{"suites.$[parent].suites": suite},
		"$set":  bson.M{"updated_at": now},
	}
	arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"parent.id": parentSuiteID},
		},
	})

	result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append nested suite: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("parent suite not found: runID=%s, parentSuiteID=%s", runID, parentSuiteID)
	}

	r.logger.Info("suite begin (nested, inserted)", "runID", runID, "suiteID", suite.ID, "parentID", parentSuiteID)
	return nil
}

// UpsertSuiteEnd updates suite end fields (status, endTime, duration).
// - runID: Required. Identifies the document (_id).
// - suiteID: Required. Identifies the suite to update.
// Returns error if runID is empty or suite not found.
func (r *MongoRepository) UpsertSuiteEnd(ctx context.Context, runID string, suiteID string, status string, endTime *time.Time, duration *int64) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if suiteID == "" {
		return fmt.Errorf("suiteID is required")
	}

	now := time.Now()
	updateFields := bson.M{"updated_at": now}
	if status != "" {
		updateFields["status"] = status
	}
	if endTime != nil {
		updateFields["end_time"] = endTime
	}
	if duration != nil {
		updateFields["duration"] = duration
	}

	// Try root-level suite
	filter := bson.M{
		"_id":       runID,
		"suites.id": suiteID,
	}
	setFields := bson.M{}
	for k, v := range updateFields {
		setFields["suites.$."+k] = v
	}
	setFields["updated_at"] = now

	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields})
	if err != nil {
		return fmt.Errorf("update suite end: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("suite end (root)", "runID", runID, "suiteID", suiteID, "status", status)
		return nil
	}

	// Try nested suite
	filter = bson.M{
		"_id":              runID,
		"suites.suites.id": suiteID,
	}
	setFields = bson.M{"updated_at": now}
	for k, v := range updateFields {
		setFields["suites.$[].suites.$[suite]."+k] = v
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"suite.id": suiteID},
		},
	})

	result, err = r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields}, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested suite end: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("suite not found: runID=%s, suiteID=%s", runID, suiteID)
	}

	r.logger.Info("suite end (nested)", "runID", runID, "suiteID", suiteID, "status", status)
	return nil
}
