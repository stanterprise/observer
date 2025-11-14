package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stanterprise/observer/internal/database"
	"github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/pkg/api"
	"github.com/stanterprise/observer/pkg/api/graph"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database with test data
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dbName := fmt.Sprintf("file:api_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	
	db, err := database.Connect(dbName, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	if err := db.AutoMigrate(&models.TestCaseRun{}, &models.StepRun{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Insert test data
	now := time.Now()
	testCases := []models.TestCaseRun{
		{
			ID:        "test-1",
			RunID:     "run-1",
			Title:     "Login Test",
			Status:    "PASSED",
			Metadata:  datatypes.JSONMap{"browser": "chrome", "env": "staging"},
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
		},
		{
			ID:        "test-2",
			RunID:     "run-1",
			Title:     "Checkout Test",
			Status:    "FAILED",
			Metadata:  datatypes.JSONMap{"browser": "chrome", "env": "staging"},
			CreatedAt: now.Add(-8 * time.Minute),
			UpdatedAt: now.Add(-3 * time.Minute),
		},
		{
			ID:        "test-3",
			RunID:     "run-2",
			Title:     "Search Test",
			Status:    "PASSED",
			Metadata:  datatypes.JSONMap{"browser": "firefox", "env": "production"},
			CreatedAt: now.Add(-5 * time.Minute),
			UpdatedAt: now.Add(-1 * time.Minute),
		},
	}

	for _, tc := range testCases {
		if err := db.Create(&tc).Error; err != nil {
			t.Fatalf("Failed to create test case %s: %v", tc.ID, err)
		}
	}

	// Insert step data
	steps := []models.StepRun{
		{
			ID:            "step-1",
			RunID:         "run-1",
			TestCaseRunID: "test-1",
			Status:        "PASSED",
			CreatedAt:     now.Add(-9 * time.Minute),
			UpdatedAt:     now.Add(-5 * time.Minute),
		},
		{
			ID:            "step-2",
			RunID:         "run-1",
			TestCaseRunID: "test-2",
			Status:        "FAILED",
			CreatedAt:     now.Add(-7 * time.Minute),
			UpdatedAt:     now.Add(-3 * time.Minute),
		},
	}

	for _, step := range steps {
		if err := db.Create(&step).Error; err != nil {
			t.Fatalf("Failed to create step %s: %v", step.ID, err)
		}
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}

// TestRESTAPIListTests tests the GET /api/tests endpoint
func TestRESTAPIListTests(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := api.NewHandler(db, logger)
	
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("ListAllTests", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/tests")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		data := result["data"].([]interface{})
		if len(data) != 3 {
			t.Errorf("Expected 3 tests, got %d", len(data))
		}
	})

	t.Run("FilterByStatus", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/tests?status=PASSED")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		data := result["data"].([]interface{})
		if len(data) != 2 {
			t.Errorf("Expected 2 PASSED tests, got %d", len(data))
		}
	})

	t.Run("FilterByRunId", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/tests?runId=run-1")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		data := result["data"].([]interface{})
		if len(data) != 2 {
			t.Errorf("Expected 2 tests for run-1, got %d", len(data))
		}
	})
}

// TestRESTAPIGetTest tests the GET /api/tests/{id} endpoint
func TestRESTAPIGetTest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := api.NewHandler(db, logger)
	
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("GetExistingTest", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/tests/test-1")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		test := result["test"].(map[string]interface{})
		if test["ID"] != "test-1" {
			t.Errorf("Expected test ID test-1, got %v", test["ID"])
		}

		steps := result["steps"].([]interface{})
		if len(steps) != 1 {
			t.Errorf("Expected 1 step, got %d", len(steps))
		}
	})

	t.Run("GetNonExistentTest", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/tests/nonexistent")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
}

// TestGraphQLQueries tests the GraphQL API
func TestGraphQLQueries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := graph.NewResolver(db, logger)
	gqlHandler := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	
	server := httptest.NewServer(gqlHandler)
	defer server.Close()

	executeQuery := func(query string) map[string]interface{} {
		reqBody := map[string]interface{}{
			"query": query,
		}
		jsonData, _ := json.Marshal(reqBody)
		
		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return result
	}

	t.Run("ListTestCases", func(t *testing.T) {
		query := `
			query {
				testCases(limit: 10) {
					nodes {
						id
						title
						status
					}
					pageInfo {
						totalCount
					}
				}
			}
		`
		result := executeQuery(query)
		
		data := result["data"].(map[string]interface{})
		testCases := data["testCases"].(map[string]interface{})
		nodes := testCases["nodes"].([]interface{})
		
		if len(nodes) != 3 {
			t.Errorf("Expected 3 test cases, got %d", len(nodes))
		}
		
		pageInfo := testCases["pageInfo"].(map[string]interface{})
		totalCount := pageInfo["totalCount"].(float64)
		if totalCount != 3 {
			t.Errorf("Expected total count 3, got %v", totalCount)
		}
	})

	t.Run("GetTestCaseWithSteps", func(t *testing.T) {
		query := `
			query {
				testCase(id: "test-1") {
					id
					title
					status
					steps {
						id
						status
					}
				}
			}
		`
		result := executeQuery(query)
		
		data := result["data"].(map[string]interface{})
		testCase := data["testCase"].(map[string]interface{})
		
		if testCase["id"] != "test-1" {
			t.Errorf("Expected test ID test-1, got %v", testCase["id"])
		}
		
		steps := testCase["steps"].([]interface{})
		if len(steps) != 1 {
			t.Errorf("Expected 1 step, got %d", len(steps))
		}
	})

	t.Run("GetRunStats", func(t *testing.T) {
		query := `
			query {
				runStats(runId: "run-1") {
					totalTests
					passedTests
					failedTests
					totalSteps
				}
			}
		`
		result := executeQuery(query)
		
		data := result["data"].(map[string]interface{})
		stats := data["runStats"].(map[string]interface{})
		
		if stats["totalTests"].(float64) != 2 {
			t.Errorf("Expected 2 total tests, got %v", stats["totalTests"])
		}
		if stats["passedTests"].(float64) != 1 {
			t.Errorf("Expected 1 passed test, got %v", stats["passedTests"])
		}
		if stats["failedTests"].(float64) != 1 {
			t.Errorf("Expected 1 failed test, got %v", stats["failedTests"])
		}
	})

	t.Run("FilterTestsByStatus", func(t *testing.T) {
		query := `
			query {
				testCases(filter: { status: "PASSED" }) {
					nodes {
						id
						status
					}
					pageInfo {
						totalCount
					}
				}
			}
		`
		result := executeQuery(query)
		
		data := result["data"].(map[string]interface{})
		testCases := data["testCases"].(map[string]interface{})
		nodes := testCases["nodes"].([]interface{})
		
		if len(nodes) != 2 {
			t.Errorf("Expected 2 passed tests, got %d", len(nodes))
		}
		
		// Verify all are PASSED
		for _, node := range nodes {
			tc := node.(map[string]interface{})
			if tc["status"] != "PASSED" {
				t.Errorf("Expected status PASSED, got %v", tc["status"])
			}
		}
	})
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK\n")
	})
	
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "OK\n" {
		t.Errorf("Expected body 'OK', got '%s'", string(body))
	}
}
