package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *MongoRepository) MarkRunningTestsAsTimedOut(ctx context.Context, runID string) error {
	if err := ValidateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{
		"_id": runID,
	}
	update := bson.M{
		"$set": bson.M{
			"tests.$[test].attempts.$[attempt].status":     "TIMEDOUT",
			"tests.$[test].attempts.$[attempt].end_time":   now,
			"tests.$[test].attempts.$[attempt].updated_at": now,
		},
	}
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.attempts": bson.M{"$elemMatch": bson.M{"status": "RUNNING"}}},
			bson.M{"attempt.status": "RUNNING"},
		},
	}
	updateOptions := options.Update().SetArrayFilters(arrayFilters)

	result, err := r.collection.UpdateMany(ctx, filter, update, updateOptions)
	if err != nil {
		return fmt.Errorf("mark running tests as timed out: %w", err)
	}

	r.logger.Info("marked running tests as timed out", "run_id", runID, "modified_count", result.ModifiedCount)
	return nil
}
