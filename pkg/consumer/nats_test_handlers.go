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

func ensureRelationalTestSuiteID(relationalTest *m.Test, suiteID string) {
	if relationalTest == nil || relationalTest.SuiteID != nil {
		return
	}
	if suiteID == "" {
		return
	}
	relationalTest.SuiteID = &suiteID
}

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
		"execution_id", req.TestCase.ExecutionId,
		"title", req.TestCase.Name)

	var startTime *time.Time
	if req.TestCase.StartTime != nil {
		t := req.TestCase.StartTime.AsTime()
		startTime = &t
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

	// runID identifies the document
	// TestSuiteRunId is the parent suite containing this test
	runID := req.TestCase.RunId
	executionID := req.TestCase.ExecutionId
	suiteID := req.TestCase.TestSuiteId
	relationalTest := m.TestCaseRunToRelationalTest(req.TestCase)
	ensureRelationalTestSuiteID(relationalTest, suiteID)
	relationalAttempt := m.TestCaseRunToRelationalAttempt(req.TestCase, attachments)
	if c.pgRepo.IsConfigured() {
		if err := c.pgRepo.UpsertTestBegin(ctx, relationalTest, relationalAttempt); err != nil {
			return fmt.Errorf("upsert relational test begin: %w", err)
		}
	}

	if c.bufferRepo != nil {
		if err := c.bufferRepo.SyncActiveTestSteps(ctx, runID, executionID, req.TestCase.Id, req.TestCase.RetryIndex, startTime); err != nil {
			return fmt.Errorf("sync active test steps: %w", err)
		}
	}

	// Replay deferred steps asynchronously so test.begin processing is not
	// blocked by a large backlog and can be acked within JetStream AckWait.
	testID := req.TestCase.Id
	retryIndex := req.TestCase.RetryIndex
	go func(runID, executionID, testID string, retryIndex int32) {
		replayCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.replayDeferredStepsForTest(replayCtx, runID, executionID, testID, retryIndex)
	}(runID, executionID, testID, retryIndex)
	// Schedule additional delayed sweeps to catch step.end events that arrive
	// out-of-order after test.begin but before step.begin creates the step.
	c.scheduleDeferredStepReplaySweep(runID, executionID, req.TestCase.Id, req.TestCase.RetryIndex)
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
		"execution_id", req.TestCase.ExecutionId,
		"status", req.TestCase.Status,
		"retry_index", req.TestCase.RetryIndex)

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
	executionID := req.TestCase.ExecutionId
	relationalTest := m.TestCaseRunToRelationalTest(req.TestCase)
	ensureRelationalTestSuiteID(relationalTest, req.TestCase.TestSuiteId)
	relationalAttempt := m.TestCaseRunToRelationalAttempt(req.TestCase, attachments)

	if c.pgRepo.IsConfigured() {
		buffered := false
		if c.bufferRepo != nil {
			steps, hasBuffer, err := c.bufferRepo.PrepareActiveTestStepsFlush(ctx, runID, executionID, req.TestCase.Id, req.TestCase.RetryIndex)
			if err != nil {
				return fmt.Errorf("prepare active test steps flush: %w", err)
			}
			if hasBuffer {
				buffered = true
				relationalAttempt.StepsCount = int32(len(steps))
				rawSteps, err := m.StepFromDocuments(steps)
				if err != nil {
					if resetErr := c.bufferRepo.ResetActiveTestStepsFlushState(ctx, runID, executionID, req.TestCase.Id, req.TestCase.RetryIndex); resetErr != nil {
						c.logger.Warn("failed to reset active test step buffer after marshal error", "run_id", runID, "test_id", req.TestCase.Id, "retry_index", req.TestCase.RetryIndex, "error", resetErr)
					}
					return fmt.Errorf("marshal attempt steps: %w", err)
				}
				relationalAttempt.Steps = rawSteps

			}
		}

		if err := c.pgRepo.FinalizeTestEnd(ctx, relationalTest, relationalAttempt); err != nil {
			if buffered && c.bufferRepo != nil {
				if resetErr := c.bufferRepo.ResetActiveTestStepsFlushState(ctx, runID, executionID, req.TestCase.Id, req.TestCase.RetryIndex); resetErr != nil {
					c.logger.Warn("failed to reset active test step buffer after postgres failure", "run_id", runID, "test_id", req.TestCase.Id, "retry_index", req.TestCase.RetryIndex, "error", resetErr)
				}
			}
			return fmt.Errorf("finalize relational test end: %w", err)
		}

		if buffered && c.bufferRepo != nil {
			if err := c.bufferRepo.DeleteActiveTestSteps(ctx, runID, executionID, req.TestCase.Id, req.TestCase.RetryIndex); err != nil {
				return fmt.Errorf("delete active test steps after flush: %w", err)
			}
		}
	}

	return nil
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
	// MongoDB GetTestFromRun and AppendTestFailure removed (legacy)
	// Use Postgres only
	retryIndex := int32(0)
	if c.pgRepo.IsConfigured() {
		if err := c.pgRepo.AppendTestFailure(ctx, req.RunId, req.ExecutionId, req.TestId, retryIndex, &failure); err != nil {
			return fmt.Errorf("append relational test failure: %w", err)
		}
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
	// MongoDB GetTestFromRun and AppendTestError removed (legacy)
	// Use Postgres only
	retryIndex := int32(0)
	if c.pgRepo.IsConfigured() {
		if err := c.pgRepo.AppendTestError(ctx, req.RunId, req.ExecutionId, req.TestId, retryIndex, &errorDoc); err != nil {
			return fmt.Errorf("append relational test error: %w", err)
		}
	}
	return nil
}

// processAttachment processes an attachment and returns a map suitable for storage.
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
func (c *NATSConsumer) scheduleDeferredStepReplaySweep(runID, executionID, testID string, retryIndex int32) {
	delays := []time.Duration{
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
	}

	go func() {
		for _, delay := range delays {
			<-time.After(delay)

			attemptCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			c.replayDeferredStepsForTest(attemptCtx, runID, executionID, testID, retryIndex)
			cancel()
		}
	}()
}
