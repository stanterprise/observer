package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	embeddedmigrations "github.com/stanterprise/observer/migrations"
)

func main() {
	action := flag.String("action", "up", "migration action: up or down")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		logger.Error("POSTGRES_DSN / DATABASE_URL not set")
		os.Exit(1)
	}

	var err error
	switch *action {
	case "up":
		err = embeddedmigrations.Up(dsn)
	case "down":
		err = embeddedmigrations.Down(dsn)
	default:
		err = fmt.Errorf("unsupported migration action %q", *action)
	}
	if err != nil {
		logger.Error("postgres migration failed", "action", *action, "error", err)
		os.Exit(1)
	}

	logger.Info("postgres migration completed", "action", *action)
}
