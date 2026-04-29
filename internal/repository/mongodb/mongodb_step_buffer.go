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

func liveStepBufferID(runID, executionID, testID string) string {
	if executionID == "" {
		return runID + ":" + stepBufferKey(testID)
	}
	return runID + ":execution:" + executionID + ":" + stepBufferKey(testID)
}

func newLiveStepBuffer(runID, executionID, testID string, retryIndex int32, eventTime, now time.Time, ttlAt time.Time) *m.LiveStepBufferDocument {
	return &m.LiveStepBufferDocument{
		ID:           liveStepBufferID(runID, executionID, testID),
		RunID:        runID,
		ExecutionID:  executionID,
		TestID:       testID,
		AttemptIndex: retryIndex,
		Status:       activeStepBufferStatusActive,
		Steps:        []*m.StepDocument{},
		FirstEventAt: &eventTime,
		LastEventAt:  &eventTime,
		TTLAt:        &ttlAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (r *MongoRepository) SyncActiveTestSteps(ctx context.Context, runID, executionID, testID string, retryIndex int32, startTime *time.Time) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	eventTime := now
	if startTime != nil {
		eventTime = *startTime
	}
	ttlAt := now.Add(db.MongoStepBufferTTL(r.logger))
	bufferID := liveStepBufferID(runID, executionID, testID)

	var existing m.LiveStepBufferDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": bufferID}).Decode(&existing)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, insertErr := r.collection.InsertOne(ctx, newLiveStepBuffer(runID, executionID, testID, retryIndex, eventTime, now, ttlAt))
			if insertErr != nil {
				return fmt.Errorf("create active test steps buffer: %w", insertErr)
			}
			return nil
		}
		return fmt.Errorf("load active test steps buffer: %w", err)
	}

	if existing.AttemptIndex != retryIndex {
		replacement := newLiveStepBuffer(runID, executionID, testID, retryIndex, eventTime, now, ttlAt)
		replacement.CreatedAt = existing.CreatedAt
		if replacement.CreatedAt.IsZero() {
			replacement.CreatedAt = now
		}
		if _, err := r.collection.ReplaceOne(ctx, bson.M{"_id": bufferID}, replacement); err != nil {
			return fmt.Errorf("replace active test steps buffer: %w", err)
		}
		return nil
	}

	existing.Status = activeStepBufferStatusActive
	existing.LastEventAt = &eventTime
	if existing.FirstEventAt == nil {
		existing.FirstEventAt = &eventTime
	}
	existing.TTLAt = &ttlAt
	existing.FlushStartedAt = nil
	existing.UpdatedAt = now

	if _, err := r.collection.ReplaceOne(ctx, bson.M{"_id": bufferID}, &existing); err != nil {
		return fmt.Errorf("touch active test steps buffer: %w", err)
	}

	return nil
}

func (r *MongoRepository) UpsertStepBegin(ctx context.Context, runID, executionID string, step *m.StepDocument, testID string, retryIndex int32) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if step == nil {
		return fmt.Errorf("step is required")
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	step.RunID = runID
	step.ExecutionID = executionID
	step.TestCaseRunID = testID
	step.RetryIndex = retryIndex
	step.UpdatedAt = now
	if step.CreatedAt.IsZero() {
		step.CreatedAt = now
	}
	if step.Steps == nil {
		step.Steps = []*m.StepDocument{}
	}

	buffer, err := r.loadLiveStepBuffer(ctx, runID, executionID, testID)
	if err != nil {
		return err
	}
	if buffer == nil || buffer.AttemptIndex != retryIndex {
		return fmt.Errorf("active step buffer not found: runID=%s, executionID=%s, testID=%s, retryIndex=%d", runID, executionID, testID, retryIndex)
	}

	buffer.Status = activeStepBufferStatusActive
	buffer.LastEventAt = &now
	ttlAt := now.Add(db.MongoStepBufferTTL(r.logger))
	buffer.TTLAt = &ttlAt
	buffer.UpdatedAt = now
	upsertStepDocument(&buffer.Steps, step)

	if err := r.saveLiveStepBuffer(ctx, buffer); err != nil {
		return err
	}

	return nil
}

