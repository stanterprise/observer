package api

import (
	m "github.com/stanterprise/observer/internal/models"
)

func FilterTestsByStatus(tests []*m.Test, status string) []*m.Test {
	if status == "" {
		return tests
	}
	filtered := make([]*m.Test, 0)
	for _, test := range tests {
		if test.Status == status {
			filtered = append(filtered, test)
		}
	}
	return filtered
}
