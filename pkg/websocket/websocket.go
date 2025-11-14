package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stanterprise/observer/pkg/publisher"
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
}

// Client represents a WebSocket client connection
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - should be restricted in production
		return true
	},
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
		broadcast:  make(chan []byte, 256),
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
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close and remove
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
			
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
				// Parse the event
				var event publisher.Event
				if err := json.Unmarshal(msg.Data(), &event); err != nil {
					h.logger.Error("unmarshal event failed", "error", err)
					msg.Nak()
					continue
				}
				
				h.logger.Debug("relaying event to WebSocket clients",
					"type", event.Type,
					"timestamp", event.Timestamp)
				
				// Broadcast to all connected WebSocket clients
				h.broadcast <- msg.Data()
				
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
	
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}
	
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
			
			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}
			
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

// noopWriter implements io.Writer but drops logs when no logger provided
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }
