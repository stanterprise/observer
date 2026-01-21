package repository

import (
	"context"
	"fmt"
	"time"

	// m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
	if err := ValidateRunID(runID); err != nil {
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
	// CRITICAL: Use array filters to match retry_index field, NOT positional indexing
	// Positional indexing (attempts.%d) can update the wrong attempt if they're out of order
	attemptSetFields := bson.M{
		"tests.$[test].attempts.$[attempt].status":     status,
		"tests.$[test].attempts.$[attempt].updated_at": now,
	}
	if endTime != nil {
		attemptSetFields["tests.$[test].attempts.$[attempt].end_time"] = endTime
	}
	if duration != nil {
		attemptSetFields["tests.$[test].attempts.$[attempt].duration"] = duration
	}

	filter := bson.M{
		"_id":               runID,
		"tests.id":          testID,
		"tests.retry_index": retryIndex,
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"attempt.retry_index": retryIndex},
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
	// Only need test filter here (not attempt filter) since we're updating test-level fields
	testLevelArrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields}, testLevelArrayFilters)
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
