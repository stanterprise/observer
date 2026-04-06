package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
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
	prefix         string
	maxDeliver     int
	ackWait        time.Duration
	dlqSubject     string

	deferCfg   deferQueueConfig
	deferQueue map[string][]deferredStepEvent
	deferMu    sync.Mutex

	integrityMu      sync.Mutex
	runIntegrityByID map[string]*runIntegrity
}

// MongoNATSConsumerConfig holds configuration for MongoDB NATS consumer
type MongoNATSConsumerConfig struct {
	URL          string
	StreamName   string
	ConsumerName string
	BatchSize    int
	MaxWait      time.Duration
	MaxDeliver   int
	AckWait      time.Duration
	DLQSubject   string

	DeferQueueMaxAttempts int
	DeferQueueTTL         time.Duration
	// RetainMessages enables storing every raw NATS message payload in a dedicated
	// MongoDB collection ("raw_messages") for auditing and debugging purposes.
	// Set via the RETAIN_MESSAGES environment variable.
	RetainMessages bool
}

type deferredStepEvent struct {
	eventType   publisher.EventType
	data        json.RawMessage
	runID       string
	testID      string
	retryIndex  int32
	queuedAt    time.Time
	attempts    int
	lastFailure string
}

type deferQueueConfig struct {
	maxAttempts int
	ttl         time.Duration
}

type runIntegrity struct {
	RunID           string
	ExpectedTests   int32
	Received        int64
	Processed       int64
	Deferred        int64
	Replayed        int64
	Failed          int64
	DLQ             int64
	PendingDeferred int64
	StartedAt       time.Time
	LastUpdatedAt   time.Time
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

	if cfg.MaxDeliver <= 0 {
		cfg.MaxDeliver = 5
	}

	if cfg.AckWait <= 0 {
		cfg.AckWait = 30 * time.Second
	}

	if cfg.DLQSubject == "" {
		cfg.DLQSubject = publisher.DefaultSubjectPrefix + ".dlq"
	}

	if cfg.DeferQueueMaxAttempts <= 0 {
		cfg.DeferQueueMaxAttempts = cfg.MaxDeliver
	}

	if cfg.DeferQueueTTL <= 0 {
		cfg.DeferQueueTTL = 5 * time.Minute
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
		prefix:         publisher.DefaultSubjectPrefix,
		maxDeliver:     cfg.MaxDeliver,
		ackWait:        cfg.AckWait,
		dlqSubject:     cfg.DLQSubject,
		deferCfg: deferQueueConfig{
			maxAttempts: cfg.DeferQueueMaxAttempts,
			ttl:         cfg.DeferQueueTTL,
		},
		deferQueue:       make(map[string][]deferredStepEvent),
		runIntegrityByID: make(map[string]*runIntegrity),
	}

	// Ensure consumer exists
	consumer, err := c.ensureConsumer(context.Background(), cfg.ConsumerName, cfg.MaxDeliver, cfg.AckWait)
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
		"max_deliver", cfg.MaxDeliver,
		"ack_wait", cfg.AckWait,
		"dlq_subject", cfg.DLQSubject,
		"defer_queue_max_attempts", cfg.DeferQueueMaxAttempts,
		"defer_queue_ttl", cfg.DeferQueueTTL,
		"retain_messages", retainMsg)

	return c, nil
}

