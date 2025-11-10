package database

import (
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect connects to Postgres using the provided DSN and returns a GORM DB.
// Example DSN: postgres://user:pass@localhost:5432/app?sslmode=disable
func Connect(dsn string, l *slog.Logger) (*gorm.DB, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "gorm ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		},
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: newLogger})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("sql db: %w", err)
	}
	// Sensible pool defaults; callers can tune if needed.
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

// ConnectFromEnv reads DATABASE_URL and returns a DB if set; otherwise returns (nil, nil).
func ConnectFromEnv(l *slog.Logger) (*gorm.DB, error) {
	// Prefer DATABASE_URL if provided.
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return Connect(dsn, l)
	}
	// Otherwise, attempt to build from split PG* envs.
	if dsn := buildDSNFromSplitEnv(); dsn != "" {
		return Connect(dsn, l)
	}
	return nil, nil
}

// AutoMigrateSchema applies database schema using GORM if db is non-nil.
// Controlled by env APPLY_MIGRATIONS (true/1/yes) or GORM_AUTO_MIGRATE (alias).
func AutoMigrateSchema(db *gorm.DB, l *slog.Logger) error {
	if db == nil {
		return nil
	}
	if l == nil {
		l = slog.Default()
	}
	enable := os.Getenv("APPLY_MIGRATIONS")
	if enable == "" {
		enable = os.Getenv("GORM_AUTO_MIGRATE")
	}
	switch enable {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		// proceed
	default:
		l.Info("automigrate disabled; set APPLY_MIGRATIONS=1 to enable")
		return nil
	}
	l.Info("running gorm automigrate")
	if err := db.AutoMigrate(&m.TestCaseRun{}, &m.StepRun{}); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	l.Info("automigrate complete")
	return nil
}

// buildDSNFromSplitEnv constructs a Postgres URL from individual PG* env vars.
// Recognized: PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE, PGSSLMODE, PGSSLCERT, PGSSLKEY, PGSSLROOTCERT
// Returns empty string if required fields are missing (at least host, user, db must be set).
func buildDSNFromSplitEnv() string {
	host := os.Getenv("PGHOST")
	user := os.Getenv("PGUSER")
	db := os.Getenv("PGDATABASE")
	if host == "" || user == "" || db == "" {
		return ""
	}
	port := os.Getenv("PGPORT")
	if port == "" {
		port = "5432"
	}
	pass := os.Getenv("PGPASSWORD")
	sslmode := os.Getenv("PGSSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	sslcert := os.Getenv("PGSSLCERT")
	sslkey := os.Getenv("PGSSLKEY")
	sslroot := os.Getenv("PGSSLROOTCERT")

	u := &url.URL{Scheme: "postgres", Host: host}
	if port != "" {
		u.Host = fmt.Sprintf("%s:%s", host, port)
	}
	if pass != "" {
		u.User = url.UserPassword(user, pass)
	} else {
		u.User = url.User(user)
	}
	u.Path = "/" + db
	q := url.Values{}
	q.Set("sslmode", sslmode)
	if sslcert != "" {
		q.Set("sslcert", sslcert)
	}
	if sslkey != "" {
		q.Set("sslkey", sslkey)
	}
	if sslroot != "" {
		q.Set("sslrootcert", sslroot)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
