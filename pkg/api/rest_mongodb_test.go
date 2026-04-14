package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	mongoRepo "github.com/stanterprise/observer/internal/repository/mongodb"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// testHandlerSetup holds test handler and collection for test data setup
type testHandlerSetup struct {
	handler    *MongoHandler
	collection *mongo.Collection
	cleanup    func()
}

// setupTestHandler creates a test handler with a MongoDB testcontainer
func setupTestHandler(t *testing.T) *testHandlerSetup {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := mongodb.RunContainer(ctx, testcontainers.WithImage("mongo:7.0"))
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		mongoContainer.Terminate(ctx)
		t.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		mongoContainer.Terminate(ctx)
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	dbName := "observer_api_test_" + time.Now().Format("20060102150405")
	collection := client.Database(dbName).Collection("test_runs")
	repo := mongoRepo.NewMongoRepository(collection, nil)
	handler := NewMongoHandler(repo, nil)

	cleanup := func() {
		client.Database(dbName).Drop(context.Background())
		client.Disconnect(context.Background())
		mongoContainer.Terminate(context.Background())
	}

	return &testHandlerSetup{
		handler:    handler,
		collection: collection,
		cleanup:    cleanup,
	}
}

func TestHandleMarkers_EmptyDatabase(t *testing.T) {
	setup := setupTestHandler(t)
	defer setup.cleanup()

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/markers", nil)
	rec := httptest.NewRecorder()

	// Call handler
	setup.handler.handleMarkers(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify structure
	markersField, exists := response["markers"]
	if !exists {
		t.Fatal("Response should have 'markers' field")
	}

	// markers can be nil (null in JSON) or empty array
	var markerCount int
	if markersField == nil {
		markerCount = 0
	} else {
		markers, ok := markersField.([]interface{})
		if !ok {
			t.Fatal("Response 'markers' should be array or null")
		}
		markerCount = len(markers)
	}

	if markerCount != 0 {
		t.Errorf("Expected 0 markers in empty database, got %d", markerCount)
	}

	count, ok := response["count"].(float64)
	if !ok {
		t.Error("Response should have 'count' field")
	}
	if int(count) != 0 {
		t.Errorf("Expected count to be 0, got %d", int(count))
	}
}

func TestHandleMarkers_WithData(t *testing.T) {
	setup := setupTestHandler(t)
	defer setup.cleanup()

	ctx := context.Background()
	now := time.Now()

	// Insert test data with markers
	testDocs := []*m.TestRunDocument{
		{
			ID:        "run-001",
			Name:      "Release Test 1",
			Status:    "completed",
			CreatedAt: now,
			UpdatedAt: now,
			Metadata: map[string]interface{}{
				"MARKER": "release-1.0",
			},
			Tests:  []*m.TestDocument{},
			Suites: []*m.SuiteDocument{},
		},
		{
			ID:        "run-002",
			Name:      "Release Test 2",
			Status:    "completed",
			CreatedAt: now.Add(1 * time.Hour),
			UpdatedAt: now.Add(1 * time.Hour),
			Metadata: map[string]interface{}{
				"MARKER": "release-1.0",
			},
			Tests:  []*m.TestDocument{},
			Suites: []*m.SuiteDocument{},
		},
		{
			ID:        "run-003",
			Name:      "Nightly Test",
			Status:    "completed",
			CreatedAt: now.Add(2 * time.Hour),
			UpdatedAt: now.Add(2 * time.Hour),
			Metadata: map[string]interface{}{
				"MARKER": "nightly",
			},
			Tests:  []*m.TestDocument{},
			Suites: []*m.SuiteDocument{},
		},
		{
			ID:        "run-004",
			Name:      "No Marker Test",
			Status:    "completed",
			CreatedAt: now.Add(3 * time.Hour),
			UpdatedAt: now.Add(3 * time.Hour),
			Metadata: map[string]interface{}{
				"environment": "staging",
			},
			Tests:  []*m.TestDocument{},
			Suites: []*m.SuiteDocument{},
		},
	}

	for _, doc := range testDocs {
		// Insert directly to the collection
		if _, err := setup.collection.InsertOne(ctx, doc); err != nil {
			t.Fatalf("Failed to insert test document: %v", err)
		}
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/markers", nil)
	rec := httptest.NewRecorder()

	// Call handler
	setup.handler.handleMarkers(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify structure
	markers, ok := response["markers"].([]interface{})
	if !ok {
		t.Fatal("Response should have 'markers' array")
	}

	// Should have 2 unique markers (release-1.0 and nightly)
	if len(markers) != 2 {
		t.Errorf("Expected 2 markers, got %d", len(markers))
	}

	count, ok := response["count"].(float64)
	if !ok {
		t.Error("Response should have 'count' field")
	}
	if int(count) != 2 {
		t.Errorf("Expected count to be 2, got %d", int(count))
	}

	// Verify first marker (should be release-1.0 with count 2)
	if len(markers) >= 1 {
		firstMarker := markers[0].(map[string]interface{})
		if firstMarker["marker"] != "release-1.0" {
			t.Errorf("Expected first marker to be 'release-1.0', got '%v'", firstMarker["marker"])
		}
		if int(firstMarker["count"].(float64)) != 2 {
			t.Errorf("Expected first marker count to be 2, got %v", firstMarker["count"])
		}
	}

	// Verify second marker (should be nightly with count 1)
	if len(markers) >= 2 {
		secondMarker := markers[1].(map[string]interface{})
		if secondMarker["marker"] != "nightly" {
			t.Errorf("Expected second marker to be 'nightly', got '%v'", secondMarker["marker"])
		}
		if int(secondMarker["count"].(float64)) != 1 {
			t.Errorf("Expected second marker count to be 1, got %v", secondMarker["count"])
		}
	}
}

func TestHandleMarkers_MethodNotAllowed(t *testing.T) {
	setup := setupTestHandler(t)
	defer setup.cleanup()

	// Test POST method (should return 405)
	req := httptest.NewRequest(http.MethodPost, "/api/markers", nil)
	rec := httptest.NewRecorder()

	setup.handler.handleMarkers(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for POST, got %d", rec.Code)
	}

	// Test PUT method (should return 405)
	req = httptest.NewRequest(http.MethodPut, "/api/markers", nil)
	rec = httptest.NewRecorder()

	setup.handler.handleMarkers(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for PUT, got %d", rec.Code)
	}
}

func TestHandleMarkers_ContentType(t *testing.T) {
	setup := setupTestHandler(t)
	defer setup.cleanup()

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/markers", nil)
	rec := httptest.NewRecorder()

	// Call handler
	setup.handler.handleMarkers(rec, req)

	// Verify Content-Type header
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}
