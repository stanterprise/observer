package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// validateRunID checks if runID is provided and returns an error if not
func validateRunID(runID string) error {
	if runID == "" {
		return fmt.Errorf("runID is required")
	}
	return nil
}

// ensureDocumentExists creates a document if it doesn't exist
func (r *MongoRepository) ensureDocumentExists(ctx context.Context, runID string) error {
	now := time.Now()
	filter := bson.M{"_id": runID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":        runID,
			"created_at": now,
			"updated_at": now,
			"suites":     bson.A{},
			"tests":      bson.A{},
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// buildStepEndUpdate creates the update document for step.end events
func buildStepEndUpdate(status string, now time.Time) bson.M {
	update := bson.M{
		"updated_at": now,
	}
	if status != "" {
		update["status"] = status
	}
	return update
}

// UpdateTestRunEnd updates a test run document with final status and metadata
func (r *MongoRepository) UpdateTestRunEnd(ctx context.Context, runID string, status string, metadata map[string]string) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{"_id": runID}

	// Convert metadata map to bson.M
	md := make(map[string]interface{})
	for k, v := range metadata {
		md[k] = v
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": now,
			"ended_at":   now,
		},
	}

	// Add metadata if provided
	if len(md) > 0 {
		update["$set"].(bson.M)["metadata"] = md
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update test run end: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("test run not found: %s", runID)
	}

	r.logger.Info("test run end updated", "run_id", runID, "status", status)
	return nil
}