// ensureConsumer creates or updates the JetStream consumer, applying the given
// MaxDeliver and AckWait values even when the durable consumer already exists.
// Using CreateOrUpdateConsumer ensures that config changes (e.g. retry count,
// ack timeout) take effect at startup rather than being silently ignored when
// a durable consumer with the same name is already registered in the stream.
func (c *MongoNATSConsumer) ensureConsumer(ctx context.Context, consumerName string, maxDeliver int, ackWait time.Duration) (jetstream.Consumer, error) {
	// Check if stream exists first
	_, err := c.js.Stream(ctx, c.stream)
	if err != nil {
		return nil, fmt.Errorf("stream %s not found: %w", c.stream, err)
	}

	// Always create-or-update so that MaxDeliver/AckWait changes take effect
	// even when the durable consumer already exists.
	consumerCfg := jetstream.ConsumerConfig{
		Durable:       consumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		MaxDeliver:    maxDeliver,
		AckWait:       ackWait,
		Description:   "MongoDB test event processor consumer",
	}

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, c.stream, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("create or update consumer: %w", err)
	}

	c.logger.Info("consumer ready", "consumer", consumerName, "max_deliver", maxDeliver, "ack_wait", ackWait)
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
				event, err := c.processMessage(ctx, msg)
				if err != nil {
					runID := extractRunID(event.Data)
					c.incrementIntegrity(runID, func(ri *runIntegrity) {
						ri.Failed++
					})
					c.logger.Error("process message failed",
						"subject", msg.Subject(),
						"type", event.Type,
						"error", err)

					meta, metaErr := msg.Metadata()
					numDelivered := uint64(1)
					if metaErr == nil {
						numDelivered = meta.NumDelivered
					}

					if int(numDelivered) >= c.maxDeliver {
						if dlqErr := c.publishDLQ(ctx, msg, event, err, numDelivered); dlqErr != nil {
							c.logger.Error("failed to publish DLQ message", "error", dlqErr)
						}
						c.incrementIntegrity(runID, func(ri *runIntegrity) {
							ri.DLQ++
						})
						if ackErr := msg.Ack(); ackErr != nil {
							c.logger.Error("failed to ack message after DLQ", "error", ackErr)
						}
						continue
					}

					if nakErr := msg.Nak(); nakErr != nil {
						c.logger.Error("failed to nak message", "error", nakErr)
					}
				} else {
					runID := extractRunID(event.Data)
					c.incrementIntegrity(runID, func(ri *runIntegrity) {
						ri.Processed++
					})
					if ackErr := msg.Ack(); ackErr != nil {
						c.logger.Error("failed to ack message", "error", ackErr)
					}
				}
			}
		}
	}
}

// processMessage handles a single NATS message
func (c *MongoNATSConsumer) processMessage(ctx context.Context, msg jetstream.Msg) (publisher.Event, error) {
	var event publisher.Event
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return event, fmt.Errorf("unmarshal event: %w", err)
	}

	runID := extractRunID(event.Data)
	c.incrementIntegrity(runID, func(ri *runIntegrity) {
		ri.Received++
	})

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
		return event, c.handleSuiteBegin(ctx, event.Data)
	case publisher.EventTypeSuiteEnd:
		return event, c.handleSuiteEnd(ctx, event.Data)
	case publisher.EventTypeTestBegin:
		return event, c.handleTestBegin(ctx, event.Data)
	case publisher.EventTypeTestEnd:
		return event, c.handleTestEnd(ctx, event.Data)
	case publisher.EventTypeStepBegin:
		err := c.handleStepBegin(ctx, event.Data)
		if err != nil && c.shouldDeferStepEvent(err) {
			if deferErr := c.deferStepEvent(event, err); deferErr == nil {
				c.incrementIntegrity(runID, func(ri *runIntegrity) {
					ri.Deferred++
				})
				return event, nil
			}
		}
		return event, err
	case publisher.EventTypeStepEnd:
		err := c.handleStepEnd(ctx, event.Data)
		if err != nil && c.shouldDeferStepEvent(err) {
			if deferErr := c.deferStepEvent(event, err); deferErr == nil {
				c.incrementIntegrity(runID, func(ri *runIntegrity) {
					ri.Deferred++
				})
				return event, nil
			}
		}
		return event, err
	case publisher.EventTypeTestFailure:
		return event, c.handleTestFailure(ctx, event.Data)
	case publisher.EventTypeTestError:
		return event, c.handleTestError(ctx, event.Data)
	case publisher.EventTypeStdOutput:
		return event, c.handleStdOutput(ctx, event.Data)
	case publisher.EventTypeStdError:
		return event, c.handleStdError(ctx, event.Data)
	case publisher.EventTypeHeartbeat:
		return event, c.handleHeartbeat(ctx, event.Data)
	case publisher.EventTypeRunEnd:
		return event, c.handleRunEnd(ctx, event.Data)
	case publisher.EventTypeRunStart:
		return event, c.handleRunStart(ctx, event.Data)
	default:
		c.logger.Warn("unknown event type", "type", event.Type)
		return event, nil
	}
}

