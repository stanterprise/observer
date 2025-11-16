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
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NATSConsumer wraps a NATS JetStream consumer for processing test events
type NATSConsumer struct {
	nc       *nats.Conn
	js       jetstream.JetStream
	logger   *slog.Logger
	db       *gorm.DB
	stream   string
	consumer jetstream.Consumer
}

// NATSConsumerConfig holds configuration for NATS consumer
type NATSConsumerConfig struct {
	URL          string
	StreamName   string
	ConsumerName string
	// BatchSize is the number of messages to fetch at once (default: 10)
	BatchSize int
	// MaxWait is the maximum time to wait for messages (default: 5s)
	MaxWait time.Duration
}

// NewNATSConsumer creates a new NATS JetStream consumer
// If logger is nil, a no-op logger is used
func NewNATSConsumer(cfg NATSConsumerConfig, logger *slog.Logger, db *gorm.DB) (*NATSConsumer, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("NATS URL is required")
	}

	if db == nil {
		return nil, fmt.Errorf("database connection is required for processor")
	}

	if cfg.StreamName == "" {
		cfg.StreamName = publisher.DefaultStreamName
	}

	if cfg.ConsumerName == "" {
		cfg.ConsumerName = "processor"
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}

	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 5 * time.Second
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.URL, nats.Name("observer-processor"))
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}

	c := &NATSConsumer{
		nc:     nc,
		js:     js,
		logger: logger,
		db:     db,
		stream: cfg.StreamName,
	}

	// Ensure consumer exists
	consumer, err := c.ensureConsumer(context.Background(), cfg.ConsumerName)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure consumer: %w", err)
	}
	c.consumer = consumer

	logger.Info("NATS consumer initialized",
		"url", cfg.URL,
		"stream", cfg.StreamName,
		"consumer", cfg.ConsumerName,
		"batch_size", cfg.BatchSize,
		"max_wait", cfg.MaxWait)

	return c, nil
}

// ensureConsumer creates the JetStream consumer if it doesn't exist
func (c *NATSConsumer) ensureConsumer(ctx context.Context, consumerName string) (jetstream.Consumer, error) {
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
		MaxDeliver:    5, // Retry up to 5 times
		AckWait:       30 * time.Second,
		Description:   "Test event processor consumer",
	}

	consumer, err = c.js.CreateOrUpdateConsumer(ctx, c.stream, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}

	c.logger.Info("consumer created", "consumer", consumerName)
	return consumer, nil
}

// Start begins consuming messages from NATS
// This function blocks until the context is cancelled
func (c *NATSConsumer) Start(ctx context.Context, cfg NATSConsumerConfig) error {
	c.logger.Info("starting consumer")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopped by context")
			return ctx.Err()
		default:
			// Fetch batch of messages
			msgs, err := c.consumer.Fetch(cfg.BatchSize, jetstream.FetchMaxWait(cfg.MaxWait))
			if err != nil {
				// Check if it's a timeout (no messages available)
				if errors.Is(err, nats.ErrTimeout) || errors.Is(err, jetstream.ErrNoMessages) {
					// Normal case - no messages available, continue
					continue
				}
				c.logger.Error("fetch messages failed", "error", err)
				// Brief sleep before retry to avoid tight loop on persistent errors
				time.Sleep(1 * time.Second)
				continue
			}

			// Process each message
			for msg := range msgs.Messages() {
				if err := c.processMessage(ctx, msg); err != nil {
					c.logger.Error("process message failed",
						"subject", msg.Subject(),
						"error", err)
					// Negative acknowledge - will be redelivered
					if nakErr := msg.Nak(); nakErr != nil {
						c.logger.Error("failed to nak message", "error", nakErr)
					}
				} else {
					// Acknowledge successful processing
					if ackErr := msg.Ack(); ackErr != nil {
						c.logger.Error("failed to ack message", "error", ackErr)
					}
				}
			}
		}
	}
}

// processMessage handles a single NATS message
func (c *NATSConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
	// Parse the event wrapper
	var event publisher.Event
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	c.logger.Debug("processing event",
		"type", event.Type,
		"subject", msg.Subject(),
		"timestamp", event.Timestamp)

	// Route to appropriate handler based on event type
	switch event.Type {
	case publisher.EventTypeTestBegin:
		return c.handleTestBegin(ctx, event.Data)
	case publisher.EventTypeTestEnd:
		return c.handleTestEnd(ctx, event.Data)
	case publisher.EventTypeStepBegin:
		return c.handleStepBegin(ctx, event.Data)
	case publisher.EventTypeStepEnd:
		return c.handleStepEnd(ctx, event.Data)
	default:
		c.logger.Warn("unknown event type", "type", event.Type)
		// Acknowledge unknown events to prevent redelivery
		return nil
	}
}

