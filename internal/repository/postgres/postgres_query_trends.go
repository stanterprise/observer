package postgres

import (
	"context"
	"fmt"

	m "github.com/stanterprise/observer/internal/models"
)

func (r *PostgresRepository) GetTestTrends(ctx context.Context, testID string, limit int64) ([]*TestTrendItem, error) {
	if testID == "" {
		return nil, fmt.Errorf("testID is required")
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}

	var tests []m.Test
	if err := r.db.WithContext(ctx).
		Where("external_test_id = ? OR id = ?", testID, testID).
		Order("created_at desc").
		Limit(int(limit)).
		Find(&tests).Error; err != nil {
		return nil, fmt.Errorf("list test trends: %w", err)
	}

	items := make([]*TestTrendItem, 0, len(tests))
	for _, test := range tests {
		item := &TestTrendItem{
			TestID:    testExternalID(test),
			RunID:     test.RunID,
			Status:    test.Status,
			Duration:  test.Duration,
			CreatedAt: test.CreatedAt,
		}
		if test.StartTime != nil {
			start := *test.StartTime
			item.StartTime = &start
		}
		if test.EndTime != nil {
			end := *test.EndTime
			item.EndTime = &end
		}
		items = append(items, item)
	}

	return items, nil
}