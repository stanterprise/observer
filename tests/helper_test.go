package main

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TestHelper struct {
	t      *testing.T
	client *observer.TestEventCollectorClient
	ctx    *context.Context
	cancel context.CancelFunc
	conn   *grpc.ClientConn
}

func NewTestHelper(t *testing.T) *TestHelper {
	// Perform setup
	t.Log("Initializing TestHelper")

	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}

	client := observer.NewTestEventCollectorClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	th := &TestHelper{
		t:      t,
		client: &client,
		ctx:    &ctx,
		cancel: cancel,
		conn:   conn,
	}

	return th
}

func (th *TestHelper) Start() {
	// Perform setup
	response, err := (*th.client).ReportTestStart(*th.ctx, &events.TestStartEventRequest{
		TestId:    "test-id",
		TestName:  "test-name",
		StartTime: timestamppb.New(time.Now()),
		Metadata:  map[string]string{"key": "value"},
	})

	if err != nil {
		th.t.Fatalf("could not send message: %v", err)
	}
	log.Printf("Response from server: %s", response)
}

func (th *TestHelper) Teardown() {
	// Perform teardown
	th.t.Log("Cleaning up TestHelper")
	response, err := (*th.client).ReportTestFinish(*th.ctx, &events.TestFinishEventRequest{
		TestId:  "test-id",
		EndTime: timestamppb.New(time.Now()),
	})
	if err != nil {
		th.t.Fatalf("could not send message: %v", err)
	}
	log.Printf("Response from server: %s", response)
	(*th.conn).Close()
	if th.cancel != nil { th.cancel() }
}

func TestWithHelper(t *testing.T) {
	helper := NewTestHelper(t)
	helper.Start()

	defer helper.Teardown()

	fmt.Println("Running test with helper")
	t.Log("Running test with helper")
}
