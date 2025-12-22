package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertTestBegin creates or updates a test within the document identified by runID.
// - runID: Required. Identifies the document (_id).
// - test: The test to create/update (test.ID identifies the test).
// - suiteID: Required. ID of parent suite containing this test.
// Returns error if runID is empty.
func (r *MongoRepository) UpsertTestBegin(ctx context.Context, runID string, test *m.TestDocument, suiteID string) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if suiteID == "" {
		return fmt.Errorf("suiteID is required")
	}

	now := time.Now()
	test.CreatedAt = now
	test.UpdatedAt = now
	test.SuiteID = suiteID
	test.RunID = runID

	if test.Steps == nil {
		test.Steps = []*m.StepDocument{}
	}

	// Try to update existing test in root-level tests array
	filter := bson.M{
		"_id": runID,
		"tests.id": test.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"tests.$[test].title":       test.Title,
			"tests.$[test].status":      test.Status,
			"tests.$[test].metadata":    test.Metadata,
			"tests.$[test].duration":    test.Duration,
			"tests.$[test].retry_count": test.RetryCount,
			"tests.$[test].retry_index": test.RetryIndex,
			"tests.$[test].timeout":     test.Timeout,
			"tests.$[test].suite_id":    test.SuiteID,
			"tests.$[test].updated_at":  now,
			"tests.$[test].run_id":      runID,
			"updated_at":                now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": test.ID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update test: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("test begin (updated)", "runID", runID, "testID", test.ID, "suiteID", suiteID)
		return nil
	}

	// Test doesn't exist, append it to root-level tests array
	filter = bson.M{
		"_id": runID,
	}
	update = bson.M{
		"$push": bson.M{"tests": test},
		"$set":  bson.M{"updated_at": now},
	}

	result, err = r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("append test: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("test run document not found: runID=%s", runID)
	}

	r.logger.Info("test begin (inserted)", "runID", runID, "testID", test.ID, "suiteID", suiteID)
	return nil
}

// UpsertTestEnd updates test end fields (status, duration).
// - runID: Required. Identifies the document (_id).
// - testID: Required. Identifies the test to update.
// Returns error if runID is empty or test not found.
func (r *MongoRepository) UpsertTestEnd(ctx context.Context, runID string, testID string, status string, duration *int64) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	updateFields := bson.M{"updated_at": now}
	if status != "" {
		updateFields["status"] = status
	}
	if duration != nil {
		updateFields["duration"] = duration
	}

	// Update test in root-level tests array
	filter := bson.M{
		"_id":      runID,
		"tests.id": testID,
	}
	setFields := bson.M{"updated_at": now}
	for k, v := range updateFields {
		setFields["tests.$[test]."+k] = v
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields}, arrayFilters)
	if err != nil {
		return fmt.Errorf("update test end: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("test not found: runID=%s, testID=%s", runID, testID)
	}

	r.logger.Info("test end", "runID", runID, "testID", testID, "status", status)
	return nil
}
