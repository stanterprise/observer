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

// extractRootSuiteID extracts the root suite ID from a potentially nested suite ID
// Example: "abc123-suite-root" -> "abc123-suite-root"
// Example: "abc123-suite-/path/to/suite" -> "abc123-suite-root"
func extractRootSuiteID(suiteID string) string {
	// Look for the pattern: {base-id}-suite-{path}
	// We want to return {base-id}-suite-root
	// Note: base-id itself might contain "-suite-" as part of the UUID
	// So we look for the LAST occurrence of "-suite-"

	suiteMarker := "-suite-"
	lastIdx := -1

	// Find last occurrence of "-suite-"
	for i := 0; i <= len(suiteID)-len(suiteMarker); i++ {
		if suiteID[i:i+len(suiteMarker)] == suiteMarker {
			lastIdx = i
		}
	}

	if lastIdx >= 0 {
		// Found "-suite-", extract base ID and append "-suite-root"
		baseID := suiteID[:lastIdx]
		return baseID + "-suite-root"
	}

	// No "-suite-" found, assume it's already a root ID or malformed
	return suiteID + "-suite-root"
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

// UpsertSuiteBegin handles suite begin events with true upsert semantics:
// - If root suite (no parent): Creates a new TestRunDocument AND adds the suite to its own suites array
// - If nested suite: Finds the suite by ID within parent and updates if exists, inserts if not
// This implements the requirement that when a suite is reported, the handler should
// find the entity in the root suite document and upsert it.
// For root suites, we create both the root document AND add the suite to the suites array
// so that tests can consistently find their parent suite in the suites array.
func (r *MongoRepository) UpsertSuiteBegin(ctx context.Context, suite *m.SuiteDocument, parentSuiteID string) error {
	now := time.Now()
	suite.CreatedAt = now
	suite.UpdatedAt = now

	if parentSuiteID == "" {
		// Root suite - create new document AND add suite to suites array
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
			Suites:          []*m.SuiteDocument{suite}, // Add the root suite to its own suites array
			Tests:           []*m.TestDocument{},
		}

		// Upsert the root document
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"_id": suite.ID}
		update := bson.M{
			"$setOnInsert": bson.M{
				"_id":        suite.ID,
				"created_at": now,
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

		result, err := r.collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			return fmt.Errorf("upsert root suite: %w", err)
		}

		// If this was an insert (not an update), also ensure the suite is in the suites array
		if result.UpsertedCount > 0 {
			// Document was created, now add the root suite to the suites array
			filter = bson.M{"_id": suite.ID}
			update = bson.M{
				"$set": bson.M{
					"suites": []*m.SuiteDocument{suite},
				},
			}
			_, err = r.collection.UpdateOne(ctx, filter, update)
			if err != nil {
				return fmt.Errorf("add root suite to suites array: %w", err)
			}
		} else {
			// Document existed, update the root suite in the suites array
			// First try to update existing suite entry
			filter = bson.M{
				"_id":        suite.ID,
				"suites.id": suite.ID,
			}
			update = bson.M{
				"$set": bson.M{
					"suites.$.name":               suite.Name,
					"suites.$.description":        suite.Description,
					"suites.$.status":             suite.Status,
					"suites.$.metadata":           suite.Metadata,
					"suites.$.test_suite_spec_id": suite.TestSuiteSpecID,
					"suites.$.initiated_by":       suite.InitiatedBy,
					"suites.$.project_name":       suite.ProjectName,
					"suites.$.start_time":         suite.StartTime,
					"suites.$.updated_at":         now,
				},
			}
			result, err = r.collection.UpdateOne(ctx, filter, update)
			if err != nil {
				return fmt.Errorf("update root suite in suites array: %w", err)
			}

			if result.MatchedCount == 0 {
				// Suite entry doesn't exist in suites array, add it
				filter = bson.M{"_id": suite.ID}
				update = bson.M{
					"$push": bson.M{
						"suites": suite,
					},
				}
				_, err = r.collection.UpdateOne(ctx, filter, update)
				if err != nil {
					return fmt.Errorf("append root suite to suites array: %w", err)
				}
			}
		}

		r.logger.Info("suite begin (root)", "id", suite.ID, "name", suite.Name)
		return nil
	}

	// Non-root suite - upsert to parent (update if exists, insert if not)
	suite.ParentSuiteID = parentSuiteID

	// First, try to update existing suite if it already exists in root document's suites array
	filter := bson.M{
		"_id":        parentSuiteID,
		"suites.id": suite.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$.name":               suite.Name,
			"suites.$.description":        suite.Description,
			"suites.$.status":             suite.Status,
			"suites.$.metadata":           suite.Metadata,
			"suites.$.test_suite_spec_id": suite.TestSuiteSpecID,
			"suites.$.initiated_by":       suite.InitiatedBy,
			"suites.$.project_name":       suite.ProjectName,
			"suites.$.start_time":         suite.StartTime,
			"suites.$.updated_at":         now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update existing suite in parent: %w", err)
	}

	if result.MatchedCount > 0 {
		// Suite was found and updated
		r.logger.Info("suite begin (nested, updated)", "id", suite.ID, "parent", parentSuiteID)
		return nil
	}

	// Suite doesn't exist, append it to root document
	filter = bson.M{"_id": parentSuiteID}
	update = bson.M{
		"$push": bson.M{
			"suites": suite,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	result, err = r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("append suite to parent: %w", err)
	}

	if result.MatchedCount == 0 {
		// Parent not found at root level, try nested update
		// First check if suite already exists in nested structure
		filter = bson.M{
			"suites.id":        parentSuiteID,
			"suites.suites.id": suite.ID,
		}
		update = bson.M{
			"$set": bson.M{
				"suites.$[parent].suites.$[suite].name":               suite.Name,
				"suites.$[parent].suites.$[suite].description":        suite.Description,
				"suites.$[parent].suites.$[suite].status":             suite.Status,
				"suites.$[parent].suites.$[suite].metadata":           suite.Metadata,
				"suites.$[parent].suites.$[suite].test_suite_spec_id": suite.TestSuiteSpecID,
				"suites.$[parent].suites.$[suite].initiated_by":       suite.InitiatedBy,
				"suites.$[parent].suites.$[suite].project_name":       suite.ProjectName,
				"suites.$[parent].suites.$[suite].start_time":         suite.StartTime,
				"suites.$[parent].suites.$[suite].updated_at":         now,
				"updated_at": now,
			},
		}
		arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"parent.id": parentSuiteID},
				bson.M{"suite.id": suite.ID},
			},
		})

		result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("update nested suite: %w", err)
		}

		if result.MatchedCount > 0 {
			r.logger.Info("suite begin (deeply nested, updated)", "id", suite.ID, "parent", parentSuiteID)
			return nil
		}

		// Suite doesn't exist in nested structure, append it
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
		arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"parent.id": parentSuiteID},
			},
		})

		_, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("append suite to nested parent: %w", err)
		}
		r.logger.Info("suite begin (deeply nested, inserted)", "id", suite.ID, "parent", parentSuiteID)
		return nil
	}

	r.logger.Info("suite begin (nested)", "id", suite.ID, "parent", parentSuiteID)
	return nil
}

