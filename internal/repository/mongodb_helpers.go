package repository

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// extractRootSuiteID extracts the root suite ID from a potentially nested suite ID
// Example: "abc123-suite-root" -> "abc123-suite-root"
// Example: "abc123-suite-/path/to/suite" -> "abc123-suite-root"
func extractRootSuiteID(suiteID string) string {
	// Look for the pattern: {base-id}-suite-{path}
	// We want to return {base-id}-suite-root
	// Note: base-id itself might contain "-suite-" as part of the UUID
	// So we look for the LAST occurrence of "-suite-"

	suiteMarker := "-suite-"
	lastIdx := -1

	// Find last occurrence of "-suite-"
	for i := 0; i <= len(suiteID)-len(suiteMarker); i++ {
		if suiteID[i:i+len(suiteMarker)] == suiteMarker {
			lastIdx = i
		}
	}

	if lastIdx >= 0 {
		// Found "-suite-", extract base ID and append "-suite-root"
		baseID := suiteID[:lastIdx]
		return baseID + "-suite-root"
	}

	// No "-suite-" found, assume it's already a root ID or malformed
return suiteID + "-suite-root"
}

// buildTestEndUpdate creates the update document for test.end events
func buildTestEndUpdate(status string, duration *int64, now time.Time) bson.M {
update := bson.M{
"updated_at": now,
}
if status != "" {
update["status"] = status
}
if duration != nil {
update["duration"] = duration
}
return update
}

// buildStepEndUpdate creates the update document for step.end events
func buildStepEndUpdate(status string, now time.Time) bson.M {
update := bson.M{
"updated_at": now,
}
if status != "" {
update["status"] = status
}
return update
}
