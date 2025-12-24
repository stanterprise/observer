package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	// DefaultStreamName is the JetStream stream for test events
	DefaultStreamName = "tests_events"
	// DefaultSubjectPrefix is the subject prefix for test events
	DefaultSubjectPrefix = "tests.events.v1"
)

// NATSPublisher wraps a NATS JetStream connection for publishing events
type NATSPublisher struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger *slog.Logger
	stream string
	prefix string
}

// NATSConfig holds configuration for NATS publisher
type NATSConfig struct {
	URL           string
	StreamName    string
	SubjectPrefix string
}

// NewNATSPublisher creates a new NATS JetStream publisher
// If logger is nil, a no-op logger is used
func NewNATSPublisher(cfg NATSConfig, logger *slog.Logger) (*NATSPublisher, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("NATS URL is required")
	}

	if cfg.StreamName == "" {
		cfg.StreamName = DefaultStreamName
	}

	if cfg.SubjectPrefix == "" {
		cfg.SubjectPrefix = DefaultSubjectPrefix
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.URL, nats.Name("observer-ingestion"))
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}

	p := &NATSPublisher{
		nc:     nc,
		js:     js,
		logger: logger,
		stream: cfg.StreamName,
		prefix: cfg.SubjectPrefix,
	}

	// Ensure stream exists
	if err := p.ensureStream(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure stream: %w", err)
	}

	logger.Info("NATS publisher initialized", "url", cfg.URL, "stream", cfg.StreamName)
	return p, nil
}

// ensureStream creates the JetStream stream if it doesn't exist
func (p *NATSPublisher) ensureStream(ctx context.Context) error {
	// Check if stream exists
	_, err := p.js.Stream(ctx, p.stream)
	if err == nil {
		p.logger.Info("stream already exists", "stream", p.stream)
		return nil
	}

	// Create stream with default configuration
	streamCfg := jetstream.StreamConfig{
		Name:        p.stream,
		Subjects:    []string{p.prefix + ".>"},
		Retention:   jetstream.LimitsPolicy, // Use LimitsPolicy to allow multiple independent consumers
		MaxAge:      24 * time.Hour,         // Keep events for 24 hours
		Storage:     jetstream.FileStorage,
		Replicas:    1,
		Description: "Test execution events stream",
	}

	_, err = p.js.CreateStream(ctx, streamCfg)
	if err != nil {
		return fmt.Errorf("create stream: %w", err)
	}

	p.logger.Info("stream created", "stream", p.stream)
	return nil
}

// EventType represents the type of event being published
type EventType string

const (
	EventTypeTestBegin   EventType = "test.begin"
	EventTypeTestEnd     EventType = "test.end"
	EventTypeStepBegin   EventType = "step.begin"
	EventTypeStepEnd     EventType = "step.end"
	EventTypeSuiteBegin  EventType = "suite.begin"
	EventTypeSuiteEnd    EventType = "suite.end"
	EventTypeTestFailure EventType = "test.failure"
	EventTypeTestError   EventType = "test.error"
	EventTypeStdOutput   EventType = "stdout"
	EventTypeStdError    EventType = "stderr"
	EventTypeHeartbeat   EventType = "heartbeat"
	EventTypeRunEnd      EventType = "run.end"
	MapSuitesEvent       EventType = "map.suites"
)

// Event represents a generic event wrapper for publishing
type Event struct {
	Type      EventType       `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// Publish publishes an event to NATS JetStream
func (p *NATSPublisher) Publish(ctx context.Context, eventType EventType, data interface{}) error {
	// Marshal the data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	// Create event wrapper
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      dataBytes,
	}

	// Marshal the complete event
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Determine subject based on event type
	subject := fmt.Sprintf("%s.%s", p.prefix, eventType)

	// Publish to JetStream
	_, err = p.js.Publish(ctx, subject, eventBytes)
	if err != nil {
		return fmt.Errorf("publish to NATS: %w", err)
	}

	p.logger.Debug("event published", "subject", subject, "type", eventType, "size", len(eventBytes))
	return nil
}

// Close closes the NATS connection
func (p *NATSPublisher) Close() error {
	if p.nc != nil {
		p.nc.Close()
		p.logger.Info("NATS publisher closed")
	}
	return nil
}

// noopWriter implements io.Writer but drops logs when no logger provided
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }
