package server

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/pkg/publisher"
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
	logger    *slog.Logger
	db        *gorm.DB
	publisher *publisher.NATSPublisher
}

// New returns a new EventServer. If logger is nil, a no-op logger is used.
// The publisher parameter is optional and can be nil.
func New(logger *slog.Logger, db *gorm.DB) *EventServer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &EventServer{logger: logger, db: db, publisher: nil}
}

// NewWithPublisher returns a new EventServer with NATS publisher support.
// If logger is nil, a no-op logger is used. The publisher parameter is optional.
func NewWithPublisher(logger *slog.Logger, db *gorm.DB, pub *publisher.NATSPublisher) *EventServer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &EventServer{logger: logger, db: db, publisher: pub}
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

	// Publish to NATS if publisher is configured (Phase 1: dual-write)
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeTestBegin, in); err != nil {
			s.logger.Error("publish to NATS failed", "id", in.TestCase.Id, "error", err)
			// Continue with DB write even if NATS publish fails (best-effort)
		}
	}

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

	// Publish to NATS if publisher is configured (Phase 1: dual-write)
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeTestEnd, in); err != nil {
			s.logger.Error("publish to NATS failed", "id", in.TestCase.Id, "error", err)
			// Continue with DB write even if NATS publish fails (best-effort)
		}
	}

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

	// Publish to NATS if publisher is configured (Phase 1: dual-write)
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeStepBegin, in); err != nil {
			s.logger.Error("publish to NATS failed", "run_id", in.Step.TestCaseRunId, "error", err)
			// Continue with DB write even if NATS publish fails (best-effort)
		}
	}

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

	// Publish to NATS if publisher is configured (Phase 1: dual-write)
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeStepEnd, in); err != nil {
			s.logger.Error("publish to NATS failed", "run_id", in.Step.TestCaseRunId, "error", err)
			// Continue with DB write even if NATS publish fails (best-effort)
		}
	}

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

func (s *EventServer) ReportSuiteBegin(ctx context.Context, in *events.SuiteBeginEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.Suite == nil {
		return nil, status.Error(codes.InvalidArgument, "suite is required")
	}
	s.logger.Info("suite start", "suite_id", in.Suite.Id, "name", in.Suite.Name)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeSuiteBegin, in); err != nil {
			s.logger.Error("publish to NATS failed", "suite_id", in.Suite.Id, "error", err)
			// Continue even if NATS publish fails (best-effort)
		}
	}

	// Persist to DB if configured
	if s.db != nil {
		// Convert metadata map[string]string to datatypes.JSONMap (map[string]any)
		md := map[string]any{}
		for k, v := range in.Suite.Metadata {
			md[k] = v
		}

		var startTime *time.Time
		if in.Suite.StartTime != nil {
			t := in.Suite.StartTime.AsTime()
			startTime = &t
		}

		suite := &m.TestSuiteRun{
			ID:              in.Suite.Id,
			Name:            in.Suite.Name,
			Description:     in.Suite.Description,
			Metadata:        md,
			TestSuiteSpecID: in.Suite.TestSuiteSpecId,
			InitiatedBy:     in.Suite.InitiatedBy,
			StartTime:       startTime,
		}

		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "description", "metadata", "test_suite_spec_id", "initiated_by", "start_time", "updated_at"}),
		}).Create(suite).Error; err != nil {
			s.logger.Error("persist suite begin failed", "suite_id", in.Suite.Id, "error", err)
			return nil, status.Error(codes.Internal, "database error")
		}
	}

	return &observer.AckResponse{Success: true, Message: "suite begin received: " + in.Suite.Id}, nil
}

func (s *EventServer) ReportSuiteEnd(ctx context.Context, in *events.SuiteEndEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.Suite == nil {
		return nil, status.Error(codes.InvalidArgument, "suite is required")
	}
	s.logger.Info("suite end", "suite_id", in.Suite.Id, "status", in.Suite.Status)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeSuiteEnd, in); err != nil {
			s.logger.Error("publish to NATS failed", "suite_id", in.Suite.Id, "error", err)
			// Continue even if NATS publish fails (best-effort)
		}
	}

	// Persist to DB if configured
	if s.db != nil {
		statusStr := in.Suite.Status.String()

		var endTime *time.Time
		if in.Suite.EndTime != nil {
			t := in.Suite.EndTime.AsTime()
			endTime = &t
		}

		suite := &m.TestSuiteRun{
			ID:      in.Suite.Id,
			Status:  statusStr,
			EndTime: endTime,
		}

		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "end_time", "updated_at"}),
		}).Create(suite).Error; err != nil {
			s.logger.Error("persist suite end failed", "suite_id", in.Suite.Id, "error", err)
			return nil, status.Error(codes.Internal, "database error")
		}
	}

	return &observer.AckResponse{Success: true, Message: "suite end received: " + in.Suite.Id}, nil
}