func (r *MongoRepository) UpsertStepEnd(ctx context.Context, runID, executionID string, stepID string, testID string, retryIndex int32, status string, metadata map[string]interface{}, errorMsg string, errors []string, duration *int64) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if stepID == "" {
		return fmt.Errorf("stepID is required")
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	buffer, err := r.loadLiveStepBuffer(ctx, runID, executionID, testID)
	if err != nil {
		return err
	}
	if buffer == nil || buffer.AttemptIndex != retryIndex {
		return fmt.Errorf("active step buffer not found: runID=%s, executionID=%s, testID=%s, retryIndex=%d", runID, executionID, testID, retryIndex)
	}

	step := findStepDocument(buffer.Steps, stepID)
	if step == nil {
		return fmt.Errorf("step not found in active buffer: runID=%s, executionID=%s, stepID=%s, testID=%s, retryIndex=%d", runID, executionID, stepID, testID, retryIndex)
	}

	now := time.Now()
	step.UpdatedAt = now
	if status != "" {
		step.Status = status
	}
	if len(metadata) > 0 {
		if step.Metadata == nil {
			step.Metadata = map[string]interface{}{}
		}
		for key, value := range metadata {
			step.Metadata[key] = value
		}
	}
	if errorMsg != "" {
		step.Error = errorMsg
	}
	if len(errors) > 0 {
		step.Errors = append([]string(nil), errors...)
	}
	if duration != nil {
		step.Duration = duration
	}

	buffer.Status = activeStepBufferStatusActive
	buffer.LastEventAt = &now
	ttlAt := now.Add(db.MongoStepBufferTTL(r.logger))
	buffer.TTLAt = &ttlAt
	buffer.UpdatedAt = now

	if err := r.saveLiveStepBuffer(ctx, buffer); err != nil {
		return err
	}

	return nil
}

func (r *MongoRepository) PrepareActiveTestStepsFlush(ctx context.Context, runID, executionID, testID string, retryIndex int32) ([]*m.StepDocument, bool, error) {
	if err := repository.ValidateRunID(runID); err != nil {
		return nil, false, err
	}
	if testID == "" {
		return nil, false, fmt.Errorf("testID is required")
	}

	now := time.Now()
	var buffer m.LiveStepBufferDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": liveStepBufferID(runID, executionID, testID), "attempt_index": retryIndex},
		bson.M{"$set": bson.M{
			"status":           activeStepBufferStatusFlushInProgress,
			"flush_started_at": now,
			"updated_at":       now,
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&buffer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("prepare active test steps flush: %w", err)
	}

	return cloneStepDocuments(buffer.Steps), true, nil
}

func (r *MongoRepository) ResetActiveTestStepsFlushState(ctx context.Context, runID, executionID, testID string, retryIndex int32) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{
		"_id":           liveStepBufferID(runID, executionID, testID),
		"attempt_index": retryIndex,
	}, bson.M{
		"$set": bson.M{
			"status":     activeStepBufferStatusActive,
			"updated_at": now,
		},
		"$unset": bson.M{
			"flush_started_at": "",
		},
	})
	if err != nil {
		return fmt.Errorf("reset active test steps flush state: %w", err)
	}
	return nil
}

func (r *MongoRepository) DeleteActiveTestSteps(ctx context.Context, runID, executionID, testID string, retryIndex int32) error {
	if err := repository.ValidateRunID(runID); err != nil {
		return err
	}
	if testID == "" {
		return fmt.Errorf("testID is required")
	}

	now := time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{
		"_id":           liveStepBufferID(runID, executionID, testID),
		"attempt_index": retryIndex,
	}, bson.M{
		"$set": bson.M{"updated_at": now},
	})
	if err != nil {
		return fmt.Errorf("delete active test steps: %w", err)
	}
	if _, err := r.collection.DeleteOne(ctx, bson.M{"_id": liveStepBufferID(runID, executionID, testID), "attempt_index": retryIndex}); err != nil {
		return fmt.Errorf("delete live step buffer: %w", err)
	}
	return nil
}

func (r *MongoRepository) loadLiveStepBuffer(ctx context.Context, runID, executionID, testID string) (*m.LiveStepBufferDocument, error) {
	var buffer m.LiveStepBufferDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": liveStepBufferID(runID, executionID, testID)}).Decode(&buffer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("load live step buffer: %w", err)
	}
	return &buffer, nil
}

func (r *MongoRepository) saveLiveStepBuffer(ctx context.Context, buffer *m.LiveStepBufferDocument) error {
	if buffer == nil {
		return fmt.Errorf("buffer is required")
	}
	if _, err := r.collection.ReplaceOne(ctx, bson.M{"_id": buffer.ID}, buffer); err != nil {
		return fmt.Errorf("save live step buffer: %w", err)
	}
	return nil
}

func upsertStepDocument(target *[]*m.StepDocument, step *m.StepDocument) bool {
	if step == nil {
		return false
	}
	if existing := findStepDocument(*target, step.ID); existing != nil {
		mergeStepDocument(existing, step)
		return true
	}
	if step.ParentStepID != "" {
		if parent := findStepDocument(*target, step.ParentStepID); parent != nil {
			parent.Steps = append(parent.Steps, cloneStepDocuments([]*m.StepDocument{step})[0])
			return true
		}
	}
	*target = append(*target, cloneStepDocuments([]*m.StepDocument{step})[0])
	return true
}

func findStepDocument(steps []*m.StepDocument, stepID string) *m.StepDocument {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if step.ID == stepID {
			return step
		}
		if nested := findStepDocument(step.Steps, stepID); nested != nil {
			return nested
		}
	}
	return nil
}

func mergeStepDocument(target, source *m.StepDocument) {
	if target == nil || source == nil {
		return
	}
	if source.RunID != "" {
		target.RunID = source.RunID
	}
	if source.ExecutionID != "" {
		target.ExecutionID = source.ExecutionID
	}
	if source.TestCaseRunID != "" {
		target.TestCaseRunID = source.TestCaseRunID
	}
	if source.ParentStepID != "" {
		target.ParentStepID = source.ParentStepID
	}
	if source.Title != "" {
		target.Title = source.Title
	}
	if source.Description != "" {
		target.Description = source.Description
	}
	if source.StartTime != nil {
		target.StartTime = source.StartTime
	}
	if source.Duration != nil {
		target.Duration = source.Duration
	}
	if source.Type != "" {
		target.Type = source.Type
	}
	if len(source.Metadata) > 0 {
		if target.Metadata == nil {
			target.Metadata = map[string]interface{}{}
		}
		for key, value := range source.Metadata {
			target.Metadata[key] = value
		}
	}
	if len(source.Tags) > 0 {
		target.Tags = append([]string(nil), source.Tags...)
	}
	if source.WorkerIndex != "" {
		target.WorkerIndex = source.WorkerIndex
	}
	if source.Status != "" {
		target.Status = source.Status
	}
	if source.Category != "" {
		target.Category = source.Category
	}
	if source.Location != "" {
		target.Location = source.Location
	}
	if source.RetryIndex != 0 {
		target.RetryIndex = source.RetryIndex
	}
	if source.Error != "" {
		target.Error = source.Error
	}
	if len(source.Errors) > 0 {
		target.Errors = append([]string(nil), source.Errors...)
	}
	if !source.CreatedAt.IsZero() {
		target.CreatedAt = source.CreatedAt
	}
	if !source.UpdatedAt.IsZero() {
		target.UpdatedAt = source.UpdatedAt
	}
	if len(source.Steps) > 0 {
		target.Steps = cloneStepDocuments(source.Steps)
	}
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
