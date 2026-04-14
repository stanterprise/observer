package graph

import (
	"log/slog"

	"github.com/stanterprise/observer/internal/repository/mongodb"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver is the root resolver for GraphQL queries.
// Currently, the API service uses MongoDB with REST endpoints, not GraphQL.
// This resolver is maintained for potential future GraphQL support.
type Resolver struct {
	repo   *mongodb.MongoRepository
	logger *slog.Logger
}

// NewResolver creates a new GraphQL resolver with MongoDB repository
func NewResolver(repo *mongodb.MongoRepository, logger *slog.Logger) *Resolver {
	if logger == nil {
		logger = slog.Default()
	}
	return &Resolver{
		repo:   repo,
		logger: logger,
	}
}
