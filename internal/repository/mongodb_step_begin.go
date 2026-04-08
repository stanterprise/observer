package repository

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertStepBegin creates or updates a step within the document identified by runID.
// With attempt-based retries: steps are stored in attempts[retry_index].steps instead of tests.steps.
// - runID: Required. Identifies the document (_id).
// - step: The step to create/update (step.ID identifies the step).
// - testID: Required. ID of parent test containing this step.
// - retry_index: Required. Retry attempt index to target for step storage.
// Returns error if runID is empty or parent test not found.
func (r *MongoRepository) UpsertStepBegin(ctx context.Context, runID string, step *m.StepDocument, testID string, retry_index int32) error {
	if err := ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	step.CreatedAt = now
	step.UpdatedAt = now
	step.TestCaseRunID = testID
	step.RunID = runID

	if step.Steps == nil {
		step.Steps = []*m.StepDocument{}
	}

	r.logger.Debug("UpsertStepBegin starting",
		"runID", runID,
		"stepID", step.ID,
		"testID", testID,
		"retryIndex", retry_index,
		"stepTitle", step.Title)

	return r.upsertStepInTestAttempt(ctx, runID, testID, retry_index, step, now)
}

// upsertStepInTestAttempt handles steps as children of attempts[retry_index] array.
// With attempt-based retries: steps are stored in attempts[retry_index].steps instead of tests.steps.
// Note: "step begin" events should ONLY insert new steps, never update existing ones.
func (r *MongoRepository) upsertStepInTestAttempt(ctx context.Context, runID string, testID string, retry_index int32, step *m.StepDocument, now time.Time) error {
	// Use a tight filter that includes the specific attempt so MatchedCount==0 reliably means
	// "the attempt does not exist yet", rather than relying on ModifiedCount==0 which is
	// unreliable when the update also contains a $set that always modifies the document.
	filter := bson.M{
		"_id": runID,
		"tests": bson.M{
			"$elemMatch": bson.M{
				"id": testID,
				"attempts": bson.M{
					"$elemMatch": bson.M{"retry_index": retry_index},
				},
			},
		},
	}
	update := bson.M{
		"$push": bson.M{"tests.$[test].attempts.$[attempt].steps": step},
		"$set":  bson.M{"updated_at": now},
	}
	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"test.id": testID},
			bson.M{"attempt.retry_index": retry_index},
		},
	})

	r.logger.Debug("Inserting new step into attempt",
		"runID", runID,
		"stepID", step.ID,
		"stepTitle", step.Title,
		"testID", testID,
		"retryIndex", retry_index)

	result, err := r.collection.UpdateOne(ctx, filter, update, arrayFilters)
	if err != nil {
		return fmt.Errorf("insert step into test attempt: %w", err)
	}

	if result.MatchedCount == 0 {
		r.logger.Warn("attempt missing for step begin, creating attempt from step context",
			"runID", runID,
			"testID", testID,
			"retryIndex", retry_index,
			"stepID", step.ID)

		if err := r.ensureAttemptExists(ctx, runID, testID, retry_index, step.StartTime, now); err != nil {
			return fmt.Errorf("ensure attempt exists for step begin: %w", err)
		}

		result, err = r.collection.UpdateOne(ctx, filter, update, arrayFilters)
		if err != nil {
			return fmt.Errorf("retry insert step into test attempt: %w", err)
		}
		if result.MatchedCount == 0 {
			r.logger.Error("parent test not found after ensuring attempt",
				"runID", runID,
				"testID", testID,
				"retryIndex", retry_index,
				"stepID", step.ID)
			return fmt.Errorf("parent test not found: runID=%s, testID=%s, retryIndex=%d", runID, testID, retry_index)
		}
	}

	r.logger.Info("step begin (inserted)",
		"runID", runID,
		"stepID", step.ID,
		"testID", testID,
		"retryIndex", retry_index,
		"matchedCount", result.MatchedCount,
		"modifiedCount", result.ModifiedCount)
	return nil
}

func (r *MongoRepository) ensureAttemptExists(ctx context.Context, runID, testID string, retryIndex int32, startTime *time.Time, now time.Time) error {
	attempt := &m.AttemptDocument{
		RetryIndex: retryIndex,
		Steps:      []*m.StepDocument{},
		Status:     "RUNNING",
		StartTime:  startTime,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Atomic pipeline update: conditionally append attempt only if retry_index is not already present.
	// This avoids the read-then-write race that could create duplicate attempts under concurrent access.
	filter := bson.M{
		"_id":      runID,
		"tests.id": testID,
	}

	existingRetryIndexesExpr := bson.M{
		"$map": bson.M{
			"input": bson.M{"$ifNull": bson.A{"$$test.attempts", bson.A{}}},
			"as":    "attempt",
			"in":    "$$attempt.retry_index",
		},
	}

	attemptsExpr := bson.M{
		"$cond": bson.A{
			bson.M{"$in": bson.A{retryIndex, existingRetryIndexesExpr}},
			bson.M{"$ifNull": bson.A{"$$test.attempts", bson.A{}}},
			bson.M{
				"$concatArrays": bson.A{
					bson.M{"$ifNull": bson.A{"$$test.attempts", bson.A{}}},
					bson.A{attempt},
				},
			},
		},
	}

	update := bson.A{
		bson.D{{
			Key: "$set",
			Value: bson.M{
				"updated_at": now,
				"tests": bson.M{
					"$map": bson.M{
						"input": "$tests",
						"as":    "test",
						"in": bson.M{
							"$cond": bson.A{
								bson.M{"$eq": bson.A{"$$test.id", testID}},
								bson.M{
									"$mergeObjects": bson.A{
										"$$test",
										bson.M{
											"retry_index": retryIndex,
											"updated_at":  now,
											"attempts":    attemptsExpr,
										},
									},
								},
								"$$test",
							},
						},
					},
				},
			},
		}},
	}

	if _, err := r.collection.UpdateOne(ctx, filter, update); err != nil {
		return err
	}

	return nil
}
