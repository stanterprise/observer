package repository

import (
	"context"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// setupRawMessageTestRepo creates a RawMessageRepository backed by a testcontainer MongoDB instance.
func setupRawMessageTestRepo(t *testing.T) (*RawMessageRepository, func()) {
	t.Helper()
	ctx := context.Background()

	mongoContainer, err := mongodb.RunContainer(ctx, testcontainers.WithImage("mongo:7.0"))
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		mongoContainer.Terminate(ctx)
		t.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		mongoContainer.Terminate(ctx)
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	dbName := "observer_raw_test_" + time.Now().Format("20060102150405")
	col := client.Database(dbName).Collection("raw_messages")
	repo := NewRawMessageRepository(col, nil)

	cleanup := func() {
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		_ = mongoContainer.Terminate(context.Background())
	}
	return repo, cleanup
}

func TestRawMessageRepository_Insert(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	doc := &m.RawMessageDocument{
		Subject:   "tests.events.v1.test.begin",
		EventType: "test.begin",
		Payload:   []byte(`{"type":"test.begin","data":{}}`),
		Stream:    "tests_events",
		Sequence:  42,
	}

	if err := repo.Insert(ctx, doc); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// ID should have been auto-generated.
	if doc.ID == "" {
		t.Fatal("Insert() should set doc.ID when empty")
	}

	// ReceivedAt should be set.
	if doc.ReceivedAt.IsZero() {
		t.Fatal("Insert() should set doc.ReceivedAt when zero")
	}

	// Verify it's actually in the collection.
	var stored m.RawMessageDocument
	if err := repo.collection.FindOne(ctx, bson.M{"_id": doc.ID}).Decode(&stored); err != nil {
		t.Fatalf("FindOne() error = %v", err)
	}

	if stored.Subject != doc.Subject {
		t.Errorf("Subject = %q, want %q", stored.Subject, doc.Subject)
	}
	if stored.EventType != doc.EventType {
		t.Errorf("EventType = %q, want %q", stored.EventType, doc.EventType)
	}
	if stored.Sequence != doc.Sequence {
		t.Errorf("Sequence = %d, want %d", stored.Sequence, doc.Sequence)
	}
	if string(stored.Payload) != string(doc.Payload) {
		t.Errorf("Payload = %q, want %q", stored.Payload, doc.Payload)
	}
}

func TestRawMessageRepository_Insert_PreservesExplicitID(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	doc := &m.RawMessageDocument{
		ID:        "explicit-id-123",
		Subject:   "tests.events.v1.suite.begin",
		EventType: "suite.begin",
		Payload:   []byte(`{}`),
	}

	if err := repo.Insert(ctx, doc); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	if doc.ID != "explicit-id-123" {
		t.Errorf("ID changed from explicit value; got %q", doc.ID)
	}

	var stored m.RawMessageDocument
	if err := repo.collection.FindOne(ctx, bson.M{"_id": "explicit-id-123"}).Decode(&stored); err != nil {
		t.Fatalf("FindOne() error = %v", err)
	}
}

func TestRawMessageRepository_Insert_SetsReceivedAt(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	before := time.Now()
	doc := &m.RawMessageDocument{
		Subject:   "tests.events.v1.test.end",
		EventType: "test.end",
		Payload:   []byte(`{}`),
	}

	if err := repo.Insert(ctx, doc); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}
	after := time.Now()

	if doc.ReceivedAt.Before(before) || doc.ReceivedAt.After(after) {
		t.Errorf("ReceivedAt = %v, want in [%v, %v]", doc.ReceivedAt, before, after)
	}
}

func TestRawMessageRepository_Insert_NilDocument(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	if err := repo.Insert(ctx, nil); err == nil {
		t.Fatal("Insert(nil) should return an error")
	}
}

