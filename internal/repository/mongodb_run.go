package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *MongoRepository) MarkRunningTestsAsTimedOut(ctx context.Context, runID string) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{
		"_id": runID,
	}
	update := bson.M{
		"$set": bson.M{
			"tests.$[test].status":     "TIMEDOUT",
			"tests.$[test].end_time":   now,
			"tests.$[test].updated_at": now,
		},
	}
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.status": "RUNNING"},
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

// UpdateTestRunEnd updates a test run document with final status, times, and duration
func (r *MongoRepository) UpdateTestRunEnd(ctx context.Context, runID string, status string, startTime *time.Time, duration *int64) error {
	if err := validateRunID(runID); err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{"_id": runID}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": now,
			"ended_at":   now,
		},
	}

	// Add start_time if provided
	if startTime != nil {
		update["$set"].(bson.M)["start_time"] = startTime
	}

	// Add duration if provided
	if duration != nil {
		update["$set"].(bson.M)["duration"] = duration
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
