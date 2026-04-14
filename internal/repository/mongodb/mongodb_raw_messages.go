package mongodb

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

// ListRunSummaries returns paginated summaries for all runs that have retained
// raw messages, newest first by updated_at.
func (r *RawMessageRepository) ListRunSummaries(ctx context.Context, limit, offset int64) ([]m.RawMessagesRunSummary, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, fmt.Errorf("count raw message runs: %w", err)
	}

	pipeline := mongo.Pipeline{
		{{Key: "$project", Value: bson.M{
			"_id":        1,
			"created_at": 1,
			"updated_at": 1,
			"message_count": bson.M{
				"$size": bson.M{
					"$ifNull": bson.A{"$messages", bson.A{}},
				},
			},
		}}},
		{{Key: "$sort", Value: bson.M{"updated_at": -1}}},
		{{Key: "$skip", Value: offset}},
		{{Key: "$limit", Value: limit}},
	}

	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, fmt.Errorf("aggregate raw message run summaries: %w", err)
	}
	defer cur.Close(ctx)

	out := make([]m.RawMessagesRunSummary, 0)
	for cur.Next(ctx) {
		var row m.RawMessagesRunSummary
		if err := cur.Decode(&row); err != nil {
			return nil, 0, fmt.Errorf("decode raw message run summary: %w", err)
		}
		out = append(out, row)
	}

	if err := cur.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate raw message run summaries: %w", err)
	}

	return out, total, nil
}
