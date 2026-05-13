package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stanterprise/observer/internal/models"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/encoding/protojson"
)

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

	// Convert protobuf Timestamp to *time.Time (removed, unused)
	// Convert protobuf Duration to *int64 (removed, unused)

	// Update the test run document with final status, times, and duration
	// MongoDB UpdateTestRunEnd and MarkRunningTestsAsTimedOut removed (legacy)

	_ = models.RunEndEventToRunExecution(&req)
	if c.pgRepo.IsConfigured() {
	}

	c.emitRunCompletenessSummary(req.RunId, req.FinalStatus.String())

	return nil
}

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

	testRun, relationalSuites := models.RunStartEventToTestRun(&req)
	runExecution := models.RunStartEventToRunExecution(&req)
	relationalTests := models.RunStartEventToTests(&req)
	if c.pgRepo.IsConfigured() {
		if err := c.pgRepo.UpsertRunStart(ctx, testRun); err != nil {
			return fmt.Errorf("upsert run start: %w", err)
		}
		if err := c.pgRepo.UpsertRunExecutionStart(ctx, runExecution); err != nil {
			return fmt.Errorf("upsert run execution start: %w", err)
		}
		if err := c.pgRepo.UpsertRunStartSuites(ctx, relationalSuites); err != nil {
			return fmt.Errorf("upsert run start suites: %w", err)
		}
		if err := c.pgRepo.UpsertRunStartTests(ctx, relationalTests); err != nil {
			return fmt.Errorf("upsert run start tests: %w", err)
		}
	}

	// MongoDB MapSuites removed (legacy)
	return nil
}
