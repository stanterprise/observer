package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertTestBegin creates or updates a test within the document identified by runID.
// - runID: Required. Identifies the document (_id).
// - test: The test to create/update (test.ID identifies the test).
// - suiteID: Required. ID of parent suite containing this test.
// Returns error if runID is empty.
func (r *MongoRepository) UpsertTestBegin(ctx context.Context, runID string, test *m.TestDocument, suiteID string) error {
	now := time.Now()
	test.UpdatedAt = now
	test.SuiteID = suiteID
	test.RunID = runID

	if test.Steps == nil {
		test.Steps = []*m.StepDocument{}
	}

	// Initialize retry_index if nil
	if test.RetryIndex == nil {
		defaultRetryIndex := int32(0)
		test.RetryIndex = &defaultRetryIndex
	}

	// Initialize retry_count if nil (default to 0 for no retries)
	if test.RetryCount == nil {
		defaultRetryCount := int32(0)
		test.RetryCount = &defaultRetryCount
	}

	r.logger.Debug("UpsertTestBegin starting",
		"runID", runID,
		"testID", test.ID,
		"retryIndex", *test.RetryIndex,
		"retryCount", *test.RetryCount,
		"status", test.Status)

	err := upsertTest(r, ctx, runID, test, suiteID, now)
	if err != nil {
		return err
	}

	return nil
}

func upsertTest(r *MongoRepository, ctx context.Context, runID string, test *m.TestDocument, suiteID string, now time.Time) error {
	if err := ValidateRunID(runID); err != nil {
		return err
	}
	if suiteID == "" {
		return fmt.Errorf("suiteID is required")
	}

	test.UpdatedAt = now
	test.SuiteID = suiteID
	test.RunID = runID

	if test.Steps == nil {
		test.Steps = []*m.StepDocument{}
	}

	// Try to update existing attempt in test's attempts array
	filter := bson.M{
		"_id":                        runID,
		"tests.id":                   test.ID,
		"tests.attempts.retry_index": test.RetryIndex,
	}

	update := bson.M{
		"$set": bson.M{
			"tests.$[test].name":                                test.Name,
			"tests.$[test].title":                               test.Title,
			"tests.$[test].description":                         test.Description,
			"tests.$[test].start_time":                          test.StartTime,
			"tests.$[test].retry_index":                         test.RetryIndex,
			"tests.$[test].updated_at":                          now,
			"tests.$[test].attempts.$[attempt].start_time":  test.StartTime,
			"tests.$[test].attempts.$[attempt].status":      test.Status,
			"tests.$[test].attempts.$[attempt].updated_at": now,
			"updated_at": now,
		},
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": test.ID},
			bson.M{"attempt.retry_index": test.RetryIndex},
		},
	})

	r.logger.Debug("Attempting to update existing attempt",
		"filter", filter,
		"retryIndex", *test.RetryIndex)

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update test attempt: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("test begin (attempt updated)",
			"runID", runID,
			"testID", test.ID,
			"retryIndex", *test.RetryIndex,
			"matchedCount", result.MatchedCount,
			"modifiedCount", result.ModifiedCount)
		return nil
	}

	// Attempt not found, try to append it or create the test
	return appendTestAttempt(r, ctx, runID, test, suiteID, now, result)
}

func appendTestAttempt(r *MongoRepository, ctx context.Context, runID string, test *m.TestDocument, suiteID string, now time.Time, testUpdateResult *mongo.UpdateResult) error {
	// Create attempt object for current retry_index
	currentAttempt := &m.AttemptDocument{
		RetryIndex: *test.RetryIndex,
		Steps:      []*m.StepDocument{},
		StartTime:  test.StartTime,
		Status:     test.Status,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	r.logger.Debug("Attempt not found, trying to append",
		"runID", runID,
		"testID", test.ID,
		"retryIndex", *test.RetryIndex)

	// Attempt doesn't exist, check if test exists to append attempt
	filter := bson.M{
		"_id":      runID,
		"tests.id": test.ID,
	}
	update := bson.M{
		"$push": bson.M{"tests.$[test].attempts": currentAttempt},
		"$set": bson.M{
			"tests.$[test].retry_index": test.RetryIndex,
			"tests.$[test].retry_count": test.RetryCount,
			// DO NOT set test-level status on TestBegin - it will be set correctly on TestEnd
			"tests.$[test].start_time": test.StartTime,
			"tests.$[test].updated_at": now,
			"updated_at":               now,
		},
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": test.ID},
		},
	})

	r.logger.Debug("Appending new attempt to existing test",
		"filter", filter,
		"currentAttempt.RetryIndex", currentAttempt.RetryIndex)

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append attempt to test: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("test begin (attempt appended)",
			"runID", runID,
			"testID", test.ID,
			"retryIndex", *test.RetryIndex,
			"matchedCount", result.MatchedCount,
			"modifiedCount", result.ModifiedCount,
			"attemptsArrayLength", "will be one more")
		return nil
	}

	// Test doesn't exist, create it with the current attempt
	r.logger.Debug("Test not found, creating new test with current attempt",
		"runID", runID,
		"testID", test.ID,
		"retryIndex", *test.RetryIndex,
		"retryCount", *test.RetryCount)

	test.CreatedAt = now
	// Save the incoming status for the attempt, but don't set test-level status yet
	attemptStatus := test.Status
	test.Status = "" // Clear test-level status - will be set correctly on TestEnd
	currentAttempt = &m.AttemptDocument{
		RetryIndex: *test.RetryIndex,
		Steps:      []*m.StepDocument{},
		StartTime:  test.StartTime,
		Status:     attemptStatus, // Use saved status for attempt
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	test.Attempts = []*m.AttemptDocument{currentAttempt}

	r.logger.Debug("Created test document with single attempt",
		"currentRetryIndex", *test.RetryIndex)

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

	r.logger.Info("test begin (test created)",
		"runID", runID,
		"testID", test.ID,
		"retryIndex", *test.RetryIndex,
		"attemptsArraySize", len(test.Attempts),
		"matchedCount", result.MatchedCount,
		"modifiedCount", result.ModifiedCount)

	return nil
}
