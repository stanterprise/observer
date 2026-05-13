package postgres

import (
	"context"
	"fmt"
	"sort"

	m "github.com/stanterprise/observer/internal/models"
	"github.com/stanterprise/observer/internal/repository"
	"gorm.io/gorm"
)

func (r *PostgresRepository) ListRuns(ctx context.Context, filter ListRunsFilter, limit, offset int64) ([]*m.TestRun, int64, error) {
	if err := r.ensureDB(); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := r.db.WithContext(ctx).Model(&m.TestRun{})
	query, err := r.applyRunListFilter(ctx, query, filter)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count runs: %w", err)
	}

	var runs []*m.TestRun
	if err := query.Order("created_at desc").Offset(int(offset)).Limit(int(limit)).Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("list runs: %w", err)
	}

	return runs, total, nil
}

func (r *PostgresRepository) GetRun(ctx context.Context, runID string, includeSteps bool) (*m.TestRun, error) {
	if err := repository.ValidateRunID(runID); err != nil {
		return nil, err
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	runDocs, err := r.buildRuns(ctx, []string{runID}, includeSteps)
	if err != nil {
		return nil, err
	}
	if len(runDocs) == 0 {
		return nil, nil
	}

	return runDocs[0], nil
}

func (r *PostgresRepository) GetRuns(ctx context.Context, filter ListRunsFilter, limit, offset int64, includeSteps bool) ([]*m.TestRun, int64, error) {
	runs, total, err := r.ListRuns(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(runs) == 0 {
		return []*m.TestRun{}, total, nil
	}

	runIDs := make([]string, 0, len(runs))
	for _, run := range runs {
		runIDs = append(runIDs, run.ID)
	}

	docs, err := r.buildRuns(ctx, runIDs, includeSteps)
	if err != nil {
		return nil, 0, err
	}

	ordered := make([]*m.TestRun, 0, len(runIDs))
	byID := make(map[string]*m.TestRun, len(docs))
	for _, doc := range docs {
		byID[doc.ID] = doc
	}
	for _, runID := range runIDs {
		if doc, ok := byID[runID]; ok {
			ordered = append(ordered, doc)
		}
	}

	return ordered, total, nil
}

func (r *PostgresRepository) GetUniqueMarkers(ctx context.Context) ([]*MarkerInfo, error) {
	if err := r.ensureDB(); err != nil {
		return nil, err
	}

	var runs []m.TestRun
	if err := r.db.WithContext(ctx).Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("list runs for markers: %w", err)
	}

	counts := map[string]int64{}
	for _, run := range runs {
		marker, ok := markerFromMetadata(run.Metadata)
		if !ok {
			continue
		}
		counts[marker]++
	}

	markers := make([]*MarkerInfo, 0, len(counts))
	for marker, count := range counts {
		markers = append(markers, &MarkerInfo{Marker: marker, Count: count})
	}

	sort.Slice(markers, func(i, j int) bool {
		if markers[i].Count == markers[j].Count {
			return markers[i].Marker < markers[j].Marker
		}
		return markers[i].Count > markers[j].Count
	})

	return markers, nil
}

func (r *PostgresRepository) applyRunListFilter(ctx context.Context, query *gorm.DB, filter ListRunsFilter) (*gorm.DB, error) {
	if filter.RunID != "" {
		query = query.Where("id = ?", filter.RunID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ProjectName != "" {
		query = query.Where("project_name = ?", filter.ProjectName)
	}
	if filter.Marker != "" {
		var runs []m.TestRun
		if err := query.Session(&gorm.Session{}).WithContext(ctx).Find(&runs).Error; err != nil {
			return nil, fmt.Errorf("list runs for marker filter: %w", err)
		}
		ids := make([]string, 0)
		for _, run := range runs {
			if marker, ok := markerFromMetadata(run.Metadata); ok && marker == filter.Marker {
				ids = append(ids, run.ID)
			}
		}
		if len(ids) == 0 {
			query = query.Where("1 = 0")
		} else {
			query = query.Where("id IN ?", ids)
		}
	}
	return query, nil
}