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
func (r *MongoRepository) UpdateTestStatus(ctx context.Context, testID string, status string) error {
	now := time.Now()

	updateFields := bson.M{
		"tests.$[test].updated_at": now,
		"tests.$[test].status":     status,
	}

	// Update test in root document's tests array
	filter := bson.M{"tests.id": testID}
	update := bson.M{"$set": updateFields}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
		},
	})

	result, err := r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update test status: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("test status updated", "id", testID, "status", status)
		return nil
	}

	// Try nested suite's tests
	nestedFields := bson.M{
		"suites.$[].tests.$[test].updated_at": now,
		"suites.$[].tests.$[test].status":     status,
	}

	filter = bson.M{"suites.tests.id": testID}
	update = bson.M{"$set": nestedFields}

	_, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested test status: %w", err)
	}

	r.logger.Info("test status updated (nested)", "id", testID, "status", status)
	return nil
}
