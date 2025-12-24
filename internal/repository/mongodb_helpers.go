package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// validateRunID checks if runID is provided and returns an error if not
func validateRunID(runID string) error {
	if runID == "" {
		return fmt.Errorf("runID is required")
	}
	return nil
}

// ensureDocumentExists creates a document if it doesn't exist
func (r *MongoRepository) ensureDocumentExists(ctx context.Context, runID string) error {
	now := time.Now()
	filter := bson.M{"_id": runID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":        runID,
			"created_at": now,
			"updated_at": now,
			"suites":     bson.A{},
			"tests":      bson.A{},
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// buildStepEndUpdate creates the update document for step.end events
func buildStepEndUpdate(status string, now time.Time) bson.M {
	update := bson.M{
		"updated_at": now,
	}
	if status != "" {
		update["status"] = status
	}
	return update
}

// UpdateTestRunEnd updates a test run document with final status, times, and duration
func (r *MongoRepository) UpdateTestRunEnd(ctx context.Context, runID string, status string, startTime *time.Time, duration *int64) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{"_id": runID}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": now,
			"ended_at":   now,
		},
	}

	// Add start_time if provided
	if startTime != nil {
		update["$set"].(bson.M)["start_time"] = startTime
	}

	// Add duration if provided
	if duration != nil {
		update["$set"].(bson.M)["duration"] = duration
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update test run end: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("test run not found: %s", runID)
	}

	r.logger.Info("test run end updated", "run_id", runID, "status", status)
	return nil
}

// AppendTestFailure adds a failure to a test document
func (r *MongoRepository) AppendTestFailure(ctx context.Context, runID, testID string, failure interface{}) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{
		"_id":             runID,
		"suites.tests.id": testID,
	}

	update := bson.M{
		"$push": bson.M{
			"suites.$[].tests.$[test].failures": failure,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	arrayFilters := []interface{}{
		bson.M{"test.id": testID},
	}

	opts := options.Update().SetArrayFilters(options.ArrayFilters{Filters: arrayFilters})
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("append test failure: %w", err)
	}

	return nil
}

// AppendTestError adds an error to a test document
func (r *MongoRepository) AppendTestError(ctx context.Context, runID, testID string, errorDoc interface{}) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{
		"_id":             runID,
		"suites.tests.id": testID,
	}

	update := bson.M{
		"$push": bson.M{
			"suites.$[].tests.$[test].errors": errorDoc,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	arrayFilters := []interface{}{
		bson.M{"test.id": testID},
	}

	opts := options.Update().SetArrayFilters(options.ArrayFilters{Filters: arrayFilters})
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("append test error: %w", err)
	}

	return nil
}

// AppendStdOutput adds stdout output to a test document
func (r *MongoRepository) AppendStdOutput(ctx context.Context, runID, testID string, output interface{}) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{
		"_id":             runID,
		"suites.tests.id": testID,
	}

	update := bson.M{
		"$push": bson.M{
			"suites.$[].tests.$[test].stdout": output,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	arrayFilters := []interface{}{
		bson.M{"test.id": testID},
	}

	opts := options.Update().SetArrayFilters(options.ArrayFilters{Filters: arrayFilters})
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("append stdout: %w", err)
	}

	return nil
}

// AppendStdError adds stderr output to a test document
func (r *MongoRepository) AppendStdError(ctx context.Context, runID, testID string, output interface{}) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{
		"_id":             runID,
		"suites.tests.id": testID,
	}

	update := bson.M{
		"$push": bson.M{
			"suites.$[].tests.$[test].stderr": output,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	arrayFilters := []interface{}{
		bson.M{"test.id": testID},
	}

	opts := options.Update().SetArrayFilters(options.ArrayFilters{Filters: arrayFilters})
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("append stderr: %w", err)
	}

	return nil
}
