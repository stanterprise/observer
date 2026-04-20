package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/encoding/protojson"
)

// handleSuiteBegin processes a suite begin event
func (c *NATSConsumer) handleSuiteBegin(ctx context.Context, data json.RawMessage) error {
	var req events.SuiteBeginEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal suite begin event: %w", err)
	}

	if req.Suite == nil {
		return errors.New("suite is nil")
	}

	c.logger.Info("suite start",
		"id", req.Suite.Id,
		"name", req.Suite.Name,
		"project", req.Suite.Project)

	// Convert metadata
	md := make(map[string]interface{})
	for k, v := range req.Suite.Metadata {
		md[k] = v
	}

	// startTime, endTime, duration removed (unused)

	// TODO: Implement Postgres UpsertSuiteBegin if needed, or remove if not required.
	return nil
}

// handleSuiteEnd processes a suite end event
func (c *NATSConsumer) handleSuiteEnd(ctx context.Context, data json.RawMessage) error {
	var req events.SuiteEndEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal suite end event: %w", err)
	}

	if req.Suite == nil {
		return errors.New("suite is nil")
	}

	c.logger.Info("suite finish",
		"id", req.Suite.Id,
		"status", req.Suite.Status)

	// TODO: Implement Postgres UpsertSuiteEnd if needed, or remove if not required.
	return nil
}
