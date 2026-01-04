package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
)

// noopWriter implements io.Writer but drops logs when no logger provided.
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// ptrInt32 returns a pointer to the given int32 value
func ptrInt32(v int32) *int32 {
	if v == 0 {
		return nil
	}
	return &v
}

// extractTestID extracts the test ID from a test case run ID.
// TestCaseRunId format is typically: {runId}-{testId}
// This function strips the runId prefix to get just the testId.
func extractTestID(testCaseRunID, runID string) string {
	// Otherwise, return as-is (backward compatibility)
	return testCaseRunID
}

// MongoNATSConsumer wraps a NATS JetStream consumer for processing test events
// and persisting them to MongoDB using a document-based data model.
//
// Event Processing Flow:
// - Suite Begin (root): Creates a new TestRunDocument if it doesn't exist
// - Suite Begin (nested): Finds parent suite in root document and upserts the nested suite
// - Test Begin: Finds parent suite in root document and upserts the test
// - Step Begin: Finds parent test in root document and upserts the step
// - End Events: Find and update existing entities within the root document
//
// All operations follow an upsert pattern: update if entity exists, insert if not found.
type MongoNATSConsumer struct {
	nc       *nats.Conn
	js       jetstream.JetStream
	logger   *slog.Logger
	repo     *repository.MongoRepository
	stream   string
	consumer jetstream.Consumer
}

// MongoNATSConsumerConfig holds configuration for MongoDB NATS consumer
type MongoNATSConsumerConfig struct {
	URL          string
	StreamName   string
	ConsumerName string
	BatchSize    int
	MaxWait      time.Duration
}

// NewMongoNATSConsumer creates a new NATS JetStream consumer with MongoDB backend
func NewMongoNATSConsumer(cfg MongoNATSConsumerConfig, logger *slog.Logger, repo *repository.MongoRepository) (*MongoNATSConsumer, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("NATS URL is required")
	}

	if repo == nil {
		return nil, fmt.Errorf("MongoDB repository is required for processor")
	}

	if cfg.StreamName == "" {
		cfg.StreamName = publisher.DefaultStreamName
	}

	if cfg.ConsumerName == "" {
		cfg.ConsumerName = "mongo-processor"
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}

	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 5 * time.Second
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.URL, nats.Name("observer-mongo-processor"))
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}

	c := &MongoNATSConsumer{
		nc:     nc,
		js:     js,
		logger: logger,
		repo:   repo,
		stream: cfg.StreamName,
	}

	// Ensure consumer exists
	consumer, err := c.ensureConsumer(context.Background(), cfg.ConsumerName)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure consumer: %w", err)
	}
	c.consumer = consumer

	logger.Info("MongoDB NATS consumer initialized",
		"url", cfg.URL,
		"stream", cfg.StreamName,
		"consumer", cfg.ConsumerName,
		"batch_size", cfg.BatchSize,
		"max_wait", cfg.MaxWait)

	return c, nil
}

// ensureConsumer creates the JetStream consumer if it doesn't exist
func (c *MongoNATSConsumer) ensureConsumer(ctx context.Context, consumerName string) (jetstream.Consumer, error) {
	// Check if stream exists first
	_, err := c.js.Stream(ctx, c.stream)
	if err != nil {
		return nil, fmt.Errorf("stream %s not found: %w", c.stream, err)
	}

	// Try to get existing consumer
	consumer, err := c.js.Consumer(ctx, c.stream, consumerName)
	if err == nil {
		c.logger.Info("consumer already exists", "consumer", consumerName)
		return consumer, nil
	}

	// Create consumer with durable name and work queue policy
	consumerCfg := jetstream.ConsumerConfig{
		Durable:       consumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		Description:   "MongoDB test event processor consumer",
	}

	consumer, err = c.js.CreateOrUpdateConsumer(ctx, c.stream, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}

	c.logger.Info("consumer created", "consumer", consumerName)
	return consumer, nil
}

