package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/stanterprise/observer/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertStepEnd updates a step within the active run-scoped step buffer keyed by test id.
// - runID: Required. Identifies the document (_id).
// - stepID: Required. Identifies the step to update.
// - testID: Required. ID of test containing the step.
// - retry_index: Required. Retry attempt index containing the buffered step.
// - status: Step status (e.g., PASSED, FAILED).
// - metadata: Step metadata including error details (error_stack, error_value, error_snippet, error_location).
// - errorMsg: Single error message.
// - errors: Array of error messages.
// - duration: Step duration in nanoseconds.
// Returns error if runID is empty or the buffered step is not found.
func (r *MongoRepository) UpsertStepEnd(ctx context.Context, runID string, stepID string, testID string, retry_index int32, status string, metadata map[string]interface{}, errorMsg string, errors []string, duration *int64) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if stepID == "" {
		return fmt.Errorf("stepID is required")
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	r.logger.Debug("UpsertStepEnd starting",
		"runID", runID,
		"stepID", stepID,
		"testID", testID,
		"retryIndex", retry_index,
		"status", status)

	now := time.Now()
	field := stepBufferField(testID)
	setFields := bson.M{
		field + ".status": activeStepBufferStatusActive,
	}

	if status != "" {
		setFields[field+".steps.$[step].status"] = status
	}

	// Update metadata (merge with existing metadata)
	if len(metadata) > 0 {
		for k, v := range metadata {
			setFields[fmt.Sprintf("%s.steps.$[step].metadata.%s", field, k)] = v
		}
	}

	// Update error fields
	if errorMsg != "" {
		setFields[field+".steps.$[step].error"] = errorMsg
	}
	if len(errors) > 0 {
		setFields[field+".steps.$[step].errors"] = errors
	}

	// Update duration
	if duration != nil {
		setFields[field+".steps.$[step].duration"] = *duration
	}

	setFields[field+".updated_at"] = now
	setFields[field+".last_event_at"] = now
	setFields["updated_at"] = now

	filter := bson.M{
		"_id":                  runID,
		field + ".retry_index": retry_index,
		field + ".steps.id":    stepID,
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
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

	if result.ModifiedCount == 0 {
		// No-op update (e.g. duplicate StepEnd event or values already match). Safe to ignore.
		r.logger.Warn("step end was a no-op (duplicate event or values unchanged)",
			"runID", runID,
			"stepID", stepID,
			"testID", testID,
			"retryIndex", retry_index)
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
