package models

import (
	"time"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
)

// RunStartEventToTestRun maps a ReportRunStartEventRequest to a TestRun row.
// The returned value is ready for an idempotent upsert into the runs table.
func RunStartEventToTestRun(req *events.ReportRunStartEventRequest) (*TestRun, []*Suite) {
	if req == nil {
		return nil, nil
	}

	now := time.Now()

	md := make(map[string]interface{})
	for k, v := range req.Metadata {
		md[k] = v
	}
	suites := make([]*Suite, 0, len(req.TestSuites))
	for _, protoSuite := range req.TestSuites {
		if protoSuite == nil {
			continue
		}
		suite := &Suite{
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
		}
		suites = append(suites, suite)
	}

	return &TestRun{
		ID:         req.RunId,
		Name:       req.Name,
		Status:     "RUNNING",
		TotalTests: req.TotalTests,
		Metadata:   md,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, suites
}

// RunEndEventToTestRun maps a TestRunEndEventRequest to TestRun.
// The returned fields are intended for a partial update to finalize the run's terminal state.
func RunEndEventToTestRun(req *events.TestRunEndEventRequest) TestRun {
	if req == nil {
		return TestRun{}
	}

	now := time.Now()
	fields := TestRun{
		ID:        req.RunId,
		Status:    req.FinalStatus.String(),
		UpdatedAt: now,
	}

	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		fields.StartTime = &t
	}

	if req.Duration != nil {
		d := req.Duration.AsDuration().Nanoseconds()
		fields.Duration = &d
	}

	return fields
}
