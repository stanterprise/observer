package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
		// Initialize suite arrays to ensure they're not nil
		if suite.Tests == nil {
			suite.Tests = []*m.TestDocument{}
		}
		if suite.Suites == nil {
			suite.Suites = []*m.SuiteDocument{}
		}

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
			// Document existed - this is a duplicate/replay event for the same run ID
			// We should update metadata but preserve existing tests and suites
			// Note: If this is truly a NEW run, it should have a different suite.ID
			filter = bson.M{
				"_id":        suite.ID,
				"suites.id": suite.ID,
			}
			update = bson.M{
				"$set": bson.M{
					"suites.$.name":               suite.Name,
					"suites.$.description":        suite.Description,
					"suites.$.metadata":           suite.Metadata,
					"suites.$.test_suite_spec_id": suite.TestSuiteSpecID,
					"suites.$.initiated_by":       suite.InitiatedBy,
					"suites.$.project_name":       suite.ProjectName,
					"suites.$.start_time":         suite.StartTime,
					"suites.$.updated_at":         now,
					// Note: We intentionally do NOT update status here to avoid overwriting
					// the final status set by SuiteEnd event
					// Note: We do NOT set tests or suites arrays to preserve existing data
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

	// Initialize suite arrays to ensure they're not nil
	if suite.Tests == nil {
		suite.Tests = []*m.TestDocument{}
	}
	if suite.Suites == nil {
		suite.Suites = []*m.SuiteDocument{}
	}

	// First, try to update existing suite if it already exists in root document's suites array
	// This handles duplicate/replay events for the same suite
	filter := bson.M{
		"_id":        parentSuiteID,
		"suites.id": suite.ID,
	}
	update := bson.M{
		"$set": bson.M{
			"suites.$.name":               suite.Name,
			"suites.$.description":        suite.Description,
			"suites.$.metadata":           suite.Metadata,
			"suites.$.test_suite_spec_id": suite.TestSuiteSpecID,
			"suites.$.initiated_by":       suite.InitiatedBy,
			"suites.$.project_name":       suite.ProjectName,
			"suites.$.start_time":         suite.StartTime,
			"suites.$.updated_at":         now,
			// Note: We intentionally do NOT update status here to avoid overwriting
			// the final status set by SuiteEnd event
			// Note: We do NOT set tests or suites arrays to preserve existing data
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update existing suite in parent: %w", err)
	}

	if result.MatchedCount > 0 {
		// Suite was found and updated (preserving tests and nested suites)
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
