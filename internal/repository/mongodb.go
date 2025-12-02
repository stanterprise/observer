package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRepository handles MongoDB operations for test runs
type MongoRepository struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository(collection *mongo.Collection, logger *slog.Logger) *MongoRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &MongoRepository{
		collection: collection,
		logger:     logger,
	}
}

// UpsertSuiteBegin handles suite begin events.
// - If root suite (no parent), creates a new TestRunDocument
// - If non-root suite, appends to the parent suite's suites array
func (r *MongoRepository) UpsertSuiteBegin(ctx context.Context, suite *m.SuiteDocument, parentSuiteID string) error {
	now := time.Now()
	suite.CreatedAt = now
	suite.UpdatedAt = now

	if parentSuiteID == "" {
		// Root suite - create new document
		doc := &m.TestRunDocument{
			ID:              suite.ID,
			Name:            suite.Name,
			Description:     suite.Description,
			Status:          suite.Status,
			Metadata:        suite.Metadata,
			TestSuiteSpecID: suite.TestSuiteSpecID,
			InitiatedBy:     suite.InitiatedBy,
			ProjectName:     suite.ProjectName,
			StartTime:       suite.StartTime,
			CreatedAt:       now,
			UpdatedAt:       now,
			Suites:          []*m.SuiteDocument{},
			Tests:           []*m.TestDocument{},
		}

		// Upsert the root document
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"_id": suite.ID}
		update := bson.M{
			"$setOnInsert": bson.M{
				"_id":        suite.ID,
				"created_at": now,
				"suites":     []*m.SuiteDocument{},
				"tests":      []*m.TestDocument{},
			},
			"$set": bson.M{
				"name":               doc.Name,
				"description":        doc.Description,
				"status":             doc.Status,
				"metadata":           doc.Metadata,
				"test_suite_spec_id": doc.TestSuiteSpecID,
				"initiated_by":       doc.InitiatedBy,
				"project_name":       doc.ProjectName,
				"start_time":         doc.StartTime,
				"updated_at":         now,
			},
		}

		_, err := r.collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			return fmt.Errorf("upsert root suite: %w", err)
		}

		r.logger.Info("suite begin (root)", "id", suite.ID, "name", suite.Name)
		return nil
	}

	// Non-root suite - append to parent
	suite.ParentSuiteID = parentSuiteID

	// Try to find and update in root document's suites array
	filter := bson.M{"_id": parentSuiteID}
	update := bson.M{
		"$push": bson.M{
			"suites": suite,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("append suite to parent: %w", err)
	}

	if result.MatchedCount == 0 {
		// Parent not found at root level, try nested update
		// Use array filters to find nested parent suite
		filter = bson.M{
			"suites.id": parentSuiteID,
		}
		update = bson.M{
			"$push": bson.M{
				"suites.$[parent].suites": suite,
			},
			"$set": bson.M{
				"updated_at": now,
			},
		}
		arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"parent.id": parentSuiteID},
			},
		})

		_, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("append suite to nested parent: %w", err)
		}
	}

	r.logger.Info("suite begin (nested)", "id", suite.ID, "parent", parentSuiteID)
	return nil
}

// UpsertSuiteEnd handles suite end events by updating the suite's attributes
func (r *MongoRepository) UpsertSuiteEnd(ctx context.Context, suiteID string, status string, endTime *time.Time, duration *int64) error {
	now := time.Now()

	updateFields := bson.M{
		"updated_at": now,
	}
	if status != "" {
		updateFields["status"] = status
	}
	if endTime != nil {
		updateFields["end_time"] = endTime
	}
	if duration != nil {
		updateFields["duration"] = duration
	}

	// Try to update root document first
	filter := bson.M{"_id": suiteID}
	update := bson.M{"$set": updateFields}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update suite end: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("suite end (root)", "id", suiteID, "status", status)
		return nil
	}

	// Try nested suite update
	nestedFields := bson.M{}
	for k, v := range updateFields {
		nestedFields["suites.$[suite]."+k] = v
	}

	filter = bson.M{"suites.id": suiteID}
	update = bson.M{"$set": nestedFields}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"suite.id": suiteID},
		},
	})

	_, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested suite end: %w", err)
	}

	r.logger.Info("suite end (nested)", "id", suiteID, "status", status)
	return nil
}

// UpsertTestBegin handles test begin events by appending to the parent suite
func (r *MongoRepository) UpsertTestBegin(ctx context.Context, test *m.TestDocument, suiteID string) error {
	now := time.Now()
	test.CreatedAt = now
	test.UpdatedAt = now
	test.SuiteID = suiteID
	test.Steps = []*m.StepDocument{}

	// Try to append to root document's tests array
	filter := bson.M{"_id": suiteID}
	update := bson.M{
		"$push": bson.M{
			"tests": test,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("append test to suite: %w", err)
	}

	if result.MatchedCount == 0 {
		// Suite not found at root level, try nested
		filter = bson.M{"suites.id": suiteID}
		update = bson.M{
			"$push": bson.M{
				"suites.$[suite].tests": test,
			},
			"$set": bson.M{
				"updated_at": now,
			},
		}
		arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"suite.id": suiteID},
			},
		})

		_, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("append test to nested suite: %w", err)
		}
	}

	r.logger.Info("test begin", "id", test.ID, "title", test.Title, "suite", suiteID)
	return nil
}

