package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
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

	autoFixDirty := parseEnvBool("MIGRATE_AUTO_FIX_DIRTY", false)

	var err error
	switch *action {
	case "up":
		err = embeddedmigrations.Up(dsn)
		if err != nil && autoFixDirty {
			var dirty migrate.ErrDirty
			if errors.As(err, &dirty) {
				logger.Warn(
					"detected dirty postgres migration state, forcing version and retrying",
					"version", dirty.Version,
				)

				if forceErr := embeddedmigrations.Force(dsn, dirty.Version); forceErr != nil {
					err = errors.Join(err, forceErr)
				} else {
					err = embeddedmigrations.Up(dsn)
				}
			}
		}
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

func parseEnvBool(key string, defaultValue bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}
