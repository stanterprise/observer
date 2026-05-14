package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/encoding/protojson"
)

func (c *NATSConsumer) handleRunStart(ctx context.Context, data json.RawMessage) error {
	var req events.ReportRunStartEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal run start event: %w", err)
	}

	c.logger.Info("run start",
		"run_id", req.RunId,
		"execution_id", req.ExecutionId,
		"name", req.Name,
		"total_tests", req.TotalTests,
		"suite_count", len(req.TestSuites))

	c.markRunStart(req.RunId, req.TotalTests)

	if c.pgRepo.IsConfigured() {
		if err := c.pgRepo.HandleRunStart(ctx, &req); err != nil {
			return fmt.Errorf("handle run start: %w", err)
		}
	}

	return nil
}

// handleRunEnd processes a test run end event
func (c *NATSConsumer) handleRunEnd(ctx context.Context, data json.RawMessage) error {
	var req events.TestRunEndEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal run end event: %w", err)
	}

	c.logger.Info("run end", "run_id", req.RunId, "execution_id", req.ExecutionId, "status", req.FinalStatus)

	if c.pgRepo.IsConfigured() {
		if err := c.pgRepo.HandleRunEnd(ctx, &req); err != nil {
			return fmt.Errorf("handle run end: %w", err)
		}
	}

	c.emitRunCompletenessSummary(req.RunId, req.FinalStatus.String())

	return nil
}
