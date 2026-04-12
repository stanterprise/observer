package database

import (
	"log/slog"
	"os"
	"testing"
)

func TestConnectPostgresFromEnv_NoDSN(t *testing.T) {
	origDSN := os.Getenv("POSTGRES_DSN")
	defer func() {
		if origDSN != "" {
			os.Setenv("POSTGRES_DSN", origDSN)
		} else {
			os.Unsetenv("POSTGRES_DSN")
		}
	}()

	os.Unsetenv("POSTGRES_DSN")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	conn, err := ConnectPostgresFromEnv(logger)

	if err != nil {
		t.Errorf("ConnectPostgresFromEnv() error = %v, want nil", err)
	}
	if conn != nil {
		t.Errorf("ConnectPostgresFromEnv() conn = %v, want nil when POSTGRES_DSN is unset", conn)
	}
}

func TestConnectPostgres_InvalidDSN(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// A syntactically valid DSN pointing to a non-existent host should fail on connect/ping.
	_, err := ConnectPostgres("postgres://user:pass@localhost:1/nonexistent?connect_timeout=1", logger)
	if err == nil {
		t.Error("ConnectPostgres() with unreachable DSN should return error, got nil")
	}
}
