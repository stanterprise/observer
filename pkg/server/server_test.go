package server

import (
	"context"
	"log/slog"
	"os"
	"testing"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/proto-go/testsystem/v1/common"
	"github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&m.TestCaseRun{}, &m.StepRun{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)

	srv := New(logger, db)
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	if srv.logger == nil {
		t.Error("New() server has nil logger")
	}
	if srv.db == nil {
		t.Error("New() server has nil db")
	}
}

func TestNew_NilLogger(t *testing.T) {
	db := setupTestDB(t)
	srv := New(nil, db)
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	if srv.logger == nil {
		t.Error("New() should create a no-op logger when nil is passed")
	}
}

func TestValidateTestID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid id", "test-123", false},
		{"empty id", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTestID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTestID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReportTestBegin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	tests := []struct {
		name      string
		req       *events.TestBeginEventRequest
		wantCode  codes.Code
		wantAck   bool
		setupFunc func()
	}{
		{
			name: "valid request",
			req: &events.TestBeginEventRequest{
				TestCase: &entities.TestCaseRun{
					Id:       "test-1",
					RunId:    "run-1",
					Name:    "Test Case 1",
					Metadata: map[string]string{"key": "value"},
				},
			},
			wantCode: codes.OK,
			wantAck:  true,
		},
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
		{
			name: "nil test case",
			req: &events.TestBeginEventRequest{
				TestCase: nil,
			},
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
		{
			name: "empty test id",
			req: &events.TestBeginEventRequest{
				TestCase: &entities.TestCaseRun{
					Id:    "",
					RunId: "run-1",
					Name: "Test Case",
				},
			},
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			resp, err := srv.ReportTestBegin(ctx, tt.req)

			if tt.wantCode != codes.OK {
				if err == nil {
					t.Errorf("ReportTestBegin() expected error with code %v, got nil", tt.wantCode)
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("ReportTestBegin() error is not a status error: %v", err)
					return
				}
				if st.Code() != tt.wantCode {
					t.Errorf("ReportTestBegin() code = %v, want %v", st.Code(), tt.wantCode)
				}
				return
			}

			if err != nil {
				t.Errorf("ReportTestBegin() unexpected error: %v", err)
				return
			}
			if resp == nil {
				t.Error("ReportTestBegin() response is nil")
				return
			}
			if resp.Success != tt.wantAck {
				t.Errorf("ReportTestBegin() success = %v, want %v", resp.Success, tt.wantAck)
			}
		})
	}
}

func TestReportTestBegin_DBPersistence(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	req := &events.TestBeginEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:       "test-persist",
			RunId:    "run-persist",
			Name:    "Persistence Test",
			Metadata: map[string]string{"env": "test"},
		},
	}

	_, err := srv.ReportTestBegin(ctx, req)
	if err != nil {
		t.Fatalf("ReportTestBegin() error = %v", err)
	}

	var tc m.TestCaseRun
	if err := db.Where("id = ?", "test-persist").First(&tc).Error; err != nil {
		t.Fatalf("Failed to find persisted test case: %v", err)
	}

	if tc.Title != "Persistence Test" {
		t.Errorf("Persisted title = %v, want Persistence Test", tc.Title)
	}
	if tc.RunID != "run-persist" {
		t.Errorf("Persisted run_id = %v, want run-persist", tc.RunID)
	}
}

func TestReportTestEnd(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	tests := []struct {
		name     string
		req      *events.TestEndEventRequest
		wantCode codes.Code
		wantAck  bool
	}{
		{
			name: "valid request",
			req: &events.TestEndEventRequest{
				TestCase: &entities.TestCaseRun{
					Id:     "test-1",
					RunId:  "run-1",
					Status: common.TestStatus_PASSED,
				},
			},
			wantCode: codes.OK,
			wantAck:  true,
		},
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
		{
			name: "nil test case",
			req: &events.TestEndEventRequest{
				TestCase: nil,
			},
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
		{
			name: "empty test id",
			req: &events.TestEndEventRequest{
				TestCase: &entities.TestCaseRun{
					Id:     "",
					RunId:  "run-1",
					Status: common.TestStatus_PASSED,
				},
			},
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := srv.ReportTestEnd(ctx, tt.req)

			if tt.wantCode != codes.OK {
				if err == nil {
					t.Errorf("ReportTestEnd() expected error with code %v, got nil", tt.wantCode)
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("ReportTestEnd() error is not a status error: %v", err)
					return
				}
				if st.Code() != tt.wantCode {
					t.Errorf("ReportTestEnd() code = %v, want %v", st.Code(), tt.wantCode)
				}
				return
			}

			if err != nil {
				t.Errorf("ReportTestEnd() unexpected error: %v", err)
				return
			}
			if resp == nil {
				t.Error("ReportTestEnd() response is nil")
				return
			}
			if resp.Success != tt.wantAck {
				t.Errorf("ReportTestEnd() success = %v, want %v", resp.Success, tt.wantAck)
			}
		})
	}
}

