package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/stanterprise/proto-go/testsystem/v1/common"
	"github.com/stanterprise/proto-go/testsystem/v1/entities"
	events "github.com/stanterprise/proto-go/testsystem/v1/events"
	observer "github.com/stanterprise/proto-go/testsystem/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to ingestion service
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := observer.NewTestEventCollectorClient(conn)
	ctx := context.Background()

	runID := "run-123"
	suiteID := fmt.Sprintf("%s-suite-root", runID)

	// Send suite begin event
	fmt.Printf("Sending suite begin event for suite ID: %s\n", suiteID)
	_, err = client.ReportSuiteBegin(ctx, &events.SuiteBeginEventRequest{
		Suite: &entities.TestSuiteRun{
			Id:      suiteID,
			Name:    "Demo Suite",
			Project: "observer-demo",
			Metadata: map[string]string{
				"suite_type": "root",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to send suite begin: %v", err)
	}
	fmt.Println("✅ Suite begin event sent")

	time.Sleep(200 * time.Millisecond)

	// Send test begin event
	testID := fmt.Sprintf("test-%d", time.Now().Unix())
	fmt.Printf("Sending test begin event for test ID: %s\n", testID)

	_, err = client.ReportTestBegin(ctx, &events.TestBeginEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:            testID,
			RunId:         runID,
			TestSuiteRunId: suiteID,
			Name:          "Demo Test for WebSocket",
			Metadata: map[string]string{
				"browser": "chrome",
				"env":     "staging",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to send test begin: %v", err)
	}
	fmt.Println("✅ Test begin event sent")

	time.Sleep(200 * time.Millisecond)

	// Send step begin event
	stepID := fmt.Sprintf("%s-%s-step-1", runID, testID)
	fmt.Println("Sending step begin event")
	_, err = client.ReportStepBegin(ctx, &events.StepBeginEventRequest{
		Step: &entities.StepRun{
			Id:            stepID,
			RunId:         runID,
			TestCaseRunId: fmt.Sprintf("%s-%s", runID, testID), // This matches the Playwright reporter format
			Title:         "Demo Step",
			Category:      "test",
		},
	})
	if err != nil {
		log.Fatalf("Failed to send step begin: %v", err)
	}
	fmt.Println("✅ Step begin event sent")

	time.Sleep(200 * time.Millisecond)

	// Send step end event
	fmt.Println("Sending step end event")
	_, err = client.ReportStepEnd(ctx, &events.StepEndEventRequest{
		Step: &entities.StepRun{
			Id:            stepID,
			TestCaseRunId: testID,
			Status:        common.TestStatus_PASSED,
		},
	})
	if err != nil {
		log.Fatalf("Failed to send step end: %v", err)
	}
	fmt.Println("✅ Step end event sent")

	time.Sleep(500 * time.Millisecond)

	// Send test end event
	fmt.Println("Sending test end event")
	_, err = client.ReportTestEnd(ctx, &events.TestEndEventRequest{
		TestCase: &entities.TestCaseRun{
			Id:     testID,
			RunId:  "run-123",
			Status: common.TestStatus_PASSED,
		},
	})
	if err != nil {
		log.Fatalf("Failed to send test end: %v", err)
	}
	fmt.Println("✅ Test end event sent")

	fmt.Println("\n✨ All events sent successfully!")
	fmt.Println("Check the WebSocket client output for received events")
}
