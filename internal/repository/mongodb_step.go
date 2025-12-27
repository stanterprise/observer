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
// Steps are stored in a flat array within the test object.
// - runID: Required. Identifies the document (_id).
// - step: The step to create/update (step.ID identifies the step).
// - testID: Required. ID of parent test containing this step.
// Returns error if runID is empty or parent test not found.
func (r *MongoRepository) UpsertStepBegin(ctx context.Context, runID string, step *m.StepDocument, testID string) error {
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

	return r.upsertStepInTest(ctx, runID, testID, step, now)
}

// upsertStepInTest handles steps as flat children of tests
func (r *MongoRepository) upsertStepInTest(ctx context.Context, runID string, testID string, step *m.StepDocument, now time.Time) error {
	// Try to update existing step
	filter := bson.M{
		"_id":            runID,
		"tests.id":       testID,
		"tests.steps.id": step.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"tests.$[test].steps.$[step].parent_step_id": step.ParentStepID,
			"tests.$[test].steps.$[step].title":          step.Title,
			"tests.$[test].steps.$[step].description":    step.Description,
			"tests.$[test].steps.$[step].start_time":     step.StartTime,
			"tests.$[test].steps.$[step].duration":       step.Duration,
			"tests.$[test].steps.$[step].type":           step.Type,
			"tests.$[test].steps.$[step].metadata":       step.Metadata,
			"tests.$[test].steps.$[step].worker_index":   step.WorkerIndex,
			"tests.$[test].steps.$[step].status":         step.Status,
			"tests.$[test].steps.$[step].category":       step.Category,
			"tests.$[test].steps.$[step].location":       step.Location,
			"tests.$[test].steps.$[step].error":          step.Error,
			"tests.$[test].steps.$[step].errors":         step.Errors,
			"tests.$[test].steps.$[step].updated_at":     now,
			"updated_at":                                 now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"step.id": step.ID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update step in test: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("step begin (updated)", "runID", runID, "stepID", step.ID, "testID", testID)
		return nil
	}

	// Step doesn't exist, append it to test's steps array
	filter = bson.M{
		"_id":      runID,
		"tests.id": testID,
	}
	update = bson.M{
		"$push": bson.M{"tests.$[test].steps": step},
		"$set":  bson.M{"updated_at": now},
	}
	arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
		},
	})

	result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append step to test: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("parent test not found: runID=%s, testID=%s", runID, testID)
	}

	r.logger.Info("step begin (inserted)", "runID", runID, "stepID", step.ID, "testID", testID)
	return nil
}

// UpsertStepEnd updates step end fields (status).
// Steps are stored in a flat array within the test object.
// - runID: Required. Identifies the document (_id).
// - stepID: Required. Identifies the step to update.
// - testID: Required. ID of test containing the step.
// Returns error if runID is empty or step not found.
func (r *MongoRepository) UpsertStepEnd(ctx context.Context, runID string, stepID string, testID string, status string) error {
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
	updateFields := bson.M{"updated_at": now}
	if status != "" {
		updateFields["status"] = status
	}

	// Update step in root-level tests array
	filter := bson.M{
		"_id":            runID,
		"tests.id":       testID,
		"tests.steps.id": stepID,
	}
	setFields := bson.M{"updated_at": now}
	for k, v := range updateFields {
		setFields["tests.$[test].steps.$[step]."+k] = v
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"step.id": stepID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields}, arrayFilters)
	if err != nil {
		return fmt.Errorf("update step end: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("step not found: runID=%s, stepID=%s, testID=%s", runID, stepID, testID)
	}

	r.logger.Info("step end", "runID", runID, "stepID", stepID, "status", status)
	return nil
}
