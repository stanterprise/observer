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

func TestRawMessageRepository_AppendMessage_CreatesDocument(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "run-append-test-1"

	msg := m.RetainedMessage{
		Subject:   "tests.events.v1.test.begin",
		EventType: "test.begin",
		Payload:   []byte(`{"type":"test.begin","data":{}}`),
		Stream:    "tests_events",
		Sequence:  42,
	}

	if err := repo.AppendMessage(ctx, runID, msg); err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	// Verify document was created with the run_id as _id.
	var stored m.RawMessagesRunDocument
	if err := repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&stored); err != nil {
		t.Fatalf("FindOne() error = %v", err)
	}

	if stored.RunID != runID {
		t.Errorf("RunID = %q, want %q", stored.RunID, runID)
	}
	if len(stored.Messages) != 1 {
		t.Fatalf("Messages count = %d, want 1", len(stored.Messages))
	}
	if stored.Messages[0].Subject != msg.Subject {
		t.Errorf("Subject = %q, want %q", stored.Messages[0].Subject, msg.Subject)
	}
	if stored.Messages[0].EventType != msg.EventType {
		t.Errorf("EventType = %q, want %q", stored.Messages[0].EventType, msg.EventType)
	}
	if stored.Messages[0].Sequence != msg.Sequence {
		t.Errorf("Sequence = %d, want %d", stored.Messages[0].Sequence, msg.Sequence)
	}
	if string(stored.Messages[0].Payload) != string(msg.Payload) {
		t.Errorf("Payload = %q, want %q", stored.Messages[0].Payload, msg.Payload)
	}
	if stored.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if stored.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestRawMessageRepository_AppendMessage_GroupsByRunID(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	runID := "run-grouping-test"

	msgs := []m.RetainedMessage{
		{Subject: "tests.events.v1.suite.begin", EventType: "suite.begin", Payload: []byte(`{}`)},
		{Subject: "tests.events.v1.test.begin", EventType: "test.begin", Payload: []byte(`{}`)},
		{Subject: "tests.events.v1.test.end", EventType: "test.end", Payload: []byte(`{}`)},
	}

	for _, msg := range msgs {
		if err := repo.AppendMessage(ctx, runID, msg); err != nil {
			t.Fatalf("AppendMessage() error = %v", err)
		}
	}

	// All three messages should be in ONE document.
	count, err := repo.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("CountDocuments() error = %v", err)
	}
	if count != 1 {
		t.Errorf("document count = %d, want 1 (all messages for a run in one doc)", count)
	}

	var stored m.RawMessagesRunDocument
	if err := repo.collection.FindOne(ctx, bson.M{"_id": runID}).Decode(&stored); err != nil {
		t.Fatalf("FindOne() error = %v", err)
	}
	if len(stored.Messages) != len(msgs) {
		t.Errorf("Messages count = %d, want %d", len(stored.Messages), len(msgs))
	}
}

func TestRawMessageRepository_AppendMessage_SeparateRunsSeparateDocuments(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	runIDs := []string{"run-A", "run-B", "run-C"}
	for _, runID := range runIDs {
		msg := m.RetainedMessage{
			Subject:   "tests.events.v1.test.begin",
			EventType: "test.begin",
			Payload:   []byte(`{}`),
		}
		if err := repo.AppendMessage(ctx, runID, msg); err != nil {
			t.Fatalf("AppendMessage(%q) error = %v", runID, err)
		}
	}

	count, err := repo.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("CountDocuments() error = %v", err)
	}
	if int(count) != len(runIDs) {
		t.Errorf("document count = %d, want %d (one per run)", count, len(runIDs))
	}
}

func TestRawMessageRepository_AppendMessage_SetsReceivedAt(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// MongoDB stores times with millisecond precision; truncate before comparison.
	before := time.Now().Truncate(time.Millisecond)
	msg := m.RetainedMessage{
		Subject:   "tests.events.v1.test.end",
		EventType: "test.end",
		Payload:   []byte(`{}`),
		// ReceivedAt deliberately zero; should be auto-populated.
	}

	if err := repo.AppendMessage(ctx, "run-time-test", msg); err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}
	after := time.Now()

	var stored m.RawMessagesRunDocument
	if err := repo.collection.FindOne(ctx, bson.M{"_id": "run-time-test"}).Decode(&stored); err != nil {
		t.Fatalf("FindOne() error = %v", err)
	}
	if len(stored.Messages) == 0 {
		t.Fatal("expected at least one message")
	}
	got := stored.Messages[0].ReceivedAt
	if got.Before(before) || got.After(after) {
		t.Errorf("ReceivedAt = %v, want in [%v, %v]", got, before, after)
	}
}

func TestRawMessageRepository_AppendMessage_EmptyRunID(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	msg := m.RetainedMessage{EventType: "test.begin", Payload: []byte(`{}`)}
	if err := repo.AppendMessage(ctx, "", msg); err == nil {
		t.Fatal("AppendMessage with empty runID should return an error")
	}
}

func TestRawMessageRepository_Accessors(t *testing.T) {
	repo, cleanup := setupRawMessageTestRepo(t)
	defer cleanup()

	if repo.CollectionName() != "raw_messages" {
		t.Errorf("CollectionName() = %q, want %q", repo.CollectionName(), "raw_messages")
	}
	if repo.DatabaseName() == "" {
		t.Error("DatabaseName() should not be empty")
	}
}
