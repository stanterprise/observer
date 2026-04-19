package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	db "github.com/stanterprise/observer/internal/database"
	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	activeStepBufferStatusActive          = "active"
	activeStepBufferStatusFlushInProgress = "flush_in_progress"
)

func stepBufferKey(testID string) string {
	replacer := strings.NewReplacer(".", "%2E", "$", "%24")
	return replacer.Replace(testID)
}

func stepBufferField(testID string) string {
	return "active_test_steps." + stepBufferKey(testID)
}

func (r *MongoRepository) SyncActiveTestSteps(ctx context.Context, runID, testID string, retryIndex int32, startTime *time.Time) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	eventTime := startTime
	if eventTime == nil {
		eventTime = &now
	}
	ttlAt := now.Add(db.MongoStepBufferTTL(r.logger))

	buffer := &m.ActiveTestStepsDocument{
		TestID:       testID,
		RetryIndex:   retryIndex,
		Status:       activeStepBufferStatusActive,
		Steps:        []*m.StepDocument{},
		FirstEventAt: eventTime,
		LastEventAt:  eventTime,
		TTLAt:        &ttlAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	field := stepBufferField(testID)
	replaceFilter := bson.M{
		"_id":      runID,
		"tests.id": testID,
		"$or": bson.A{
			bson.M{field: bson.M{"$exists": false}},
			bson.M{field + ".retry_index": bson.M{"$ne": retryIndex}},
		},
	}
	replaceUpdate := bson.M{
		"$set": bson.M{
			field:        buffer,
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, replaceFilter, replaceUpdate)
	if err != nil {
		return fmt.Errorf("sync active test steps: %w", err)
	}
	if result.MatchedCount > 0 {
		return nil
	}

	touchFilter := bson.M{
		"_id":                  runID,
		"tests.id":             testID,
		field + ".retry_index": retryIndex,
	}
	touchUpdate := bson.M{
		"$set": bson.M{
			field + ".status":        activeStepBufferStatusActive,
			field + ".last_event_at": eventTime,
			field + ".ttl_at":        ttlAt,
			field + ".updated_at":    now,
			"updated_at":             now,
		},
		"$unset": bson.M{
			field + ".flush_started_at": "",
		},
	}

	result, err = r.collection.UpdateOne(ctx, touchFilter, touchUpdate)
	if err != nil {
		return fmt.Errorf("touch active test steps: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("parent test not found: runID=%s, testID=%s", runID, testID)
	}

	return nil
}

func (r *MongoRepository) PrepareActiveTestStepsFlush(ctx context.Context, runID, testID string, retryIndex int32) ([]*m.StepDocument, bool, error) {
	if err := repository.ValidateRunID(runID); err != nil {
		return nil, false, err
	}
	if testID == "" {
		return nil, false, fmt.Errorf("testID is required")
	}

	now := time.Now()
	field := stepBufferField(testID)
	filter := bson.M{
		"_id":                  runID,
		field + ".retry_index": retryIndex,
	}
	update := bson.M{
		"$set": bson.M{
			field + ".status":           activeStepBufferStatusFlushInProgress,
			field + ".flush_started_at": now,
			field + ".updated_at":       now,
			"updated_at":                now,
		},
	}
	projection := bson.M{field: 1}
	var doc bson.M
	err := r.collection.FindOneAndUpdate(
		ctx,
		filter,
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(projection),
	).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("prepare active test steps flush: %w", err)
	}

	buffer, found, err := decodeActiveTestStepsBuffer(doc, testID)
	if err != nil {
		return nil, false, err
	}
	if !found || buffer == nil {
		return nil, false, nil
	}

	return cloneStepDocuments(buffer.Steps), true, nil
}

func (r *MongoRepository) ResetActiveTestStepsFlushState(ctx context.Context, runID, testID string, retryIndex int32) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	field := stepBufferField(testID)
	_, err := r.collection.UpdateOne(ctx, bson.M{
		"_id":                  runID,
		field + ".retry_index": retryIndex,
	}, bson.M{
		"$set": bson.M{
			field + ".status":     activeStepBufferStatusActive,
			field + ".updated_at": now,
			"updated_at":          now,
		},
		"$unset": bson.M{
			field + ".flush_started_at": "",
		},
	})
	if err != nil {
		return fmt.Errorf("reset active test steps flush state: %w", err)
	}
	return nil
}

func (r *MongoRepository) DeleteActiveTestSteps(ctx context.Context, runID, testID string, retryIndex int32) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	field := stepBufferField(testID)
	_, err := r.collection.UpdateOne(ctx, bson.M{
		"_id":                  runID,
		field + ".retry_index": retryIndex,
	}, bson.M{
		"$unset": bson.M{field: ""},
		"$set":   bson.M{"updated_at": now},
	})
	if err != nil {
		return fmt.Errorf("delete active test steps: %w", err)
	}
	return nil
}

func decodeActiveTestStepsBuffer(doc bson.M, testID string) (*m.ActiveTestStepsDocument, bool, error) {
	rawActive, ok := doc["active_test_steps"]
	if !ok {
		return nil, false, nil
	}

	active, ok := rawActive.(bson.M)
	if !ok {
		return nil, false, fmt.Errorf("decode active test steps: invalid active_test_steps shape")
	}

	rawBuffer, ok := active[stepBufferKey(testID)]
	if !ok {
		return nil, false, nil
	}

	bsonBytes, err := bson.Marshal(rawBuffer)
	if err != nil {
		return nil, false, fmt.Errorf("marshal active test steps buffer: %w", err)
	}

	var buffer m.ActiveTestStepsDocument
	if err := bson.Unmarshal(bsonBytes, &buffer); err != nil {
		return nil, false, fmt.Errorf("unmarshal active test steps buffer: %w", err)
	}

	return &buffer, true, nil
}

func cloneStepDocuments(input []*m.StepDocument) []*m.StepDocument {
	if len(input) == 0 {
		return []*m.StepDocument{}
	}
	output := make([]*m.StepDocument, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		copied := *item
		if item.Metadata != nil {
			copied.Metadata = make(map[string]interface{}, len(item.Metadata))
			for key, value := range item.Metadata {
				copied.Metadata[key] = value
			}
		}
		copied.Tags = append([]string(nil), item.Tags...)
		copied.Errors = append([]string(nil), item.Errors...)
		copied.Steps = cloneStepDocuments(item.Steps)
		output = append(output, &copied)
	}
	return output
}
