package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/proto-go/testsystem/v1/events"
)

// handleTestBegin processes a test begin event
func (c *MongoNATSConsumer) handleTestBegin(ctx context.Context, data json.RawMessage) error {
	var req events.TestBeginEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test begin event: %w", err)
	}

	if req.TestCase == nil {
		return errors.New("test_case is nil")
	}

	c.logger.Info("test start",
		"id", req.TestCase.Id,
		"run_id", req.TestCase.RunId,
		"title", req.TestCase.Name)

	// Convert metadata
	md := make(map[string]interface{})
	for k, v := range req.TestCase.Metadata {
		md[k] = v
	}

	var startTime *time.Time
	if req.TestCase.StartTime != nil {
		t := req.TestCase.StartTime.AsTime()
		startTime = &t
	}

	var endTime *time.Time
	if req.TestCase.EndTime != nil {
		t := req.TestCase.EndTime.AsTime()
		endTime = &t
	}

	// Convert attachments
	var attachments []map[string]interface{}
	for _, att := range req.TestCase.Attachments {
		attMap := make(map[string]interface{})
		attMap["name"] = att.Name
		attMap["mime_type"] = att.MimeType
		if content := att.GetContent(); len(content) > 0 {
			attMap["content"] = string(content)
		} else if att.GetUri() != "" {
			attMap["uri"] = att.GetUri()
		}
		attachments = append(attachments, attMap)
	}

	test := &m.TestDocument{
		ID:           req.TestCase.Id,
		Name:         req.TestCase.Name,
		Title:        req.TestCase.Name,
		Description:  req.TestCase.Description,
		RunID:        req.TestCase.RunId,
		Status:       req.TestCase.Status.String(),
		StartTime:    startTime,
		EndTime:      endTime,
		Metadata:     md,
		Tags:         req.TestCase.Tags,
		Location:     req.TestCase.Location,
		RetryCount:   ptrInt32(req.TestCase.RetryCount),
		RetryIndex:   ptrInt32(req.TestCase.RetryIndex),
		Timeout:      ptrInt32(req.TestCase.Timeout),
		Attachments:  attachments,
		ErrorMessage: req.TestCase.ErrorMessage,
		StackTrace:   req.TestCase.StackTrace,
		ErrorList:    req.TestCase.Errors,
	}

	if req.TestCase.Duration != nil {
		nanos := req.TestCase.Duration.AsDuration().Nanoseconds()
		test.Duration = &nanos
	}

	// runID identifies the document
	// TestSuiteRunId is the parent suite containing this test
	runID := req.TestCase.RunId
	suiteID := req.TestCase.TestSuiteId
	if suiteID == "" {
		// Fallback: if no TestSuiteRunId, use RunId as both
		suiteID = runID
	}

	return c.repo.UpsertTestBegin(ctx, runID, test, suiteID)
}

// handleTestEnd processes a test end event
func (c *MongoNATSConsumer) handleTestEnd(ctx context.Context, data json.RawMessage) error {
	var req events.TestEndEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test end event: %w", err)
	}

	if req.TestCase == nil {
		return errors.New("test_case is nil")
	}

	c.logger.Info("test finish",
		"id", req.TestCase.Id,
		"run_id", req.TestCase.RunId,
		"status", req.TestCase.Status)

	var duration *int64
	if req.TestCase.Duration != nil {
		nanos := req.TestCase.Duration.AsDuration().Nanoseconds()
		duration = &nanos
	}

	runID := req.TestCase.RunId
	return c.repo.UpsertTestEnd(ctx, runID, req.TestCase.Id, req.TestCase.Status.String(), req.TestCase.RetryIndex, duration)
}

// handleTestFailure processes a test failure event
func (c *MongoNATSConsumer) handleTestFailure(ctx context.Context, data json.RawMessage) error {
	var req events.TestFailureEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test failure event: %w", err)
	}

	c.logger.Info("test failure",
		"run_id", req.RunId,
		"test_id", req.TestId,
		"message_len", len(req.FailureMessage))

	if req.RunId == "" {
		c.logger.Warn("test failure event missing run_id", "test_id", req.TestId)
		return nil
	}

	// Convert protobuf Timestamp to *time.Time
	var timestamp *time.Time
	if req.Timestamp != nil {
		t := req.Timestamp.AsTime()
		timestamp = &t
	}

	// Convert attachments to map slice
	attachments := make([]map[string]interface{}, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attMap := map[string]interface{}{
			"name":      att.Name,
			"mime_type": att.MimeType,
		}
		// Handle oneof payload
		if content := att.GetContent(); content != nil {
			attMap["content"] = content
		}
		if uri := att.GetUri(); uri != "" {
			attMap["uri"] = uri
		}
		attachments = append(attachments, attMap)
	}

	failure := m.TestFailureDocument{
		FailureMessage: req.FailureMessage,
		StackTrace:     req.StackTrace,
		Timestamp:      timestamp,
		Attachments:    attachments,
	}

	if err := c.repo.AppendTestFailure(ctx, req.RunId, req.TestId, failure); err != nil {
		return fmt.Errorf("append test failure: %w", err)
	}

	return nil
}

// handleTestError processes a test error event
func (c *MongoNATSConsumer) handleTestError(ctx context.Context, data json.RawMessage) error {
	var req events.TestErrorEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test error event: %w", err)
	}

	c.logger.Info("test error",
		"run_id", req.RunId,
		"test_id", req.TestId,
		"message_len", len(req.ErrorMessage))

	if req.RunId == "" {
		c.logger.Warn("test error event missing run_id", "test_id", req.TestId)
		return nil
	}

	// Convert protobuf Timestamp to *time.Time
	var timestamp *time.Time
	if req.Timestamp != nil {
		t := req.Timestamp.AsTime()
		timestamp = &t
	}

	// Convert attachments to map slice
	attachments := make([]map[string]interface{}, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attMap := map[string]interface{}{
			"name":      att.Name,
			"mime_type": att.MimeType,
		}
		// Handle oneof payload
		if content := att.GetContent(); content != nil {
			attMap["content"] = content
		}
		if uri := att.GetUri(); uri != "" {
			attMap["uri"] = uri
		}
		attachments = append(attachments, attMap)
	}

	errorDoc := m.TestErrorDocument{
		ErrorMessage: req.ErrorMessage,
		StackTrace:   req.StackTrace,
		Timestamp:    timestamp,
		Attachments:  attachments,
	}

	if err := c.repo.AppendTestError(ctx, req.RunId, req.TestId, errorDoc); err != nil {
		return fmt.Errorf("append test error: %w", err)
	}

	return nil
}
