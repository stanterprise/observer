package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stanterprise/observer/internal/models"
	m "github.com/stanterprise/observer/internal/models"
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

	c.logger.Info("run end", "run_id", req.RunId, "status", req.FinalStatus)

	// Convert protobuf Timestamp to *time.Time (removed, unused)
	// Convert protobuf Duration to *int64 (removed, unused)

	// Update the test run document with final status, times, and duration
	// MongoDB UpdateTestRunEnd and MarkRunningTestsAsTimedOut removed (legacy)

	testRun := models.RunEndEventToTestRun(&req)
	if c.pgRepo.IsConfigured() {
		if runShard := models.RunEndEventToRunShard(&req); runShard != nil {
			if _, err := c.pgRepo.FinalizeRunShardEnd(ctx, runShard); err != nil {
				return fmt.Errorf("finalize run shard end: %w", err)
			}
		} else {
			if err := c.pgRepo.FinalizeRunEnd(ctx, testRun); err != nil {
				return fmt.Errorf("finalize run end: %w", err)
			}
		}
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
		"name", req.Name,
		"total_tests", req.TotalTests,
		"suite_count", len(req.TestSuites))

	c.markRunStart(req.RunId, req.TotalTests)

	// Convert run-level metadata
	runMetadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		runMetadata[k] = v
	}

	// Convert protobuf entities to Suite models
	suites := make([]m.Suite, 0, len(req.TestSuites))
	for _, protoSuite := range req.TestSuites {
		if protoSuite == nil {
			continue
		}

		// Convert metadata
		md := make(map[string]interface{})
		for k, v := range protoSuite.Metadata {
			md[k] = v
		}

		var startTime *time.Time
		if protoSuite.StartTime != nil {
			t := protoSuite.StartTime.AsTime()
			startTime = &t
		}

		var endTime *time.Time
		if protoSuite.EndTime != nil {
			t := protoSuite.EndTime.AsTime()
			endTime = &t
		}

		var duration *int64
		if protoSuite.Duration != nil {
			d := protoSuite.Duration.AsDuration().Nanoseconds()
			duration = &d
		}

		suite := m.Suite{
			ID:            protoSuite.Id,
			RunID:         protoSuite.RunId,
			ParentSuiteID: &protoSuite.ParentSuiteId,
			Name:          protoSuite.Name,
			Description:   protoSuite.Description,
			Metadata:      md,
			Location:      protoSuite.Location,
			Type:          protoSuite.Type.String(),
			InitiatedBy:   protoSuite.InitiatedBy,
			ProjectName:   protoSuite.Project,
			Author:        protoSuite.Author,
			Owner:         protoSuite.Owner,
			TestCaseIDs:   protoSuite.TestCaseIds,
			SubSuiteIDs:   protoSuite.SubSuiteIds,
			StartTime:     startTime,
			EndTime:       endTime,
			Duration:      duration,
			Status:        protoSuite.Status.String(),
		}

		suites = append(suites, suite)
	}

	testRun, relationalSuites := models.RunStartEventToTestRun(&req)
	relationalTests := models.RunStartEventToTests(&req)
	if c.pgRepo.IsConfigured() {
		if runShard := models.RunStartEventToRunShard(&req); runShard != nil {
			if err := c.pgRepo.UpsertRunStart(ctx, testRun); err != nil {
				return fmt.Errorf("upsert run start: %w", err)
			}
			if err := c.pgRepo.UpsertRunShardStart(ctx, runShard); err != nil {
				return fmt.Errorf("upsert run shard start: %w", err)
			}
		} else {
			if err := c.pgRepo.UpsertRunStart(ctx, testRun); err != nil {
				return fmt.Errorf("upsert run start: %w", err)
			}
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