func TestReportStepBegin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	tests := []struct {
		name     string
		req      *events.StepBeginEventRequest
		wantCode codes.Code
		wantAck  bool
	}{
		{
			name: "valid request",
			req: &events.StepBeginEventRequest{
				Step: &entities.StepRun{
					TestCaseRunId: "test-1",
				},
			},
			wantCode: codes.OK,
			wantAck:  true,
		},
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
		{
			name: "nil step",
			req: &events.StepBeginEventRequest{
				Step: nil,
			},
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
		{
			name: "empty test case run id",
			req: &events.StepBeginEventRequest{
				Step: &entities.StepRun{
					TestCaseRunId: "",
				},
			},
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := srv.ReportStepBegin(ctx, tt.req)

			if tt.wantCode != codes.OK {
				if err == nil {
					t.Errorf("ReportStepBegin() expected error with code %v, got nil", tt.wantCode)
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("ReportStepBegin() error is not a status error: %v", err)
					return
				}
				if st.Code() != tt.wantCode {
					t.Errorf("ReportStepBegin() code = %v, want %v", st.Code(), tt.wantCode)
				}
				return
			}

			if err != nil {
				t.Errorf("ReportStepBegin() unexpected error: %v", err)
				return
			}
			if resp == nil {
				t.Error("ReportStepBegin() response is nil")
				return
			}
			if resp.Success != tt.wantAck {
				t.Errorf("ReportStepBegin() success = %v, want %v", resp.Success, tt.wantAck)
			}
		})
	}
}

func TestReportStepEnd(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	tests := []struct {
		name     string
		req      *events.StepEndEventRequest
		wantCode codes.Code
		wantAck  bool
	}{
		{
			name: "valid request",
			req: &events.StepEndEventRequest{
				Step: &entities.StepRun{
					TestCaseRunId: "test-1",
					Status:        common.TestStatus_PASSED,
				},
			},
			wantCode: codes.OK,
			wantAck:  true,
		},
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
		{
			name: "nil step",
			req: &events.StepEndEventRequest{
				Step: nil,
			},
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
		{
			name: "empty test case run id",
			req: &events.StepEndEventRequest{
				Step: &entities.StepRun{
					TestCaseRunId: "",
					Status:        common.TestStatus_PASSED,
				},
			},
			wantCode: codes.InvalidArgument,
			wantAck:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := srv.ReportStepEnd(ctx, tt.req)

			if tt.wantCode != codes.OK {
				if err == nil {
					t.Errorf("ReportStepEnd() expected error with code %v, got nil", tt.wantCode)
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("ReportStepEnd() error is not a status error: %v", err)
					return
				}
				if st.Code() != tt.wantCode {
					t.Errorf("ReportStepEnd() code = %v, want %v", st.Code(), tt.wantCode)
				}
				return
			}

			if err != nil {
				t.Errorf("ReportStepEnd() unexpected error: %v", err)
				return
			}
			if resp == nil {
				t.Error("ReportStepEnd() response is nil")
				return
			}
			if resp.Success != tt.wantAck {
				t.Errorf("ReportStepEnd() success = %v, want %v", resp.Success, tt.wantAck)
			}
		})
	}
}

func TestReportStepEnd_UpdatesExistingStep(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	// Create initial step
	beginReq := &events.StepBeginEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-step-update",
		},
	}
	_, err := srv.ReportStepBegin(ctx, beginReq)
	if err != nil {
		t.Fatalf("ReportStepBegin() error = %v", err)
	}

	// End the step with status
	endReq := &events.StepEndEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-step-update",
			Status:        common.TestStatus_PASSED,
		},
	}
	_, err = srv.ReportStepEnd(ctx, endReq)
	if err != nil {
		t.Fatalf("ReportStepEnd() error = %v", err)
	}

	// Verify the step was updated
	var step m.StepRun
	if err := db.Where("test_case_run_id = ?", "test-step-update").First(&step).Error; err != nil {
		t.Fatalf("Failed to find step: %v", err)
	}

	if step.Status != "PASSED" {
		t.Errorf("Step status = %v, want PASSED", step.Status)
	}
}

