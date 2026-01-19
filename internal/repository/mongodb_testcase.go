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
			// DO NOT set test-level status on TestBegin - it will be set correctly on TestEnd
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
			// DO NOT set test-level status on TestBegin - it will be set correctly on TestEnd
			"tests.$[test].start_time": test.StartTime,
			"tests.$[test].updated_at": now,
			"updated_at":               now,
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

	r.logger.Debug("Test not found, creating new test with current attempt only",
		"runID", runID,
		"testID", test.ID,
		"retryIndex", *test.RetryIndex,
		"retryCount", *test.RetryCount)

	// Test doesn't exist, create it with only the current attempt
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

	r.logger.Debug("Created test with single attempt",
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

	// Step 1: Update the current attempt's status, end_time, and duration
	attemptSetFields := bson.M{
		fmt.Sprintf("tests.$[test].attempts.%d.status", retryIndex):     status,
		fmt.Sprintf("tests.$[test].attempts.%d.updated_at", retryIndex): now,
	}
	if endTime != nil {
		attemptSetFields[fmt.Sprintf("tests.$[test].attempts.%d.end_time", retryIndex)] = endTime
	}
	if duration != nil {
		attemptSetFields[fmt.Sprintf("tests.$[test].attempts.%d.duration", retryIndex)] = duration
	}

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

	// Update attempt fields first
	_, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": attemptSetFields}, arrayFilters)
	if err != nil {
		return fmt.Errorf("update test attempt: %w", err)
	}

	// Step 2: Fetch the test to determine the overall status based on all attempts
	// This is necessary because test-level status should represent the BEST outcome across all attempts
	// Following Playwright/Jest convention: if ANY attempt passed, the test is PASSED overall
	testDoc, err := r.GetTestFromRun(ctx, testID)
	if err != nil {
		return fmt.Errorf("fetch test for status aggregation: %w", err)
	}
	if testDoc == nil {
		return fmt.Errorf("test not found after attempt update: %s", testID)
	}

	// Determine overall test status based on all attempts
	// Rule: If ANY attempt has status PASSED, the test is PASSED (retry success scenario)
	//       Otherwise, use the current attempt's status
	overallStatus := status
	if len(testDoc.Attempts) > 0 {
		hasPassedAttempt := false
		for _, attempt := range testDoc.Attempts {
			if attempt.Status == "PASSED" {
				hasPassedAttempt = true
				break
			}
		}
		if hasPassedAttempt {
			overallStatus = "PASSED"
		}
	}

	r.logger.Debug("Computed overall test status",
		"runID", runID,
		"testID", testID,
		"currentAttemptStatus", status,
		"overallStatus", overallStatus,
		"totalAttempts", len(testDoc.Attempts))

	// Step 3: Update test-level fields with aggregated status and timing
	setFields := bson.M{
		"updated_at": now,
	}

	// Update test-level status with aggregated status (may differ from current attempt)
	if overallStatus != "" {
		setFields["tests.$[test].status"] = overallStatus
	}

	// Update test-level end_time (latest attempt end_time)
	if endTime != nil {
		setFields["tests.$[test].end_time"] = endTime
	}

	// Update test-level duration (current attempt duration)
	if duration != nil {
		setFields["tests.$[test].duration"] = duration
	}

	// Update test-level updated_at
	setFields["tests.$[test].updated_at"] = now

	// Update test in root-level tests array with aggregated status
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
		"currentAttemptStatus", status,
		"overallStatus", overallStatus,
		"retryIndex", retryIndex,
		"matchedCount", result.MatchedCount,
		"modifiedCount", result.ModifiedCount)
	return nil
}
