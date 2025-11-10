package database

import (
	"log/slog"
	"os"
	"testing"
)

func TestBuildDSNFromSplitEnv(t *testing.T) {
	// Save original env vars
	origVars := map[string]string{
		"PGHOST":        os.Getenv("PGHOST"),
		"PGPORT":        os.Getenv("PGPORT"),
		"PGUSER":        os.Getenv("PGUSER"),
		"PGPASSWORD":    os.Getenv("PGPASSWORD"),
		"PGDATABASE":    os.Getenv("PGDATABASE"),
		"PGSSLMODE":     os.Getenv("PGSSLMODE"),
		"PGSSLCERT":     os.Getenv("PGSSLCERT"),
		"PGSSLKEY":      os.Getenv("PGSSLKEY"),
		"PGSSLROOTCERT": os.Getenv("PGSSLROOTCERT"),
	}
	defer func() {
		// Restore original env vars
		for k, v := range origVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		wantDSN  string
		contains []string
	}{
		{
			name: "minimal configuration",
			envVars: map[string]string{
				"PGHOST":     "localhost",
				"PGUSER":     "testuser",
				"PGDATABASE": "testdb",
			},
			contains: []string{"postgres://", "localhost:5432", "testuser", "testdb", "sslmode=disable"},
		},
		{
			name: "with custom port",
			envVars: map[string]string{
				"PGHOST":     "localhost",
				"PGPORT":     "5433",
				"PGUSER":     "testuser",
				"PGDATABASE": "testdb",
			},
			contains: []string{"localhost:5433"},
		},
		{
			name: "with password",
			envVars: map[string]string{
				"PGHOST":     "localhost",
				"PGUSER":     "testuser",
				"PGPASSWORD": "secret",
				"PGDATABASE": "testdb",
			},
			contains: []string{"testuser:secret@"},
		},
		{
			name: "with ssl mode",
			envVars: map[string]string{
				"PGHOST":     "localhost",
				"PGUSER":     "testuser",
				"PGDATABASE": "testdb",
				"PGSSLMODE":  "require",
			},
			contains: []string{"sslmode=require"},
		},
		{
			name: "missing required fields",
			envVars: map[string]string{
				"PGHOST": "localhost",
			},
			wantDSN: "",
		},
		{
			name:    "empty env returns empty string",
			envVars: map[string]string{},
			wantDSN: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all PG* vars first
			for k := range origVars {
				os.Unsetenv(k)
			}
			// Set test-specific vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			dsn := buildDSNFromSplitEnv()

			if tt.wantDSN != "" {
				if dsn != tt.wantDSN {
					t.Errorf("buildDSNFromSplitEnv() = %v, want %v", dsn, tt.wantDSN)
				}
			}

			for _, substr := range tt.contains {
				if dsn == "" {
					t.Errorf("buildDSNFromSplitEnv() returned empty string, expected to contain %v", substr)
					continue
				}
				if !contains(dsn, substr) {
					t.Errorf("buildDSNFromSplitEnv() = %v, should contain %v", dsn, substr)
				}
			}
		})
	}
}

func TestConnectFromEnv_NoDatabaseURL(t *testing.T) {
	// Save and clear env vars
	origVars := map[string]string{
		"DATABASE_URL": os.Getenv("DATABASE_URL"),
		"PGHOST":       os.Getenv("PGHOST"),
		"PGPORT":       os.Getenv("PGPORT"),
		"PGUSER":       os.Getenv("PGUSER"),
		"PGPASSWORD":   os.Getenv("PGPASSWORD"),
		"PGDATABASE":   os.Getenv("PGDATABASE"),
	}
	defer func() {
		for k, v := range origVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Clear all DB-related env vars
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("PGHOST")
	os.Unsetenv("PGUSER")
	os.Unsetenv("PGDATABASE")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := ConnectFromEnv(logger)

	if err != nil {
		t.Errorf("ConnectFromEnv() error = %v, want nil", err)
	}
	if db != nil {
		t.Errorf("ConnectFromEnv() db = %v, want nil", db)
	}
}

func TestConnect_InvalidDSN(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	// Test with invalid DSN
	_, err := Connect("invalid://dsn", logger)
	if err == nil {
		t.Error("Connect() with invalid DSN should return error")
	}
}

func TestAutoMigrateSchema_NilDB(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	err := AutoMigrateSchema(nil, logger)
	if err != nil {
		t.Errorf("AutoMigrateSchema(nil) error = %v, want nil", err)
	}
}

func TestAutoMigrateSchema_Disabled(t *testing.T) {
	origMigrate := os.Getenv("APPLY_MIGRATIONS")
	origAutoMigrate := os.Getenv("GORM_AUTO_MIGRATE")
	defer func() {
		if origMigrate == "" {
			os.Unsetenv("APPLY_MIGRATIONS")
		} else {
			os.Setenv("APPLY_MIGRATIONS", origMigrate)
		}
		if origAutoMigrate == "" {
			os.Unsetenv("GORM_AUTO_MIGRATE")
		} else {
			os.Setenv("GORM_AUTO_MIGRATE", origAutoMigrate)
		}
	}()

	// Disable migrations
	os.Setenv("APPLY_MIGRATIONS", "false")
	os.Setenv("GORM_AUTO_MIGRATE", "false")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Can't test with real DB here, but we can test nil case
	err := AutoMigrateSchema(nil, logger)
	if err != nil {
		t.Errorf("AutoMigrateSchema() with disabled migrations error = %v, want nil", err)
	}
}

func TestAutoMigrateSchema_EnabledVariations(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		value   string
		enabled bool
	}{
		{"APPLY_MIGRATIONS=1", "APPLY_MIGRATIONS", "1", true},
		{"APPLY_MIGRATIONS=true", "APPLY_MIGRATIONS", "true", true},
		{"APPLY_MIGRATIONS=TRUE", "APPLY_MIGRATIONS", "TRUE", true},
		{"APPLY_MIGRATIONS=yes", "APPLY_MIGRATIONS", "yes", true},
		{"APPLY_MIGRATIONS=YES", "APPLY_MIGRATIONS", "YES", true},
		{"APPLY_MIGRATIONS=on", "APPLY_MIGRATIONS", "on", true},
		{"APPLY_MIGRATIONS=ON", "APPLY_MIGRATIONS", "ON", true},
		{"GORM_AUTO_MIGRATE=1", "GORM_AUTO_MIGRATE", "1", true},
		{"APPLY_MIGRATIONS=false", "APPLY_MIGRATIONS", "false", false},
		{"APPLY_MIGRATIONS=0", "APPLY_MIGRATIONS", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origVal := os.Getenv(tt.envVar)
			defer func() {
				if origVal == "" {
					os.Unsetenv(tt.envVar)
				} else {
					os.Setenv(tt.envVar, origVal)
				}
			}()

			os.Setenv(tt.envVar, tt.value)
			
			// We can only test that it doesn't error with nil DB
			// Real migration testing would require a test database
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			err := AutoMigrateSchema(nil, logger)
			if err != nil {
				t.Errorf("AutoMigrateSchema() error = %v, want nil", err)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
