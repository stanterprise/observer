#!/bin/bash
# Script to insert test data with tags into MongoDB for demonstration

# MongoDB connection details
MONGO_URI="mongodb://root:password@localhost:27017/observer?authSource=admin"

# Insert a test run with tags using mongosh
docker exec -i observer-mongodb mongosh "$MONGO_URI" <<'EOF'

// Clear existing data (optional)
db.test_runs.deleteMany({});

// Insert a test run with suite, tests, and steps - all with tags
db.test_runs.insertOne({
  "_id": "run-tags-demo-001",
  "name": "Tags Feature Demo Test Run",
  "description": "Demonstrating tags on all entities",
  "status": "PASSED",
  "created_at": new Date(),
  "updated_at": new Date(),
  "suites": [
    {
      "id": "suite-001",
      "run_id": "run-tags-demo-001",
      "name": "Authentication Suite",
      "description": "Tests for user authentication",
      "status": "PASSED",
      "tags": ["@auth", "@smoke", "@api"],
      "created_at": new Date(),
      "updated_at": new Date(),
      "tests": [
        {
          "id": "test-001",
          "name": "Login with valid credentials",
          "title": "Login with valid credentials",
          "run_id": "run-tags-demo-001",
          "suite_id": "suite-001",
          "status": "PASSED",
          "tags": ["@login", "@positive", "@critical"],
          "duration": 1500000000,
          "created_at": new Date(),
          "updated_at": new Date(),
          "steps": [
            {
              "id": "step-001",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-001",
              "title": "Navigate to login page",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@navigation", "@ui"],
              "duration": 500000000,
              "created_at": new Date(),
              "updated_at": new Date()
            },
            {
              "id": "step-002",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-001",
              "title": "Enter username and password",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@input", "@ui"],
              "duration": 300000000,
              "created_at": new Date(),
              "updated_at": new Date()
            },
            {
              "id": "step-003",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-001",
              "title": "Click login button",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@click", "@ui", "@interaction"],
              "duration": 700000000,
              "created_at": new Date(),
              "updated_at": new Date()
            }
          ]
        },
        {
          "id": "test-002",
          "name": "Login with invalid credentials",
          "title": "Login with invalid credentials",
          "run_id": "run-tags-demo-001",
          "suite_id": "suite-001",
          "status": "PASSED",
          "tags": ["@login", "@negative", "@error-handling"],
          "duration": 1200000000,
          "created_at": new Date(),
          "updated_at": new Date(),
          "steps": [
            {
              "id": "step-004",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-002",
              "title": "Navigate to login page",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@navigation", "@ui"],
              "duration": 500000000,
              "created_at": new Date(),
              "updated_at": new Date()
            },
            {
              "id": "step-005",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-002",
              "title": "Enter invalid credentials",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@input", "@ui", "@negative"],
              "duration": 300000000,
              "created_at": new Date(),
              "updated_at": new Date()
            },
            {
              "id": "step-006",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-002",
              "title": "Verify error message",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@assertion", "@ui", "@error"],
              "duration": 400000000,
              "created_at": new Date(),
              "updated_at": new Date()
            }
          ]
        }
      ]
    },
    {
      "id": "suite-002",
      "run_id": "run-tags-demo-001",
      "name": "API Testing Suite",
      "description": "Tests for REST API endpoints",
      "status": "PASSED",
      "tags": ["@api", "@integration", "@backend"],
      "created_at": new Date(),
      "updated_at": new Date(),
      "tests": [
        {
          "id": "test-003",
          "name": "GET /users endpoint",
          "title": "GET /users endpoint",
          "run_id": "run-tags-demo-001",
          "suite_id": "suite-002",
          "status": "PASSED",
          "tags": ["@api", "@get", "@users"],
          "duration": 800000000,
          "created_at": new Date(),
          "updated_at": new Date(),
          "steps": [
            {
              "id": "step-007",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-003",
              "title": "Send GET request",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@http", "@request"],
              "duration": 400000000,
              "created_at": new Date(),
              "updated_at": new Date()
            },
            {
              "id": "step-008",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-003",
              "title": "Verify response status",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@assertion", "@status-code"],
              "duration": 200000000,
              "created_at": new Date(),
              "updated_at": new Date()
            },
            {
              "id": "step-009",
              "run_id": "run-tags-demo-001",
              "test_case_run_id": "test-003",
              "title": "Verify response body",
              "status": "PASSED",
              "category": "test.step",
              "tags": ["@assertion", "@json", "@validation"],
              "duration": 200000000,
              "created_at": new Date(),
              "updated_at": new Date()
            }
          ]
        }
      ]
    }
  ]
});

print("Test data with tags inserted successfully!");
print("Run ID: run-tags-demo-001");

EOF

echo "Demo data inserted! View at http://localhost:3000/suite_runs/run-tags-demo-001"
