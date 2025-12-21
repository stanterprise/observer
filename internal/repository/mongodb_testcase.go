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
// Returns error if runID is empty or parent suite not found.
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

	// Try to update existing test in root-level suite
	filter := bson.M{
		"_id":             runID,
		"suites.id":       suiteID,
		"suites.tests.id": test.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$[suite].tests.$[test].title":       test.Title,
			"suites.$[suite].tests.$[test].status":      test.Status,
			"suites.$[suite].tests.$[test].metadata":    test.Metadata,
			"suites.$[suite].tests.$[test].duration":    test.Duration,
			"suites.$[suite].tests.$[test].retry_count": test.RetryCount,
			"suites.$[suite].tests.$[test].retry_index": test.RetryIndex,
			"suites.$[suite].tests.$[test].timeout":     test.Timeout,
			"suites.$[suite].tests.$[test].updated_at":  now,
			"updated_at": now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
Filters: []interface{}{
bson.M{"suite.id": suiteID},
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

	// Test doesn't exist, append it to suite's tests array
	filter = bson.M{
		"_id":       runID,
		"suites.id": suiteID,
	}
	update = bson.M{
		"$push": bson.M{"suites.$[suite].tests": test},
		"$set":  bson.M{"updated_at": now},
	}
	arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
Filters: []interface{}{
bson.M{"suite.id": suiteID},
},
})

	result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append test: %w", err)
	}

	if result.MatchedCount == 0 {
		// Try nested suite (one level deep)
		filter = bson.M{
			"_id":               runID,
			"suites.suites.id": suiteID,
		}
		update = bson.M{
			"$push": bson.M{"suites.$[].suites.$[suite].tests": test},
			"$set":  bson.M{"updated_at": now},
		}
		arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
Filters: []interface{}{
bson.M{"suite.id": suiteID},
},
})

		result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("append test to nested suite: %w", err)
		}

		if result.MatchedCount == 0 {
			return fmt.Errorf("parent suite not found: runID=%s, suiteID=%s", runID, suiteID)
		}
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

	// Try root-level suite tests
	filter := bson.M{
		"_id":             runID,
		"suites.tests.id": testID,
	}
	setFields := bson.M{"updated_at": now}
	for k, v := range updateFields {
		setFields["suites.$[].tests.$[test]."+k] = v
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

	if result.MatchedCount > 0 {
		r.logger.Info("test end", "runID", runID, "testID", testID, "status", status)
		return nil
	}

	// Try nested suite tests
	filter = bson.M{
		"_id":                    runID,
		"suites.suites.tests.id": testID,
	}
	setFields = bson.M{"updated_at": now}
	for k, v := range updateFields {
		setFields["suites.$[].suites.$[].tests.$[test]."+k] = v
	}

	result, err = r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields}, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested test end: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("test not found: runID=%s, testID=%s", runID, testID)
	}

	r.logger.Info("test end", "runID", runID, "testID", testID, "status", status)
	return nil
}
