package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stanterprise/observer/pkg/publisher"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
)

// Hub manages WebSocket connections and broadcasts events to connected clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe access to clients map
	mu sync.RWMutex

	// Logger
	logger *slog.Logger

	// NATS consumer for event relay
	nc       *nats.Conn
	js       jetstream.JetStream
	consumer jetstream.Consumer
	stream   string

	// Metrics for monitoring (atomic operations)
	droppedMessages   int64 // Messages dropped due to full client buffers
	droppedBroadcasts int64 // Broadcasts dropped due to full hub channel
}

// EventFilters holds filters for selective event streaming
type EventFilters struct {
	// EventTypes filters events by type (e.g., test.begin, test.end)
	// Empty slice means all event types
	EventTypes []string

	// RunID filters events by run ID
	RunID string

	// TestID filters events by test ID
	TestID string

	// SuiteID filters events by suite ID
	SuiteID string
}

// Client represents a WebSocket client connection
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	filters EventFilters
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - should be restricted in production
		return true
	},
}

// isLowPriorityEvent returns true if the event type is low priority (e.g., steps)
// Low priority events are only sent to clients that explicitly filter for them
func isLowPriorityEvent(eventType publisher.EventType) bool {
	return eventType == publisher.EventTypeStepBegin ||
		eventType == publisher.EventTypeStepEnd
}

// isHighPriorityEvent returns true if the event type is high priority (e.g., tests, runs)
// High priority events are broadcast to all clients matching filters
func isHighPriorityEvent(eventType publisher.EventType) bool {
	return eventType == publisher.EventTypeRunStart ||
		eventType == publisher.EventTypeRunEnd ||
		eventType == publisher.EventTypeTestBegin ||
		eventType == publisher.EventTypeTestEnd ||
		eventType == publisher.EventTypeTestFailure ||
		eventType == publisher.EventTypeTestError
}

// NATSConfig holds configuration for NATS WebSocket integration
type NATSConfig struct {
	URL          string
	StreamName   string
	ConsumerName string
	BatchSize    int
	MaxWait      time.Duration
}

// NewHub creates a new WebSocket hub
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}

	return &Hub{
		broadcast:  make(chan []byte, 4096), // Increased from 1024 to handle high load (4x capacity)
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		logger:     logger,
	}
}

// InitNATS initializes NATS consumer for the WebSocket hub
func (h *Hub) InitNATS(cfg NATSConfig) error {
	if cfg.URL == "" {
		h.logger.Info("NATS URL not provided; WebSocket will run without NATS relay")
		return nil
	}

	if cfg.StreamName == "" {
		cfg.StreamName = publisher.DefaultStreamName
	}

	if cfg.ConsumerName == "" {
		cfg.ConsumerName = "websocket"
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}

	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 5 * time.Second
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.URL, nats.Name("observer-websocket"))
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("create jetstream context: %w", err)
	}

	h.nc = nc
	h.js = js
	h.stream = cfg.StreamName

	// Ensure consumer exists
	consumer, err := h.ensureConsumer(context.Background(), cfg.ConsumerName)
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure consumer: %w", err)
	}
	h.consumer = consumer

	h.logger.Info("NATS consumer initialized for WebSocket",
		"url", cfg.URL,
		"stream", cfg.StreamName,
		"consumer", cfg.ConsumerName)

	return nil
}

