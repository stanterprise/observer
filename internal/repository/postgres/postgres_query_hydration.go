package postgres

import (
	"context"
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
)

func (r *PostgresRepository) buildRuns(ctx context.Context, runIDs []string, includeSteps bool) ([]*m.TestRun, error) {
	if len(runIDs) == 0 {
		return []*m.TestRun{}, nil
	}

	var runs []*m.TestRun
	if err := r.db.WithContext(ctx).Where("id IN ?", runIDs).Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("load runs: %w", err)
	}

	if len(runs) == 0 {
		return []*m.TestRun{}, nil
	}

	var suites []*m.Suite
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("created_at asc, id asc").
		Find(&suites).Error; err != nil {
		return nil, fmt.Errorf("load suites: %w", err)
	}

	var tests []*m.Test
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("created_at asc, id asc").
		Find(&tests).Error; err != nil {
		return nil, fmt.Errorf("load tests: %w", err)
	}

	var executions []*m.RunExecution
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("created_at asc, id asc").
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("load run executions: %w", err)
	}

	var attempts []m.TestAttempt
	if err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("test_id asc, attempt_index asc").
		Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("load test attempts: %w", err)
	}

	if !includeSteps {
		for i := range attempts {
			attempts[i].Steps = nil
		}
	}

	attemptsByTestID := make(map[string][]m.TestAttempt, len(attempts))
	for _, attempt := range attempts {
		key := runScopedEntityKey(attempt.RunID, attempt.TestID)
		attemptsByTestID[key] = append(attemptsByTestID[key], attempt)
	}

	runByID := make(map[string]*m.TestRun, len(runs))
	for _, run := range runs {
		run.Executions = nil
		run.Suites = nil
		run.Tests = nil
		runByID[run.ID] = run
	}

	for _, execution := range executions {
		if run, ok := runByID[execution.RunID]; ok {
			run.Executions = append(run.Executions, execution)
		}
	}

	suiteByID := make(map[string]*m.Suite, len(suites))
	for _, suite := range suites {
		suite.Suites = nil
		suite.Tests = nil
		suiteByID[runScopedEntityKey(suite.RunID, suite.ID)] = suite
	}

	for _, suite := range suites {
		if suite.ParentSuiteID != nil {
			if parent, ok := suiteByID[runScopedEntityKey(suite.RunID, *suite.ParentSuiteID)]; ok {
				parent.Suites = append(parent.Suites, suite)
				continue
			}
		}
		if run, ok := runByID[suite.RunID]; ok {
			run.Suites = append(run.Suites, suite)
		}
	}

	for _, test := range tests {
		if attachedAttempts, ok := attemptsByTestID[runScopedEntityKey(test.RunID, test.ID)]; ok {
			test.Attempts = attachedAttempts
			hydrateTestFromAttempts(test)
		} else {
			test.Attempts = nil
		}

		if test.SuiteID != nil {
			if suite, ok := suiteByID[runScopedEntityKey(test.RunID, *test.SuiteID)]; ok {
				suite.Tests = append(suite.Tests, test)
				continue
			}
		}
		if run, ok := runByID[test.RunID]; ok {
			run.Tests = append(run.Tests, test)
		}
	}

	return runs, nil
}

func runScopedEntityKey(runID, entityID string) string {
	return runID + ":" + entityID
}

func hydrateTestFromAttempts(test *m.Test) {
	if test == nil || len(test.Attempts) == 0 {
		return
	}

	latestAttempts, _ := latestExecutionAttemptSet(test.Attempts)
	if len(latestAttempts) == 0 {
		return
	}

	test.Status = aggregateTestAttemptStatuses(latestAttempts)

	var startedAt *time.Time
	var finishedAt *time.Time
	for _, attempt := range latestAttempts {
		if attempt.StartTime != nil && (startedAt == nil || attempt.StartTime.Before(*startedAt)) {
			startedAt = cloneTimePtr(attempt.StartTime)
		}
		if attempt.EndTime != nil && (finishedAt == nil || attempt.EndTime.After(*finishedAt)) {
			finishedAt = cloneTimePtr(attempt.EndTime)
		}
	}
	if test.StartTime == nil && startedAt != nil {
		test.StartTime = startedAt
	}
	if test.EndTime == nil && finishedAt != nil {
		test.EndTime = finishedAt
	}
	if test.Duration == nil && startedAt != nil && finishedAt != nil {
		duration := finishedAt.Sub(*startedAt).Nanoseconds()
		test.Duration = &duration
	}
}
