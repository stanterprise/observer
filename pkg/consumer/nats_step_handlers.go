package consumer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	"github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/encoding/protojson"
)

// handleStepBegin processes a step begin event
func (c *NATSConsumer) handleStepBegin(ctx context.Context, data json.RawMessage) error {
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
	testID := extractTestID(req.Step.TestCaseId, req.Step.RunId)
	retryIndex := req.Step.RetryIndex
	if c.bufferRepo != nil {
		if err := c.bufferRepo.UpsertStepBegin(ctx, runID, step, testID, retryIndex); err != nil {
			return err
		}
	}

	// After the step is created, schedule deferred replay sweeps so any
	// step.end events that arrived out-of-order (deferred while waiting for
	// this step to exist) are applied promptly.
	c.scheduleDeferredStepReplaySweep(runID, testID, retryIndex)
	return nil
}

// handleStepEnd processes a step end event
func (c *NATSConsumer) handleStepEnd(ctx context.Context, data json.RawMessage) error {
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

	testID := extractTestID(req.Step.TestCaseId, req.Step.RunId)
	runID := req.Step.RunId
	retryIndex := req.Step.RetryIndex

	// Convert metadata (including error metadata from Playwright reporter)
	metadata := make(map[string]interface{})
	for k, v := range req.Step.Metadata {
		metadata[k] = v
	}

	// Extract step attachments from metadata if present
	stepAttachments := c.extractStepAttachments(ctx, req.Step.Metadata)

	errorMsg := req.Step.Error
	errors := req.Step.Errors

	var duration *int64
	if req.Step.Duration != nil {
		nanos := req.Step.Duration.AsDuration().Nanoseconds()
		duration = &nanos
	}

	if c.bufferRepo != nil {
		if err := c.bufferRepo.UpsertStepEnd(ctx, runID, req.Step.Id, testID, retryIndex, statusToString(req.Step.Status), metadata, errorMsg, errors, duration); err != nil {
			return err
		}
	}

	if len(stepAttachments) > 0 {
		// TODO: Implement Postgres AppendTestAttachments if needed, or remove if not required.
	}

	return nil
}

type stepAttachmentPayload struct {
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	MimeType    string `json:"mime_type"`
	Body        string `json:"body"`
	Encoding    string `json:"encoding"`
	URI         string `json:"uri"`
}

func (c *NATSConsumer) extractStepAttachments(ctx context.Context, metadata map[string]string) []map[string]interface{} {
	keys := []string{"attachments", "attachments_json", "step.attachments", "pw:attachments"}
	var raw string
	for _, key := range keys {
		if value, ok := metadata[key]; ok && value != "" {
			raw = value
			break
		}
	}
	if raw == "" {
		return nil
	}

	var payloads []stepAttachmentPayload
	if err := json.Unmarshal([]byte(raw), &payloads); err != nil {
		var single stepAttachmentPayload
		if err := json.Unmarshal([]byte(raw), &single); err != nil {
			c.logger.Warn("failed to parse step attachments", "error", err)
			return nil
		}
		payloads = []stepAttachmentPayload{single}
	}

	attachments := make([]map[string]interface{}, 0, len(payloads))
	for _, payload := range payloads {
		mimeType := payload.MimeType
		if mimeType == "" {
			mimeType = payload.ContentType
		}
		name := payload.Name

		if payload.URI != "" {
			att := &common.Attachment{
				Name:     name,
				MimeType: mimeType,
				Payload:  &common.Attachment_Uri{Uri: payload.URI},
			}
			if attMap, err := c.processAttachment(ctx, att); err == nil {
				attachments = append(attachments, attMap)
			}
			continue
		}

		body := payload.Body
		if body == "" {
			continue
		}

		var content []byte
		if payload.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				c.logger.Warn("failed to decode step attachment body", "error", err)
				continue
			}
			content = decoded
		} else {
			content = []byte(body)
		}

		att := &common.Attachment{
			Name:     name,
			MimeType: mimeType,
			Payload:  &common.Attachment_Content{Content: content},
		}
		attMap, err := c.processAttachment(ctx, att)
		if err != nil {
			c.logger.Warn("failed to process step attachment", "error", err)
			continue
		}
		attachments = append(attachments, attMap)
	}

	return attachments
}
