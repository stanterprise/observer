package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/stanterprise/observer/pkg/publisher"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
)

// EventClassification indicates how an event should be processed
type EventClassification string

const (
	// ClassifyImmediate means the event can be processed immediately
	ClassifyImmediate EventClassification = "immediate"
	// ClassifyBuffer means the event should be buffered for reconciliation
	ClassifyBuffer EventClassification = "buffer"
	// ClassifyReconcile means the event is a root suite end that triggers reconciliation
	ClassifyReconcile EventClassification = "reconcile"
)

// Classifier determines if an event can be immediately processed or needs buffering
type Classifier struct {
	logger *slog.Logger
}

// NewClassifier creates a new event classifier
func NewClassifier(logger *slog.Logger) *Classifier {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &Classifier{
		logger: logger,
	}
}

// Classify determines how an event should be processed
func (c *Classifier) Classify(ctx context.Context, event publisher.Event) (EventClassification, error) {
	switch event.Type {
	case publisher.EventTypeSuiteBegin:
		return c.classifySuiteBegin(ctx, event)
	case publisher.EventTypeSuiteEnd:
		return c.classifySuiteEnd(ctx, event)
	case publisher.EventTypeTestBegin:
		return c.classifyTestBegin(ctx, event)
	case publisher.EventTypeTestEnd:
		return c.classifyTestEnd(ctx, event)
	case publisher.EventTypeStepBegin:
		return c.classifyStepBegin(ctx, event)
	case publisher.EventTypeStepEnd:
		return c.classifyStepEnd(ctx, event)
	case publisher.EventTypeRunStart:
		return c.classifyRunStart(ctx, event)
	case publisher.EventTypeRunEnd:
		return c.classifyRunEnd(ctx, event)
	default:
		// Unknown or heartbeat events can be processed immediately
		return ClassifyImmediate, nil
	}
}

// classifySuiteBegin checks if a suite begin event can be processed immediately
func (c *Classifier) classifySuiteBegin(ctx context.Context, event publisher.Event) (EventClassification, error) {
	var req events.SuiteBeginEventRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		return ClassifyBuffer, fmt.Errorf("unmarshal suite begin: %w", err)
	}

	if req.Suite == nil {
		return ClassifyBuffer, fmt.Errorf("suite begin missing suite")
	}

	// Check if this is a root suite (no parent_suite_id in metadata)
	parentSuiteID := ""
	if req.Suite.Metadata != nil {
		if parent, ok := req.Suite.Metadata["parent_suite_id"]; ok {
			parentSuiteID = parent
		}
	}

	// Root suite can always be processed immediately
	if parentSuiteID == "" {
		c.logger.Debug("root suite begin - immediate",
			"suite_id", req.Suite.Id)
		return ClassifyImmediate, nil
	}

	c.logger.Debug("nested suite begin - parent existence unknown - buffer",
		"suite_id", req.Suite.Id,
		"parent_suite_id", parentSuiteID)
	return ClassifyBuffer, nil
}

// classifySuiteEnd checks if a suite end event triggers reconciliation
func (c *Classifier) classifySuiteEnd(ctx context.Context, event publisher.Event) (EventClassification, error) {
	var req events.SuiteEndEventRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		return ClassifyBuffer, fmt.Errorf("unmarshal suite end: %w", err)
	}

	if req.Suite == nil {
		return ClassifyBuffer, fmt.Errorf("suite end missing suite")
	}

	suiteID := req.Suite.Id

	c.logger.Debug("suite end - suite existence unknown - buffer",
		"suite_id", suiteID)
	return ClassifyBuffer, nil
}

