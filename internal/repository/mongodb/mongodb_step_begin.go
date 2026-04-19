package mongodb

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
)

// UpsertStepBegin appends a step to the active run-scoped step buffer keyed by test id.
// - runID: Required. Identifies the document (_id).
// - step: The step to create/update (step.ID identifies the step).
// - testID: Required. ID of parent test containing this step.
// - retry_index: Required. Retry attempt index for the active buffer.
// Returns error if runID is empty or the active test buffer has not been initialized.
func (r *MongoRepository) UpsertStepBegin(ctx context.Context, runID string, step *m.StepDocument, testID string, retry_index int32) error {
	if err := repository.ValidateRunID(runID); err != nil {
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

	field := stepBufferField(testID)
	filter := bson.M{
		"_id":                  runID,
		field + ".retry_index": retry_index,
	}
	update := bson.M{
		"$push": bson.M{field + ".steps": step},
		"$set": bson.M{
			field + ".status":        activeStepBufferStatusActive,
			field + ".last_event_at": now,
			field + ".updated_at":    now,
			"updated_at":             now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("insert step into active test buffer: %w", err)
	}

	if result.MatchedCount == 0 {
		r.logger.Warn("active test buffer missing for step begin",
			"runID", runID,
			"testID", testID,
			"retryIndex", retry_index,
			"stepID", step.ID)
		return fmt.Errorf("parent test not found: runID=%s, testID=%s, retryIndex=%d", runID, testID, retry_index)
	}

	r.logger.Info("step begin buffered",
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
