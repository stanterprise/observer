package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConnection wraps a pgxpool.Pool and provides lifecycle management.
type PostgresConnection struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

// ConnectPostgres connects to PostgreSQL using the provided DSN.
// The DSN should be in the format: postgres://user:pass@host:port/database?sslmode=disable
func ConnectPostgres(dsn string, logger *slog.Logger) (*PostgresConnection, error) {
	if logger == nil {
		logger = slog.Default()
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	// Conservative pool defaults matching the MongoDB connection module pattern.
	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 10 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	// Verify connectivity with a ping.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	logger.Info("connected to postgres",
		"host", cfg.ConnConfig.Host,
		"port", cfg.ConnConfig.Port,
		"database", cfg.ConnConfig.Database)

	return &PostgresConnection{
		Pool:   pool,
		logger: logger,
	}, nil
}

// ConnectPostgresFromEnv reads POSTGRES_DSN env variable and connects to PostgreSQL.
// Returns (nil, nil) if no PostgreSQL DSN is configured.
func ConnectPostgresFromEnv(logger *slog.Logger) (*PostgresConnection, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return nil, nil
	}
	return ConnectPostgres(dsn, logger)
}

// Close closes the connection pool.
func (p *PostgresConnection) Close() {
	if p.Pool != nil {
		p.Pool.Close()
		p.logger.Info("postgres connection closed")
	}
}

// Ping checks that the database is reachable.
func (p *PostgresConnection) Ping(ctx context.Context) error {
	return p.Pool.Ping(ctx)
}

// DatabaseName returns the configured database name.
func (p *PostgresConnection) DatabaseName() string {
	return p.Pool.Config().ConnConfig.Database
}
