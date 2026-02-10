package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func (r *MongoRepository) MarkRunStarts(ctx context.Context, runID string) error {
	if err := ValidateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{"_id": runID}
	update := bson.M{
		"$set": bson.M{
			"status":     "RUNNING",
			"updated_at": now,
			"started_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("mark run starts: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("test run not found: %s", runID)
	}

	r.logger.Info("test run started", "run_id", runID)
	return nil
}
