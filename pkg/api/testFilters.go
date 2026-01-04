package api

import (
	m "github.com/stanterprise/observer/internal/models"
)

func FilterTestsByStatus(tests []*m.TestDocument, status string) []*m.TestDocument {
	if status == "" {
		return tests
	}
	filtered := make([]*m.TestDocument, 0)
	for _, test := range tests {
		if test.Status == status {
			filtered = append(filtered, test)
		}
	}
	return filtered
}
