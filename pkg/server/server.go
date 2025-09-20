package server

import (
	"context"
	"log"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	grpc "google.golang.org/grpc"
)

type server struct {
	observer.UnimplementedTestEventCollectorServer
}

func (s *server) ReportTestStart(ctx context.Context, in *events.TestStartEventRequest) (*observer.AckResponse, error) {
	log.Printf("Received Start Event: %v\n", in)
	return &observer.AckResponse{Success: true, Message: "Received: " + in.TestId}, nil
}

func (s *server) ReportTestFinish(ctx context.Context, in *events.TestFinishEventRequest) (*observer.AckResponse, error) {
	log.Printf("Received Finish Event: %v\n", in)
	return &observer.AckResponse{Success: true, Message: "Received: " + in.TestId}, nil
}

func (s *server) ReportTestStep(ctx context.Context, in *events.TestStepEventRequest) (*observer.AckResponse, error) {
	log.Printf("Received Step Event: %v\n", in)
	return &observer.AckResponse{Success: true, Message: "Received: " + in.TestId}, nil
}

func RegisterServices(s *grpc.Server) {
	observer.RegisterTestEventCollectorServer(s, &server{})
}
