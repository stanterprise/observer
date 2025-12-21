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
// - runID: Required. Identifies the document (_id).
// - step: The step to create/update (step.ID identifies the step).
// - testID: Required. ID of parent test containing this step.
// - parentStepID: Empty for direct child of test, or ID of parent step for nested steps.
// Returns error if runID is empty or parent test not found.
func (r *MongoRepository) UpsertStepBegin(ctx context.Context, runID string, step *m.StepDocument, testID string, parentStepID string) error {
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
	step.ParentStepID = parentStepID

	if step.Steps == nil {
		step.Steps = []*m.StepDocument{}
	}

	if parentStepID == "" {
		// Direct child of test
		return r.upsertStepInTest(ctx, runID, testID, step, now)
	}

	// Nested step (child of another step)
	return r.upsertNestedStep(ctx, runID, testID, parentStepID, step, now)
}

// upsertStepInTest handles steps that are direct children of tests
func (r *MongoRepository) upsertStepInTest(ctx context.Context, runID string, testID string, step *m.StepDocument, now time.Time) error {
	// Try to update existing step in root-level suite tests
	filter := bson.M{
		"_id":                   runID,
		"suites.tests.id":       testID,
		"suites.tests.steps.id": step.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$[].tests.$[test].steps.$[step].status":     step.Status,
			"suites.$[].tests.$[test].steps.$[step].category":   step.Category,
			"suites.$[].tests.$[test].steps.$[step].title":      step.Title,
			"suites.$[].tests.$[test].steps.$[step].updated_at": now,
			"updated_at": now,
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
		"_id":             runID,
		"suites.tests.id": testID,
	}
	update = bson.M{
		"$push": bson.M{"suites.$[].tests.$[test].steps": step},
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
		// Try nested suite tests
		filter = bson.M{
			"_id":                    runID,
			"suites.suites.tests.id": testID,
		}
		update = bson.M{
			"$push": bson.M{"suites.$[].suites.$[].tests.$[test].steps": step},
			"$set":  bson.M{"updated_at": now},
		}

		result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("append step to nested test: %w", err)
		}

		if result.MatchedCount == 0 {
			return fmt.Errorf("parent test not found: runID=%s, testID=%s", runID, testID)
		}
	}

	r.logger.Info("step begin (inserted)", "runID", runID, "stepID", step.ID, "testID", testID)
	return nil
}

// upsertNestedStep handles steps that are children of other steps
func (r *MongoRepository) upsertNestedStep(ctx context.Context, runID string, testID string, parentStepID string, step *m.StepDocument, now time.Time) error {
	// Try to update existing nested step
	filter := bson.M{
		"_id":                          runID,
		"suites.tests.id":              testID,
		"suites.tests.steps.id":        parentStepID,
		"suites.tests.steps.steps.id": step.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$[].tests.$[test].steps.$[parent].steps.$[step].status":     step.Status,
			"suites.$[].tests.$[test].steps.$[parent].steps.$[step].category":   step.Category,
			"suites.$[].tests.$[test].steps.$[parent].steps.$[step].title":      step.Title,
			"suites.$[].tests.$[test].steps.$[parent].steps.$[step].updated_at": now,
			"updated_at": now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
Filters: []interface{}{
bson.M{"test.id": testID},
bson.M{"parent.id": parentStepID},
bson.M{"step.id": step.ID},
},
})

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested step: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("step begin (nested, updated)", "runID", runID, "stepID", step.ID, "parentStepID", parentStepID)
		return nil
	}

	// Step doesn't exist, append it to parent step's steps array
	filter = bson.M{
		"_id":                   runID,
		"suites.tests.id":       testID,
		"suites.tests.steps.id": parentStepID,
	}
	update = bson.M{
		"$push": bson.M{"suites.$[].tests.$[test].steps.$[parent].steps": step},
		"$set":  bson.M{"updated_at": now},
	}
	arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
Filters: []interface{}{
bson.M{"test.id": testID},
bson.M{"parent.id": parentStepID},
},
})

	result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append nested step: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("parent step not found: runID=%s, testID=%s, parentStepID=%s", runID, testID, parentStepID)
	}

	r.logger.Info("step begin (nested, inserted)", "runID", runID, "stepID", step.ID, "parentStepID", parentStepID)
	return nil
}

// UpsertStepEnd updates step end fields (status).
// - runID: Required. Identifies the document (_id).
// - stepID: Required. Identifies the step to update.
// - testID: Required. ID of test containing the step (helps narrow search).
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

	// Try direct child of test
	filter := bson.M{
		"_id":                   runID,
		"suites.tests.id":       testID,
		"suites.tests.steps.id": stepID,
	}
	setFields := bson.M{"updated_at": now}
	for k, v := range updateFields {
		setFields["suites.$[].tests.$[test].steps.$[step]."+k] = v
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

	if result.MatchedCount > 0 {
		r.logger.Info("step end", "runID", runID, "stepID", stepID, "status", status)
		return nil
	}

	// Try nested step
	filter = bson.M{
		"_id":                          runID,
		"suites.tests.id":              testID,
		"suites.tests.steps.steps.id": stepID,
	}
	setFields = bson.M{"updated_at": now}
	for k, v := range updateFields {
		setFields["suites.$[].tests.$[test].steps.$[].steps.$[step]."+k] = v
	}

	result, err = r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields}, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested step end: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("step not found: runID=%s, stepID=%s, testID=%s", runID, stepID, testID)
	}

	r.logger.Info("step end", "runID", runID, "stepID", stepID, "status", status)
	return nil
}