func TestReportStepEnd_NoExistingStep(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	// Try to end a step that doesn't exist
	endReq := &events.StepEndEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-no-step",
			Status:        common.TestStatus_PASSED,
		},
	}
	_, err := srv.ReportStepEnd(ctx, endReq)
	if err != nil {
		t.Fatalf("ReportStepEnd() error = %v", err)
	}

	// Verify a step was created
	var step m.StepRun
	if err := db.Where("test_case_run_id = ?", "test-no-step").First(&step).Error; err != nil {
		t.Fatalf("Failed to find created step: %v", err)
	}

	if step.Status != "PASSED" {
		t.Errorf("Step status = %v, want PASSED", step.Status)
	}
}

func TestRegisterServices(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	grpcServer := NewGRPCServer(logger)

	srv := RegisterServices(grpcServer, logger, db)
	if srv == nil {
		t.Error("RegisterServices() returned nil")
	}
	if srv.logger == nil {
		t.Error("RegisterServices() server has nil logger")
	}
	if srv.db == nil {
		t.Error("RegisterServices() server has nil db")
	}
}

func TestNewGRPCServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := NewGRPCServer(logger)
	if srv == nil {
		t.Error("NewGRPCServer() returned nil")
	}
}

func TestNewGRPCServer_NilLogger(t *testing.T) {
	srv := NewGRPCServer(nil)
	if srv == nil {
		t.Error("NewGRPCServer() with nil logger returned nil")
	}
}

func TestReportTestBegin_NoDB(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger, nil)
	ctx := context.Background()

	req := &events.TestBeginEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:    "test-no-db",
			RunId: "run-no-db",
			Name: "Test Without DB",
		},
	}

	resp, err := srv.ReportTestBegin(ctx, req)
	if err != nil {
		t.Errorf("ReportTestBegin() without DB error = %v", err)
	}
	if resp == nil || !resp.Success {
		t.Error("ReportTestBegin() without DB should still succeed")
	}
}

func TestReportTestEnd_NoDB(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger, nil)
	ctx := context.Background()

	req := &events.TestEndEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:     "test-no-db",
			RunId:  "run-no-db",
			Status: common.TestStatus_PASSED,
		},
	}

	resp, err := srv.ReportTestEnd(ctx, req)
	if err != nil {
		t.Errorf("ReportTestEnd() without DB error = %v", err)
	}
	if resp == nil || !resp.Success {
		t.Error("ReportTestEnd() without DB should still succeed")
	}
}

func TestReportStepBegin_NoDB(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger, nil)
	ctx := context.Background()

	req := &events.StepBeginEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-no-db",
		},
	}

	resp, err := srv.ReportStepBegin(ctx, req)
	if err != nil {
		t.Errorf("ReportStepBegin() without DB error = %v", err)
	}
	if resp == nil || !resp.Success {
		t.Error("ReportStepBegin() without DB should still succeed")
	}
}

func TestReportStepEnd_NoDB(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger, nil)
	ctx := context.Background()

	req := &events.StepEndEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-no-db",
			Status:        common.TestStatus_PASSED,
		},
	}

	resp, err := srv.ReportStepEnd(ctx, req)
	if err != nil {
		t.Errorf("ReportStepEnd() without DB error = %v", err)
	}
	if resp == nil || !resp.Success {
		t.Error("ReportStepEnd() without DB should still succeed")
	}
}

func TestNoopWriter(t *testing.T) {
	w := &noopWriter{}
	data := []byte("test data")
	n, err := w.Write(data)
	if err != nil {
		t.Errorf("noopWriter.Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("noopWriter.Write() n = %v, want %v", n, len(data))
	}
}

// Mock DB error test
func TestReportTestBegin_DBError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)

	// Close the DB to force an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	srv := New(logger, db)
	ctx := context.Background()

	req := &events.TestBeginEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:    "test-db-error",
			RunId: "run-db-error",
			Name: "Test DB Error",
		},
	}

	_, err := srv.ReportTestBegin(ctx, req)
	if err == nil {
		t.Error("ReportTestBegin() with closed DB should return error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("ReportTestBegin() error is not a status error: %v", err)
		return
	}
	if st.Code() != codes.Internal {
		t.Errorf("ReportTestBegin() with DB error code = %v, want %v", st.Code(), codes.Internal)
	}
}

