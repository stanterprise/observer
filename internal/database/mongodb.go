package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultMongoStepBufferTTL = 15 * time.Minute

// MongoDBConnection wraps a MongoDB client and database
type MongoDBConnection struct {
	Client   *mongo.Client
	Database *mongo.Database
	logger   *slog.Logger
}

// ConnectMongoDB connects to MongoDB using the provided URI
// The URI should be in the format: mongodb://[user:pass@]host[:port]/database[?options]
// or mongodb+srv://... for Atlas clusters
func ConnectMongoDB(uri string, logger *slog.Logger) (*MongoDBConnection, error) {
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set up client options
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(50).
		SetMinPoolSize(10).
		SetMaxConnIdleTime(30 * time.Minute)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	// Verify connection with ping
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	// Extract database name from URI
	dbName := extractDBName(uri)
	if dbName == "" {
		dbName = "observer" // default database name
	}

	logger.Info("connected to mongodb",
		"database", dbName)

	connection := &MongoDBConnection{
		Client:   client,
		Database: client.Database(dbName),
		logger:   logger,
	}

	if err := connection.EnsureLiveStepBufferIndexes(ctx); err != nil {
		return nil, err
	}

	return connection, nil
}

// ConnectMongoDBFromEnv reads MONGODB_URI or MONGO_URI env variable and connects to MongoDB.
// Returns (nil, nil) if no MongoDB URI is configured.
func ConnectMongoDBFromEnv(logger *slog.Logger) (*MongoDBConnection, error) {
	// Check for MongoDB URI in environment
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = os.Getenv("MONGO_URI")
	}
	if uri == "" {
		// Try building from split env vars
		uri = buildMongoURIFromSplitEnv()
	}
	if uri == "" {
		return nil, nil
	}

	return ConnectMongoDB(uri, logger)
}

// buildMongoURIFromSplitEnv constructs a MongoDB URI from individual env vars.
// Recognized: MONGO_HOST, MONGO_PORT, MONGO_USER, MONGO_PASSWORD, MONGO_DATABASE, MONGO_AUTH_SOURCE
// Returns empty string if required fields (at least host) are not set.
func buildMongoURIFromSplitEnv() string {
	host := os.Getenv("MONGO_HOST")
	if host == "" {
		return ""
	}

	port := os.Getenv("MONGO_PORT")
	if port == "" {
		port = "27017"
	}

	user := os.Getenv("MONGO_USER")
	password := os.Getenv("MONGO_PASSWORD")
	database := os.Getenv("MONGO_DATABASE")
	if database == "" {
		database = "observer"
	}
	authSource := os.Getenv("MONGO_AUTH_SOURCE")
	if authSource == "" {
		authSource = "admin"
	}

	var uri string
	if user != "" && password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s",
			user, password, host, port, database, authSource)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%s/%s", host, port, database)
	}

	return uri
}

// extractDBName extracts the database name from a MongoDB URI.
// For mongodb://host:port/dbname, returns "dbname"
// For mongodb://host:port/dbname?options, returns "dbname"
func extractDBName(uri string) string {
	// Remove scheme prefix
	uri = strings.TrimPrefix(uri, "mongodb://")
	uri = strings.TrimPrefix(uri, "mongodb+srv://")

	// Remove credentials if present
	if atIdx := strings.Index(uri, "@"); atIdx != -1 {
		uri = uri[atIdx+1:]
	}

	// Find the path part (after host:port)
	slashIdx := strings.Index(uri, "/")
	if slashIdx == -1 {
		return ""
	}

	// Extract database name (before query string)
	path := uri[slashIdx+1:]
	if qIdx := strings.Index(path, "?"); qIdx != -1 {
		path = path[:qIdx]
	}

	return path
}

// Close closes the MongoDB connection
func (m *MongoDBConnection) Close(ctx context.Context) error {
	if m.Client != nil {
		if err := m.Client.Disconnect(ctx); err != nil {
			return fmt.Errorf("disconnect mongodb: %w", err)
		}
		m.logger.Info("mongodb connection closed")
	}
	return nil
}

// Collection returns a collection handle for the specified collection name
func (m *MongoDBConnection) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// LiveStepBuffersCollection returns the live_step_buffers collection handle used
// for the standalone active-step-buffer contract.
func (m *MongoDBConnection) LiveStepBuffersCollection() *mongo.Collection {
	return m.Collection("live_step_buffers")
}

