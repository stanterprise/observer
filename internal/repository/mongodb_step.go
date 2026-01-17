package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertStepBegin creates or updates a step within the document identified by runID.
// With attempt-based retries: steps are stored in attempts[retry_index].steps instead of tests.steps.
// - runID: Required. Identifies the document (_id).
// - step: The step to create/update (step.ID identifies the step).
// - testID: Required. ID of parent test containing this step.
// - retry_index: Required. Retry attempt index to target for step storage.
// Returns error if runID is empty or parent test not found.
func (r *MongoRepository) UpsertStepBegin(ctx context.Context, runID string, step *m.StepDocument, testID string, retry_index int32) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	step.CreatedAt = now
	step.UpdatedAt = now
	step.TestCaseRunID = testID
	step.RunID = runID

	if step.Steps == nil {
		step.Steps = []*m.StepDocument{}
	}

	return r.upsertStepInTestAttempt(ctx, runID, testID, retry_index, step, now)
}

// upsertStepInTestAttempt handles steps as children of attempts[retry_index] array.
// With attempt-based retries: steps are stored in attempts[retry_index].steps instead of tests.steps.
func (r *MongoRepository) upsertStepInTestAttempt(ctx context.Context, runID string, testID string, retry_index int32, step *m.StepDocument, now time.Time) error {
	// Use arrayFilters for ALL levels: test, attempt, step
	// This avoids MongoDB error "The path 'tests.X.attempts.0.steps' must exist"
	filter := bson.M{
		"_id":      runID,
		"tests.id": testID,
	}

	update := bson.M{
		"$set": bson.M{
			"tests.$[test].attempts.$[attempt].steps.$[step].parent_step_id": step.ParentStepID,
			"tests.$[test].attempts.$[attempt].steps.$[step].title":          step.Title,
			"tests.$[test].attempts.$[attempt].steps.$[step].description":    step.Description,
			"tests.$[test].attempts.$[attempt].steps.$[step].start_time":     step.StartTime,
			"tests.$[test].attempts.$[attempt].steps.$[step].duration":       step.Duration,
			"tests.$[test].attempts.$[attempt].steps.$[step].type":           step.Type,
			"tests.$[test].attempts.$[attempt].steps.$[step].tags":           step.Tags,
			"tests.$[test].attempts.$[attempt].steps.$[step].metadata":       step.Metadata,
			"tests.$[test].attempts.$[attempt].steps.$[step].worker_index":   step.WorkerIndex,
			"tests.$[test].attempts.$[attempt].steps.$[step].status":         step.Status,
			"tests.$[test].attempts.$[attempt].steps.$[step].category":       step.Category,
			"tests.$[test].attempts.$[attempt].steps.$[step].location":       step.Location,
			"tests.$[test].attempts.$[attempt].steps.$[step].error":          step.Error,
			"tests.$[test].attempts.$[attempt].steps.$[step].errors":         step.Errors,
			"tests.$[test].attempts.$[attempt].steps.$[step].updated_at":     now,
			"updated_at": now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"attempt.retry_index": retry_index},
			bson.M{"step.id": step.ID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update step in test attempt: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("step begin (updated)", "runID", runID, "stepID", step.ID, "testID", testID, "retryIndex", retry_index)
		return nil
	}

	// Step doesn't exist, append it to attempts[retry_index].steps array
	// Use arrayFilter for attempt to avoid MongoDB path existence error
	filter = bson.M{
		"_id":      runID,
		"tests.id": testID,
	}
	update = bson.M{
		"$push": bson.M{"tests.$[test].attempts.$[attempt].steps": step},
		"$set":  bson.M{"updated_at": now},
	}
	arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"attempt.retry_index": retry_index},
		},
	})

	result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append step to test attempt: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("parent test not found: runID=%s, testID=%s, retryIndex=%d", runID, testID, retry_index)
	}

	r.logger.Info("step begin (inserted)",
		"runID", runID,
		"stepID", step.ID,
		"testID", testID,
		"retryIndex", retry_index,
		"matchedCount", result.MatchedCount,
		"modifiedCount", result.ModifiedCount)
	return nil
}

// UpsertStepEnd updates step end fields (status).
// With attempt-based retries: steps are stored in attempts[retry_index].steps.
// - runID: Required. Identifies the document (_id).
// - stepID: Required. Identifies the step to update.
// - testID: Required. ID of test containing the step.
// - retry_index: Required. Retry attempt index containing the step.
// Returns error if runID is empty or step not found.
func (r *MongoRepository) UpsertStepEnd(ctx context.Context, runID string, stepID string, testID string, retry_index int32, status string) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if stepID == "" {
		return fmt.Errorf("stepID is required")
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()

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
		return fmt.Errorf("step not found: runID=%s, stepID=%s, testID=%s, retryIndex=%d", runID, stepID, testID, retry_index)
	}

	r.logger.Info("step end", "runID", runID, "stepID", stepID, "testID", testID, "retryIndex", retry_index, "status", status)
	return nil
}
