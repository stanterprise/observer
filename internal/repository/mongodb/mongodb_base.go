package mongodb

import (
	"log/slog"

	"go.mongodb.org/mongo-driver/mongo"
)

// MongoRepository handles MongoDB operations for live step buffers.
type MongoRepository struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// NewMongoRepository creates a repository bound to a specific MongoDB collection.
func NewMongoRepository(collection *mongo.Collection, logger *slog.Logger) *MongoRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &MongoRepository{
		collection: collection,
		logger:     logger,
	}
}