// Start begins consuming messages from NATS
func (c *MongoNATSConsumer) Start(ctx context.Context, cfg MongoNATSConsumerConfig) error {
	c.logger.Info("starting MongoDB consumer")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopped by context")
			return ctx.Err()
		default:
			msgs, err := c.consumer.Fetch(cfg.BatchSize, jetstream.FetchMaxWait(cfg.MaxWait))
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) || errors.Is(err, jetstream.ErrNoMessages) {
					continue
				}
				c.logger.Error("fetch messages failed", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for msg := range msgs.Messages() {
				if err := c.processMessage(ctx, msg); err != nil {
					c.logger.Error("process message failed",
						"subject", msg.Subject(),
						"error", err)
					if nakErr := msg.Nak(); nakErr != nil {
						c.logger.Error("failed to nak message", "error", nakErr)
					}
				} else {
					if ackErr := msg.Ack(); ackErr != nil {
						c.logger.Error("failed to ack message", "error", ackErr)
					}
				}
			}
		}
	}
}

// processMessage handles a single NATS message
func (c *MongoNATSConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
	var event publisher.Event
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	c.logger.Debug("processing event",
		"type", event.Type,
		"subject", msg.Subject(),
		"timestamp", event.Timestamp)

	switch event.Type {
	case publisher.EventTypeSuiteBegin:
		return c.handleSuiteBegin(ctx, event.Data)
	case publisher.EventTypeSuiteEnd:
		return c.handleSuiteEnd(ctx, event.Data)
	case publisher.EventTypeTestBegin:
		return c.handleTestBegin(ctx, event.Data)
	case publisher.EventTypeTestEnd:
		return c.handleTestEnd(ctx, event.Data)
	case publisher.EventTypeStepBegin:
		return c.handleStepBegin(ctx, event.Data)
	case publisher.EventTypeStepEnd:
		return c.handleStepEnd(ctx, event.Data)
	case publisher.EventTypeTestFailure:
		return c.handleTestFailure(ctx, event.Data)
	case publisher.EventTypeTestError:
		return c.handleTestError(ctx, event.Data)
	case publisher.EventTypeStdOutput:
		return c.handleStdOutput(ctx, event.Data)
	case publisher.EventTypeStdError:
		return c.handleStdError(ctx, event.Data)
	case publisher.EventTypeHeartbeat:
		return c.handleHeartbeat(ctx, event.Data)
	case publisher.EventTypeRunEnd:
		return c.handleRunEnd(ctx, event.Data)
	case publisher.MapSuitesEvent:
		return c.handleMapSuites(ctx, event.Data)
	default:
		c.logger.Warn("unknown event type", "type", event.Type)
		return nil
	}
}

// handleSuiteBegin processes a suite begin event
func (c *MongoNATSConsumer) handleSuiteBegin(ctx context.Context, data json.RawMessage) error {
	var req events.SuiteBeginEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
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
		StartTime:       startTime,
		EndTime:         endTime,
	}

	// Use ParentSuiteId directly from protobuf (already set in suite object)
	// For root suites: ParentSuiteId will be empty string
	// For nested suites: ParentSuiteId will be set to parent's ID
	runID := req.Suite.RunId

	return c.repo.UpsertSuiteBegin(ctx, runID, suite, suite.ParentSuiteID)
}