// ensureConsumer creates the JetStream consumer if it doesn't exist
func (h *Hub) ensureConsumer(ctx context.Context, consumerName string) (jetstream.Consumer, error) {
	// Check if stream exists first
	_, err := h.js.Stream(ctx, h.stream)
	if err != nil {
		return nil, fmt.Errorf("stream %s not found: %w", h.stream, err)
	}

	// Try to get existing consumer
	consumer, err := h.js.Consumer(ctx, h.stream, consumerName)
	if err == nil {
		h.logger.Info("consumer already exists", "consumer", consumerName)
		return consumer, nil
	}

	// Create consumer with durable name
	consumerCfg := jetstream.ConsumerConfig{
		Durable:       consumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy, // Start from beginning for WebSocket (can be customized)
		MaxDeliver:    3,
		AckWait:       10 * time.Second,
		Description:   "WebSocket event relay consumer",
	}

	consumer, err = h.js.CreateOrUpdateConsumer(ctx, h.stream, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}

	h.logger.Info("consumer created", "consumer", consumerName)
	return consumer, nil
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context, cfg NATSConfig) {
	// Start NATS consumer in separate goroutine if configured
	if h.consumer != nil {
		go h.consumeNATSEvents(ctx, cfg)
	}

	// Periodic metrics logging ticker
	metricsTicker := time.NewTicker(60 * time.Second)
	defer metricsTicker.Stop()

	// Main hub loop
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("client connected", "total_clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info("client disconnected", "total_clients", len(h.clients))

		case message := <-h.broadcast:
			// Parse the event to check filters
			var event publisher.Event
			if err := json.Unmarshal(message, &event); err != nil {
				h.logger.Error("failed to parse event for filtering", "error", err)
				continue
			}

			h.mu.RLock()
			sentCount := 0
			filteredCount := 0

			for client := range h.clients {
				// SMART FILTERING: Skip low-priority events if client doesn't have matching filter
				if isLowPriorityEvent(event.Type) && !client.matchesFilters(&event) {
					filteredCount++
					continue
				}

				// High-priority events OR matching low-priority events - check general filter
				if !client.matchesFilters(&event) {
					continue
				}

				select {
				case client.send <- message:
					sentCount++
				default:
					// Client's send channel is full - drop oldest message to make room
					// This keeps the client connected and ensures they see latest events
					select {
					case <-client.send: // Remove oldest message
						atomic.AddInt64(&h.droppedMessages, 1)
					default:
						// Channel already drained by another goroutine
					}
					// Try to add new message
					select {
					case client.send <- message:
						sentCount++
					default:
						// Still couldn't send - client is extremely slow
						// Log but keep connection alive
						droppedCount := atomic.LoadInt64(&h.droppedMessages)
						if droppedCount%100 == 0 { // Log every 100th drop to avoid spam
							h.logger.Warn("client buffer overflow, dropping messages",
								"total_dropped", droppedCount,
								"event_type", event.Type)
						}
					}
				}
			}
			h.mu.RUnlock()

			// Log filtering effectiveness for low-priority events
			if filteredCount > 0 {
				h.logger.Debug("filtered low-priority event",
					"type", event.Type,
					"filtered_clients", filteredCount,
					"sent_to_clients", sentCount)
			}

		case <-metricsTicker.C:
			// Log metrics every 60 seconds
			h.LogMetrics()

		case <-ctx.Done():
			h.logger.Info("hub stopping")
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
			}
			h.mu.Unlock()
			if h.nc != nil {
				h.nc.Close()
			}
			return
		}
	}
}

// consumeNATSEvents consumes events from NATS and broadcasts to WebSocket clients
func (h *Hub) consumeNATSEvents(ctx context.Context, cfg NATSConfig) {
	h.logger.Info("starting NATS consumer for WebSocket relay")

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("NATS consumer stopped by context")
			return
		default:
			// Fetch batch of messages
			msgs, err := h.consumer.Fetch(cfg.BatchSize, jetstream.FetchMaxWait(cfg.MaxWait))
			if err != nil {
				// Check if it's a timeout (no messages available)
				if err == nats.ErrTimeout || err == jetstream.ErrNoMessages {
					continue
				}
				h.logger.Error("fetch messages failed", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Process each message
			for msg := range msgs.Messages() {
				// Parse the event envelope
				var event publisher.Event
				if err := json.Unmarshal(msg.Data(), &event); err != nil {
					h.logger.Error("unmarshal event failed", "error", err)
					msg.Nak()
					continue
				}

				h.logger.Debug("relaying event to WebSocket clients",
					"type", event.Type,
					"timestamp", event.Timestamp)

				// Normalize protobuf data to camelCase before broadcasting
				normalizedData, err := h.normalizeEventData(&event)
				if err != nil {
					h.logger.Error("normalize event data failed", "error", err, "type", event.Type)
					// Fall back to raw data if normalization fails
					select {
					case h.broadcast <- msg.Data():
					default:
						atomic.AddInt64(&h.droppedBroadcasts, 1)
						droppedCount := atomic.LoadInt64(&h.droppedBroadcasts)
						if droppedCount%50 == 0 { // Log every 50th drop
							h.logger.Warn("broadcast channel full, dropping event",
								"type", event.Type,
								"total_dropped_broadcasts", droppedCount)
						}
					}
				} else {
					select {
					case h.broadcast <- normalizedData:
					default:
						atomic.AddInt64(&h.droppedBroadcasts, 1)
						droppedCount := atomic.LoadInt64(&h.droppedBroadcasts)
						if droppedCount%50 == 0 {
							h.logger.Warn("broadcast channel full, dropping event",
								"type", event.Type,
								"total_dropped_broadcasts", droppedCount)
						}
					}
				}

				// Acknowledge message
				msg.Ack()
			}
		}
	}
}

