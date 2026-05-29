package postgres

import (
	"context"
	"fmt"

	m "github.com/stanterprise/observer/internal/models"
	"gorm.io/gorm/clause"
)

func (r *PostgresRepository) GetAllRunStats(ctx context.Context, limit int64, offset int64) ([]m.RunStat, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	var stats []m.RunStat
	if err := r.db.
		WithContext(ctx).
		Order(clause.OrderBy{Columns: []clause.OrderByColumn{
			{
				Column: clause.Column{Name: "created_at"},
				Desc:   true,
			},
		}}).
		Limit(int(limit)).
		Offset(int(offset)).
		Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("load all run stats: %w", err)
	}

	return stats, nil
}

func (r *PostgresRepository) GetRunStatsByMarker(ctx context.Context, marker string) ([]m.RunStat, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	var stats []m.RunStat
	if err := r.db.
		WithContext(ctx).
		Joins("JOIN runs ON runs.id = run_stats.run_id").
		Where("runs.metadata ->> 'MARKER' = ?", marker).
		Order(clause.OrderBy{Columns: []clause.OrderByColumn{
			{
				Column: clause.Column{Name: "created_at"},
				Desc:   true,
			},
		}}).
		Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("load run stats by marker: %w", err)
	}

	return stats, nil
}
