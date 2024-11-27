package server

import (
	"context"

	pb "github.com/stanterprise/observer/proto"
	"google.golang.org/grpc"
)

type TestSignalObserverServer struct {
	pb.UnimplementedTestSignalObserverServer
}

func (s *TestSignalObserverServer) SubmitSignal(ctx context.Context, in *pb.Signal) (*pb.SignalResponse, error) {
	return &pb.SignalResponse{Response: "Received: " + in.Signal}, nil
}

func RegisterServices(s *grpc.Server) {
	pb.RegisterTestSignalObserverServer(s, &TestSignalObserverServer{})
}
