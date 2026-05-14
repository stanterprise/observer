package websocket

import (
	"time"

	"github.com/stanterprise/observer/internal/models"
	entities "github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
)

// convertMetadata converts protobuf map[string]string to map[string]interface{}
func convertMetadata(protoMeta map[string]string) map[string]interface{} {
	if protoMeta == nil {
		return nil
	}
	md := make(map[string]interface{})
	for k, v := range protoMeta {
		md[k] = v
	}
	return md
}

// protoToTestRunDocument converts ReportRunStartEventRequest to models.TestRun
// Only populates fields relevant for WebSocket streaming (omits CreatedAt/UpdatedAt)
func protoToTestRunDocument(req *events.ReportRunStartEventRequest) *models.TestRun {
	if req == nil {
		return nil
	}

	return &models.TestRun{
		ID:       req.RunId,
		Name:     req.Name,
		Metadata: convertMetadata(req.Metadata),
	}
}

// protoToTest converts protobuf TestCaseRun to TestDocument
// Only populates fields relevant for WebSocket streaming (omits CreatedAt/UpdatedAt)
func protoToTest(tc *entities.TestCaseRun) *models.Test {
	if tc == nil {
		return nil
	}

	var startTime, endTime *time.Time
	if tc.StartTime != nil {
		t := tc.StartTime.AsTime()
		startTime = &t
	}
	if tc.EndTime != nil {
		t := tc.EndTime.AsTime()
		endTime = &t
	}

	return &models.Test{
		ID:          tc.Id,
		RunID:       tc.RunId,
		SuiteID:     &tc.TestSuiteId,
		Name:        tc.Name,
		Description: tc.Description,
		Status:      tc.Status.String(),
		Metadata:    convertMetadata(tc.Metadata),
		StartTime:   startTime,
		EndTime:     endTime,
	}
}

func protoToStepDocument(step *entities.StepRun) *models.StepDocument {
	if step == nil {
		return nil
	}

	var startTime *time.Time
	var durationNanos *int64
	if step.StartTime != nil {
		t := step.StartTime.AsTime()
		startTime = &t
	}
	if step.Duration != nil {
		nanos := step.Duration.AsDuration().Nanoseconds()
		durationNanos = &nanos
	}

	return &models.StepDocument{
		ID:            step.Id,
		TestCaseRunID: step.TestCaseId,
		Title:         step.Title,
		Description:   step.Description,
		Status:        step.Status.String(),
		Metadata:      convertMetadata(step.Metadata),
		StartTime:     startTime,
		Duration:      durationNanos,
		ParentStepID:  step.ParentStepId,
		RunID:         step.RunId,
		Type:          step.Type,
		Category:      step.Category,
	}
}
