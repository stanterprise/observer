package server

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"time"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type EventServer struct {
	observer.UnimplementedTestEventCollectorServer
	logger *slog.Logger
}

// New returns a new EventServer. If logger is nil, a no-op logger is used.
func New(logger *slog.Logger) *EventServer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
	}
	return &EventServer{logger: logger}
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
	if err := validateTestID(in.TestCase.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.logger.Info("test start", "test_id", in.TestCase.Id, "name", in.TestCase.Name, "metadata_count", len(in.TestCase.Metadata))
	return &observer.AckResponse{Success: true, Message: "start received: " + in.TestCase.Id}, nil
}

func (s *EventServer) ReportTestEnd(ctx context.Context, in *events.TestEndEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if err := validateTestID(in.TestCase.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.logger.Info("test finish", "test_id", in.TestCase.Id, "status", in.TestCase.Status)
	return &observer.AckResponse{Success: true, Message: "finish received: " + in.TestCase.Id}, nil
}

func (s *EventServer) ReportStepBegin(ctx context.Context, in *events.StepBeginEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if err := validateTestID(in.Step.TestId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Logging limited fields (TestId); extend when proto adds step-specific identifiers.
	s.logger.Info("test step", "test_id", in.Step.TestId)
	return &observer.AckResponse{Success: true, Message: "step received: " + in.Step.TestId}, nil
}

func (s *EventServer) ReportStepEnd(ctx context.Context, in *events.StepEndEventRequest) (*observer.AckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if err := validateTestID(in.Step.TestId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Logging limited fields (TestId); extend when proto adds step-specific identifiers.
	s.logger.Info("test step end", "test_id", in.Step.TestId, "status", in.Step.Status)
	return &observer.AckResponse{Success: true, Message: "step end received: " + in.Step.TestId}, nil
}

func tsToTime(ts interface{ GetSeconds() int64; GetNanos() int32 }) time.Time {
	if ts == nil { return time.Time{} }
	return time.Unix(ts.GetSeconds(), int64(ts.GetNanos()))
}

// RegisterServices keeps backward compatibility; returns the created server for further customization in callers.
func RegisterServices(s *grpc.Server, logger *slog.Logger) *EventServer {
	srv := New(logger)
	observer.RegisterTestEventCollectorServer(s, srv)
	return srv
}

// loggingInterceptor provides basic structured logging for unary calls.
func loggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil { logger = slog.Default() }
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()
		p, _ := peer.FromContext(ctx)
		resp, err = handler(ctx, req)
		dur := time.Since(start)
		attrs := []any{"method", info.FullMethod, "duration_ms", dur.Milliseconds()}
		if p != nil { attrs = append(attrs, "peer", p.Addr.String()) }
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
	if logger == nil { logger = slog.Default() }
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
