package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/stanterprise/observer/internal/repository"
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
	repo   *repository.MongoRepository
	logger *slog.Logger
}

// NewClassifier creates a new event classifier
func NewClassifier(repo *repository.MongoRepository, logger *slog.Logger) *Classifier {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &Classifier{
		repo:   repo,
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

	// Non-root suite: check if parent exists
	exists, err := c.repo.SuiteExists(ctx, parentSuiteID)
	if err != nil {
		c.logger.Error("failed to check parent suite existence",
			"suite_id", req.Suite.Id,
			"parent_suite_id", parentSuiteID,
			"error", err)
		return ClassifyBuffer, fmt.Errorf("check parent suite: %w", err)
	}

	if exists {
		c.logger.Debug("nested suite begin - parent exists - immediate",
			"suite_id", req.Suite.Id,
			"parent_suite_id", parentSuiteID)
		return ClassifyImmediate, nil
	}

	c.logger.Debug("nested suite begin - parent missing - buffer",
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

	// Check if this is a root suite by checking if a document with this ID exists
	// Root suite ID is used as the document _id
	exists, err := c.repo.SuiteExists(ctx, suiteID)
	if err != nil {
		c.logger.Error("failed to check suite existence",
			"suite_id", suiteID,
			"error", err)
		return ClassifyImmediate, fmt.Errorf("check suite: %w", err)
	}

	// If suite exists as a document (root suite), this triggers reconciliation
	// We check by attempting to get the run document with this ID
	run, err := c.repo.GetTestRun(ctx, suiteID)
	if err != nil {
		return ClassifyImmediate, fmt.Errorf("get test run: %w", err)
	}

	if run != nil && run.ID == suiteID {
		c.logger.Debug("root suite end - trigger reconciliation",
			"suite_id", suiteID)
		return ClassifyReconcile, nil
	}

	// Non-root suite end can be processed immediately if suite exists
	if exists {
		c.logger.Debug("nested suite end - immediate",
			"suite_id", suiteID)
		return ClassifyImmediate, nil
	}

	c.logger.Debug("suite end - suite missing - buffer",
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

	// Check if parent suite exists
	exists, err := c.repo.SuiteExists(ctx, parentSuiteID)
	if err != nil {
		c.logger.Error("failed to check parent suite existence",
			"test_id", req.TestCase.Id,
			"parent_suite_id", parentSuiteID,
			"error", err)
		return ClassifyBuffer, fmt.Errorf("check parent suite: %w", err)
	}

	if exists {
		c.logger.Debug("test begin - parent suite exists - immediate",
			"test_id", req.TestCase.Id,
			"parent_suite_id", parentSuiteID)
		return ClassifyImmediate, nil
	}

	c.logger.Debug("test begin - parent suite missing - buffer",
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

	// Check if test exists
	exists, err := c.repo.TestExists(ctx, testID)
	if err != nil {
		c.logger.Error("failed to check test existence",
			"test_id", testID,
			"error", err)
		return ClassifyBuffer, fmt.Errorf("check test: %w", err)
	}

	if exists {
		c.logger.Debug("test end - test exists - immediate",
			"test_id", testID)
		return ClassifyImmediate, nil
	}

	c.logger.Debug("test end - test missing - buffer",
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
	testID := req.Step.TestCaseRunId
	if testID == "" {
		c.logger.Debug("step begin - no parent test - buffer",
			"step_id", req.Step.Id)
		return ClassifyBuffer, nil
	}

	// Check if parent test exists
	exists, err := c.repo.TestExists(ctx, testID)
	if err != nil {
		c.logger.Error("failed to check parent test existence",
			"step_id", req.Step.Id,
			"test_id", testID,
			"error", err)
		return ClassifyBuffer, fmt.Errorf("check parent test: %w", err)
	}

	if exists {
		c.logger.Debug("step begin - parent test exists - immediate",
			"step_id", req.Step.Id,
			"test_id", testID)
		return ClassifyImmediate, nil
	}

	c.logger.Debug("step begin - parent test missing - buffer",
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

	// Check if step exists
	exists, err := c.repo.StepExists(ctx, stepID)
	if err != nil {
		c.logger.Error("failed to check step existence",
			"step_id", stepID,
			"error", err)
		return ClassifyBuffer, fmt.Errorf("check step: %w", err)
	}

	if exists {
		c.logger.Debug("step end - step exists - immediate",
			"step_id", stepID)
		return ClassifyImmediate, nil
	}

	c.logger.Debug("step end - step missing - buffer",
		"step_id", stepID)
	return ClassifyBuffer, nil
}
