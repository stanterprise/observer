package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

func (r *MongoRepository) MapSuites(ctx context.Context, runID string, name string, metadata map[string]interface{}, totalTests int32, suites []m.SuiteDocument) error {
	if err := ValidateRunID(runID); err != nil {
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

	// Update run-level fields (name, metadata, total_tests)
	// For sharded test execution: accumulate total_tests, merge metadata at key level
	runUpdate := bson.M{
		"$set": bson.M{
			"updated_at": now,
		},
		"$inc": bson.M{},
	}

	// Name: use first-write wins via $setOnInsert (handled in ensureDocumentExists)
	// For subsequent calls, only update if name is different
	if name != "" {
		runUpdate["$set"].(bson.M)["name"] = name
	}

	// Metadata: merge at key level to preserve metadata from all shards
	if len(metadata) > 0 {
		for k, v := range metadata {
			runUpdate["$set"].(bson.M)[fmt.Sprintf("metadata.%s", k)] = v
		}
	}

	// Total tests: accumulate across shards using $inc
	if totalTests > 0 {
		runUpdate["$inc"].(bson.M)["total_tests"] = totalTests
		// Track number of shards that reported
		runUpdate["$inc"].(bson.M)["shard_count"] = 1
	}

	// Remove empty $inc if not used
	if len(runUpdate["$inc"].(bson.M)) == 0 {
		delete(runUpdate, "$inc")
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
