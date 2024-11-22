package main

import (
	"context"
	"log"

	pb "github.com/stanterprise/observer/proto"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewTestSignalObserverClient(conn)

	response, err := client.SubmitSignal(context.Background(), &pb.Signal{Signal: "test"})
	if err != nil {
		log.Fatalf("could not submit signal: %v", err)
	}
	log.Printf("Response: %s", response.Response)
}
