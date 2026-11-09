package main

import (
	"log/slog"
	"testing"
)

func TestNewIngestionGRPCServerMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		key     string
		wantErr bool
	}{
		{name: "local mode is keyless", mode: "local"},
		{name: "local mode rejects a key", mode: "local", key: "key", wantErr: true},
		{name: "distributed mode requires a key", mode: "distributed", wantErr: true},
		{name: "distributed mode accepts a strong key", mode: "distributed", key: "test-ingestion-auth-key-with-32-bytes-minimum"},
		{name: "unknown mode rejected", mode: "unsupported", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server, err := newIngestionGRPCServer(slog.Default(), tc.mode, tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected configuration error")
				}
				return
			}
			if err != nil {
				t.Fatalf("create gRPC server: %v", err)
			}
			server.Stop()
		})
	}
}
