package graph

import (
	"log/slog"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver is the root resolver for GraphQL queries.
// GraphQL is currently a stub and does not bind to an active persistence backend.
type Resolver struct {
	logger *slog.Logger
}

// NewResolver creates a new GraphQL resolver.
func NewResolver(logger *slog.Logger) *Resolver {
	if logger == nil {
		logger = slog.Default()
	}
	return &Resolver{
		logger: logger,
	}
}