func (s *EventServer) ReportTestFailure(ctx context.Context, in *events.TestFailureEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if err := validateTestID(in.TestId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.logger.Info("test failure", "test_id", in.TestId, "message_len", len(in.FailureMessage))

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeTestFailure, in); err != nil {
			s.logger.Error("publish to NATS failed", "test_id", in.TestId, "error", err)
			// Continue even if NATS publish fails (best-effort)
		}
	}

	// Note: Database persistence for failures not yet implemented
	// Failure info will be available via WebSocket relay in real-time

	return &observer.AckResponse{Success: true, Message: "test failure received: " + in.TestId}, nil
}

func (s *EventServer) ReportTestError(ctx context.Context, in *events.TestErrorEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if err := validateTestID(in.TestId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.logger.Info("test error", "test_id", in.TestId, "message_len", len(in.ErrorMessage))

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeTestError, in); err != nil {
			s.logger.Error("publish to NATS failed", "test_id", in.TestId, "error", err)
			// Continue even if NATS publish fails (best-effort)
		}
	}

	// Note: Database persistence for errors not yet implemented
	// Error info will be available via WebSocket relay in real-time

	return &observer.AckResponse{Success: true, Message: "test error received: " + in.TestId}, nil
}

func (s *EventServer) ReportStdOutput(ctx context.Context, in *events.StdOutputEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if err := validateTestID(in.TestId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.logger.Debug("stdout", "test_id", in.TestId, "message_len", len(in.Message))

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeStdOutput, in); err != nil {
			s.logger.Error("publish to NATS failed", "test_id", in.TestId, "error", err)
			// Continue even if NATS publish fails (best-effort)
		}
	}

	// Note: stdout typically not persisted to DB due to volume
	// Available via WebSocket relay in real-time

	return &observer.AckResponse{Success: true, Message: "stdout received"}, nil
}

func (s *EventServer) ReportStdError(ctx context.Context, in *events.StdErrorEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if err := validateTestID(in.TestId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.logger.Debug("stderr", "test_id", in.TestId, "message_len", len(in.Message))

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeStdError, in); err != nil {
			s.logger.Error("publish to NATS failed", "test_id", in.TestId, "error", err)
			// Continue even if NATS publish fails (best-effort)
		}
	}

	// Note: stderr typically not persisted to DB due to volume
	// Available via WebSocket relay in real-time

	return &observer.AckResponse{Success: true, Message: "stderr received"}, nil
}

func (s *EventServer) Heartbeat(ctx context.Context, in *events.HeartbeatEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	s.logger.Debug("heartbeat", "source_id", in.SourceId)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeHeartbeat, in); err != nil {
			s.logger.Error("publish to NATS failed", "source_id", in.SourceId, "error", err)
			// Continue even if NATS publish fails (best-effort)
		}
	}

	// Note: Heartbeats typically not persisted to DB
	// Available for monitoring via WebSocket relay

	return &observer.AckResponse{Success: true, Message: "heartbeat received"}, nil
}

// Note: timestamp conversion helpers will be added when timestamp fields are persisted.

// RegisterServices keeps backward compatibility; returns the created server for further customization in callers.
func RegisterServices(s *grpc.Server, logger *slog.Logger, db *gorm.DB) *EventServer {
	srv := New(logger, db)
	observer.RegisterTestEventCollectorServer(s, srv)
	return srv
}

// RegisterServicesWithPublisher registers the gRPC services with NATS publisher support.
func RegisterServicesWithPublisher(s *grpc.Server, logger *slog.Logger, db *gorm.DB, pub *publisher.NATSPublisher) *EventServer {
	srv := NewWithPublisher(logger, db, pub)
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
