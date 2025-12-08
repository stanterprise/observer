package repository

import (
"log/slog"

"go.mongodb.org/mongo-driver/mongo"
)

// MongoRepository handles MongoDB operations for test runs
type MongoRepository struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository(collection *mongo.Collection, logger *slog.Logger) *MongoRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &MongoRepository{
		collection: collection,
		logger:     logger,
	}
}