// handleTestBegin processes a test begin event
func (c *NATSConsumer) handleTestBegin(ctx context.Context, data json.RawMessage) error {
	var req events.TestBeginEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test begin event: %w", err)
	}

	if req.TestCase == nil {
		return errors.New("test_case is nil")
	}

	c.logger.Info("test start",
		"run_id", req.TestCase.RunId,
		"title", req.TestCase.Title,
		"metadata_count", len(req.TestCase.Metadata))

	// Convert metadata map[string]string to datatypes.JSONMap (map[string]any)
	md := map[string]any{}
	for k, v := range req.TestCase.Metadata {
		md[k] = v
	}

	tc := &m.TestCaseRun{
		RunID:    req.TestCase.RunId,
		Title:    req.TestCase.Title,
		Metadata: md,
		ID:       req.TestCase.Id,
	}

	// Upsert to database
	if err := c.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "metadata", "updated_at"}),
	}).Create(tc).Error; err != nil {
		return fmt.Errorf("persist test start: %w", err)
	}

	return nil
}

// handleTestEnd processes a test end event
func (c *NATSConsumer) handleTestEnd(ctx context.Context, data json.RawMessage) error {
	var req events.TestEndEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal test end event: %w", err)
	}

	if req.TestCase == nil {
		return errors.New("test_case is nil")
	}

	c.logger.Info("test finish",
		"run_id", req.TestCase.RunId,
		"status", req.TestCase.Status)

	statusStr := req.TestCase.Status.String()
	tc := &m.TestCaseRun{
		ID:     req.TestCase.Id,
		RunID:  req.TestCase.RunId,
		Status: statusStr,
	}

	// Upsert status on finish
	if err := c.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
	}).Create(tc).Error; err != nil {
		return fmt.Errorf("persist test end: %w", err)
	}

	return nil
}

// handleStepBegin processes a step begin event
func (c *NATSConsumer) handleStepBegin(ctx context.Context, data json.RawMessage) error {
	var req events.StepBeginEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal step begin event: %w", err)
	}

	if req.Step == nil {
		return errors.New("step is nil")
	}

	c.logger.Info("test step", "run_id", req.Step.TestCaseRunId)

	st := &m.StepRun{
		TestCaseRunID: req.Step.TestCaseRunId,
		Status:        "RUNNING",
	}

	if err := c.db.WithContext(ctx).Create(st).Error; err != nil {
		return fmt.Errorf("persist step begin: %w", err)
	}

	return nil
}

// handleStepEnd processes a step end event
func (c *NATSConsumer) handleStepEnd(ctx context.Context, data json.RawMessage) error {
	var req events.StepEndEventRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal step end event: %w", err)
	}

	if req.Step == nil {
		return errors.New("step is nil")
	}

	c.logger.Info("test step end",
		"run_id", req.Step.TestCaseRunId,
		"status", req.Step.Status)

	// Use transaction to ensure atomic read+update
	err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var step m.StepRun
		// Lock the latest step row for this test case
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("test_case_run_id = ?", req.Step.TestCaseRunId).
			Order("created_at DESC").
			Limit(1).Take(&step)
		if q.Error != nil {
			if errors.Is(q.Error, gorm.ErrRecordNotFound) {
				// No step row exists; create one inside the tx
				st := &m.StepRun{
					TestCaseRunID: req.Step.TestCaseRunId,
					Status:        statusToString(req.Step.Status),
				}
				if err := tx.Create(st).Error; err != nil {
					return err
				}
				return nil
			}
			return q.Error
		}
		// Update the locked row
		if err := tx.Model(&m.StepRun{}).
			Where("id = ?", step.ID).
			Update("status", statusToString(req.Step.Status)).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("persist step end: %w", err)
	}

	return nil
}

// statusToString converts protobuf status to string
func statusToString(status common.TestStatus) string {
	return status.String()
}

// Close closes the NATS connection
func (c *NATSConsumer) Close() error {
	if c.nc != nil {
		c.nc.Close()
		c.logger.Info("NATS consumer closed")
	}
	return nil
}

// noopWriter implements io.Writer but drops logs when no logger provided
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }
