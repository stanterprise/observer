package main

import (
	"os"
	"os/exec"
	"testing"
)

var testClientCmd *exec.Cmd

func TestMain(m *testing.M) {
	// Setup: Initialize resources
	setup()

	// Run tests
	code := m.Run()

	// Teardown: Cleanup resources
	teardown()

	// Exit with the code from `m.Run`
	os.Exit(code)
}

func setup() {
	// Start the test client as a separate process
	testClientCmd = exec.Command("go", "run", "../server/main.go")
	testClientCmd.Stdout = os.Stdout
	testClientCmd.Stderr = os.Stderr

	if err := testClientCmd.Start(); err != nil {
		panic("Failed to start test client: " + err.Error())
	}
}

func teardown() {
	// Stop the test client process
	if testClientCmd != nil && testClientCmd.Process != nil {
		if err := testClientCmd.Process.Kill(); err != nil {
			panic("Failed to stop test client: " + err.Error())
		}
	}
}