func (c *MongoNATSConsumer) shouldDeferStepEvent(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "parent test not found") || strings.Contains(errMsg, "step not found")
}

func (c *MongoNATSConsumer) deferStepEvent(event publisher.Event, cause error) error {
	runID, testID, retryIndex, err := extractStepIdentity(event)
	if err != nil {
		return err
	}

	key := deferredQueueKey(runID, testID, retryIndex)
	c.deferMu.Lock()
	deferred := deferredStepEvent{
		eventType:   event.Type,
		data:        event.Data,
		runID:       runID,
		testID:      testID,
		retryIndex:  retryIndex,
		queuedAt:    time.Now(),
		attempts:    1,
		lastFailure: cause.Error(),
	}
	c.deferQueue[key] = append(c.deferQueue[key], deferred)
	pendingCount := len(c.deferQueue[key])
	c.deferMu.Unlock()

	c.incrementIntegrity(runID, func(ri *runIntegrity) {
		ri.PendingDeferred++
	})

	c.logger.Warn("deferred orphan step event",
		"type", event.Type,
		"run_id", runID,
		"test_id", testID,
		"retry_index", retryIndex,
		"pending_for_key", pendingCount,
		"error", cause)

	return nil
}

func (c *MongoNATSConsumer) replayDeferredStepsForTest(ctx context.Context, runID, testID string, retryIndex int32) {
	key := deferredQueueKey(runID, testID, retryIndex)

	c.deferMu.Lock()
	events := c.deferQueue[key]
	delete(c.deferQueue, key)
	c.deferMu.Unlock()

	if len(events) == 0 {
		return
	}

	c.logger.Info("replaying deferred step events",
		"run_id", runID,
		"test_id", testID,
		"retry_index", retryIndex,
		"count", len(events))

	now := time.Now()
	requeue := make([]deferredStepEvent, 0)
	replayed := int64(0)

	for _, deferred := range events {
		if now.Sub(deferred.queuedAt) > c.deferCfg.ttl {
			c.logger.Warn("dropping deferred step event after TTL",
				"run_id", deferred.runID,
				"test_id", deferred.testID,
				"retry_index", deferred.retryIndex,
				"type", deferred.eventType,
				"ttl", c.deferCfg.ttl)
			continue
		}

		var err error
		switch deferred.eventType {
		case publisher.EventTypeStepBegin:
			err = c.handleStepBegin(ctx, deferred.data)
		case publisher.EventTypeStepEnd:
			err = c.handleStepEnd(ctx, deferred.data)
		default:
			continue
		}

		if err == nil {
			replayed++
			continue
		}

		deferred.attempts++
		deferred.lastFailure = err.Error()
		if deferred.attempts >= c.deferCfg.maxAttempts {
			c.logger.Error("deferred step event exceeded max attempts",
				"run_id", deferred.runID,
				"test_id", deferred.testID,
				"retry_index", deferred.retryIndex,
				"type", deferred.eventType,
				"attempts", deferred.attempts,
				"error", err)
			continue
		}

		requeue = append(requeue, deferred)
	}

	if len(requeue) > 0 {
		c.deferMu.Lock()
		c.deferQueue[key] = append(c.deferQueue[key], requeue...)
		c.deferMu.Unlock()
	}

	c.incrementIntegrity(runID, func(ri *runIntegrity) {
		ri.Replayed += replayed
		if len(events) > 0 {
			if ri.PendingDeferred >= int64(len(events)) {
				ri.PendingDeferred -= int64(len(events))
			} else {
				ri.PendingDeferred = 0
			}
		}
		ri.PendingDeferred += int64(len(requeue))
	})
}

