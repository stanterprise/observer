package repository

import "fmt"

// validateRunID checks if runID is provided and returns an error if not
func ValidateRunID(runID string) error {
	if runID == "" {
		return fmt.Errorf("runID is required")
	}
	return nil
}
