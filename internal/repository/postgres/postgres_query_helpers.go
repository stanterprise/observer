package postgres

import (
	"fmt"
	"strings"

	m "github.com/stanterprise/observer/internal/models"
)

func markerFromMetadata(metadata map[string]interface{}) (string, bool) {
	if metadata == nil {
		return "", false
	}
	value, ok := metadata["MARKER"]
	if !ok || value == nil {
		return "", false
	}
	marker := strings.TrimSpace(fmt.Sprint(value))
	if marker == "" || marker == "<nil>" {
		return "", false
	}
	return marker, true
}

func testExternalID(test m.Test) string {
	if test.ExternalTestID != "" {
		return test.ExternalTestID
	}
	parts := strings.SplitN(test.ID, ":test:", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return test.ID
}

func cloneMetadata(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]interface{}, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func findAttachmentInMaps(attachments []map[string]interface{}, storageKey string) map[string]interface{} {
	for _, attachment := range attachments {
		if key, ok := attachment["storage_key"].(string); ok && key == storageKey {
			return cloneMetadata(attachment)
		}
	}
	return nil
}
