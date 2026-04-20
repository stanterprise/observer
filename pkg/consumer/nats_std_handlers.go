package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/encoding/protojson"
)

// handleStdOutput processes a stdout event
func (c *NATSConsumer) handleStdOutput(ctx context.Context, data json.RawMessage) error {
	var req events.StdOutputEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal stdout event: %w", err)
	}

	c.logger.Debug("stdout",
		"run_id", req.RunId,
		"test_id", req.TestId,
		"message_len", len(req.Message))

	if req.RunId == "" {
		c.logger.Warn("stdout event missing run_id", "test_id", req.TestId)
		return nil
	}

	// timestamp removed (unused)

	// TODO: Implement Postgres AppendStdOutput if needed, or remove if not required.

	return nil
}

// handleStdError processes a stderr event
func (c *NATSConsumer) handleStdError(ctx context.Context, data json.RawMessage) error {
	var req events.StdErrorEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal stderr event: %w", err)
	}

	c.logger.Debug("stderr",
		"run_id", req.RunId,
		"test_id", req.TestId,
		"message_len", len(req.Message))

	if req.RunId == "" {
		c.logger.Warn("stderr event missing run_id", "test_id", req.TestId)
		return nil
	}

	// timestamp removed (unused)

	// TODO: Implement Postgres AppendStdError if needed, or remove if not required.

	return nil
}
