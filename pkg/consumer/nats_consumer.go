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
	"github.com/stanterprise/observer/internal/repository/mongodb"
	"github.com/stanterprise/observer/internal/repository/postgres"
	"github.com/stanterprise/observer/internal/telemetry"
	"github.com/stanterprise/observer/pkg/publisher"
	"github.com/stanterprise/observer/pkg/storage"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/encoding/protojson"
)

// noopWriter implements io.Writer but drops logs when no logger provided.
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// extractTestID extracts the test ID from a test case run ID.
// TestCaseRunId format is typically: {runId}-{testId}
// This function strips the runId prefix to get just the testId.
func extractTestID(testCaseRunID, runID string) string {
	if runID != "" {
		prefix := runID + "-"
		if strings.HasPrefix(testCaseRunID, prefix) {
			return strings.TrimPrefix(testCaseRunID, prefix)
		}
	}
	// Otherwise, return as-is (backward compatibility)
	return testCaseRunID
}

// NATSConsumer wraps a NATS JetStream consumer for processing test events.
// Structured run, suite, test, and attempt data is persisted to PostgreSQL,
// while MongoDB is retained only for the live step buffer used during in-flight
// execution.
type NATSConsumer struct {
	nc            *nats.Conn
	js            jetstream.JetStream
	logger        *slog.Logger
	bufferRepo    *mongodb.MongoRepository
	pgRepo        *postgres.PostgresRepository
	storageDriver storage.Driver
	stream        string
	consumer      jetstream.Consumer
	prefix        string
	maxDeliver    int
	ackWait       time.Duration
	dlqSubject    string

	deferCfg   deferQueueConfig
	deferQueue map[string][]deferredStepEvent
	deferMu    sync.Mutex

	integrityMu      sync.Mutex
	runIntegrityByID map[string]*runIntegrity

	metrics         *consumerMetrics
	metricsReg      metric.Registration
}

// NATSConsumerConfig holds configuration for NATS consumer
type NATSConsumerConfig struct {
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
	RetainMessages        bool // Deprecated no-op; legacy raw message retention was removed.
}

type deferredStepEvent struct {
	eventType   publisher.EventType
	data        json.RawMessage
	runID       string
	executionID string
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

// NewNATSConsumer creates a new NATS JetStream consumer with Postgres backend and MongoDB-backed live step buffer support.
func NewNATSConsumer(cfg NATSConsumerConfig, logger *slog.Logger, bufferRepo *mongodb.MongoRepository, pgRepo *postgres.PostgresRepository) (*NATSConsumer, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("NATS URL is required")
	}
	if pgRepo == nil {
		return nil, fmt.Errorf("Postgres repository is required for processor")
	}
	if bufferRepo == nil {
		return nil, fmt.Errorf("MongoDB live step buffer repository is required for processor")
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
	nc, err := nats.Connect(cfg.URL, nats.Name("observer-processor"))
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

	c := &NATSConsumer{
		nc:            nc,
		js:            js,
		logger:        logger,
		bufferRepo:    bufferRepo,
		pgRepo:        pgRepo,
		storageDriver: storageDriver,
		stream:        cfg.StreamName,
		prefix:        publisher.DefaultSubjectPrefix,
		maxDeliver:    cfg.MaxDeliver,
		ackWait:       cfg.AckWait,
		dlqSubject:    cfg.DLQSubject,
		deferCfg: deferQueueConfig{
			maxAttempts: cfg.DeferQueueMaxAttempts,
			ttl:         cfg.DeferQueueTTL,
		},
		deferQueue:       make(map[string][]deferredStepEvent),
		integrityMu:      sync.Mutex{},
		deferMu:          sync.Mutex{},
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

	logger.Info("NATS consumer initialized",
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
	)

	// Initialise OTel metrics instruments.  Uses the global MeterProvider which
	// is a no-op if telemetry.Setup has not been called, so this never fails in
	// tests or when metrics are intentionally disabled.
	meter := telemetry.Meter("observer/processor")
	metrics, reg, err := initConsumerMetrics(meter, func() int64 {
		c.deferMu.Lock()
		total := 0
		for _, evts := range c.deferQueue {
			total += len(evts)
		}
		c.deferMu.Unlock()
		return int64(total)
	})
	if err != nil {
		logger.Warn("consumer metrics init failed – metrics disabled for this consumer", "error", err)
	} else {
		c.metrics = metrics
		c.metricsReg = reg
	}

	return c, nil
}

// ensureConsumer creates or updates the JetStream consumer, applying the given
// MaxDeliver and AckWait values even when the durable consumer already exists.
// Using CreateOrUpdateConsumer ensures that config changes (e.g. retry count,
// ack timeout) take effect at startup rather than being silently ignored when
// a durable consumer with the same name is already registered in the stream.
func (c *NATSConsumer) ensureConsumer(ctx context.Context, consumerName string, maxDeliver int, ackWait time.Duration) (jetstream.Consumer, error) {
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
		Description:   "Observer test event processor consumer",
	}

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, c.stream, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("create or update consumer: %w", err)
	}

	c.logger.Info("consumer ready", "consumer", consumerName, "max_deliver", maxDeliver, "ack_wait", ackWait)
	return consumer, nil
}

