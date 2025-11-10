package server

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventServer struct {
	observer.UnimplementedTestEventCollectorServer
	logger *slog.Logger
	db     *gorm.DB
}

// New returns a new EventServer. If logger is nil, a no-op logger is used.
func New(logger *slog.Logger, db *gorm.DB) *EventServer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &EventServer{logger: logger, db: db}
}

// noopWriter implements io.Writer but drops logs when no logger provided.
type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

func validateTestID(id string) error {
	if id == "" {
		return errors.New("test_id is required")
	}
	return nil
}

func (s *EventServer) ReportTestBegin(ctx context.Context, in *events.TestBeginEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.TestCase == nil {
		return nil, status.Error(codes.InvalidArgument, "test_case is required")
	}
	if err := validateTestID(in.TestCase.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.logger.Info("test start", "run_id", in.TestCase.RunId, "title", in.TestCase.Title, "metadata_count", len(in.TestCase.Metadata))

	// Persist or update TestCase if DB is configured.
	if s.db != nil {
		// Convert metadata map[string]string to datatypes.JSONMap (map[string]any)
		md := map[string]any{}
		for k, v := range in.TestCase.Metadata {
			md[k] = v
		}
		tc := &m.TestCaseRun{
			RunID:    in.TestCase.RunId,
			Title:    in.TestCase.Title,
			Metadata: md,
			ID:       in.TestCase.Id,
		}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"title", "metadata", "updated_at"}),
		}).Create(tc).Error; err != nil {
			s.logger.Error("persist test start failed", "id", in.TestCase.Id, "error", err)
			return nil, status.Error(codes.Internal, "database error")
		}
	}
	return &observer.AckResponse{Success: true, Message: "start received: " + in.TestCase.Id}, nil
}

func (s *EventServer) ReportTestEnd(ctx context.Context, in *events.TestEndEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.TestCase == nil {
		return nil, status.Error(codes.InvalidArgument, "test_case is required")
	}
	if err := validateTestID(in.TestCase.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.logger.Info("test finish", "run_id", in.TestCase.RunId, "status", in.TestCase.Status)

	// Upsert status on finish if DB is configured.
	if s.db != nil {
		statusStr := in.TestCase.Status.String()
		tc := &m.TestCaseRun{
			ID:     in.TestCase.Id,
			RunID:  in.TestCase.RunId,
			Status: statusStr,
		}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
		}).Create(tc).Error; err != nil {
			s.logger.Error("persist test end failed", "id", in.TestCase.Id, "error", err)
			return nil, status.Error(codes.Internal, "database error")
		}
	}
	return &observer.AckResponse{Success: true, Message: "finish received: " + in.TestCase.Id}, nil
}

func (s *EventServer) ReportStepBegin(ctx context.Context, in *events.StepBeginEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.Step == nil {
		return nil, status.Error(codes.InvalidArgument, "step is required")
	}
	if err := validateTestID(in.Step.TestCaseRunId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Logging limited fields (RunId); extend when proto adds step-specific identifiers.
	s.logger.Info("test step", "run_id", in.Step.TestCaseRunId)

	if s.db != nil {
		st := &m.StepRun{TestCaseRunID: in.Step.TestCaseRunId, Status: "RUNNING"}
		if err := s.db.WithContext(ctx).Create(st).Error; err != nil {
			s.logger.Error("persist step begin failed", "run_id", in.Step.TestCaseRunId, "error", err)
			return nil, status.Error(codes.Internal, "database error")
		}
	}
	return &observer.AckResponse{Success: true, Message: "step received: " + in.Step.TestCaseRunId}, nil
}

func (s *EventServer) ReportStepEnd(ctx context.Context, in *events.StepEndEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.Step == nil {
		return nil, status.Error(codes.InvalidArgument, "step is required")
	}
	if err := validateTestID(in.Step.TestCaseRunId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Logging limited fields (RunId); extend when proto adds step-specific identifiers.
	s.logger.Info("test step end", "run_id", in.Step.TestCaseRunId, "status", in.Step.Status)

	if s.db != nil {
		// Make read+update atomic to avoid races among concurrent step-end reports.
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var step m.StepRun
			// Lock the latest step row for this test case.
			q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("test_case_run_id = ?", in.Step.TestCaseRunId).
				Order("created_at DESC").
				Limit(1).Take(&step)
			if q.Error != nil {
				if errors.Is(q.Error, gorm.ErrRecordNotFound) {
					// No step row exists; create one inside the tx.
					st := &m.StepRun{TestCaseRunID: in.Step.TestCaseRunId, Status: in.Step.Status.String()}
					if err := tx.Create(st).Error; err != nil {
						return err
					}
					return nil
				}
				return q.Error
			}
			// Update the locked row.
			if err := tx.Model(&m.StepRun{}).Where("id = ?", step.ID).Update("status", in.Step.Status.String()).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			s.logger.Error("persist step end failed", "run_id", in.Step.TestCaseRunId, "error", err)
			return nil, status.Error(codes.Internal, "database error")
		}
	}
	return &observer.AckResponse{Success: true, Message: "step end received: " + in.Step.TestCaseRunId}, nil
}

// Note: timestamp conversion helpers will be added when timestamp fields are persisted.

// RegisterServices keeps backward compatibility; returns the created server for further customization in callers.
func RegisterServices(s *grpc.Server, logger *slog.Logger, db *gorm.DB) *EventServer {
	srv := New(logger, db)
	observer.RegisterTestEventCollectorServer(s, srv)
	return srv
}

// loggingInterceptor provides basic structured logging for unary calls.
func loggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()
		p, _ := peer.FromContext(ctx)
		resp, err = handler(ctx, req)
		dur := time.Since(start)
		attrs := []any{"method", info.FullMethod, "duration_ms", dur.Milliseconds()}
		if p != nil {
			attrs = append(attrs, "peer", p.Addr.String())
		}
		if err != nil {
			st, _ := status.FromError(err)
			attrs = append(attrs, "code", st.Code(), "error", st.Message())
			logger.Error("rpc", attrs...)
		} else {
			attrs = append(attrs, "code", codes.OK)
			logger.Info("rpc", attrs...)
		}
		return resp, err
	}
}

// recoveryInterceptor converts panics into Internal errors and logs stack trace.
func recoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// NewGRPCServer constructs a gRPC server with standard interceptors.
func NewGRPCServer(logger *slog.Logger, opts ...grpc.ServerOption) *grpc.Server {
	chain := grpc.ChainUnaryInterceptor(
		recoveryInterceptor(logger),
		loggingInterceptor(logger),
	)
	opts = append(opts, chain)
	return grpc.NewServer(opts...)
}
