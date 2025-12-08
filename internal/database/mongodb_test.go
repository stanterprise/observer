package database

import (
	"log/slog"
	"os"
	"testing"
)

func TestExtractDBName(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "simple uri",
			uri:      "mongodb://localhost:27017/mydb",
			expected: "mydb",
		},
		{
			name:     "with credentials",
			uri:      "mongodb://user:pass@localhost:27017/testdb",
			expected: "testdb",
		},
		{
			name:     "with query string",
			uri:      "mongodb://localhost:27017/dbname?authSource=admin",
			expected: "dbname",
		},
		{
			name:     "srv format",
			uri:      "mongodb+srv://user:pass@cluster.example.com/production",
			expected: "production",
		},
		{
			name:     "no database",
			uri:      "mongodb://localhost:27017",
			expected: "",
		},
		{
			name:     "empty uri",
			uri:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDBName(tt.uri)
			if result != tt.expected {
				t.Errorf("extractDBName(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestIsMongoDBURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		{
			name:     "mongodb scheme",
			uri:      "mongodb://localhost:27017/db",
			expected: true,
		},
		{
			name:     "mongodb+srv scheme",
			uri:      "mongodb+srv://cluster.example.com/db",
			expected: true,
		},
		{
			name:     "postgres scheme",
			uri:      "postgres://localhost:5432/db",
			expected: false,
		},
		{
			name:     "file path",
			uri:      "/path/to/db.sqlite",
			expected: false,
		},
		{
			name:     "empty string",
			uri:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMongoDBURI(tt.uri)
			if result != tt.expected {
				t.Errorf("IsMongoDBURI(%q) = %v, want %v", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestBuildMongoURIFromSplitEnv(t *testing.T) {
	// Save original env vars
	origVars := map[string]string{
		"MONGO_HOST":        os.Getenv("MONGO_HOST"),
		"MONGO_PORT":        os.Getenv("MONGO_PORT"),
		"MONGO_USER":        os.Getenv("MONGO_USER"),
		"MONGO_PASSWORD":    os.Getenv("MONGO_PASSWORD"),
		"MONGO_DATABASE":    os.Getenv("MONGO_DATABASE"),
		"MONGO_AUTH_SOURCE": os.Getenv("MONGO_AUTH_SOURCE"),
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
		contains []string
		wantURI  string
	}{
		{
			name: "minimal configuration",
			envVars: map[string]string{
				"MONGO_HOST": "localhost",
			},
			contains: []string{"mongodb://", "localhost:27017", "observer"},
		},
		{
			name: "with custom port",
			envVars: map[string]string{
				"MONGO_HOST": "localhost",
				"MONGO_PORT": "27018",
			},
			contains: []string{"localhost:27018"},
		},
		{
			name: "with credentials",
			envVars: map[string]string{
				"MONGO_HOST":     "localhost",
				"MONGO_USER":     "testuser",
				"MONGO_PASSWORD": "secret",
			},
			contains: []string{"testuser:secret@"},
		},
		{
			name: "with custom database",
			envVars: map[string]string{
				"MONGO_HOST":     "localhost",
				"MONGO_DATABASE": "customdb",
			},
			contains: []string{"/customdb"},
		},
		{
			name:    "missing host",
			envVars: map[string]string{},
			wantURI: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all MONGO* vars first
			for k := range origVars {
				os.Unsetenv(k)
			}
			// Set test-specific vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			uri := buildMongoURIFromSplitEnv()

			if tt.wantURI != "" {
				if uri != tt.wantURI {
					t.Errorf("buildMongoURIFromSplitEnv() = %v, want %v", uri, tt.wantURI)
				}
			}

			for _, substr := range tt.contains {
				if uri == "" {
					if len(tt.contains) > 0 {
						t.Errorf("buildMongoURIFromSplitEnv() returned empty string, expected to contain %v", substr)
					}
					continue
				}
				if !findSubstring(uri, substr) {
					t.Errorf("buildMongoURIFromSplitEnv() = %v, should contain %v", uri, substr)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func findSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || contains(s, substr))
}

// contains checks if string s contains substr
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestConnectMongoDBFromEnv_NoURI(t *testing.T) {
	// Save and clear env vars
	origVars := map[string]string{
		"MONGODB_URI": os.Getenv("MONGODB_URI"),
		"MONGO_URI":   os.Getenv("MONGO_URI"),
		"MONGO_HOST":  os.Getenv("MONGO_HOST"),
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

	// Clear all MongoDB-related env vars
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("MONGO_URI")
	os.Unsetenv("MONGO_HOST")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := ConnectMongoDBFromEnv(logger)

	if err != nil {
		t.Errorf("ConnectMongoDBFromEnv() error = %v, want nil", err)
	}
	if db != nil {
		t.Errorf("ConnectMongoDBFromEnv() db = %v, want nil", db)
	}
}
