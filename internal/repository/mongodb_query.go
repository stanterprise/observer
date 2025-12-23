package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetTestRun retrieves a complete test run document by ID
func (r *MongoRepository) GetTestRun(ctx context.Context, id string) (*m.TestRunDocument, error) {
	var doc m.TestRunDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find test run: %w", err)
	}
	return &doc, nil
}

// ListTestRuns retrieves test runs with pagination and optional filters
func (r *MongoRepository) ListTestRuns(ctx context.Context, filter bson.M, limit, offset int64) ([]*m.TestRunDocument, int64, error) {
	// Get total count
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count test runs: %w", err)
	}

	// Find documents with pagination
	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetSkip(offset).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("find test runs: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []*m.TestRunDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, 0, fmt.Errorf("decode test runs: %w", err)
	}

	return docs, count, nil
}

// GetTestFromRun retrieves a specific test from within a test run document
func (r *MongoRepository) GetTestFromRun(ctx context.Context, testID string) (*m.TestDocument, error) {
	// Try to find test in top-level tests array
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$tests"}},
		{{Key: "$match", Value: bson.M{"tests.id": testID}}},
		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$tests"}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate test: %w", err)
	}

	var tests []*m.TestDocument
	if err := cursor.All(ctx, &tests); err != nil {
		cursor.Close(ctx)
		return nil, fmt.Errorf("decode test: %w", err)
	}
	cursor.Close(ctx)

	if len(tests) > 0 {
		return tests[0], nil
	}

	// If not found in top-level tests, search in nested suites
	nestedPipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$suites"}},
		{{Key: "$unwind", Value: "$suites.tests"}},
		{{Key: "$match", Value: bson.M{"suites.tests.id": testID}}},
		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$suites.tests"}}},
	}

	cursor, err = r.collection.Aggregate(ctx, nestedPipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate nested test: %w", err)
	}
	defer cursor.Close(ctx)

	tests = []*m.TestDocument{}
	if err := cursor.All(ctx, &tests); err != nil {
		return nil, fmt.Errorf("decode nested test: %w", err)
	}

	if len(tests) == 0 {
		return nil, nil
	}

	return tests[0], nil
}

// UpdateTestStatus updates the status of a test case run
func (r *MongoRepository) UpdateTestStatus(ctx context.Context, runID string, testID string, status string) error {
	if err := validateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()

	// Try root-level suite tests
	filter := bson.M{
		"_id":             runID,
		"suites.tests.id": testID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$[].tests.$[test].status":     status,
			"suites.$[].tests.$[test].updated_at": now,
			"updated_at":                          now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update test status: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("test status updated", "runID", runID, "testID", testID, "status", status)
		return nil
	}

	// Try nested suite tests
	filter = bson.M{
		"_id":                    runID,
		"suites.suites.tests.id": testID,
	}
	update = bson.M{
		"$set": bson.M{
			"suites.$[].suites.$[].tests.$[test].status":     status,
			"suites.$[].suites.$[].tests.$[test].updated_at": now,
			"updated_at": now,
		},
	}

	result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested test status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("test not found: runID=%s, testID=%s", runID, testID)
	}

	r.logger.Info("test status updated", "runID", runID, "testID", testID, "status", status)
	return nil
}

// SuiteExists checks if a suite exists in the repository
// For nested suites, it extracts the root document ID and checks if the suite exists in the document hierarchy
func (r *MongoRepository) SuiteExists(ctx context.Context, suiteID string) (bool, error) {
	// Root suite check - suite is the document itself
	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": suiteID})
	if err != nil {
		return false, fmt.Errorf("count root suite: %w", err)
	}
	if count > 0 {
		return true, nil
	}

	// Check if suite exists in nested suites array
	count, err = r.collection.CountDocuments(ctx, bson.M{"suites.id": suiteID})
	if err != nil {
		return false, fmt.Errorf("count nested suite: %w", err)
	}

	return count > 0, nil
}

// TestExists checks if a test exists in the repository
// It checks both root-level tests array and nested suite tests arrays
func (r *MongoRepository) TestExists(ctx context.Context, testID string) (bool, error) {
	// Check root-level tests array
	count, err := r.collection.CountDocuments(ctx, bson.M{"tests.id": testID})
	if err != nil {
		return false, fmt.Errorf("count root test: %w", err)
	}
	if count > 0 {
		return true, nil
	}

	// Check nested suite tests
	count, err = r.collection.CountDocuments(ctx, bson.M{"suites.tests.id": testID})
	if err != nil {
		return false, fmt.Errorf("count nested test: %w", err)
	}

	return count > 0, nil
}

// StepExists checks if a step exists within a test
// It checks both root-level and nested suite tests
func (r *MongoRepository) StepExists(ctx context.Context, stepID string) (bool, error) {
	// Check steps in root-level tests
	count, err := r.collection.CountDocuments(ctx, bson.M{"tests.steps.id": stepID})
	if err != nil {
		return false, fmt.Errorf("count root test step: %w", err)
	}
	if count > 0 {
		return true, nil
	}

	// Check steps in nested suite tests
	count, err = r.collection.CountDocuments(ctx, bson.M{"suites.tests.steps.id": stepID})
	if err != nil {
		return false, fmt.Errorf("count nested test step: %w", err)
	}

	return count > 0, nil
}
