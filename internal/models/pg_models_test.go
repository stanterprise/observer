package models

import "testing"

func TestIsTerminalStatus(t *testing.T) {
	terminal := []string{StatusPassed, StatusFailed, StatusTimedOut, StatusCancelled, StatusAborted, StatusSkipped}
	for _, s := range terminal {
		if !IsTerminalStatus(s) {
			t.Errorf("IsTerminalStatus(%q) = false, want true", s)
		}
	}

	active := []string{StatusRunning, StatusInProgress, "", "UNKNOWN"}
	for _, s := range active {
		if IsTerminalStatus(s) {
			t.Errorf("IsTerminalStatus(%q) = true, want false", s)
		}
	}
}