// handleSuiteEnd processes a suite end event
func (c *MongoNATSConsumer) handleSuiteEnd(ctx context.Context, data json.RawMessage) error {
	var req events.SuiteEndEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
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

// handleTestBegin processes a test begin event
func (c *MongoNATSConsumer) handleTestBegin(ctx context.Context, data json.RawMessage) error {
	var req events.TestBeginEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test begin event: %w", err)
	}

	if req.TestCase == nil {
		return errors.New("test_case is nil")
	}

	c.logger.Info("test start",
		"id", req.TestCase.Id,
		"run_id", req.TestCase.RunId,
		"title", req.TestCase.Name)

	// Convert metadata
	md := make(map[string]interface{})
	for k, v := range req.TestCase.Metadata {
		md[k] = v
	}

	var startTime *time.Time
	if req.TestCase.StartTime != nil {
		t := req.TestCase.StartTime.AsTime()
		startTime = &t
	}

	var endTime *time.Time
	if req.TestCase.EndTime != nil {
		t := req.TestCase.EndTime.AsTime()
		endTime = &t
	}

	// Convert attachments
	var attachments []map[string]interface{}
	for _, att := range req.TestCase.Attachments {
		attMap := make(map[string]interface{})
		attMap["name"] = att.Name
		attMap["mime_type"] = att.MimeType
		if content := att.GetContent(); len(content) > 0 {
			attMap["content"] = string(content)
		} else if att.GetUri() != "" {
			attMap["uri"] = att.GetUri()
		}
		attachments = append(attachments, attMap)
	}

	test := &m.TestDocument{
		ID:           req.TestCase.Id,
		Name:         req.TestCase.Name,
		Title:        req.TestCase.Name,
		Description:  req.TestCase.Description,
		RunID:        req.TestCase.RunId,
		Status:       req.TestCase.Status.String(),
		StartTime:    startTime,
		EndTime:      endTime,
		Metadata:     md,
		Tags:         req.TestCase.Tags,
		Location:     req.TestCase.Location,
		RetryCount:   ptrInt32(req.TestCase.RetryCount),
		RetryIndex:   ptrInt32(req.TestCase.RetryIndex),
		Timeout:      ptrInt32(req.TestCase.Timeout),
		Attachments:  attachments,
		ErrorMessage: req.TestCase.ErrorMessage,
		StackTrace:   req.TestCase.StackTrace,
		ErrorList:    req.TestCase.Errors,
	}

	if req.TestCase.Duration != nil {
		nanos := req.TestCase.Duration.AsDuration().Nanoseconds()
		test.Duration = &nanos
	}

	// runID identifies the document
	// TestSuiteRunId is the parent suite containing this test
	runID := req.TestCase.RunId
	suiteID := req.TestCase.TestSuiteId
	if suiteID == "" {
		// Fallback: if no TestSuiteRunId, use RunId as both
		suiteID = runID
	}

	return c.repo.UpsertTestBegin(ctx, runID, test, suiteID)
}

// handleTestEnd processes a test end event
func (c *MongoNATSConsumer) handleTestEnd(ctx context.Context, data json.RawMessage) error {
	var req events.TestEndEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test end event: %w", err)
	}

	if req.TestCase == nil {
		return errors.New("test_case is nil")
	}

	c.logger.Info("test finish",
		"id", req.TestCase.Id,
		"run_id", req.TestCase.RunId,
		"status", req.TestCase.Status)

	var duration *int64
	if req.TestCase.Duration != nil {
		nanos := req.TestCase.Duration.AsDuration().Nanoseconds()
		duration = &nanos
	}

	runID := req.TestCase.RunId
	return c.repo.UpsertTestEnd(ctx, runID, req.TestCase.Id, req.TestCase.Status.String(), duration)
}

// handleStepBegin processes a step begin event
func (c *MongoNATSConsumer) handleStepBegin(ctx context.Context, data json.RawMessage) error {
	var req events.StepBeginEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal step begin event: %w", err)
	}

	if req.Step == nil {
		return errors.New("step is nil")
	}

	// Extract the actual test ID from TestCaseRunId
	// TestCaseRunId format is typically: {runId}-{testId}
	// But tests are stored with just {testId}, so we need to extract it

	c.logger.Info("step start",
		"id", req.Step.Id,
		"test_case_run_id", req.Step.TestCaseId)

	// Convert metadata
	md := make(map[string]interface{})
	for k, v := range req.Step.Metadata {
		md[k] = v
	}

	var startTime *time.Time
	if req.Step.StartTime != nil {
		t := req.Step.StartTime.AsTime()
		startTime = &t
	}

	step := &m.StepDocument{
		ID:            req.Step.Id,
		RunID:         req.Step.RunId,
		TestCaseRunID: req.Step.TestCaseId,
		ParentStepID:  req.Step.ParentStepId,
		Title:         req.Step.Title,
		Description:   req.Step.Description,
		StartTime:     startTime,
		Type:          req.Step.Type,
		Metadata:      md,
		WorkerIndex:   req.Step.WorkerIndex,
		Status:        req.Step.Status.String(),
		Category:      req.Step.Category,
		Location:      req.Step.Location,
		Error:         req.Step.Error,
		Errors:        req.Step.Errors,
	}

	if req.Step.Duration != nil {
		nanos := req.Step.Duration.AsDuration().Nanoseconds()
		step.Duration = &nanos
	}

	runID := req.Step.RunId
	return c.repo.UpsertStepBegin(ctx, runID, step, req.Step.TestCaseId)
}

// handleStepEnd processes a step end event
func (c *MongoNATSConsumer) handleStepEnd(ctx context.Context, data json.RawMessage) error {
	var req events.StepEndEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal step end event: %w", err)
	}

	if req.Step == nil {
		return errors.New("step is nil")
	}

	c.logger.Info("step end",
		"id", req.Step.Id,
		"status", req.Step.Status)

	// Extract testID from TestCaseRunId (same as in handleStepBegin)
	testID := extractTestID(req.Step.TestCaseId, req.Step.RunId)
	runID := req.Step.RunId

	return c.repo.UpsertStepEnd(ctx, runID, req.Step.Id, testID, mongoStatusToString(req.Step.Status))
}

// handleTestFailure processes a test failure event
func (c *MongoNATSConsumer) handleTestFailure(ctx context.Context, data json.RawMessage) error {
	var req events.TestFailureEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test failure event: %w", err)
	}

	c.logger.Info("test failure",
		"run_id", req.RunId,
		"test_id", req.TestId,
		"message_len", len(req.FailureMessage))

	if req.RunId == "" {
		c.logger.Warn("test failure event missing run_id", "test_id", req.TestId)
		return nil
	}

	// Convert protobuf Timestamp to *time.Time
	var timestamp *time.Time
	if req.Timestamp != nil {
		t := req.Timestamp.AsTime()
		timestamp = &t
	}

	// Convert attachments to map slice
	attachments := make([]map[string]interface{}, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attMap := map[string]interface{}{
			"name":      att.Name,
			"mime_type": att.MimeType,
		}
		// Handle oneof payload
		if content := att.GetContent(); content != nil {
			attMap["content"] = content
		}
		if uri := att.GetUri(); uri != "" {
			attMap["uri"] = uri
		}
		attachments = append(attachments, attMap)
	}

	failure := m.TestFailureDocument{
		FailureMessage: req.FailureMessage,
		StackTrace:     req.StackTrace,
		Timestamp:      timestamp,
		Attachments:    attachments,
	}

	if err := c.repo.AppendTestFailure(ctx, req.RunId, req.TestId, failure); err != nil {
		return fmt.Errorf("append test failure: %w", err)
	}

	return nil
}

// handleTestError processes a test error event
func (c *MongoNATSConsumer) handleTestError(ctx context.Context, data json.RawMessage) error {
	var req events.TestErrorEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test error event: %w", err)
	}

	c.logger.Info("test error",
		"run_id", req.RunId,
		"test_id", req.TestId,
		"message_len", len(req.ErrorMessage))

	if req.RunId == "" {
		c.logger.Warn("test error event missing run_id", "test_id", req.TestId)
		return nil
	}

	// Convert protobuf Timestamp to *time.Time
	var timestamp *time.Time
	if req.Timestamp != nil {
		t := req.Timestamp.AsTime()
		timestamp = &t
	}

	// Convert attachments to map slice
	attachments := make([]map[string]interface{}, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attMap := map[string]interface{}{
			"name":      att.Name,
			"mime_type": att.MimeType,
		}
		// Handle oneof payload
		if content := att.GetContent(); content != nil {
			attMap["content"] = content
		}
		if uri := att.GetUri(); uri != "" {
			attMap["uri"] = uri
		}
		attachments = append(attachments, attMap)
	}

	errorDoc := m.TestErrorDocument{
		ErrorMessage: req.ErrorMessage,
		StackTrace:   req.StackTrace,
		Timestamp:    timestamp,
		Attachments:  attachments,
	}

	if err := c.repo.AppendTestError(ctx, req.RunId, req.TestId, errorDoc); err != nil {
		return fmt.Errorf("append test error: %w", err)
	}

	return nil
}

// handleStdOutput processes a stdout event
func (c *MongoNATSConsumer) handleStdOutput(ctx context.Context, data json.RawMessage) error {
	var req events.StdOutputEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
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

	// Convert protobuf Timestamp to *time.Time
	var timestamp *time.Time
	if req.Timestamp != nil {
		t := req.Timestamp.AsTime()
		timestamp = &t
	}

	output := m.OutputDocument{
		Message:   req.Message,
		Timestamp: timestamp,
	}

	if err := c.repo.AppendStdOutput(ctx, req.RunId, req.TestId, output); err != nil {
		return fmt.Errorf("append stdout: %w", err)
	}

	return nil
}

// handleStdError processes a stderr event
func (c *MongoNATSConsumer) handleStdError(ctx context.Context, data json.RawMessage) error {
	var req events.StdErrorEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
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

	// Convert protobuf Timestamp to *time.Time
	var timestamp *time.Time
	if req.Timestamp != nil {
		t := req.Timestamp.AsTime()
		timestamp = &t
	}

	output := m.OutputDocument{
		Message:   req.Message,
		Timestamp: timestamp,
	}

	if err := c.repo.AppendStdError(ctx, req.RunId, req.TestId, output); err != nil {
		return fmt.Errorf("append stderr: %w", err)
	}

	return nil
}

// handleHeartbeat processes a heartbeat event
func (c *MongoNATSConsumer) handleHeartbeat(ctx context.Context, data json.RawMessage) error {
	var req events.HeartbeatEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal heartbeat event: %w", err)
	}

	c.logger.Debug("heartbeat", "source_id", req.SourceId)
	return nil
}

// handleRunEnd processes a test run end event
func (c *MongoNATSConsumer) handleRunEnd(ctx context.Context, data json.RawMessage) error {
	var req events.TestRunEndEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal run end event: %w", err)
	}

	c.logger.Info("run end", "run_id", req.RunId, "status", req.FinalStatus)

	// Convert protobuf Timestamp to *time.Time
	var startTime *time.Time
	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		startTime = &t
	}

	// Convert protobuf Duration to *int64 (nanoseconds)
	var duration *int64
	if req.Duration != nil {
		d := req.Duration.AsDuration().Nanoseconds()
		duration = &d
	}

	// Update the test run document with final status, times, and duration
	if err := c.repo.UpdateTestRunEnd(ctx, req.RunId, req.FinalStatus.String(), startTime, duration); err != nil {
		return fmt.Errorf("update run end: %w", err)
	}

	return nil
}

func (c *MongoNATSConsumer) handleMapSuites(ctx context.Context, data json.RawMessage) error {
	var req events.ReportRunStartEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal map suites event: %w", err)
	}

	c.logger.Info("map suites",
		"run_id", req.RunId,
		"name", req.Name,
		"total_tests", req.TotalTests,
		"suite_count", len(req.TestSuites))

	// Convert run-level metadata
	runMetadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		runMetadata[k] = v
	}

	// Convert protobuf entities to SuiteDocument models
	suites := make([]m.SuiteDocument, 0, len(req.TestSuites))
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

		suite := m.SuiteDocument{
			ID:            protoSuite.Id,
			RunID:         protoSuite.RunId,
			ParentSuiteID: protoSuite.ParentSuiteId,
			Name:          protoSuite.Name,
			Description:   protoSuite.Description,
			Metadata:      md,
			Location:      protoSuite.Location,
			Type:          protoSuite.Type.String(),
			InitiatedBy:   protoSuite.InitiatedBy,
			ProjectName:   protoSuite.Project,
			Author:        protoSuite.Author,
			Owner:         protoSuite.Owner,
			TestCaseIds:   protoSuite.TestCaseIds,
			SubSuiteIds:   protoSuite.SubSuiteIds,
			StartTime:     startTime,
			EndTime:       endTime,
			Duration:      duration,
			Status:        protoSuite.Status.String(),
		}

		suites = append(suites, suite)
	}

	return c.repo.MapSuites(ctx, req.RunId, req.Name, runMetadata, req.TotalTests, suites)
}

// mongoStatusToString converts protobuf status to string
func mongoStatusToString(status common.TestStatus) string {
	return status.String()
}

// Close closes the NATS connection
func (c *MongoNATSConsumer) Close() error {
	if c.nc != nil {
		c.nc.Close()
		c.logger.Info("MongoDB NATS consumer closed")
	}
	return nil
}
