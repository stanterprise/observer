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
func (r *MongoRepository) UpsertStepBegin(ctx context.Context, step *m.StepDocument, testID string, runID string, parentStepID string) error {
	now := time.Now()
	step.CreatedAt = now
	step.UpdatedAt = now
	step.TestCaseRunID = testID
	step.ParentStepID = parentStepID
	if step.Steps == nil {
		step.Steps = []*m.StepDocument{}
	}

	// Extract root document ID to ensure we only update steps in the correct test run
	// If runID is empty, we'll search without the root document ID filter
	rootDocID := ""
	if runID != "" {
		rootDocID = extractRootSuiteID(runID)
	}

	if parentStepID == "" {
		// Direct child of test - check if step already exists, then update or append
		// First try to update existing step in root-level tests
		filter := bson.M{
			"tests.id":       testID,
			"tests.steps.id": step.ID,
		}
		if rootDocID != "" {
			filter["_id"] = rootDocID // Add root document ID filter if available
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
		filter = bson.M{
			"tests.id": testID,
		}
		if rootDocID != "" {
			filter["_id"] = rootDocID // Add root document ID filter if available
		}
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

		if result.MatchedCount == 0 && rootDocID != "" {
			// Try in nested suites - only if we have a valid rootDocID
			// First check if step exists
			filter = bson.M{
				"_id":                   rootDocID, // CRITICAL: Prevent cross-document mutation
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

			// Step doesn't exist in nested suite, append it using aggregation pipeline
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
																	bson.M{
																		"steps": bson.M{
																			"$concatArrays": bson.A{
																				bson.M{"$ifNull": bson.A{"$$test.steps", bson.A{}}},
																				bson.A{step},
																			},
																		},
																	},
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
				"_id": rootDocID, // CRITICAL (conditionally added): Prevent cross-document mutation
				"suites.tests.id": testID,
			}
			_, err = r.collection.UpdateMany(ctx, filter, pipeline)
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
	if rootDocID != "" {
		filter["_id"] = rootDocID // Add root document ID filter if available
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
	filter = bson.M{
		"tests.id": testID,
	}
	if rootDocID != "" {
		filter["_id"] = rootDocID // Add root document ID filter if available
	}
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
func (r *MongoRepository) UpsertStepEnd(ctx context.Context, stepID string, runID string, status string) error {
	now := time.Now()

	// Extract root document ID to ensure we only update steps in the correct test run
	// If runID is empty, we'll search without the root document ID filter
	rootDocID := ""
	if runID != "" {
		rootDocID = extractRootSuiteID(runID)
	}

	updateFields := bson.M{
		"tests.$[].steps.$[step].updated_at": now,
	}
	if status != "" {
		updateFields["tests.$[].steps.$[step].status"] = status
	}

	filter := bson.M{
		"tests.steps.id": stepID,
	}
	if rootDocID != "" {
		filter["_id"] = rootDocID // Add root document ID filter if available
	}
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

	// Try in nested suites using aggregation pipeline (only if we have a valid rootDocID)
	if rootDocID != "" {
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
												"$mergeObjects": bson.A{
													"$$test",
													bson.M{
														"steps": bson.M{
															"$map": bson.M{
																"input": bson.M{
																	"$ifNull": bson.A{"$$test.steps", bson.A{}},
																},
																"as": "step",
																"in": bson.M{
																	"$cond": bson.A{
																		bson.M{"$eq": bson.A{"$$step.id", stepID}},
																		bson.M{
																			"$mergeObjects": bson.A{
																				"$$step",
																				buildStepEndUpdate(status, now),
																			},
																		},
																		"$$step",
																	},
																},
															},
														},
																	},
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
			"_id":                   rootDocID, // CRITICAL: Prevent cross-document mutation
			"suites.tests.steps.id": stepID,
		}
		_, err = r.collection.UpdateMany(ctx, filter, pipeline)
		if err != nil {
			return fmt.Errorf("update nested step end: %w", err)
		}

		r.logger.Info("step end (nested)", "id", stepID, "status", status)
	}

	return nil
}