// UpsertSuiteEnd handles suite end events by finding the suite within the root document
// structure and updating its attributes (status, end_time, duration).
// Searches both root-level suites and nested suites.
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

// UpsertTestBegin handles test begin events by upserting to the parent suite
// (update if exists, insert if not)
// Tests should always be added to suite's tests array, never to root document's tests array
func (r *MongoRepository) UpsertTestBegin(ctx context.Context, test *m.TestDocument, suiteID string) error {
	now := time.Now()
	test.CreatedAt = now
	test.UpdatedAt = now
	test.SuiteID = suiteID
	if test.Steps == nil {
		test.Steps = []*m.StepDocument{}
	}

	// First, try to find and update the test in a nested suite (level 1 - direct child of root)
	filter := bson.M{
		"suites.id":        suiteID,
		"suites.tests.id": test.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$[suite].tests.$[test].title":       test.Title,
			"suites.$[suite].tests.$[test].status":      test.Status,
			"suites.$[suite].tests.$[test].metadata":    test.Metadata,
			"suites.$[suite].tests.$[test].duration":    test.Duration,
			"suites.$[suite].tests.$[test].retry_count": test.RetryCount,
			"suites.$[suite].tests.$[test].retry_index": test.RetryIndex,
			"suites.$[suite].tests.$[test].timeout":     test.Timeout,
			"suites.$[suite].tests.$[test].updated_at":  now,
			"updated_at": now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"suite.id": suiteID},
			bson.M{"test.id": test.ID},
		},
	})

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update test in nested suite: %w", err)
	}

	if result.MatchedCount > 0 {
		// Test was found and updated in nested suite
		r.logger.Info("test begin (nested, updated)", "id", test.ID, "title", test.Title, "suite", suiteID)
		return nil
	}

	// Test doesn't exist in nested suite, try to append it
	filter = bson.M{"suites.id": suiteID}
	update = bson.M{
		"$push": bson.M{
			"suites.$[suite].tests": test,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}
	arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"suite.id": suiteID},
		},
	})

	result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append test to nested suite: %w", err)
	}

	if result.MatchedCount > 0 {
		// Test was appended to nested suite
		r.logger.Info("test begin (nested, inserted)", "id", test.ID, "title", test.Title, "suite", suiteID)
		return nil
	}

	// Suite not found in level 1, this means the suiteID might be the root document ID
	// In this case, we should NOT add to root document's tests array
	// Instead, log a warning - tests should always belong to a suite
	r.logger.Warn("test has no parent suite, suite may not have been created yet",
		"test_id", test.ID,
		"suite_id", suiteID,
		"run_id", test.RunID)

	// As a fallback, check if the root document exists with this ID
	// If it does, we should NOT add tests to root level - they need a suite
	var doc m.TestRunDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": suiteID}).Decode(&doc)
	if err == nil {
		// First check if test already exists in nested suite
		filter = bson.M{
			"suites.id":        suiteID,
			"suites.tests.id": test.ID,
		}
		update = bson.M{
			"$set": bson.M{
				"suites.$[suite].tests.$[test].title":       test.Title,
				"suites.$[suite].tests.$[test].status":      test.Status,
				"suites.$[suite].tests.$[test].metadata":    test.Metadata,
				"suites.$[suite].tests.$[test].duration":    test.Duration,
				"suites.$[suite].tests.$[test].retry_count": test.RetryCount,
				"suites.$[suite].tests.$[test].retry_index": test.RetryIndex,
				"suites.$[suite].tests.$[test].timeout":     test.Timeout,
				"suites.$[suite].tests.$[test].updated_at":  now,
				"updated_at": now,
			},
		}
		arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"suite.id": suiteID},
				bson.M{"test.id": test.ID},
			},
		})

		result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("update test in nested suite: %w", err)
		}

		if result.MatchedCount > 0 {
			r.logger.Info("test begin (nested, updated)", "id", test.ID, "title", test.Title, "suite", suiteID)
			return nil
		}

		// Test doesn't exist in nested suite, try to append it
		filter = bson.M{"suites.id": suiteID}
		update = bson.M{
			"$push": bson.M{
				"suites.$[suite].tests": test,
			},
			"$set": bson.M{
				"updated_at": now,
			},
		}
		arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"suite.id": suiteID},
			},
		})

		result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("append test to nested suite: %w", err)
		}

		if result.MatchedCount > 0 {
			r.logger.Info("test begin (nested, inserted)", "id", test.ID, "title", test.Title, "suite", suiteID)
			return nil
		}

		// Suite not found anywhere - this is an error
		// Root document exists but test should belong to a suite, not directly to root
		return fmt.Errorf("test %s belongs to run/root %s but no suite structure exists - suite.begin event may be missing", test.ID, suiteID)
	}

	// Root document doesn't exist either - this is a more severe error
	return fmt.Errorf("suite and run not found: %s (test: %s)", suiteID, test.ID)
}

