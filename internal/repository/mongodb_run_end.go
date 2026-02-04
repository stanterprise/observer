package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpdateTestRunEnd updates a test run document with final status, times, and duration
func (r *MongoRepository) UpdateTestRunEnd(ctx context.Context, runID string, status string, startTime *time.Time, duration *int64) error {
	if err := ValidateRunID(runID); err != nil {
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
		"$inc": bson.M{
			"shards.finished": 1,
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

func (r *MongoRepository) MarkRunningTestsAsTimedOut(ctx context.Context, runID string) error {
	if err := ValidateRunID(runID); err != nil {
		return err
	}

	if !checkShardCompletion(ctx, r, runID) {
		return nil
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

func checkShardCompletion(ctx context.Context, r *MongoRepository, runID string) bool {
	var shardState struct {
		Shards struct {
			Finished *int64 `bson:"finished"`
		} `bson:"shards"`
		Metadata map[string]interface{} `bson:"metadata"`
	}

	projection := options.FindOne().SetProjection(bson.M{
		"shards.finished": 1,
		"metadata":        1,
	})
	if err := r.collection.FindOne(ctx, bson.M{"_id": runID}, projection).Decode(&shardState); err != nil {
		if err == mongo.ErrNoDocuments {
			return false
		}
		return false
	}

	finished := int64(0)
	if shardState.Shards.Finished != nil {
		finished = *shardState.Shards.Finished
	}

	totalRaw, ok := shardState.Metadata["shard.total"]
	if !ok {
		return false
	}

	var total int64
	switch v := totalRaw.(type) {
	case int64:
		total = v
	case int32:
		total = int64(v)
	case float64:
		total = int64(v)
	case float32:
		total = int64(v)
	default:
		return false
	}

	if finished != total {
		return false
	}

	return true
}
