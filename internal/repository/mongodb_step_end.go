package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertStepEnd updates step end fields (status, metadata, error fields, duration).
// With attempt-based retries: steps are stored in attempts[retry_index].steps.
// - runID: Required. Identifies the document (_id).
// - stepID: Required. Identifies the step to update.
// - testID: Required. ID of test containing the step.
// - retry_index: Required. Retry attempt index containing the step.
// - status: Step status (e.g., PASSED, FAILED).
// - metadata: Step metadata including error details (error_stack, error_value, error_snippet, error_location).
// - errorMsg: Single error message.
// - errors: Array of error messages.
// - duration: Step duration in nanoseconds.
// Returns error if runID is empty or step not found.
func (r *MongoRepository) UpsertStepEnd(ctx context.Context, runID string, stepID string, testID string, retry_index int32, status string, metadata map[string]interface{}, errorMsg string, errors []string, duration *int64) error {
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

	// Update metadata (merge with existing metadata)
	if metadata != nil && len(metadata) > 0 {
		for k, v := range metadata {
			setFields[fmt.Sprintf("tests.$[test].attempts.$[attempt].steps.$[step].metadata.%s", k)] = v
		}
	}

	// Update error fields
	if errorMsg != "" {
		setFields["tests.$[test].attempts.$[attempt].steps.$[step].error"] = errorMsg
	}
	if errors != nil && len(errors) > 0 {
		setFields["tests.$[test].attempts.$[attempt].steps.$[step].errors"] = errors
	}

	// Update duration
	if duration != nil {
		setFields["tests.$[test].attempts.$[attempt].steps.$[step].duration"] = *duration
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
		// The run document or parent test was not found.
		r.logger.Warn("parent test not found for UpsertStepEnd",
			"runID", runID,
			"stepID", stepID,
			"testID", testID,
			"retryIndex", retry_index)
		return &ErrParentNotFound{
			ParentType: "test",
			ParentID:   testID,
			ChildType:  "step",
			ChildID:    stepID,
		}
	}

	if result.ModifiedCount == 0 {
		// The run document and test were found but the step was not found within
		// the attempt. This is a race condition: StepEnd arrived before StepBegin.
		r.logger.Warn("step not found within attempt for UpsertStepEnd",
			"runID", runID,
			"stepID", stepID,
			"testID", testID,
			"retryIndex", retry_index)
		return &ErrParentNotFound{
			ParentType: "step",
			ParentID:   stepID,
			ChildType:  "step_end",
			ChildID:    stepID,
		}
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