// ServeWS handles WebSocket requests from clients
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	// Parse filters from query parameters
	filters := parseFilters(r)

	client := &Client{
		hub:     h,
		conn:    conn,
		send:    make(chan []byte, 2048), // Increased from 1024 to handle high load (2x capacity)
		filters: filters,
	}

	h.logger.Info("client connecting with filters",
		"eventTypes", filters.EventTypes,
		"runID", filters.RunID,
		"testID", filters.TestID,
		"suiteID", filters.SuiteID)

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Error("websocket read error", "error", err)
			}
			break
		}
		// We don't process messages from clients currently
		// This is just to keep the connection alive and detect disconnects
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// parseFilters extracts event filters from URL query parameters
func parseFilters(r *http.Request) EventFilters {
	query := r.URL.Query()

	filters := EventFilters{
		RunID:   query.Get("runId"),
		TestID:  query.Get("testId"),
		SuiteID: query.Get("suiteId"),
	}

	// Parse event types (comma-separated)
	if eventTypes := query.Get("eventTypes"); eventTypes != "" {
		for _, et := range splitAndTrim(eventTypes, ",") {
			if et != "" {
				filters.EventTypes = append(filters.EventTypes, et)
			}
		}
	}

	return filters
}

// splitAndTrim splits a string by delimiter and trims whitespace from each part
func splitAndTrim(s, delimiter string) []string {
	if s == "" {
		return nil
	}
	parts := []string{}
	for _, part := range splitString(s, delimiter) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString splits string by delimiter (simple implementation)
func splitString(s, delimiter string) []string {
	if s == "" {
		return nil
	}
	result := []string{}
	current := ""
	for i := 0; i < len(s); i++ {
		if i+len(delimiter) <= len(s) && s[i:i+len(delimiter)] == delimiter {
			result = append(result, current)
			current = ""
			i += len(delimiter) - 1
		} else {
			current += string(s[i])
		}
	}
	if current != "" || len(result) > 0 {
		result = append(result, current)
	}
	return result
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// matchesFilters checks if an event matches the client's filters
func (c *Client) matchesFilters(event *publisher.Event) bool {
	// If no filters are set, match all events
	if len(c.filters.EventTypes) == 0 && c.filters.RunID == "" &&
		c.filters.TestID == "" && c.filters.SuiteID == "" {
		return true
	}

	// Parse event data to extract IDs
	var eventData map[string]interface{}
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		// If we can't parse the data, allow the event (fail open)
		return true
	}

	// Check event type filter
	if len(c.filters.EventTypes) > 0 {
		matched := false
		for _, et := range c.filters.EventTypes {
			if string(event.Type) == et {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check runID filter
	if c.filters.RunID != "" {
		if runID, ok := extractID(eventData, "run_id", "runId"); ok {
			if runID != c.filters.RunID {
				return false
			}
		} else {
			// If runID is required but not found in event, filter it out
			return false
		}
	}

	// Check testID filter
	if c.filters.TestID != "" {
		if testID, ok := extractID(eventData, "test_case.id", "testCase.id", "test_id", "testId", "id"); ok {
			if testID != c.filters.TestID {
				return false
			}
		} else {
			return false
		}
	}

	// Check suiteID filter
	if c.filters.SuiteID != "" {
		if suiteID, ok := extractID(eventData, "suite.id", "suiteId", "suite_id"); ok {
			if suiteID != c.filters.SuiteID {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

// extractID attempts to extract an ID from event data using multiple possible field names
func extractID(data map[string]interface{}, fieldNames ...string) (string, bool) {
	for _, fieldName := range fieldNames {
		// Handle nested fields (e.g., "test_case.id")
		if value, ok := getNestedField(data, fieldName); ok {
			if strValue, ok := value.(string); ok {
				return strValue, true
			}
		}
	}
	return "", false
}

// getNestedField retrieves a nested field from a map using dot notation
func getNestedField(data map[string]interface{}, fieldPath string) (interface{}, bool) {
	fields := splitString(fieldPath, ".")
	current := data

	for i, field := range fields {
		if i == len(fields)-1 {
			// Last field - return its value
			value, ok := current[field]
			return value, ok
		}

		// Intermediate field - must be a map
		value, ok := current[field]
		if !ok {
			return nil, false
		}

		nextMap, ok := value.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = nextMap
	}

	return nil, false
}

// normalizeEventData converts protobuf events to model-based JSON for consistency with REST API
// This ensures WebSocket events match the MongoDB document structure used by the REST API
func (h *Hub) normalizeEventData(event *publisher.Event) ([]byte, error) {
	// Convert protobuf to model-based structure
	var modelData interface{}

	switch event.Type {
	case publisher.EventTypeRunStart:
		var req events.ReportRunStartEventRequest
		if err := json.Unmarshal(event.Data, &req); err != nil {
			return nil, fmt.Errorf("unmarshal run start: %w", err)
		}
		modelData = protoToTestRunDocument(&req)

	case publisher.EventTypeTestBegin:
		var req events.TestBeginEventRequest
		if err := json.Unmarshal(event.Data, &req); err != nil {
			return nil, fmt.Errorf("unmarshal test begin: %w", err)
		}
		modelData = protoToTestDocument(req.TestCase)

	case publisher.EventTypeTestEnd:
		var req events.TestEndEventRequest
		if err := json.Unmarshal(event.Data, &req); err != nil {
			return nil, fmt.Errorf("unmarshal test end: %w", err)
		}
		modelData = protoToTestDocument(req.TestCase)

	case publisher.EventTypeStepBegin:
		var req events.StepBeginEventRequest
		if err := json.Unmarshal(event.Data, &req); err != nil {
			return nil, fmt.Errorf("unmarshal step begin: %w", err)
		}
		modelData = protoToStepDocument(req.Step)

	case publisher.EventTypeStepEnd:
		var req events.StepEndEventRequest
		if err := json.Unmarshal(event.Data, &req); err != nil {
			return nil, fmt.Errorf("unmarshal step end: %w", err)
		}
		modelData = protoToStepDocument(req.Step)

	case publisher.EventTypeSuiteBegin, publisher.EventTypeSuiteEnd,
		publisher.EventTypeRunEnd:
		// For events not yet converted to models, pass through raw data
		// TODO: Add model converters for these event types
		if err := json.Unmarshal(event.Data, &modelData); err != nil {
			return nil, fmt.Errorf("parse event data: %w", err)
		}

	default:
		// For unknown event types, pass through the raw data
		if err := json.Unmarshal(event.Data, &modelData); err != nil {
			return nil, fmt.Errorf("parse unknown event type: %w", err)
		}
	}

	// Re-wrap in event envelope with model-based data
	normalizedEvent := publisher.Event{
		Type:      event.Type,
		Timestamp: event.Timestamp,
		Data:      nil, // Will be filled during marshal
	}

	// Marshal model data
	dataBytes, err := json.Marshal(modelData)
	if err != nil {
		return nil, fmt.Errorf("marshal model data: %w", err)
	}
	normalizedEvent.Data = json.RawMessage(dataBytes)

	// Marshal complete event
	return json.Marshal(normalizedEvent)
}

// Metrics returns current hub metrics for monitoring
type HubMetrics struct {
	ConnectedClients   int
	DroppedMessages    int64
	DroppedBroadcasts  int64
	BroadcastQueueSize int
	BroadcastCapacity  int
}

// GetMetrics returns current hub metrics (safe for concurrent access)
func (h *Hub) GetMetrics() HubMetrics {
	h.mu.RLock()
	clients := len(h.clients)
	h.mu.RUnlock()

	return HubMetrics{
		ConnectedClients:   clients,
		DroppedMessages:    atomic.LoadInt64(&h.droppedMessages),
		DroppedBroadcasts:  atomic.LoadInt64(&h.droppedBroadcasts),
		BroadcastQueueSize: len(h.broadcast),
		BroadcastCapacity:  cap(h.broadcast),
	}
}

// LogMetrics logs current metrics (useful for periodic health checks)
func (h *Hub) LogMetrics() {
	m := h.GetMetrics()
	h.logger.Info("websocket hub metrics",
		"connected_clients", m.ConnectedClients,
		"dropped_messages", m.DroppedMessages,
		"dropped_broadcasts", m.DroppedBroadcasts,
		"broadcast_queue_size", m.BroadcastQueueSize,
		"broadcast_capacity", m.BroadcastCapacity,
		"queue_utilization_pct", float64(m.BroadcastQueueSize)/float64(m.BroadcastCapacity)*100)
}

// noopWriter implements io.Writer but drops logs when no logger provided
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }
