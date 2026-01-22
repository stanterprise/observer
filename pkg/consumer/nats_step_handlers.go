package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/encoding/protojson"
)

// handleStepBegin processes a step begin event
func (c *MongoNATSConsumer) handleStepBegin(ctx context.Context, data json.RawMessage) error {
	var req events.StepBeginEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal step begin event: %w", err)
	}

	if req.Step == nil {
		return errors.New("step is nil")
	}

	// Extract the actual test ID from TestCaseRunId
	// TestCaseRunId format is typically: {runId}-{testId}
	// But tests are stored with just {testId}, so we need to extract it

	c.logger.Info("step start",
		"id", req.Step.Id,
		"test_case_run_id", req.Step.TestCaseId,
		"retry_index", req.Step.RetryIndex)

	// Convert metadata
	md := make(map[string]interface{})
	for k, v := range req.Step.Metadata {
		md[k] = v
	}

	var startTime *time.Time
	if req.Step.StartTime != nil {
		t := req.Step.StartTime.AsTime()
		startTime = &t
	}

	step := &m.StepDocument{
		ID:            req.Step.Id,
		RunID:         req.Step.RunId,
		TestCaseRunID: req.Step.TestCaseId,
		ParentStepID:  req.Step.ParentStepId,
		Title:         req.Step.Title,
		Description:   req.Step.Description,
		StartTime:     startTime,
		Type:          req.Step.Type,
		Metadata:      md,
		// Tags:          req.Step.Tags, // TODO: Add when available in protobuf
		WorkerIndex: req.Step.WorkerIndex,
		Status:      req.Step.Status.String(),
		Category:    req.Step.Category,
		Location:    req.Step.Location,
		Error:       req.Step.Error,
		Errors:      req.Step.Errors,
	}

	if req.Step.Duration != nil {
		nanos := req.Step.Duration.AsDuration().Nanoseconds()
		step.Duration = &nanos
	}

	runID := req.Step.RunId
	return c.repo.UpsertStepBegin(ctx, runID, step, req.Step.TestCaseId, req.Step.RetryIndex)
}

// handleStepEnd processes a step end event
func (c *MongoNATSConsumer) handleStepEnd(ctx context.Context, data json.RawMessage) error {
	var req events.StepEndEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal step end event: %w", err)
	}

	if req.Step == nil {
		return errors.New("step is nil")
	}

	c.logger.Info("step end",
		"id", req.Step.Id,
		"status", req.Step.Status,
		"retry_index", req.Step.RetryIndex)

	// Extract testID from TestCaseRunId (same as in handleStepBegin)
	testID := extractTestID(req.Step.TestCaseId, req.Step.RunId)
	runID := req.Step.RunId

	// Extract retry_index from Step (defaults to 0 if not set)
	retryIndex := req.Step.RetryIndex

	// Convert metadata (including error metadata from Playwright reporter)
	metadata := make(map[string]interface{})
	for k, v := range req.Step.Metadata {
		metadata[k] = v
	}

	// Extract error fields
	errorMsg := req.Step.Error
	errors := req.Step.Errors

	// Calculate duration if available
	var duration *int64
	if req.Step.Duration != nil {
		nanos := req.Step.Duration.AsDuration().Nanoseconds()
		duration = &nanos
	}

	return c.repo.UpsertStepEnd(ctx, runID, req.Step.Id, testID, retryIndex, mongoStatusToString(req.Step.Status), metadata, errorMsg, errors, duration)
}
