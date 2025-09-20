package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	obsrv "github.com/stanterprise/observer/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var (
	testBufListener *bufconn.Listener
	testGRPCServer  *grpc.Server
)

const bufSize = 1024 * 1024

func TestMain(m *testing.M) {
	// Setup in-process server
	testBufListener = bufconn.Listen(bufSize)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	testGRPCServer = obsrv.NewGRPCServer(logger)
	obsrv.RegisterServices(testGRPCServer, logger)
	go func() {
		_ = testGRPCServer.Serve(&listenerWrapper{Listener: testBufListener})
	}()

	code := m.Run()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		testGRPCServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-shutdownCtx.Done():
		testGRPCServer.Stop()
	}
	os.Exit(code)
}

// listenerWrapper adapts bufconn to satisfy only the Accept / Close / Addr used by Serve.
type listenerWrapper struct { *bufconn.Listener }
