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

// protoToTestRunDocument converts ReportRunStartEventRequest to TestRunDocument
// Only populates fields relevant for WebSocket streaming (omits CreatedAt/UpdatedAt)
func protoToTestRunDocument(req *events.ReportRunStartEventRequest) *models.TestRunDocument {
	if req == nil {
		return nil
	}

	return &models.TestRunDocument{
		ID:         req.RunId,
		Name:       req.Name,
		TotalTests: req.TotalTests,
		Metadata:   convertMetadata(req.Metadata),
	}
}

// protoToTestDocument converts protobuf TestCaseRun to TestDocument
// Only populates fields relevant for WebSocket streaming (omits CreatedAt/UpdatedAt)
func protoToTestDocument(tc *entities.TestCaseRun) *models.TestDocument {
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

	return &models.TestDocument{
		ID:          tc.Id,
		RunID:       tc.RunId,
		SuiteID:     tc.TestSuiteId,
		Name:        tc.Name,
		Description: tc.Description,
		Status:      tc.Status.String(),
		Metadata:    convertMetadata(tc.Metadata),
		StartTime:   startTime,
		EndTime:     endTime,
	}
}