func extractStepIdentity(event publisher.Event) (runID string, testID string, retryIndex int32, err error) {
	switch event.Type {
	case publisher.EventTypeStepBegin:
		var req events.StepBeginEventRequest
		if unmarshalErr := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(event.Data, &req); unmarshalErr != nil {
			return "", "", 0, fmt.Errorf("parse step begin for defer queue: %w", unmarshalErr)
		}
		if req.Step == nil {
			return "", "", 0, fmt.Errorf("parse step begin for defer queue: step is nil")
		}
		return req.Step.RunId, extractTestID(req.Step.TestCaseId, req.Step.RunId), req.Step.RetryIndex, nil
	case publisher.EventTypeStepEnd:
		var req events.StepEndEventRequest
		if unmarshalErr := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(event.Data, &req); unmarshalErr != nil {
			return "", "", 0, fmt.Errorf("parse step end for defer queue: %w", unmarshalErr)
		}
		if req.Step == nil {
			return "", "", 0, fmt.Errorf("parse step end for defer queue: step is nil")
		}
		return req.Step.RunId, extractTestID(req.Step.TestCaseId, req.Step.RunId), req.Step.RetryIndex, nil
	default:
		return "", "", 0, fmt.Errorf("not a step event: %s", event.Type)
	}
}

func deferredQueueKey(runID, testID string, retryIndex int32) string {
	return runID + "::" + testID + "::" + strconv.Itoa(int(retryIndex))
}

func (c *MongoNATSConsumer) incrementIntegrity(runID string, mutate func(ri *runIntegrity)) {
	if runID == "" {
		return
	}
	c.integrityMu.Lock()
	defer c.integrityMu.Unlock()
	ri, exists := c.runIntegrityByID[runID]
	if !exists {
		ri = &runIntegrity{RunID: runID, StartedAt: time.Now(), LastUpdatedAt: time.Now()}
		c.runIntegrityByID[runID] = ri
	}
	mutate(ri)
	ri.LastUpdatedAt = time.Now()
}

func (c *MongoNATSConsumer) markRunStart(runID string, expectedTests int32) {
	c.incrementIntegrity(runID, func(ri *runIntegrity) {
		if ri.StartedAt.IsZero() {
			ri.StartedAt = time.Now()
		}
		ri.ExpectedTests = expectedTests
	})
}

func (c *MongoNATSConsumer) emitRunCompletenessSummary(runID string, finalStatus string) {
	c.integrityMu.Lock()
	ri, exists := c.runIntegrityByID[runID]
	if !exists {
		c.integrityMu.Unlock()
		return
	}
	snapshot := *ri
	// Remove the entry so a long-running processor does not accumulate one
	// struct per completed run indefinitely.
	delete(c.runIntegrityByID, runID)
	c.integrityMu.Unlock()

	// Purge any leftover deferred-queue entries for this run.
	c.purgeDeferQueueForRun(runID)

	completeness := 0.0
	if snapshot.Received > 0 {
		completeness = float64(snapshot.Processed+snapshot.DLQ) / float64(snapshot.Received)
	}

	c.logger.Info("run completeness summary",
		"run_id", runID,
		"final_status", finalStatus,
		"expected_tests", snapshot.ExpectedTests,
		"received", snapshot.Received,
		"processed", snapshot.Processed,
		"deferred", snapshot.Deferred,
		"replayed", snapshot.Replayed,
		"failed", snapshot.Failed,
		"dlq", snapshot.DLQ,
		"pending_deferred", snapshot.PendingDeferred,
		"completeness_ratio", completeness,
		"started_at", snapshot.StartedAt,
		"last_updated_at", snapshot.LastUpdatedAt)
}

// purgeDeferQueueForRun removes all deferred step event entries whose runID
// matches the given run. This is called when a run ends to bound memory usage
// and prevent stale entries from accumulating in long-running processors.
func (c *MongoNATSConsumer) purgeDeferQueueForRun(runID string) {
	prefix := runID + "::"
	c.deferMu.Lock()
	for key := range c.deferQueue {
		if strings.HasPrefix(key, prefix) {
			delete(c.deferQueue, key)
		}
	}
	c.deferMu.Unlock()
}

