package consumer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/encoding/protojson"
)

// handleTestBegin processes a test begin event
func (c *NATSConsumer) handleTestBegin(ctx context.Context, data json.RawMessage) error {
	var req events.TestBeginEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
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
	attachments := make([]map[string]interface{}, 0, len(req.TestCase.Attachments))
	for _, att := range req.TestCase.Attachments {
		attMap, err := c.processAttachment(ctx, att)
		if err != nil {
			c.logger.Error("failed to process attachment", "error", err)
			continue
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

	if err := c.repo.UpsertTestBegin(ctx, runID, test, suiteID); err != nil {
		return err
	}

	// Replay deferred steps asynchronously so test.begin processing is not
	// blocked by a large backlog and can be acked within JetStream AckWait.
	testID := req.TestCase.Id
	retryIndex := req.TestCase.RetryIndex
	go func(runID, testID string, retryIndex int32) {
		replayCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.replayDeferredStepsForTest(replayCtx, runID, testID, retryIndex)
	}(runID, testID, retryIndex)
	// Schedule additional delayed sweeps to catch step.end events that arrive
	// out-of-order after test.begin but before step.begin creates the step.
	c.scheduleDeferredStepReplaySweep(runID, req.TestCase.Id, req.TestCase.RetryIndex)
	return nil
}

// handleTestEnd processes a test end event
func (c *NATSConsumer) handleTestEnd(ctx context.Context, data json.RawMessage) error {
	var req events.TestEndEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test end event: %w", err)
	}

	if req.TestCase == nil {
		return errors.New("test_case is nil")
	}

	c.logger.Info("test finish",
		"id", req.TestCase.Id,
		"run_id", req.TestCase.RunId,
		"status", req.TestCase.Status,
		"retry_index", req.TestCase.RetryIndex)

	var duration *int64
	if req.TestCase.Duration != nil {
		nanos := req.TestCase.Duration.AsDuration().Nanoseconds()
		duration = &nanos
	}

	var endTime *time.Time
	if req.TestCase.EndTime != nil {
		t := req.TestCase.EndTime.AsTime()
		endTime = &t
	}

	// Convert attachments
	attachments := make([]map[string]interface{}, 0, len(req.TestCase.Attachments))
	for _, att := range req.TestCase.Attachments {
		attMap, err := c.processAttachment(ctx, att)
		if err != nil {
			c.logger.Error("failed to process attachment", "error", err)
			continue
		}
		attachments = append(attachments, attMap)
	}

	runID := req.TestCase.RunId
	return c.repo.UpsertTestEnd(ctx, runID, req.TestCase.Id, req.TestCase.Status.String(), req.TestCase.RetryIndex, endTime, duration, attachments)
}

// handleTestFailure processes a test failure event
func (c *NATSConsumer) handleTestFailure(ctx context.Context, data json.RawMessage) error {
	var req events.TestFailureEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
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
		attMap, err := c.processAttachment(ctx, att)
		if err != nil {
			c.logger.Error("failed to process attachment", "error", err)
			continue
		}
		attachments = append(attachments, attMap)
	}

	failure := m.TestFailureDocument{
		FailureMessage: req.FailureMessage,
		StackTrace:     req.StackTrace,
		Timestamp:      timestamp,
		Attachments:    attachments,
	}

	// Fetch test document to get retry_index (failure events don't include it)
	testDoc, err := c.repo.GetTestFromRun(ctx, req.TestId)
	if err != nil {
		return fmt.Errorf("get test for failure: %w", err)
	}
	if testDoc == nil {
		c.logger.Warn("test not found for failure event", "test_id", req.TestId)
		return nil
	}

	// Use retry_index from test document (defaults to 0 if nil)
	retryIndex := int32(0)
	if testDoc.RetryIndex != nil {
		retryIndex = *testDoc.RetryIndex
	}

	if err := c.repo.AppendTestFailure(ctx, req.RunId, req.TestId, retryIndex, failure); err != nil {
		return fmt.Errorf("append test failure: %w", err)
	}

	return nil
}

// handleTestError processes a test error event
func (c *NATSConsumer) handleTestError(ctx context.Context, data json.RawMessage) error {
	var req events.TestErrorEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
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
		attMap, err := c.processAttachment(ctx, att)
		if err != nil {
			c.logger.Error("failed to process attachment", "error", err)
			continue
		}
		attachments = append(attachments, attMap)
	}

	errorDoc := m.TestErrorDocument{
		ErrorMessage: req.ErrorMessage,
		StackTrace:   req.StackTrace,
		Timestamp:    timestamp,
		Attachments:  attachments,
	}

	// Fetch test document to get retry_index (error events don't include it)
	testDoc, err := c.repo.GetTestFromRun(ctx, req.TestId)
	if err != nil {
		return fmt.Errorf("get test for error: %w", err)
	}
	if testDoc == nil {
		c.logger.Warn("test not found for error event", "test_id", req.TestId)
		return nil
	}

	// Use retry_index from test document (defaults to 0 if nil)
	retryIndex := int32(0)
	if testDoc.RetryIndex != nil {
		retryIndex = *testDoc.RetryIndex
	}

	if err := c.repo.AppendTestError(ctx, req.RunId, req.TestId, retryIndex, errorDoc); err != nil {
		return fmt.Errorf("append test error: %w", err)
	}

	return nil
}

// processAttachment processes an attachment and returns a map suitable for MongoDB storage.
// It uses a size-based strategy:
// - < 100KB: Store inline as base64 content
// - >= 100KB: Store in external storage (if configured)
// Falls back to inline storage if external storage is not configured.
func (c *NATSConsumer) processAttachment(ctx context.Context, att *common.Attachment) (map[string]interface{}, error) {
	attMap := make(map[string]interface{})
	attMap["name"] = att.Name
	attMap["mime_type"] = att.MimeType

	// Handle attachment content
	if content := att.GetContent(); len(content) > 0 {
		const inlineThreshold = 100 * 1024 // 100KB

		if len(content) < inlineThreshold {
			// Small attachments: store inline
			attMap["content"] = base64.StdEncoding.EncodeToString(content)
			attMap["content_encoding"] = "base64"
			attMap["storage"] = "inline"
			attMap["size"] = len(content)
		} else if c.storageDriver != nil {
			// Large attachments: store in external storage
			reader := bytes.NewReader(content)
			metadata, err := c.storageDriver.Upload(ctx, att.Name, att.MimeType, reader)
			if err != nil {
				// Log error but don't fail the event processing
				c.logger.Error("storage upload failed, falling back to inline",
					"name", att.Name,
					"size", len(content),
					"error", err)
				attMap["content"] = base64.StdEncoding.EncodeToString(content)
				attMap["content_encoding"] = "base64"
				attMap["storage"] = "inline"
				attMap["size"] = len(content)
			} else {
				attMap["storage_key"] = metadata.StorageKey
				attMap["storage_uri"] = metadata.StorageURI
				attMap["size"] = metadata.Size
				attMap["storage"] = c.storageDriver.Name()
				attMap["uploaded_at"] = metadata.UploadedAt
				c.logger.Info("attachment stored externally",
					"name", att.Name,
					"size", metadata.Size,
					"storage", c.storageDriver.Name())
			}
		} else {
			// No storage driver configured: store inline
			attMap["content"] = base64.StdEncoding.EncodeToString(content)
			attMap["content_encoding"] = "base64"
			attMap["storage"] = "inline"
			attMap["size"] = len(content)
		}
	} else if uri := att.GetUri(); uri != "" {
		// External URI reference
		attMap["uri"] = uri
		attMap["storage"] = "external"
	}

	return attMap, nil
}

// scheduleDeferredStepReplaySweep performs a small number of delayed replay
// attempts after test or step creation succeeds. This resolves the common
// out-of-order case where step.end arrives after test.begin but before
// step.begin has created the step: the step.end is deferred, but no further
// test.begin will arrive to trigger another replay after step.begin runs.
func (c *NATSConsumer) scheduleDeferredStepReplaySweep(runID, testID string, retryIndex int32) {
	delays := []time.Duration{
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
	}

	go func() {
		for _, delay := range delays {
			<-time.After(delay)

			attemptCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			c.replayDeferredStepsForTest(attemptCtx, runID, testID, retryIndex)
			cancel()
		}
	}()
}
