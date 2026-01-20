package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertSuiteEnd updates suite end fields (status, endTime, duration).
// - runID: Required. Identifies the document (_id).
// - suiteID: Required. Identifies the suite to update.
// Returns error if runID is empty or suite not found.
func (r *MongoRepository) UpsertSuiteEnd(ctx context.Context, runID string, suiteID string, status string, endTime *time.Time, duration *int64) error {
	if err := ValidateRunID(runID); err != nil {
		return err
	}
	if suiteID == "" {
		return fmt.Errorf("suiteID is required")
	}

	now := time.Now()
	updateFields := bson.M{"updated_at": now}
	if status != "" {
		updateFields["status"] = status
	}
	if endTime != nil {
		updateFields["end_time"] = endTime
	}
	if duration != nil {
		updateFields["duration"] = duration
	}

	// Try root-level suite
	filter := bson.M{
		"_id":       runID,
		"suites.id": suiteID,
	}
	setFields := bson.M{}
	for k, v := range updateFields {
		setFields["suites.$."+k] = v
	}
	setFields["updated_at"] = now

	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields})
	if err != nil {
		return fmt.Errorf("update suite end: %w", err)
	}

	if result.MatchedCount > 0 {
		r.logger.Info("suite end (root)", "runID", runID, "suiteID", suiteID, "status", status)
		return nil
	}

	// Try nested suite
	filter = bson.M{
		"_id":              runID,
		"suites.suites.id": suiteID,
	}
	setFields = bson.M{"updated_at": now}
	for k, v := range updateFields {
		setFields["suites.$[].suites.$[suite]."+k] = v
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"suite.id": suiteID},
		},
	})

	result, err = r.collection.UpdateOne(ctx, filter, bson.M{"$set": setFields}, arrayFilters)
	if err != nil {
		return fmt.Errorf("update nested suite end: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("suite not found: runID=%s, suiteID=%s", runID, suiteID)
	}

	r.logger.Info("suite end (nested)", "runID", runID, "suiteID", suiteID, "status", status)
	return nil
}