// classifyTestBegin checks if a test begin event can be processed immediately
func (c *Classifier) classifyTestBegin(ctx context.Context, event publisher.Event) (EventClassification, error) {
	var req events.TestBeginEventRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		return ClassifyBuffer, fmt.Errorf("unmarshal test begin: %w", err)
	}

	if req.TestCase == nil {
		return ClassifyBuffer, fmt.Errorf("test begin missing test_case")
	}

	// Extract parent suite ID from metadata
	parentSuiteID := ""
	if req.TestCase.Metadata != nil {
		if parent, ok := req.TestCase.Metadata["suite_id"]; ok {
			parentSuiteID = parent
		}
	}

	if parentSuiteID == "" {
		c.logger.Debug("test begin - no parent suite - buffer",
			"test_id", req.TestCase.Id)
		return ClassifyBuffer, nil
	}

	c.logger.Debug("test begin - parent suite existence unknown - buffer",
		"test_id", req.TestCase.Id,
		"parent_suite_id", parentSuiteID)
	return ClassifyBuffer, nil
}

// classifyTestEnd checks if a test end event can be processed immediately
func (c *Classifier) classifyTestEnd(ctx context.Context, event publisher.Event) (EventClassification, error) {
	var req events.TestEndEventRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		return ClassifyBuffer, fmt.Errorf("unmarshal test end: %w", err)
	}

	if req.TestCase == nil {
		return ClassifyBuffer, fmt.Errorf("test end missing test_case")
	}

	testID := req.TestCase.Id

	c.logger.Debug("test end - test existence unknown - buffer",
		"test_id", testID)
	return ClassifyBuffer, nil
}

// classifyStepBegin checks if a step begin event can be processed immediately
func (c *Classifier) classifyStepBegin(ctx context.Context, event publisher.Event) (EventClassification, error) {
	var req events.StepBeginEventRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		return ClassifyBuffer, fmt.Errorf("unmarshal step begin: %w", err)
	}

	if req.Step == nil {
		return ClassifyBuffer, fmt.Errorf("step begin missing step")
	}

	// Extract parent test ID
	testID := req.Step.TestCaseId
	if testID == "" {
		c.logger.Debug("step begin - no parent test - buffer",
			"step_id", req.Step.Id)
		return ClassifyBuffer, nil
	}

	c.logger.Debug("step begin - parent test existence unknown - buffer",
		"step_id", req.Step.Id,
		"test_id", testID)
	return ClassifyBuffer, nil
}

// classifyStepEnd checks if a step end event can be processed immediately
func (c *Classifier) classifyStepEnd(ctx context.Context, event publisher.Event) (EventClassification, error) {
	var req events.StepEndEventRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		return ClassifyBuffer, fmt.Errorf("unmarshal step end: %w", err)
	}

	if req.Step == nil {
		return ClassifyBuffer, fmt.Errorf("step end missing step")
	}

	stepID := req.Step.Id

	c.logger.Debug("step end - step existence unknown - buffer",
		"step_id", stepID)
	return ClassifyBuffer, nil
}

// classifyRunStart checks if a run start event can be processed immediately
func (c *Classifier) classifyRunStart(ctx context.Context, event publisher.Event) (EventClassification, error) {
	var req events.ReportRunStartEventRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		return ClassifyBuffer, fmt.Errorf("unmarshal run start: %w", err)
	}

	if req.RunId == "" {
		return ClassifyBuffer, fmt.Errorf("run start missing run")
	}

	// Run start events are always immediate as they create new test runs
	c.logger.Debug("run start - immediate",
		"run_id", req.RunId)
	return ClassifyImmediate, nil
}

// classifyRunEnd checks if a run end event can be processed immediately
func (c *Classifier) classifyRunEnd(ctx context.Context, event publisher.Event) (EventClassification, error) {
	var req events.TestRunEndEventRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		return ClassifyBuffer, fmt.Errorf("unmarshal run end: %w", err)
	}

	if req.RunId == "" {
		return ClassifyBuffer, fmt.Errorf("run end missing run")
	}

	runID := req.RunId
	c.logger.Debug("run end - run existence unknown - buffer",
		"run_id", runID)
	return ClassifyBuffer, nil
}