// Start begins consuming messages from NATS
func (c *NATSConsumer) Start(ctx context.Context, cfg NATSConsumerConfig) error {
	c.logger.Info("starting NATS consumer")

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

			batchCount := int64(0)
			for msg := range msgs.Messages() {
				batchCount++
				msgStart := time.Now()
				event, err := c.processMessage(ctx, msg)
				elapsed := time.Since(msgStart).Seconds()

				if c.metrics != nil {
					c.metrics.processingDuration.Record(ctx, elapsed, eventAttr(string(event.Type)))
				}

				if err != nil {
					runID := extractRunID(event.Data)
					c.incrementIntegrity(runID, func(ri *runIntegrity) {
						ri.Failed++
					})
					if c.metrics != nil {
						c.metrics.eventsFailed.Add(ctx, 1, eventAttr(string(event.Type)))
					}
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
						if c.metrics != nil {
							c.metrics.eventsDLQ.Add(ctx, 1, eventAttr(string(event.Type)))
						}
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
					if c.metrics != nil {
						c.metrics.eventsProcessed.Add(ctx, 1, eventAttr(string(event.Type)))
					}
					if ackErr := msg.Ack(); ackErr != nil {
						c.logger.Error("failed to ack message", "error", ackErr)
					}
				}
			}

			if c.metrics != nil && batchCount > 0 {
				c.metrics.batchSize.Record(ctx, batchCount)
			}
		}
	}
}

// processMessage handles a single NATS message
func (c *NATSConsumer) processMessage(ctx context.Context, msg jetstream.Msg) (publisher.Event, error) {
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

	// (Legacy raw message retention removed)

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

func (c *NATSConsumer) shouldDeferStepEvent(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "parent test not found") ||
		strings.Contains(errMsg, "active step buffer not found") ||
		strings.Contains(errMsg, "step not found")
}