// UpsertTestEnd handles test end events by updating the test's attributes
func (r *MongoRepository) UpsertTestEnd(ctx context.Context, testID string, status string, duration *int64) error {
	now := time.Now()

	updateFields := bson.M{
		"tests.$[test].updated_at": now,
	}
	if status != "" {
		updateFields["tests.$[test].status"] = status
	}
	if duration != nil {
		updateFields["tests.$[test].duration"] = duration
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
		return fmt.Errorf("update test end: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("test end", "id", testID, "status", status)
		return nil
	}

	// Try nested suite's tests
	nestedFields := bson.M{
		"suites.$[].tests.$[test].updated_at": now,
	}
	if status != "" {
		nestedFields["suites.$[].tests.$[test].status"] = status
	}
	if duration != nil {
		nestedFields["suites.$[].tests.$[test].duration"] = duration
	}

	filter = bson.M{"suites.tests.id": testID}
	update = bson.M{"$set": nestedFields}

	_, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested test end: %w", err)
	}

	r.logger.Info("test end (nested)", "id", testID, "status", status)
	return nil
}

// UpsertStepBegin handles step begin events
func (r *MongoRepository) UpsertStepBegin(ctx context.Context, step *m.StepDocument, testID string, parentStepID string) error {
	now := time.Now()
	step.CreatedAt = now
	step.UpdatedAt = now
	step.TestCaseRunID = testID
	step.ParentStepID = parentStepID
	step.Steps = []*m.StepDocument{}

	if parentStepID == "" {
		// Direct child of test - append to test's steps array
		filter := bson.M{"tests.id": testID}
		update := bson.M{
			"$push": bson.M{
				"tests.$[test].steps": step,
			},
			"$set": bson.M{
				"updated_at": now,
			},
		}
		arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"test.id": testID},
			},
		})

		result, err := r.collection.UpdateMany(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("append step to test: %w", err)
		}

		if result.MatchedCount == 0 {
			// Try in nested suites
			filter = bson.M{"suites.tests.id": testID}
			update = bson.M{
				"$push": bson.M{
					"suites.$[].tests.$[test].steps": step,
				},
				"$set": bson.M{
					"updated_at": now,
				},
			}

			_, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
			if err != nil {
				return fmt.Errorf("append step to nested test: %w", err)
			}
		}

		r.logger.Info("step begin", "id", step.ID, "test", testID)
		return nil
	}

	// Nested step - append to parent step's steps array
	// Note: Deep nesting of steps (step -> step -> step) requires recursive updates
	// which are complex with MongoDB's array update operators. For now, we log a warning
	// and store the parent step reference. The step will still be queryable by ID.
	r.logger.Warn("nested step support is limited; step will be stored with parent reference only",
		"id", step.ID, "parent", parentStepID)

	// Store the nested step at the test level with parent reference
	// This allows later retrieval and tree reconstruction
	filter := bson.M{"tests.id": testID}
	update := bson.M{
		"$push": bson.M{
			"tests.$[test].steps": step,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
		},
	})

	_, err := r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append nested step to test: %w", err)
	}

	r.logger.Info("step begin (nested)", "id", step.ID, "parent", parentStepID)
	return nil
}

// UpsertStepEnd handles step end events by updating the step's attributes
func (r *MongoRepository) UpsertStepEnd(ctx context.Context, stepID string, status string) error {
	now := time.Now()

	updateFields := bson.M{
		"tests.$[].steps.$[step].updated_at": now,
	}
	if status != "" {
		updateFields["tests.$[].steps.$[step].status"] = status
	}

	filter := bson.M{"tests.steps.id": stepID}
	update := bson.M{"$set": updateFields}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"step.id": stepID},
		},
	})

	result, err := r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update step end: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("step end", "id", stepID, "status", status)
		return nil
	}

	// Try in nested suites
	nestedFields := bson.M{
		"suites.$[].tests.$[].steps.$[step].updated_at": now,
	}
	if status != "" {
		nestedFields["suites.$[].tests.$[].steps.$[step].status"] = status
	}

	filter = bson.M{"suites.tests.steps.id": stepID}
	update = bson.M{"$set": nestedFields}

	_, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested step end: %w", err)
	}

	r.logger.Info("step end (nested)", "id", stepID, "status", status)
	return nil
}

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
	// Use aggregation to find the specific test
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$tests"}},
		{{Key: "$match", Value: bson.M{"tests.id": testID}}},
		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$tests"}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate test: %w", err)
	}
	defer cursor.Close(ctx)

	var tests []*m.TestDocument
	if err := cursor.All(ctx, &tests); err != nil {
		return nil, fmt.Errorf("decode test: %w", err)
	}

	if len(tests) == 0 {
		return nil, nil
	}

	return tests[0], nil
}