func (c *MongoNATSConsumer) publishDLQ(ctx context.Context, msg jetstream.Msg, event publisher.Event, processingErr error, deliveries uint64) error {
	runID := extractRunID(event.Data)
	payload := map[string]interface{}{
		"reason":            processingErr.Error(),
		"run_id":            runID,
		"stream":            c.stream,
		"original_subject":  msg.Subject(),
		"num_delivered":     deliveries,
		"max_deliver":       c.maxDeliver,
		"failed_at":         time.Now().UTC(),
		"event_type":        event.Type,
		"original_envelope": json.RawMessage(msg.Data()),
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal dlq payload: %w", err)
	}

	if _, err := c.js.Publish(ctx, c.dlqSubject, b); err != nil {
		return fmt.Errorf("publish dlq payload: %w", err)
	}

	c.logger.Error("message sent to DLQ",
		"run_id", runID,
		"subject", msg.Subject(),
		"dlq_subject", c.dlqSubject,
		"num_delivered", deliveries,
		"error", processingErr)

	return nil
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

// retainRawMessage persists the raw NATS message into the per-run retention document.
// The run_id is extracted from the event data and used as the document identifier so
// that all messages belonging to the same run are stored in a single document.
func (c *MongoNATSConsumer) retainRawMessage(ctx context.Context, msg jetstream.Msg, event publisher.Event) error {
	runID := extractRunID(event.Data)
	if runID == "" {
		c.logger.Warn("could not extract run_id from message, skipping retention",
			"event_type", event.Type,
			"subject", msg.Subject())
		return nil
	}

	// Parse the raw message bytes into a JSON map so the payload is stored as a
	// readable BSON document rather than binary bytes.  We keep the full event
	// envelope (type, timestamp, data) so the audit trail is complete.
	var parsedPayload interface{}
	if err := json.Unmarshal(msg.Data(), &parsedPayload); err != nil {
		// Fallback: wrap in a map so it remains a BSON document.
		parsedPayload = map[string]interface{}{
			"raw":   string(msg.Data()),
			"error": err.Error(),
		}
	}

	retained := m.RetainedMessage{
		Subject:    msg.Subject(),
		EventType:  string(event.Type),
		Payload:    parsedPayload,
		Stream:     c.stream,
		ReceivedAt: time.Now(),
	}

	// Attach JetStream sequence number when available.
	if meta, err := msg.Metadata(); err == nil {
		retained.Sequence = meta.Sequence.Stream
	}

	return c.rawMessageRepo.AppendMessage(ctx, runID, retained)
}

// extractRunID extracts the run_id from a JSON-encoded event data payload.
// It handles the different nesting structures used by each event type:
//   - run.start / run.end / test.failure / test.error / stdout / stderr → top-level "run_id"
//   - suite.begin / suite.end → nested under "suite.run_id"
//   - test.begin / test.end → nested under "test_case.run_id"
//   - step.begin / step.end → nested under "step.run_id"
func extractRunID(data json.RawMessage) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}

	// Events with run_id at top level
	if v, ok := raw["run_id"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil && s != "" {
			return s
		}
	}

	// Suite events: { "suite": { "run_id": "..." } }
	if suite, ok := raw["suite"]; ok {
		if runID := nestedRunID(suite); runID != "" {
			return runID
		}
	}

	// Test events: { "test_case": { "run_id": "..." } }
	if testCase, ok := raw["test_case"]; ok {
		if runID := nestedRunID(testCase); runID != "" {
			return runID
		}
	}

	// Step events: { "step": { "run_id": "..." } }
	if step, ok := raw["step"]; ok {
		if runID := nestedRunID(step); runID != "" {
			return runID
		}
	}

	return ""
}

// nestedRunID extracts the "run_id" string from a JSON object.
func nestedRunID(data json.RawMessage) string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return ""
	}
	if v, ok := obj["run_id"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			return s
		}
	}
	return ""
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
