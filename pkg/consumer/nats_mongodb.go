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

// MongoNATSConsumer wraps a NATS JetStream consumer for processing test events
// and persisting them to MongoDB using a document-based data model
type MongoNATSConsumer struct {
	nc         *nats.Conn
	js         jetstream.JetStream
	logger     *slog.Logger
	repo       *repository.MongoRepository
	stream     string
	consumer   jetstream.Consumer
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

	suite := &m.SuiteDocument{
		ID:              req.Suite.Id,
		Name:            req.Suite.Name,
		Description:     req.Suite.Description,
		Metadata:        md,
		TestSuiteSpecID: "",
		InitiatedBy:     req.Suite.InitiatedBy,
		ProjectName:     req.Suite.Project,
		StartTime:       startTime,
	}

	// Extract parent suite ID from metadata if present
	// For non-root suites, the parent ID should be passed via metadata["parent_suite_id"]
	parentSuiteID := ""
	if parentID, ok := req.Suite.Metadata["parent_suite_id"]; ok && parentID != "" {
		parentSuiteID = parentID
	}

	return c.repo.UpsertSuiteBegin(ctx, suite, parentSuiteID)
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

	return c.repo.UpsertSuiteEnd(ctx, req.Suite.Id, req.Suite.Status.String(), endTime, duration)
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

	test := &m.TestDocument{
		ID:         req.TestCase.Id,
		RunID:      req.TestCase.RunId,
		Title:      req.TestCase.Name,
		Metadata:   md,
		RetryCount: ptrInt32(req.TestCase.RetryCount),
		RetryIndex: ptrInt32(req.TestCase.RetryIndex),
		Timeout:    ptrInt32(req.TestCase.Timeout),
	}

	if req.TestCase.Duration != nil {
		nanos := req.TestCase.Duration.AsDuration().Nanoseconds()
		test.Duration = &nanos
	}

	// Use TestSuiteRunId or RunID as the suite ID - tests belong to a run/suite
	suiteID := req.TestCase.RunId
	if req.TestCase.TestSuiteRunId != "" {
		suiteID = req.TestCase.TestSuiteRunId
	}

	return c.repo.UpsertTestBegin(ctx, test, suiteID)
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

	return c.repo.UpsertTestEnd(ctx, req.TestCase.Id, req.TestCase.Status.String(), duration)
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

	c.logger.Info("step start",
		"id", req.Step.Id,
		"test_id", req.Step.TestCaseRunId)

	step := &m.StepDocument{
		ID:            req.Step.Id,
		RunID:         req.Step.RunId,
		TestCaseRunID: req.Step.TestCaseRunId,
		ParentStepID:  req.Step.ParentStepId,
		Status:        "RUNNING",
		Category:      req.Step.Category,
		Title:         req.Step.Title,
	}

	return c.repo.UpsertStepBegin(ctx, step, req.Step.TestCaseRunId, req.Step.ParentStepId)
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

	return c.repo.UpsertStepEnd(ctx, req.Step.Id, mongoStatusToString(req.Step.Status))
}

// handleTestFailure processes a test failure event
func (c *MongoNATSConsumer) handleTestFailure(ctx context.Context, data json.RawMessage) error {
	var req events.TestFailureEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test failure event: %w", err)
	}

	c.logger.Info("test failure",
		"test_id", req.TestId,
		"message_len", len(req.FailureMessage))

	// Update test status to FAILED
	return c.repo.UpdateTestStatus(ctx, req.TestId, "FAILED")
}

// handleTestError processes a test error event
func (c *MongoNATSConsumer) handleTestError(ctx context.Context, data json.RawMessage) error {
	var req events.TestErrorEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test error event: %w", err)
	}

	c.logger.Info("test error",
		"test_id", req.TestId,
		"message_len", len(req.ErrorMessage))

	// Update test status to ERROR
	return c.repo.UpdateTestStatus(ctx, req.TestId, "ERROR")
}

// handleStdOutput processes a stdout event
func (c *MongoNATSConsumer) handleStdOutput(ctx context.Context, data json.RawMessage) error {
	var req events.StdOutputEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal stdout event: %w", err)
	}

	c.logger.Debug("stdout",
		"test_id", req.TestId,
		"message_len", len(req.Message))

	return nil
}

// handleStdError processes a stderr event
func (c *MongoNATSConsumer) handleStdError(ctx context.Context, data json.RawMessage) error {
	var req events.StdErrorEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal stderr event: %w", err)
	}

	c.logger.Debug("stderr",
		"test_id", req.TestId,
		"message_len", len(req.Message))

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
