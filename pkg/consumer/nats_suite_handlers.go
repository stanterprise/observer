package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
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

	var startTime *time.Time
	if req.Suite.StartTime != nil {
		t := req.Suite.StartTime.AsTime()
		startTime = &t
	}

	var endTime *time.Time
	if req.Suite.EndTime != nil {
		t := req.Suite.EndTime.AsTime()
		endTime = &t
	}

	var duration *int64
	if req.Suite.Duration != nil {
		d := req.Suite.Duration.AsDuration().Nanoseconds()
		duration = &d
	}

	suite := &m.SuiteDocument{
		ID:              req.Suite.Id,
		RunID:           req.Suite.RunId,
		ParentSuiteID:   req.Suite.ParentSuiteId,
		Name:            req.Suite.Name,
		Description:     req.Suite.Description,
		Status:          req.Suite.Status.String(),
		Metadata:        md,
		Duration:        duration,
		Location:        req.Suite.Location,
		Type:            req.Suite.Type.String(),
		TestSuiteSpecID: "",
		InitiatedBy:     req.Suite.InitiatedBy,
		ProjectName:     req.Suite.Project,
		Author:          req.Suite.Author,
		Owner:           req.Suite.Owner,
		TestCaseIds:     req.Suite.TestCaseIds,
		SubSuiteIds:     req.Suite.SubSuiteIds,
		// Tags:            req.Suite.Tags, // TODO: Add when available in protobuf
		StartTime: startTime,
		EndTime:   endTime,
	}

	// Use ParentSuiteId directly from protobuf (already set in suite object)
	// For root suites: ParentSuiteId will be empty string
	// For nested suites: ParentSuiteId will be set to parent's ID
	runID := req.Suite.RunId

	return c.repo.UpsertSuiteBegin(ctx, runID, suite, suite.ParentSuiteID)
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

	var endTime *time.Time
	if req.Suite.EndTime != nil {
		t := req.Suite.EndTime.AsTime()
		endTime = &t
	}

	var duration *int64
	if req.Suite.Duration != nil {
		d := req.Suite.Duration.AsDuration().Nanoseconds()
		duration = &d
	}

	// Use RunId directly from protobuf
	runID := req.Suite.RunId

	return c.repo.UpsertSuiteEnd(ctx, runID, req.Suite.Id, req.Suite.Status.String(), endTime, duration)
}
