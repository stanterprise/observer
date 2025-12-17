package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// (update if exists, insert if not)
func (r *MongoRepository) UpsertTestBegin(ctx context.Context, test *m.TestDocument, suiteID string) error {
	now := time.Now()
	test.CreatedAt = now
	test.UpdatedAt = now
	test.SuiteID = suiteID
	if test.Steps == nil {
		test.Steps = []*m.StepDocument{}
	}

	// Extract root document ID to ensure we only update tests in the correct test run.
	// Note: extractRootSuiteID appends "-suite-root" to the base suite ID. Therefore,
	// suiteID == rootDocID is only true when the incoming suiteID already represents
	// the root suite (i.e., it ends with "-suite-root").
	rootDocID := extractRootSuiteID(suiteID)

	// Check if we're adding to root document's tests array (root suite tests)
	if suiteID == rootDocID {
		// First, try to update existing test in root document
		filter := bson.M{
			"_id":      rootDocID,
			"tests.id": test.ID,
		}
		update := bson.M{
			"$set": bson.M{
				"tests.$[test].title":       test.Title,
				"tests.$[test].status":      test.Status,
				"tests.$[test].metadata":    test.Metadata,
				"tests.$[test].duration":    test.Duration,
				"tests.$[test].retry_count": test.RetryCount,
				"tests.$[test].retry_index": test.RetryIndex,
				"tests.$[test].timeout":     test.Timeout,
				"tests.$[test].updated_at":  now,
				"updated_at": now,
			},
		}
		arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"test.id": test.ID},
			},
		})

		result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("update test in root document: %w", err)
		}

		if result.MatchedCount > 0 {
			r.logger.Info("test begin (root, updated)", "id", test.ID, "title", test.Title)
			return nil
		}

		// Test doesn't exist, append to root document's tests array
		filter = bson.M{"_id": rootDocID}
		update = bson.M{
			"$push": bson.M{
				"tests": test,
			},
			"$set": bson.M{
				"updated_at": now,
			},
		}

		_, err = r.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("append test to root document: %w", err)
		}

		r.logger.Info("test begin (root, inserted)", "id", test.ID, "title", test.Title)
		return nil
	}

	// First, try to find and update the test in a nested suite (level 1 - direct child of root)
	filter := bson.M{
		"_id":             rootDocID, // CRITICAL: Prevent cross-document mutation
		"suites.id":       suiteID,
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

	// Test doesn't exist in nested suite, try to append it at level 1
	filter = bson.M{
		"_id":       rootDocID, // CRITICAL: Prevent cross-document mutation
		"suites.id": suiteID,
	}
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
		// Test was appended to nested suite (level 1)
		r.logger.Info("test begin (nested, inserted)", "id", test.ID, "title", test.Title, "suite", suiteID)
		return nil
	}

	// Try level 2 nested suite (suites.suites.id)
	filter = bson.M{
		"_id":              rootDocID, // CRITICAL: Prevent cross-document mutation
		"suites.suites.id": suiteID,
	}
	update = bson.M{
		"$push": bson.M{
			"suites.$[].suites.$[suite].tests": test,
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
		return fmt.Errorf("append test to level-2 nested suite: %w", err)
	}

	if result.MatchedCount > 0 {
		// Test was appended to level-2 nested suite
		r.logger.Info("test begin (level-2 nested, inserted)", "id", test.ID, "title", test.Title, "suite", suiteID)
		return nil
	}

	// As a fallback, check if the root document exists with this ID
	// (for nested suites where the nested suite doesn't exist yet)
	var doc m.TestRunDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": suiteID}).Decode(&doc)
	if err == nil {
		// First check if test already exists in nested suite
		filter = bson.M{
			"suites.id":       suiteID,
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

		// Suite not found anywhere - create placeholder suite
		// Root document exists but suite doesn't, so create a placeholder
		r.logger.Warn("test arrived before suite, creating placeholder suite in existing run",
			"test_id", test.ID,
			"suite_id", suiteID,
			"root_run_id", suiteID)

		// Create placeholder suite
		placeholderSuite := &m.SuiteDocument{
			ID:        suiteID,
			Name:      "(pending)",
			CreatedAt: now,
			UpdatedAt: now,
			Tests:     []*m.TestDocument{test},
			Suites:    []*m.SuiteDocument{},
			Metadata: map[string]interface{}{
				"placeholder": true,
				"created_by":  "test_begin_event",
			},
		}

		// Add placeholder suite to root document
		filter = bson.M{"_id": suiteID}
		update = bson.M{
			"$push": bson.M{
				"suites": placeholderSuite,
			},
			"$set": bson.M{
				"updated_at": now,
			},
		}

		_, err = r.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("create placeholder suite in root: %w", err)
		}

		r.logger.Info("test begin (placeholder suite created in root)", "id", test.ID, "title", test.Title, "suite", suiteID)
		return nil
	}

	// Root document doesn't exist with suiteID - extract root suite ID and try there
	// rootDocID was already extracted at the beginning of the function
	r.logger.Warn("test arrived before suite, creating placeholder suite",
		"test_id", test.ID,
		"suite_id", suiteID,
		"root_suite_id", rootDocID)

	// Create placeholder suite
	placeholderSuite := &m.SuiteDocument{
		ID:        suiteID,
		Name:      "(pending)",
		CreatedAt: now,
		UpdatedAt: now,
		Tests:     []*m.TestDocument{test},
		Suites:    []*m.SuiteDocument{},
		Metadata: map[string]interface{}{
			"placeholder": true,
			"created_by":  "test_begin_event",
		},
	}

	// Try to add placeholder suite to root document
	filter = bson.M{"_id": rootDocID}
	update = bson.M{
		"$push": bson.M{
			"suites": placeholderSuite,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	result, err = r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("create placeholder suite: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("root suite document not found: %s (for suite: %s, test: %s)", rootDocID, suiteID, test.ID)
	}

	r.logger.Info("test begin (placeholder suite created)", "id", test.ID, "title", test.Title, "suite", suiteID)
	return nil
}

// UpsertTestEnd handles test end events by finding the test within the root document
// structure and updating its attributes (status, duration).
// Searches both root-level tests and nested suite tests.
func (r *MongoRepository) UpsertTestEnd(ctx context.Context, testID string, runID string, status string, duration *int64) error {
	now := time.Now()

	// Extract root document ID to ensure we only update tests in the correct test run
	// If runID is empty, we'll search without the root document ID filter
	rootDocID := ""
	if runID != "" {
		rootDocID = extractRootSuiteID(runID)
	}

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
	filter := bson.M{
		"tests.id": testID,
	}
	if rootDocID != "" {
		filter["_id"] = rootDocID // Add root document ID filter if available
	}
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

	// Try nested suite's tests using aggregation pipeline
	// This approach works even when intermediate arrays don't exist in the document
	pipeline := bson.A{
		bson.M{
			"$set": bson.M{
				"suites": bson.M{
					"$map": bson.M{
						"input": "$suites",
						"as":    "suite",
						"in": bson.M{
							"$mergeObjects": bson.A{
								"$$suite",
								bson.M{
									"tests": bson.M{
										"$map": bson.M{
											"input": bson.M{
												"$ifNull": bson.A{"$$suite.tests", bson.A{}},
											},
											"as": "test",
											"in": bson.M{
												"$cond": bson.A{
													bson.M{"$eq": bson.A{"$$test.id", testID}},
													bson.M{
														"$mergeObjects": bson.A{
															"$$test",
															buildTestEndUpdate(status, duration, now),
														},
													},
													"$$test",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				"updated_at": now,
			},
		},
	}

	filter = bson.M{
		"_id":             rootDocID, // CRITICAL: Prevent cross-document mutation
		"suites.tests.id": testID,
	}
	_, err = r.collection.UpdateMany(ctx, filter, pipeline)
	if err != nil {
		return fmt.Errorf("update nested test end: %w", err)
	}

	r.logger.Info("test end (nested)", "id", testID, "status", status)
	return nil
}

// UpsertStepBegin handles step begin events by upserting to the parent test
