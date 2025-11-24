package main

import (
	"log/slog"
	"os"

	"github.com/stanterprise/observer/internal/database"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := database.ConnectFromEnv(logger)
	if err != nil {
		logger.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	if db == nil {
		logger.Error("DATABASE_URL not set")
		os.Exit(1)
	}

	logger.Info("running database migrations")
	if err := database.AutoMigrateSchema(db, logger); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	logger.Info("migrations completed successfully")
}
