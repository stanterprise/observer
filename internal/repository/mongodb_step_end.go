package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertStepEnd updates step end fields (status).
// With attempt-based retries: steps are stored in attempts[retry_index].steps.
// - runID: Required. Identifies the document (_id).
// - stepID: Required. Identifies the step to update.
// - testID: Required. ID of test containing the step.
// - retry_index: Required. Retry attempt index containing the step.
// Returns error if runID is empty or step not found.
func (r *MongoRepository) UpsertStepEnd(ctx context.Context, runID string, stepID string, testID string, retry_index int32, status string) error {
	if err := ValidateRunID(runID); err != nil {
		return err
	}
	if stepID == "" {
		return fmt.Errorf("stepID is required")
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()

	r.logger.Debug("UpsertStepEnd starting",
		"runID", runID,
		"stepID", stepID,
		"testID", testID,
		"retryIndex", retry_index,
		"status", status)

	// Use arrayFilters for ALL levels for consistency with UpsertStepBegin
	setFields := bson.M{
		"updated_at": now,
		"tests.$[test].attempts.$[attempt].steps.$[step].updated_at": now,
	}

	if status != "" {
		setFields["tests.$[test].attempts.$[attempt].steps.$[step].status"] = status
	}

	// Update step in attempts[retry_index].steps array
	filter := bson.M{
		"_id":      runID,
		"tests.id": testID,
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"attempt.retry_index": retry_index},
			bson.M{"step.id": stepID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields}, arrayFilters)
	if err != nil {
		return fmt.Errorf("update step end: %w", err)
	}

	if result.MatchedCount == 0 {
		r.logger.Error("step not found for UpsertStepEnd",
			"runID", runID,
			"stepID", stepID,
			"testID", testID,
			"retryIndex", retry_index,
			"filter", filter)
		return fmt.Errorf("step not found: runID=%s, stepID=%s, testID=%s, retryIndex=%d", runID, stepID, testID, retry_index)
	}

	r.logger.Info("step end",
		"runID", runID,
		"stepID", stepID,
		"testID", testID,
		"retryIndex", retry_index,
		"status", status,
		"matchedCount", result.MatchedCount,
		"modifiedCount", result.ModifiedCount)
	return nil
}