// UpsertTestEnd handles test end events by finding the test within the root document
// structure and updating its attributes (status, duration).
// Searches both root-level tests and nested suite tests.
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

// UpsertStepBegin handles step begin events by upserting to the parent test
// (update if exists, insert if not)
func (r *MongoRepository) UpsertStepBegin(ctx context.Context, step *m.StepDocument, testID string, parentStepID string) error {
	now := time.Now()
	step.CreatedAt = now
	step.UpdatedAt = now
	step.TestCaseRunID = testID
	step.ParentStepID = parentStepID
	if step.Steps == nil {
		step.Steps = []*m.StepDocument{}
	}

	if parentStepID == "" {
		// Direct child of test - check if step already exists, then update or append
		// First try to update existing step in root-level tests
		filter := bson.M{
			"tests.id":       testID,
			"tests.steps.id": step.ID,
		}
		update := bson.M{
			"$set": bson.M{
				"tests.$[test].steps.$[step].status":     step.Status,
				"tests.$[test].steps.$[step].category":   step.Category,
				"tests.$[test].steps.$[step].title":      step.Title,
				"tests.$[test].steps.$[step].updated_at": now,
				"updated_at": now,
			},
		}
		arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"test.id": testID},
				bson.M{"step.id": step.ID},
			},
		})

		result, err := r.collection.UpdateMany(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("update existing step in test: %w", err)
		}

		if result.MatchedCount > 0 {
			r.logger.Info("step begin (updated)", "id", step.ID, "test", testID)
			return nil
		}

		// Step doesn't exist, append it to test
		filter = bson.M{"tests.id": testID}
		update = bson.M{
			"$push": bson.M{
				"tests.$[test].steps": step,
			},
			"$set": bson.M{
				"updated_at": now,
			},
		}
		arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"test.id": testID},
			},
		})

		result, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("append step to test: %w", err)
		}

		if result.MatchedCount == 0 {
			// Try in nested suites - first check if step exists
			filter = bson.M{
				"suites.tests.id":       testID,
				"suites.tests.steps.id": step.ID,
			}
			update = bson.M{
				"$set": bson.M{
					"suites.$[].tests.$[test].steps.$[step].status":     step.Status,
					"suites.$[].tests.$[test].steps.$[step].category":   step.Category,
					"suites.$[].tests.$[test].steps.$[step].title":      step.Title,
					"suites.$[].tests.$[test].steps.$[step].updated_at": now,
					"updated_at": now,
				},
			}
			arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
				Filters: []interface{}{
					bson.M{"test.id": testID},
					bson.M{"step.id": step.ID},
				},
			})

			result, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
			if err != nil {
				return fmt.Errorf("update step in nested test: %w", err)
			}

			if result.MatchedCount > 0 {
				r.logger.Info("step begin (nested, updated)", "id", step.ID, "test", testID)
				return nil
			}

			// Step doesn't exist in nested suite, append it
			filter = bson.M{"suites.tests.id": testID}
			update = bson.M{
				"$push": bson.M{
					"suites.$[].tests.$[test].steps": step,
				},
				"$set": bson.M{
					"updated_at": now,
				},
			}
			arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
				Filters: []interface{}{
					bson.M{"test.id": testID},
				},
			})

			_, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
			if err != nil {
				return fmt.Errorf("append step to nested test: %w", err)
			}
			r.logger.Info("step begin (nested, inserted)", "id", step.ID, "test", testID)
			return nil
		}

		r.logger.Info("step begin (inserted)", "id", step.ID, "test", testID)
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
	// First check if it exists
	filter := bson.M{
		"tests.id":       testID,
		"tests.steps.id": step.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"tests.$[test].steps.$[step].status":        step.Status,
			"tests.$[test].steps.$[step].category":      step.Category,
			"tests.$[test].steps.$[step].title":         step.Title,
			"tests.$[test].steps.$[step].parent_step_id": step.ParentStepID,
			"tests.$[test].steps.$[step].updated_at":    now,
			"updated_at": now,
		},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"step.id": step.ID},
		},
	})

	result, err := r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested step in test: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("step begin (nested, updated)", "id", step.ID, "parent", parentStepID)
		return nil
	}

	// Doesn't exist, append it
	filter = bson.M{"tests.id": testID}
	update = bson.M{
		"$push": bson.M{
			"tests.$[test].steps": step,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}
	arrayFilters = options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
		},
	})

	_, err = r.collection.UpdateMany(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("append nested step to test: %w", err)
	}

	r.logger.Info("step begin (nested, inserted)", "id", step.ID, "parent", parentStepID)
	return nil
}

// UpsertStepEnd handles step end events by finding the step within the root document
// structure and updating its status attribute.
// Searches both root-level test steps and nested suite test steps.
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
