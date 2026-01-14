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

// TestRunExists checks if a test run document exists by ID
func (r *MongoRepository) TestRunExists(ctx context.Context, runID string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": runID})
	if err != nil {
		return false, fmt.Errorf("count test run: %w", err)
	}
	return count > 0, nil
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

// TestTrendItem represents a single test execution in the history
type TestTrendItem struct {
	TestID    string     `json:"testId"`
	RunID     string     `json:"runId"`
	Status    string     `json:"status"`
	Duration  *int64     `json:"duration,omitempty"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// GetTestTrends retrieves historical test execution data for a specific test ID across multiple runs
// Returns test executions sorted by created_at in descending order (most recent first)
func (r *MongoRepository) GetTestTrends(ctx context.Context, testID string, limit int64) ([]*TestTrendItem, error) {
	if testID == "" {
		return nil, fmt.Errorf("testID is required")
	}

	if limit <= 0 {
		limit = 50 // Default limit
	}

	// Aggregate pipeline to find all tests with the given testID across all runs
	// This searches both root-level tests and nested suite tests
	pipeline := mongo.Pipeline{
		// Stage 1: Match documents that contain the test ID in either root or nested tests
		{{Key: "$match", Value: bson.M{
			"$or": []bson.M{
				{"tests.id": testID},
				{"suites.tests.id": testID},
			},
		}}},
		// Stage 2: Add a field with all matching tests
		{{Key: "$addFields", Value: bson.M{
			"allTests": bson.M{
				"$concatArrays": []interface{}{
					bson.M{"$ifNull": []interface{}{"$tests", []interface{}{}}},
					bson.M{
						"$reduce": bson.M{
							"input":        "$suites",
							"initialValue": []interface{}{},
							"in": bson.M{
								"$concatArrays": []interface{}{
									"$$value",
									bson.M{"$ifNull": []interface{}{"$$this.tests", []interface{}{}}},
								},
							},
						},
					},
				},
			},
		}}},
		// Stage 3: Unwind the allTests array
		{{Key: "$unwind", Value: "$allTests"}},
		// Stage 4: Match only the specific test ID
		{{Key: "$match", Value: bson.M{"allTests.id": testID}}},
		// Stage 5: Project the required fields
		{{Key: "$project", Value: bson.M{
			"testId":    "$allTests.id",
			"runId":     "$_id",
			"status":    "$allTests.status",
			"duration":  "$allTests.duration",
			"startTime": "$allTests.start_time",
			"endTime":   "$allTests.end_time",
			"createdAt": "$allTests.created_at",
		}}},
		// Stage 6: Sort by created_at descending (most recent first)
		{{Key: "$sort", Value: bson.M{"createdAt": -1}}},
		// Stage 7: Limit results
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate test trends: %w", err)
	}
	defer cursor.Close(ctx)

	var trends []*TestTrendItem
	if err := cursor.All(ctx, &trends); err != nil {
		return nil, fmt.Errorf("decode test trends: %w", err)
	}

	return trends, nil
}
