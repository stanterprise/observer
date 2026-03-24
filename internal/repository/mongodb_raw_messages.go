package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RawMessageRepository handles persistence of raw NATS messages for auditing and debugging.
// All messages for a given run are stored in a single document identified by the run_id.
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

// CollectionName returns the MongoDB collection name that raw messages are inserted into.
func (r *RawMessageRepository) CollectionName() string {
	return r.collection.Name()
}

// DatabaseName returns the MongoDB database name that raw messages are inserted into.
func (r *RawMessageRepository) DatabaseName() string {
	return r.collection.Database().Name()
}

// AppendMessage appends a single retained message to the run document identified by
// runID.  If the document does not yet exist it is created (upsert).  All messages
// belonging to the same run are therefore stored in one document.
func (r *RawMessageRepository) AppendMessage(ctx context.Context, runID string, msg m.RetainedMessage) error {
	if runID == "" {
		return fmt.Errorf("runID is required")
	}

	now := time.Now()
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = now
	}

	filter := bson.M{"_id": runID}
	update := bson.M{
		"$push": bson.M{
			"messages": msg,
		},
		"$set": bson.M{
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}
	opts := options.Update().SetUpsert(true)

	if _, err := r.collection.UpdateOne(ctx, filter, update, opts); err != nil {
		return fmt.Errorf("append raw message: %w", err)
	}

	r.logger.Debug("raw message appended",
		"run_id", runID,
		"event_type", msg.EventType,
		"sequence", msg.Sequence)

	return nil
}

// GetByRunID retrieves the full raw messages document for the given run ID.
// Returns nil, nil if no document exists for that run (retention not enabled or
// run predates retention being turned on).
func (r *RawMessageRepository) GetByRunID(ctx context.Context, runID string) (*m.RawMessagesRunDocument, error) {
	if runID == "" {
		return nil, fmt.Errorf("runID is required")
	}

	var doc m.RawMessagesRunDocument
	if err := r.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find raw messages for run %q: %w", runID, err)
	}

	return &doc, nil
}
