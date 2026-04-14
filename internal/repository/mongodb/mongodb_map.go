package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
)

// MapSuites adds or updates test suites for a test run.
// For sharded test runs (detected by shard.total and shard.current in metadata):
// - Metadata is merged to preserve shard info from multiple shards
// - total_tests is accumulated across shards using $inc
// For non-sharded runs:
// - Metadata is replaced entirely
// - total_tests is set directly
func (r *MongoRepository) MapSuites(ctx context.Context, runID string, name string, metadata map[string]interface{}, totalTests int32, suites []m.SuiteDocument) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	var errs []error
	now := time.Now()

	filter := bson.M{
		"_id": runID,
	}

	// Ensure document exists
	if err := r.ensureDocumentExists(ctx, runID); err != nil {
		errs = append(errs, fmt.Errorf("ensure document exists: %w", err))
	}

	// Detect if this is a sharded run
	isSharded := false
	if metadata != nil {
		_, hasTotal := metadata["shard.total"]
		_, hasCurrent := metadata["shard.current"]
		isSharded = hasTotal && hasCurrent
		if isSharded {
			r.logger.Info("detected sharded run",
				"run_id", runID,
				"shard.total", metadata["shard.total"],
				"shard.current", metadata["shard.current"],
				"shard_total_tests", totalTests)
		}
	}

	// Update run-level fields (name, metadata, total_tests)
	runUpdate := bson.M{
		"$set": bson.M{
			"updated_at": now,
		},
	}
	if name != "" {
		runUpdate["$set"].(bson.M)["name"] = name
	}
	if len(metadata) > 0 {
		if isSharded {
			// For sharded runs, merge metadata to preserve shard info from multiple shards
			// Since metadata keys contain periods (e.g., "shard.total"), we can't use dot notation
			// Instead, we read existing metadata, merge, and update the entire object
			var existingDoc bson.M
			err := r.collection.FindOne(ctx, filter).Decode(&existingDoc)
			if err == nil {
				if existingMeta, ok := existingDoc["metadata"].(bson.M); ok {
					// Merge existing metadata with new metadata
					for k, v := range existingMeta {
						if _, exists := metadata[k]; !exists {
							metadata[k] = v
						}
					}
				}
			}
			runUpdate["$set"].(bson.M)["metadata"] = metadata
		} else {
			runUpdate["$set"].(bson.M)["metadata"] = metadata
		}
	}
	if totalTests > 0 {
		if isSharded {
			// For sharded runs, accumulate total_tests across shards
			if runUpdate["$inc"] == nil {
				runUpdate["$inc"] = bson.M{}
			}
			runUpdate["$inc"].(bson.M)["total_tests"] = totalTests
		} else {
			// For non-sharded runs, set total_tests directly
			runUpdate["$set"].(bson.M)["total_tests"] = totalTests
		}
	}

	_, err := r.collection.UpdateOne(ctx, filter, runUpdate)
	if err != nil {
		errs = append(errs, fmt.Errorf("update run metadata: %w", err))
	}

	for _, suite := range suites {

		suite.CreatedAt = now
		suite.UpdatedAt = now

		// Initialize child arrays
		if suite.Tests == nil {
			suite.Tests = []*m.TestDocument{}
		}
		if suite.Suites != nil {
			for _, childSuite := range suite.Suites {
				childSuite.CreatedAt = now
				childSuite.UpdatedAt = now
				childSuite.ParentSuiteID = suite.ID
			}
		}

	}

	// Suite doesn't exist, append it
	filter = bson.M{"_id": runID}
	update := bson.M{
		"$push": bson.M{"suites": bson.M{"$each": suites}},
		"$set":  bson.M{"updated_at": now},
	}

	_, err = r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		errs = append(errs, fmt.Errorf("append root suite: %w", err))
	}

	return errors.Join(errs...)
}