func TestReportTestEnd_DBError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)

	// Close the DB to force an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	srv := New(logger, db)
	ctx := context.Background()

	req := &events.TestEndEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:     "test-db-error",
			RunId:  "run-db-error",
			Status: common.TestStatus_PASSED,
		},
	}

	_, err := srv.ReportTestEnd(ctx, req)
	if err == nil {
		t.Error("ReportTestEnd() with closed DB should return error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("ReportTestEnd() error is not a status error: %v", err)
		return
	}
	if st.Code() != codes.Internal {
		t.Errorf("ReportTestEnd() with DB error code = %v, want %v", st.Code(), codes.Internal)
	}
}

func TestReportStepBegin_DBError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)

	// Close the DB to force an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	srv := New(logger, db)
	ctx := context.Background()

	req := &events.StepBeginEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-db-error",
		},
	}

	_, err := srv.ReportStepBegin(ctx, req)
	if err == nil {
		t.Error("ReportStepBegin() with closed DB should return error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("ReportStepBegin() error is not a status error: %v", err)
		return
	}
	if st.Code() != codes.Internal {
		t.Errorf("ReportStepBegin() with DB error code = %v, want %v", st.Code(), codes.Internal)
	}
}

func TestReportStepEnd_DBTransactionError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)

	// Close the DB to force an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	srv := New(logger, db)
	ctx := context.Background()

	req := &events.StepEndEventRequest{
		Step: &entities.StepRun{
			TestCaseRunId: "test-db-error",
			Status:        common.TestStatus_PASSED,
		},
	}

	_, err := srv.ReportStepEnd(ctx, req)
	if err == nil {
		t.Error("ReportStepEnd() with closed DB should return error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("ReportStepEnd() error is not a status error: %v", err)
		return
	}
	if st.Code() != codes.Internal {
		t.Errorf("ReportStepEnd() with DB error code = %v, want %v", st.Code(), codes.Internal)
	}
}

// Test upsert behavior
func TestReportTestBegin_Upsert(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	// First insert
	req1 := &events.TestBeginEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:       "test-upsert",
			RunId:    "run-1",
			Name:    "Original Title",
			Metadata: map[string]string{"version": "1"},
		},
	}
	_, err := srv.ReportTestBegin(ctx, req1)
	if err != nil {
		t.Fatalf("First ReportTestBegin() error = %v", err)
	}

	// Update with same ID
	req2 := &events.TestBeginEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:       "test-upsert",
			RunId:    "run-1",
			Name:    "Updated Title",
			Metadata: map[string]string{"version": "2"},
		},
	}
	_, err = srv.ReportTestBegin(ctx, req2)
	if err != nil {
		t.Fatalf("Second ReportTestBegin() error = %v", err)
	}

	// Verify only one record exists with updated values
	var count int64
	db.Model(&m.TestCaseRun{}).Where("id = ?", "test-upsert").Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 record, got %v", count)
	}

	var tc m.TestCaseRun
	db.Where("id = ?", "test-upsert").First(&tc)
	if tc.Title != "Updated Title" {
		t.Errorf("Title = %v, want Updated Title", tc.Title)
	}
}

func TestReportTestEnd_Upsert(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := setupTestDB(t)
	srv := New(logger, db)
	ctx := context.Background()

	// First end
	req1 := &events.TestEndEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:     "test-end-upsert",
			RunId:  "run-1",
			Status: common.TestStatus_BROKEN,
		},
	}
	_, err := srv.ReportTestEnd(ctx, req1)
	if err != nil {
		t.Fatalf("First ReportTestEnd() error = %v", err)
	}

	// Second end with different status
	req2 := &events.TestEndEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:     "test-end-upsert",
			RunId:  "run-1",
			Status: common.TestStatus_PASSED,
		},
	}
	_, err = srv.ReportTestEnd(ctx, req2)
	if err != nil {
		t.Fatalf("Second ReportTestEnd() error = %v", err)
	}

	// Verify status was updated
	var tc m.TestCaseRun
	db.Where("id = ?", "test-end-upsert").First(&tc)
	if tc.Status != "PASSED" {
		t.Errorf("Status = %v, want PASSED", tc.Status)
	}
}

// Test interceptors indirectly by verifying they don't panic
func TestInterceptors_NoError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test that creating a server with interceptors doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Creating gRPC server with interceptors panicked: %v", r)
		}
	}()

	srv := NewGRPCServer(logger)
	if srv == nil {
		t.Error("NewGRPCServer() returned nil")
	}
}
