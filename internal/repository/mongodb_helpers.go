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
