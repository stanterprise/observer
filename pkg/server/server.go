package server

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/stanterprise/observer/pkg/publisher"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type EventServer struct {
	observer.UnimplementedTestEventCollectorServer
	logger    *slog.Logger
	publisher *publisher.NATSPublisher
}

// New returns a new EventServer. If logger is nil, a no-op logger is used.
func New(logger *slog.Logger) *EventServer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &EventServer{logger: logger, publisher: nil}
}

// NewWithPublisher returns a new EventServer with NATS publisher support.
// If logger is nil, a no-op logger is used. The publisher parameter is optional.
func NewWithPublisher(logger *slog.Logger, pub *publisher.NATSPublisher) *EventServer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &EventServer{logger: logger, publisher: pub}
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
	s.logger.Info("test start", "run_id", in.TestCase.RunId, "title", in.TestCase.Name, "metadata_count", len(in.TestCase.Metadata))

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeTestBegin, in); err != nil {
			s.logger.Error("publish to NATS failed", "id", in.TestCase.Id, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Test begin received"}, nil
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
	s.logger.Info("test end", "id", in.TestCase.Id, "status", in.TestCase.Status)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeTestEnd, in); err != nil {
			s.logger.Error("publish to NATS failed", "id", in.TestCase.Id, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Test end received"}, nil
}

func (s *EventServer) ReportStepBegin(ctx context.Context, in *events.StepBeginEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.Step == nil {
		return nil, status.Error(codes.InvalidArgument, "step is required")
	}
	if in.Step.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "step.id is required")
	}
	s.logger.Info("step start", "id", in.Step.Id, "title", in.Step.Title, "category", in.Step.Category)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeStepBegin, in); err != nil {
			s.logger.Error("publish to NATS failed", "id", in.Step.Id, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Step begin received"}, nil
}

func (s *EventServer) ReportStepEnd(ctx context.Context, in *events.StepEndEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.Step == nil {
		return nil, status.Error(codes.InvalidArgument, "step is required")
	}
	if in.Step.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "step.id is required")
	}
	s.logger.Info("step end", "id", in.Step.Id, "status", in.Step.Status)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeStepEnd, in); err != nil {
			s.logger.Error("publish to NATS failed", "id", in.Step.Id, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Step end received"}, nil
}

func (s *EventServer) ReportSuiteBegin(ctx context.Context, in *events.SuiteBeginEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.Suite == nil {
		return nil, status.Error(codes.InvalidArgument, "suite is required")
	}
	if in.Suite.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "suite.id is required")
	}
	s.logger.Info("suite start", "id", in.Suite.Id, "name", in.Suite.Name)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeSuiteBegin, in); err != nil {
			s.logger.Error("publish to NATS failed", "id", in.Suite.Id, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Suite begin received"}, nil
}

func (s *EventServer) ReportSuiteEnd(ctx context.Context, in *events.SuiteEndEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.Suite == nil {
		return nil, status.Error(codes.InvalidArgument, "suite is required")
	}
	if in.Suite.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "suite.id is required")
	}
	s.logger.Info("suite end", "id", in.Suite.Id, "status", in.Suite.Status)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeSuiteEnd, in); err != nil {
			s.logger.Error("publish to NATS failed", "id", in.Suite.Id, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Suite end received"}, nil
}

func (s *EventServer) ReportTestFailure(ctx context.Context, in *events.TestFailureEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.TestId == "" {
		return nil, status.Error(codes.InvalidArgument, "test_id is required")
	}
	s.logger.Info("test failure", "test_id", in.TestId, "message", in.FailureMessage)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeTestFailure, in); err != nil {
			s.logger.Error("publish to NATS failed", "test_id", in.TestId, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Test failure received"}, nil
}

func (s *EventServer) ReportTestError(ctx context.Context, in *events.TestErrorEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.TestId == "" {
		return nil, status.Error(codes.InvalidArgument, "test_id is required")
	}
	s.logger.Info("test error", "test_id", in.TestId, "message", in.ErrorMessage)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeTestError, in); err != nil {
			s.logger.Error("publish to NATS failed", "test_id", in.TestId, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Test error received"}, nil
}

func (s *EventServer) ReportStdError(ctx context.Context, in *events.StdErrorEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.TestId == "" {
		return nil, status.Error(codes.InvalidArgument, "test_id is required")
	}
	s.logger.Info("stderr", "test_id", in.TestId, "length", len(in.Message))

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeStdError, in); err != nil {
			s.logger.Error("publish to NATS failed", "test_id", in.TestId, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Stderr received"}, nil
}

func (s *EventServer) ReportStdOutput(ctx context.Context, in *events.StdOutputEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.TestId == "" {
		return nil, status.Error(codes.InvalidArgument, "test_id is required")
	}
	s.logger.Info("stdout", "test_id", in.TestId, "length", len(in.Message))

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeStdOutput, in); err != nil {
			s.logger.Error("publish to NATS failed", "test_id", in.TestId, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Stdout received"}, nil
}

func (s *EventServer) ReportRunEnd(ctx context.Context, in *events.TestRunEndEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.RunId == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	s.logger.Info("run end", "run_id", in.RunId, "status", in.FinalStatus)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeRunEnd, in); err != nil {
			s.logger.Error("publish to NATS failed", "run_id", in.RunId, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Run end received"}, nil
}

func (s *EventServer) Heartbeat(ctx context.Context, in *events.HeartbeatEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.SourceId == "" {
		return nil, status.Error(codes.InvalidArgument, "source_id is required")
	}
	s.logger.Debug("heartbeat", "source_id", in.SourceId, "timestamp", in.Timestamp)

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.EventTypeHeartbeat, in); err != nil {
			s.logger.Error("publish to NATS failed", "source_id", in.SourceId, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Heartbeat received"}, nil
}

func (s *EventServer) MapTestRun(ctx context.Context, in *events.MapTestRunEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if in.RunId == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	if len(in.TestSuites) == 0 {
		return nil, status.Error(codes.InvalidArgument, "test_suites is required")
	}
	s.logger.Info("map test run", "run_id", in.RunId, "suite_count", len(in.TestSuites))

	// Publish to NATS if publisher is configured
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, publisher.MapSuitesEvent, in); err != nil {
			s.logger.Error("publish to NATS failed", "run_id", in.RunId, "error", err)
			return nil, status.Error(codes.Internal, "failed to publish event")
		}
	}

	return &observer.AckResponse{Success: true, Message: "Map test run received"}, nil
}

// NewGRPCServer creates a gRPC server with panic recovery and logging interceptors.
func NewGRPCServer(logger *slog.Logger) *grpc.Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(logger),
			loggingInterceptor(logger),
		),
	)
}

// recoveryInterceptor catches panics in gRPC handlers and returns Internal error.
func recoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic in grpc handler",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// loggingInterceptor logs all gRPC calls with duration and status.
func loggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		peer, _ := peer.FromContext(ctx)
		logger.Info("grpc call",
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
			"status", code.String(),
			"peer", peer.Addr.String())

		return resp, err
	}
}

// RegisterServices registers the gRPC services without database or publisher.
// Kept for backward compatibility with ingestion service.
func RegisterServices(s *grpc.Server, logger *slog.Logger, _ interface{}) *EventServer {
	srv := New(logger)
	observer.RegisterTestEventCollectorServer(s, srv)

	// Register health service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthServer)

	// Register reflection for grpcurl
	reflection.Register(s)

	return srv
}

// RegisterServicesWithPublisher registers the gRPC services with NATS publisher support.
func RegisterServicesWithPublisher(s *grpc.Server, logger *slog.Logger, _ interface{}, pub *publisher.NATSPublisher) *EventServer {
	srv := NewWithPublisher(logger, pub)
	observer.RegisterTestEventCollectorServer(s, srv)

	// Register health service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthServer)

	// Register reflection for grpcurl
	reflection.Register(s)

	return srv
}
