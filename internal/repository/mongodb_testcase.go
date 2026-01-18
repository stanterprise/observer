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

	// Create attempt object for current retry_index
	currentAttempt := &m.AttemptDocument{
		RetryIndex: *test.RetryIndex,
		Steps:      []*m.StepDocument{},
		StartTime:  test.StartTime,
		Status:     test.Status,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Try to update existing attempt in test's attempts array
	filter := bson.M{
		"_id":                        runID,
		"tests.id":                   test.ID,
		"tests.attempts.retry_index": test.RetryIndex,
	}
	update := bson.M{
		"$set": bson.M{
			"tests.$[test].name":        test.Name,
			"tests.$[test].title":       test.Title,
			"tests.$[test].description": test.Description,
			"tests.$[test].status":      test.Status,
			"tests.$[test].start_time":  test.StartTime,
			"tests.$[test].retry_index": test.RetryIndex,
			"tests.$[test].updated_at":  now,
			fmt.Sprintf("tests.$[test].attempts.%d.start_time", *test.RetryIndex): test.StartTime,
			fmt.Sprintf("tests.$[test].attempts.%d.status", *test.RetryIndex):     test.Status,
			fmt.Sprintf("tests.$[test].attempts.%d.updated_at", *test.RetryIndex): now,
			"updated_at": now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": test.ID},
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

	r.logger.Debug("Attempt not found, trying to append",
		"runID", runID,
		"testID", test.ID,
		"retryIndex", *test.RetryIndex)

	// Attempt doesn't exist, check if test exists to append attempt
	filter = bson.M{
		"_id":      runID,
		"tests.id": test.ID,
	}
	update = bson.M{
		"$push": bson.M{"tests.$[test].attempts": currentAttempt},
		"$set": bson.M{
			"tests.$[test].retry_index": test.RetryIndex,
			"tests.$[test].retry_count": test.RetryCount,
			"tests.$[test].status":      test.Status,
			"tests.$[test].start_time":  test.StartTime,
			"tests.$[test].updated_at":  now,
			"updated_at":                now,
		},
	}

	r.logger.Debug("Appending new attempt to existing test",
		"filter", filter,
		"currentAttempt.RetryIndex", currentAttempt.RetryIndex)

	result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
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

	r.logger.Debug("Test not found, creating new test with full attempts array",
		"runID", runID,
		"testID", test.ID,
		"retryIndex", *test.RetryIndex,
		"retryCount", *test.RetryCount)

	// Test doesn't exist, create it with full attempts array pre-allocated
	test.CreatedAt = now
	attemptsSize := int(*test.RetryCount + 1)
	test.Attempts = make([]*m.AttemptDocument, attemptsSize)
	for i := 0; i < attemptsSize; i++ {
		test.Attempts[i] = &m.AttemptDocument{
			RetryIndex: int32(i),
			Steps:      []*m.StepDocument{},
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}
	test.Attempts[*test.RetryIndex].StartTime = test.StartTime
	test.Attempts[*test.RetryIndex].Status = test.Status

	r.logger.Debug("Pre-allocated attempts array",
		"attemptsSize", attemptsSize,
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

// UpsertTestEnd updates test end fields (status, duration) and corresponding attempt fields.
// With attempt-based retries: updates both test-level status and attempts[retry_index] status/end_time/duration.
// - runID: Required. Identifies the document (_id).
// - testID: Required. Identifies the test to update.
// - retryIndex: Required. Identifies the test attempt to update.
// - status: New status to set (optional).
// - endTime: End time for the attempt (optional).
// - duration: New duration to set (optional).
// Returns error if runID is empty or test not found.
func (r *MongoRepository) UpsertTestEnd(ctx context.Context, runID string, testID string, status string, retryIndex int32, endTime *time.Time, duration *int64) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()

	r.logger.Debug("UpsertTestEnd starting",
		"runID", runID,
		"testID", testID,
		"retryIndex", retryIndex,
		"status", status)

	// Build update fields for both test-level and attempt-level
	setFields := bson.M{
		"updated_at": now,
	}

	// Update test-level status (mirrors current attempt status)
	if status != "" {
		setFields["tests.$[test].status"] = status
		// Also update the attempt status using literal index
		setFields[fmt.Sprintf("tests.$[test].attempts.%d.status", retryIndex)] = status
	}

	// Update test-level end_time (latest attempt end_time)
	if endTime != nil {
		setFields["tests.$[test].end_time"] = endTime
		setFields[fmt.Sprintf("tests.$[test].attempts.%d.end_time", retryIndex)] = endTime
	}

	// Update test-level duration (current attempt duration)
	if duration != nil {
		setFields["tests.$[test].duration"] = duration
		setFields[fmt.Sprintf("tests.$[test].attempts.%d.duration", retryIndex)] = duration
	}

	// Update attempt updated_at
	setFields[fmt.Sprintf("tests.$[test].attempts.%d.updated_at", retryIndex)] = now

	// Update test in root-level tests array
	filter := bson.M{
		"_id":               runID,
		"tests.id":          testID,
		"tests.retry_index": retryIndex,
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
		r.logger.Error("test not found for UpsertTestEnd",
			"runID", runID,
			"testID", testID,
			"retryIndex", retryIndex,
			"filter", filter)
		return fmt.Errorf("test not found: runID=%s, testID=%s, retryIndex=%v", runID, testID, retryIndex)
	}

	r.logger.Info("test end",
		"runID", runID,
		"testID", testID,
		"status", status,
		"retryIndex", retryIndex,
		"matchedCount", result.MatchedCount,
		"modifiedCount", result.ModifiedCount)
	return nil
}
