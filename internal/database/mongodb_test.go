package database

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	mongocontainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func TestMongoStepBufferTTL(t *testing.T) {
	original := os.Getenv("MONGO_STEP_BUFFER_TTL")
	defer func() {
		if original == "" {
			os.Unsetenv("MONGO_STEP_BUFFER_TTL")
		} else {
			os.Setenv("MONGO_STEP_BUFFER_TTL", original)
		}
	}()

	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "default when unset", value: "", want: defaultMongoStepBufferTTL},
		{name: "duration syntax", value: "45m", want: 45 * time.Minute},
		{name: "plain seconds", value: "120", want: 120 * time.Second},
		{name: "invalid falls back", value: "garbage", want: defaultMongoStepBufferTTL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				os.Unsetenv("MONGO_STEP_BUFFER_TTL")
			} else {
				os.Setenv("MONGO_STEP_BUFFER_TTL", tt.value)
			}

			got := MongoStepBufferTTL(nil)
			if got != tt.want {
				t.Fatalf("MongoStepBufferTTL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureLiveStepBufferIndexes(t *testing.T) {
	ctx := context.Background()
	mongoContainer, err := mongocontainer.RunContainer(ctx, testcontainers.WithImage("mongo:7.0"))
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}
	defer mongoContainer.Terminate(ctx)

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("ConnectionString failed: %v", err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("mongo.Connect failed: %v", err)
	}
	defer client.Disconnect(ctx)

	dbName := "observer_index_test_" + time.Now().Format("20060102150405")
	connection := &MongoDBConnection{
		Client:   client,
		Database: client.Database(dbName),
		logger:   slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	original := os.Getenv("MONGO_STEP_BUFFER_TTL")
	defer func() {
		if original == "" {
			os.Unsetenv("MONGO_STEP_BUFFER_TTL")
		} else {
			os.Setenv("MONGO_STEP_BUFFER_TTL", original)
		}
	}()
	os.Setenv("MONGO_STEP_BUFFER_TTL", "30m")

	if err := connection.EnsureLiveStepBufferIndexes(ctx); err != nil {
		t.Fatalf("EnsureLiveStepBufferIndexes failed: %v", err)
	}

	if got := connection.LiveStepBuffersCollection().Name(); got != "live_step_buffers" {
		t.Fatalf("LiveStepBuffersCollection().Name() = %q, want %q", got, "live_step_buffers")
	}

	cursor, err := connection.LiveStepBuffersCollection().Indexes().List(ctx)
	if err != nil {
		t.Fatalf("Indexes().List failed: %v", err)
	}
	defer cursor.Close(ctx)

	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		t.Fatalf("decode indexes failed: %v", err)
	}

	var ttlFound, runIDFound bool
	for _, index := range indexes {
		name, _ := index["name"].(string)
		key, _ := index["key"].(bson.M)
		switch name {
		case "live_step_buffers_ttl_at_ttl":
			ttlFound = key["ttl_at"] == int32(1) || key["ttl_at"] == int64(1) || key["ttl_at"] == 1
			expire, ok := index["expireAfterSeconds"]
			if !ok || expire == nil {
				t.Fatalf("ttl index missing expireAfterSeconds: %#v", index)
			}
			if expire != int32(0) && expire != int64(0) && expire != 0 {
				t.Fatalf("ttl index expireAfterSeconds = %v, want 0", expire)
			}
		case "live_step_buffers_run_id_idx":
			runIDFound = key["run_id"] == int32(1) || key["run_id"] == int64(1) || key["run_id"] == 1
		}
	}

	if !ttlFound {
		t.Fatal("expected ttl_at index to be created")
	}
	if !runIDFound {
		t.Fatal("expected run_id index to be created")
	}
}

func TestEnsureLiveStepBufferIndexesReconcilesLegacyTTLIndex(t *testing.T) {
	ctx := context.Background()
	mongoContainer, err := mongocontainer.RunContainer(ctx, testcontainers.WithImage("mongo:7.0"))
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}
	defer mongoContainer.Terminate(ctx)

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("ConnectionString failed: %v", err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("mongo.Connect failed: %v", err)
	}
	defer client.Disconnect(ctx)

	dbName := "observer_legacy_index_test_" + time.Now().Format("20060102150405")
	connection := &MongoDBConnection{
		Client:   client,
		Database: client.Database(dbName),
		logger:   slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	legacyTTLIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "ttl_at", Value: 1}},
		Options: options.Index().
			SetName("ttl_idx").
			SetExpireAfterSeconds(900),
	}

	if _, err := connection.LiveStepBuffersCollection().Indexes().CreateOne(ctx, legacyTTLIndex); err != nil {
		t.Fatalf("CreateOne legacy ttl index failed: %v", err)
	}

	if err := connection.EnsureLiveStepBufferIndexes(ctx); err != nil {
		t.Fatalf("EnsureLiveStepBufferIndexes failed: %v", err)
	}

	indexes, err := listCollectionIndexes(ctx, connection.LiveStepBuffersCollection())
	if err != nil {
		t.Fatalf("listCollectionIndexes failed: %v", err)
	}

	var legacyFound, reconciledFound bool
	for _, index := range indexes {
		name, _ := index["name"].(string)
		switch name {
		case "ttl_idx":
			legacyFound = true
		case "live_step_buffers_ttl_at_ttl":
			reconciledFound = true
			expire, ok := extractInt64(index["expireAfterSeconds"])
			if !ok || expire != 0 {
				t.Fatalf("reconciled ttl index expireAfterSeconds = %v, want 0", index["expireAfterSeconds"])
			}
		}
	}

	if legacyFound {
		t.Fatal("expected legacy ttl_idx index to be removed")
	}
	if !reconciledFound {
		t.Fatal("expected live_step_buffers_ttl_at_ttl index to be created")
	}
}