// EnsureLiveStepBufferIndexes creates the TTL and run sweep indexes required by
// the standalone live step buffer collection.
func (m *MongoDBConnection) EnsureLiveStepBufferIndexes(ctx context.Context) error {
	if m == nil || m.Database == nil {
		return fmt.Errorf("mongodb connection is not initialized")
	}

	if err := m.ensureLiveStepBufferTTLIndex(ctx); err != nil {
		return err
	}

	if err := m.ensureLiveStepBufferRunIDIndex(ctx); err != nil {
		return err
	}

	return nil
}

func (m *MongoDBConnection) ensureLiveStepBufferTTLIndex(ctx context.Context) error {
	collection := m.LiveStepBuffersCollection()
	existingIndexes, err := listCollectionIndexes(ctx, collection)
	if err != nil {
		return fmt.Errorf("list live step buffer indexes: %w", err)
	}

	for _, index := range existingIndexes {
		if !hasSingleFieldIndex(index["key"], "ttl_at") {
			continue
		}

		expireAfterSeconds, ok := extractInt64(index["expireAfterSeconds"])
		if ok && expireAfterSeconds == 0 {
			return nil
		}

		name, _ := index["name"].(string)
		if name == "" {
			return fmt.Errorf("found incompatible live step buffer ttl index without a name")
		}

		if _, err := collection.Indexes().DropOne(ctx, name); err != nil {
			return fmt.Errorf("drop incompatible live step buffer ttl index %q: %w", name, err)
		}
		break
	}

	model := mongo.IndexModel{
		Keys: bson.D{{Key: "ttl_at", Value: 1}},
		Options: options.Index().
			SetName("live_step_buffers_ttl_at_ttl").
			SetExpireAfterSeconds(0),
	}

	if _, err := collection.Indexes().CreateOne(ctx, model); err != nil {
		return fmt.Errorf("create live step buffer ttl index: %w", err)
	}

	return nil
}

func (m *MongoDBConnection) ensureLiveStepBufferRunIDIndex(ctx context.Context) error {
	collection := m.LiveStepBuffersCollection()
	existingIndexes, err := listCollectionIndexes(ctx, collection)
	if err != nil {
		return fmt.Errorf("list live step buffer indexes: %w", err)
	}

	for _, index := range existingIndexes {
		if hasSingleFieldIndex(index["key"], "run_id") {
			return nil
		}
	}

	model := mongo.IndexModel{
		Keys: bson.D{{Key: "run_id", Value: 1}},
		Options: options.Index().
			SetName("live_step_buffers_run_id_idx"),
	}

	if _, err := collection.Indexes().CreateOne(ctx, model); err != nil {
		return fmt.Errorf("create live step buffer run_id index: %w", err)
	}

	return nil
}

func listCollectionIndexes(ctx context.Context, collection *mongo.Collection) ([]bson.M, error) {
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		return nil, err
	}

	return indexes, nil
}

func hasSingleFieldIndex(rawKey interface{}, field string) bool {
	switch key := rawKey.(type) {
	case bson.M:
		if len(key) != 1 {
			return false
		}
		value, ok := key[field]
		if !ok {
			return false
		}
		order, ok := extractInt64(value)
		return ok && order == 1
	case bson.D:
		if len(key) != 1 {
			return false
		}
		order, ok := extractInt64(key[0].Value)
		return key[0].Key == field && ok && order == 1
	default:
		return false
	}
}

func extractInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// MongoStepBufferTTL resolves the configured TTL for live step buffers. Invalid
// values fall back to the default so service startup remains safe.
func MongoStepBufferTTL(logger *slog.Logger) time.Duration {
	if logger == nil {
		logger = slog.Default()
	}

	raw := strings.TrimSpace(os.Getenv("MONGO_STEP_BUFFER_TTL"))
	if raw == "" {
		return defaultMongoStepBufferTTL
	}

	if duration, err := time.ParseDuration(raw); err == nil && duration > 0 {
		return duration
	}

	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	logger.Warn("invalid MONGO_STEP_BUFFER_TTL; using default", "value", raw, "default", defaultMongoStepBufferTTL.String())
	return defaultMongoStepBufferTTL
}
