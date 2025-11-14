package graph

import (
	"log/slog"

	"gorm.io/gorm"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewResolver creates a new resolver with database connection
func NewResolver(db *gorm.DB, logger *slog.Logger) *Resolver {
	if logger == nil {
		logger = slog.Default()
	}
	return &Resolver{
		db:     db,
		logger: logger,
	}
}
