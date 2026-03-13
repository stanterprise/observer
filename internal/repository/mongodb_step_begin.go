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
	if err := ValidateRunID(runID); err != nil {
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

	r.logger.Debug("UpsertStepBegin starting",
		"runID", runID,
		"stepID", step.ID,
		"testID", testID,
		"retryIndex", retry_index,
		"stepTitle", step.Title)

	return r.upsertStepInTestAttempt(ctx, runID, testID, retry_index, step, now)
}

// upsertStepInTestAttempt handles steps as children of attempts[retry_index] array.
// With attempt-based retries: steps are stored in attempts[retry_index].steps instead of tests.steps.
// Note: "step begin" events should ONLY insert new steps, never update existing ones.
func (r *MongoRepository) upsertStepInTestAttempt(ctx context.Context, runID string, testID string, retry_index int32, step *m.StepDocument, now time.Time) error {
	// Step begin event: always insert a new step into the attempts[retry_index].steps array
	filter := bson.M{
		"_id":      runID,
		"tests.id": testID,
	}
	update := bson.M{
		"$push": bson.M{"tests.$[test].attempts.$[attempt].steps": step},
		"$set":  bson.M{"updated_at": now},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"attempt.retry_index": retry_index},
		},
	})

	r.logger.Debug("Inserting new step into attempt",
		"runID", runID,
		"stepID", step.ID,
		"stepTitle", step.Title,
		"testID", testID,
		"retryIndex", retry_index)

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("insert step into test attempt: %w", err)
	}

	if result.MatchedCount == 0 {
		r.logger.Error("parent test or attempt not found for step",
			"runID", runID,
			"testID", testID,
			"retryIndex", retry_index,
			"stepID", step.ID,
			"filter", filter)
		return &ErrParentNotFound{
			ParentType: "test",
			ParentID:   testID,
			ChildType:  "step",
			ChildID:    step.ID,
		}
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
