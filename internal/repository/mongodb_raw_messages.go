package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// RawMessageRepository handles persistence of raw NATS messages for auditing and debugging.
type RawMessageRepository struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// NewRawMessageRepository creates a new RawMessageRepository backed by the given collection.
func NewRawMessageRepository(collection *mongo.Collection, logger *slog.Logger) *RawMessageRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &RawMessageRepository{
		collection: collection,
		logger:     logger,
	}
}

// Insert persists a raw message document to MongoDB.
// The document ID is auto-generated if empty.
func (r *RawMessageRepository) Insert(ctx context.Context, doc *m.RawMessageDocument) error {
	if doc == nil {
		return fmt.Errorf("raw message document is nil")
	}

	if doc.ID == "" {
		doc.ID = primitive.NewObjectID().Hex()
	}

	if doc.ReceivedAt.IsZero() {
		doc.ReceivedAt = time.Now()
	}

	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("insert raw message: %w", err)
	}

	r.logger.Debug("raw message stored",
		"id", doc.ID,
		"subject", doc.Subject,
		"event_type", doc.EventType,
		"sequence", doc.Sequence)

	return nil
}