func (c *NATSConsumer) deferStepEvent(event publisher.Event, cause error) error {
	runID, executionID, testID, retryIndex, err := extractStepIdentity(event)
	if err != nil {
		return err
	}

	key := deferredQueueKey(runID, executionID, testID, retryIndex)
	c.deferMu.Lock()
	deferred := deferredStepEvent{
		eventType:   event.Type,
		data:        event.Data,
		runID:       runID,
		executionID: executionID,
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

	if c.metrics != nil {
		c.metrics.eventsDeferred.Add(context.Background(), 1, eventAttr(string(event.Type)))
	}

	c.logger.Warn("deferred orphan step event",
		"type", event.Type,
		"run_id", runID,
		"execution_id", executionID,
		"test_id", testID,
		"retry_index", retryIndex,
		"pending_for_key", pendingCount,
		"error", cause)

	return nil
}

func (c *NATSConsumer) replayDeferredStepsForTest(ctx context.Context, runID, executionID, testID string, retryIndex int32) {
	key := deferredQueueKey(runID, executionID, testID, retryIndex)

	c.deferMu.Lock()
	events := c.deferQueue[key]
	delete(c.deferQueue, key)
	c.deferMu.Unlock()

	if len(events) == 0 {
		return
	}

	c.logger.Info("replaying deferred step events",
		"run_id", runID,
		"execution_id", executionID,
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
				"execution_id", deferred.executionID,
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
				"execution_id", deferred.executionID,
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

func extractStepIdentity(event publisher.Event) (runID string, executionID string, testID string, retryIndex int32, err error) {
	switch event.Type {
	case publisher.EventTypeStepBegin:
		var req events.StepBeginEventRequest
		if unmarshalErr := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(event.Data, &req); unmarshalErr != nil {
			return "", "", "", 0, fmt.Errorf("parse step begin for defer queue: %w", unmarshalErr)
		}
		if req.Step == nil {
			return "", "", "", 0, fmt.Errorf("parse step begin for defer queue: step is nil")
		}
		return req.Step.RunId, req.Step.ExecutionId, extractTestID(req.Step.TestCaseId, req.Step.RunId), req.Step.RetryIndex, nil
	case publisher.EventTypeStepEnd:
		var req events.StepEndEventRequest
		if unmarshalErr := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(event.Data, &req); unmarshalErr != nil {
			return "", "", "", 0, fmt.Errorf("parse step end for defer queue: %w", unmarshalErr)
		}
		if req.Step == nil {
			return "", "", "", 0, fmt.Errorf("parse step end for defer queue: step is nil")
		}
		return req.Step.RunId, req.Step.ExecutionId, extractTestID(req.Step.TestCaseId, req.Step.RunId), req.Step.RetryIndex, nil
	default:
		return "", "", "", 0, fmt.Errorf("not a step event: %s", event.Type)
	}
}

func deferredQueueKey(runID, executionID, testID string, retryIndex int32) string {
	return runID + "::" + executionID + "::" + testID + "::" + strconv.Itoa(int(retryIndex))
}

func (c *NATSConsumer) incrementIntegrity(runID string, mutate func(ri *runIntegrity)) {
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

func (c *NATSConsumer) markRunStart(runID string, expectedTests int32) {
	c.incrementIntegrity(runID, func(ri *runIntegrity) {
		if ri.StartedAt.IsZero() {
			ri.StartedAt = time.Now()
		}
		ri.ExpectedTests = expectedTests
	})
}

func (c *NATSConsumer) emitRunCompletenessSummary(runID string, finalStatus string) {
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
func (c *NATSConsumer) purgeDeferQueueForRun(runID string) {
	prefix := runID + "::"
	c.deferMu.Lock()
	for key := range c.deferQueue {
		if strings.HasPrefix(key, prefix) {
			delete(c.deferQueue, key)
		}
	}
	c.deferMu.Unlock()
}

func (c *NATSConsumer) publishDLQ(ctx context.Context, msg jetstream.Msg, event publisher.Event, processingErr error, deliveries uint64) error {
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
func (c *NATSConsumer) handleHeartbeat(ctx context.Context, data json.RawMessage) error {
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

// statusToString converts protobuf status to string
func statusToString(status common.TestStatus) string {
	return status.String()
}

// Close closes the NATS connection
func (c *NATSConsumer) Close() error {
	if c.metricsReg != nil {
		if err := c.metricsReg.Unregister(); err != nil {
			c.logger.Warn("failed to unregister metrics callback", "error", err)
		}
	}
	if c.storageDriver != nil {
		if err := c.storageDriver.Close(); err != nil {
			c.logger.Warn("failed to close storage driver", "error", err)
		}
	}
	if c.nc != nil {
		c.nc.Close()
		c.logger.Info("NATS consumer closed")
	}
	return nil
}
