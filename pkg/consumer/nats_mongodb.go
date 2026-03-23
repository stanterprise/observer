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
	"github.com/stanterprise/observer/pkg/storage"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/protobuf/encoding/protojson"
)

// noopWriter implements io.Writer but drops logs when no logger provided.
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// ptrInt32 returns a pointer to the given int32 value
func ptrInt32(v int32) *int32 {
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
	nc             *nats.Conn
	js             jetstream.JetStream
	logger         *slog.Logger
	repo           *repository.MongoRepository
	rawMessageRepo *repository.RawMessageRepository
	storageDriver  storage.Driver
	stream         string
	consumer       jetstream.Consumer
}

// MongoNATSConsumerConfig holds configuration for MongoDB NATS consumer
type MongoNATSConsumerConfig struct {
	URL            string
	StreamName     string
	ConsumerName   string
	BatchSize      int
	MaxWait        time.Duration
	// RetainMessages enables storing every raw NATS message payload in a dedicated
	// MongoDB collection ("raw_messages") for auditing and debugging purposes.
	// Set via the RETAIN_MESSAGES environment variable.
	RetainMessages bool
}

// NewMongoNATSConsumer creates a new NATS JetStream consumer with MongoDB backend.
// If rawMessageRepo is non-nil and cfg.RetainMessages is true, every received message
// will be persisted to the raw_messages collection before processing.
func NewMongoNATSConsumer(cfg MongoNATSConsumerConfig, logger *slog.Logger, repo *repository.MongoRepository, rawMessageRepo *repository.RawMessageRepository) (*MongoNATSConsumer, error) {
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

	// Initialize storage driver (optional)
	storageDriver, err := storage.NewDriverFromEnv(logger)
	if err != nil {
		return nil, fmt.Errorf("initialize storage driver: %w", err)
	}
	if storageDriver != nil {
		logger.Info("storage driver initialized", "driver", storageDriver.Name())
	} else {
		logger.Info("storage driver not configured; using inline attachment storage")
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.URL, nats.Name("observer-mongo-processor"))
	if err != nil {
		if storageDriver != nil {
			storageDriver.Close()
		}
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		if storageDriver != nil {
			storageDriver.Close()
		}
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}

	c := &MongoNATSConsumer{
		nc:             nc,
		js:             js,
		logger:         logger,
		repo:           repo,
		rawMessageRepo: rawMessageRepo,
		storageDriver:  storageDriver,
		stream:         cfg.StreamName,
	}

	// Ensure consumer exists
	consumer, err := c.ensureConsumer(context.Background(), cfg.ConsumerName)
	if err != nil {
		nc.Close()
		if storageDriver != nil {
			storageDriver.Close()
		}
		return nil, fmt.Errorf("ensure consumer: %w", err)
	}
	c.consumer = consumer

	retainMsg := cfg.RetainMessages && rawMessageRepo != nil
	logger.Info("MongoDB NATS consumer initialized",
		"url", cfg.URL,
		"stream", cfg.StreamName,
		"consumer", cfg.ConsumerName,
		"batch_size", cfg.BatchSize,
		"max_wait", cfg.MaxWait,
		"retain_messages", retainMsg)

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

	// Persist the raw message when retention is enabled.
	if c.rawMessageRepo != nil {
		if err := c.retainRawMessage(ctx, msg, event); err != nil {
			// Log but do not fail processing – retention is best-effort.
			c.logger.Warn("failed to retain raw message",
				"subject", msg.Subject(),
				"event_type", event.Type,
				"error", err)
		}
	}

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
	case publisher.EventTypeRunStart:
		return c.handleRunStart(ctx, event.Data)
	default:
		c.logger.Warn("unknown event type", "type", event.Type)
		return nil
	}
}

// handleHeartbeat processes a heartbeat event
func (c *MongoNATSConsumer) handleHeartbeat(ctx context.Context, data json.RawMessage) error {
	var req events.HeartbeatEventRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("unmarshal heartbeat event: %w", err)
	}

	c.logger.Debug("heartbeat", "source_id", req.SourceId)
	return nil
}

// retainRawMessage persists the raw NATS message to the raw_messages collection.
func (c *MongoNATSConsumer) retainRawMessage(ctx context.Context, msg jetstream.Msg, event publisher.Event) error {
	doc := &m.RawMessageDocument{
		Subject:    msg.Subject(),
		EventType:  string(event.Type),
		Payload:    msg.Data(),
		Stream:     c.stream,
		ReceivedAt: time.Now(),
	}

	// Attach JetStream sequence number when available.
	if meta, err := msg.Metadata(); err == nil {
		doc.Sequence = meta.Sequence.Stream
	}

	return c.rawMessageRepo.Insert(ctx, doc)
}

// mongoStatusToString converts protobuf status to string
func mongoStatusToString(status common.TestStatus) string {
	return status.String()
}

// Close closes the NATS connection
func (c *MongoNATSConsumer) Close() error {
	if c.storageDriver != nil {
		if err := c.storageDriver.Close(); err != nil {
			c.logger.Warn("failed to close storage driver", "error", err)
		}
	}
	if c.nc != nil {
		c.nc.Close()
		c.logger.Info("MongoDB NATS consumer closed")
	}
	return nil
}
