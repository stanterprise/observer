package postgres

import (
	"fmt"
	"time"

	m "github.com/stanterprise/observer/internal/models"
	"gorm.io/gorm"
)

func upsertRunExecutionStart(tx *gorm.DB, execution *m.RunExecution, now time.Time) error {
	stored, err := loadRunExecution(tx, execution.RunID, execution.ID)
	if err != nil {
		return err
	}

	if stored == nil {
		create := m.RunExecution{
			ID:                 execution.ID,
			RunID:              execution.RunID,
			Name:               execution.Name,
			Status:             execution.Status,
			IsShard:            execution.IsShard,
			ShardIndex:         cloneInt32Ptr(execution.ShardIndex),
			ShardCountExpected: cloneInt32Ptr(execution.ShardCountExpected),
			Metadata:           execution.Metadata,
			StartTime:          cloneTimePtr(execution.StartTime),
			EndTime:            cloneTimePtr(execution.EndTime),
			Duration:           cloneInt64Ptr(execution.Duration),
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if create.Status == "" {
			create.Status = "RUNNING"
		}
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create run execution: %w", err)
		}
		return nil
	}

	if execution.Name != "" {
		stored.Name = execution.Name
	}
	if execution.Status != "" {
		stored.Status = execution.Status
	}
	if execution.IsShard {
		stored.IsShard = true
	}
	if execution.ShardIndex != nil {
		stored.ShardIndex = cloneInt32Ptr(execution.ShardIndex)
	}
	if execution.ShardCountExpected != nil {
		stored.ShardCountExpected = cloneInt32Ptr(execution.ShardCountExpected)
	}
	if len(execution.Metadata) > 0 {
		stored.Metadata = mergeRunStartMetadata(stored.Metadata, execution.Metadata)
	}
	if execution.StartTime != nil && (stored.StartTime == nil || execution.StartTime.Before(*stored.StartTime)) {
		stored.StartTime = cloneTimePtr(execution.StartTime)
	}
	if execution.EndTime != nil {
		stored.EndTime = cloneTimePtr(execution.EndTime)
	}
	if execution.Duration != nil {
		stored.Duration = cloneInt64Ptr(execution.Duration)
	}
	stored.UpdatedAt = now

	if err := tx.Save(stored).Error; err != nil {
		return fmt.Errorf("update run execution: %w", err)
	}
	return nil
}

func upsertRunExecutionEnd(tx *gorm.DB, execution *m.RunExecution, now time.Time) error {
	stored, err := loadRunExecution(tx, execution.RunID, execution.ID)
	if err != nil {
		return err
	}

	if stored == nil {
		create := m.RunExecution{
			ID:                 execution.ID,
			RunID:              execution.RunID,
			Name:               execution.Name,
			Status:             execution.Status,
			IsShard:            execution.IsShard,
			ShardIndex:         cloneInt32Ptr(execution.ShardIndex),
			ShardCountExpected: cloneInt32Ptr(execution.ShardCountExpected),
			Metadata:           execution.Metadata,
			StartTime:          cloneTimePtr(execution.StartTime),
			EndTime:            cloneTimePtr(execution.EndTime),
			Duration:           cloneInt64Ptr(execution.Duration),
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if create.EndTime == nil {
			create.EndTime = &now
		}
		if err := tx.Create(&create).Error; err != nil {
			return fmt.Errorf("create finalized run execution: %w", err)
		}
		return nil
	}

	if execution.Name != "" {
		stored.Name = execution.Name
	}
	if execution.Status != "" {
		stored.Status = execution.Status
	}
	if execution.IsShard {
		stored.IsShard = true
	}
	if execution.ShardIndex != nil {
		stored.ShardIndex = cloneInt32Ptr(execution.ShardIndex)
	}
	if execution.ShardCountExpected != nil {
		stored.ShardCountExpected = cloneInt32Ptr(execution.ShardCountExpected)
	}
	if len(execution.Metadata) > 0 {
		stored.Metadata = mergeRunStartMetadata(stored.Metadata, execution.Metadata)
	}
	if execution.StartTime != nil && (stored.StartTime == nil || execution.StartTime.Before(*stored.StartTime)) {
		stored.StartTime = cloneTimePtr(execution.StartTime)
	}
	if execution.EndTime != nil {
		stored.EndTime = cloneTimePtr(execution.EndTime)
	} else if stored.EndTime == nil {
		stored.EndTime = &now
	}
	if execution.Duration != nil {
		stored.Duration = cloneInt64Ptr(execution.Duration)
	}
	stored.UpdatedAt = now

	if err := tx.Save(stored).Error; err != nil {
		return fmt.Errorf("update finalized run execution: %w", err)
	}
	return nil
}

func refreshLogicalRunAggregate(tx *gorm.DB, runID string, now time.Time) error {
	var executions []m.RunExecution
	if err := tx.Where("run_id = ?", runID).Order("created_at asc, id asc").Find(&executions).Error; err != nil {
		return fmt.Errorf("load run executions: %w", err)
	}
	if len(executions) == 0 {
		return nil
	}

	aggregate, ok := buildAggregatedRunFromExecutions(runID, executions, now)
	if !ok {
		return nil
	}

	assignment := m.TestRun{
		Status:    aggregate.Status,
		StartTime: aggregate.StartTime,
		EndTime:   aggregate.EndTime,
		Duration:  aggregate.Duration,
		UpdatedAt: now,
	}
	result := tx.Where(m.TestRun{ID: runID}).Assign(assignment).FirstOrCreate(&m.TestRun{ID: runID, CreatedAt: now, UpdatedAt: now})
	if result.Error != nil {
		return fmt.Errorf("refresh logical run aggregate: %w", result.Error)
	}
	return nil
}

func buildAggregatedRunFromExecutions(runID string, executions []m.RunExecution, now time.Time) (*m.TestRun, bool) {
	if len(executions) == 0 {
		return nil, false
	}

	status := aggregateRunExecutionStatuses(executions)
	if status == "" {
		return nil, false
	}

	var (
		startedAt  *time.Time
		finishedAt *time.Time
	)
	for _, execution := range executions {
		if execution.StartTime != nil && (startedAt == nil || execution.StartTime.Before(*startedAt)) {
			t := *execution.StartTime
			startedAt = &t
		}
		if execution.EndTime != nil && (finishedAt == nil || execution.EndTime.After(*finishedAt)) {
			t := *execution.EndTime
			finishedAt = &t
		}
	}

	var duration *int64
	if startedAt != nil && finishedAt != nil {
		d := finishedAt.Sub(*startedAt).Nanoseconds()
		duration = &d
	}

	return &m.TestRun{
		ID:        runID,
		Status:    status,
		StartTime: startedAt,
		EndTime:   finishedAt,
		Duration:  duration,
		CreatedAt: now,
		UpdatedAt: now,
	}, true
}

func aggregateRunExecutionStatuses(executions []m.RunExecution) string {
	if len(executions) == 0 {
		return "UNKNOWN"
	}

	allPassed := true
	anyRunning := false
	anyFailed := false
	anyTimedOut := false
	anyInterrupted := false

	for _, execution := range executions {
		switch execution.Status {
		case "PASSED":
			// keep allPassed true
		case "RUNNING", "NOT_RUN", "":
			allPassed = false
			anyRunning = true
		default:
			allPassed = false
		}
		if execution.EndTime == nil && execution.Status != "PASSED" {
			anyRunning = true
		}
		if execution.Status == "FAILED" {
			anyFailed = true
		}
		if execution.Status == "TIMEDOUT" {
			anyTimedOut = true
		}
		if execution.Status == "INTERRUPTED" {
			anyInterrupted = true
		}
	}

	switch {
	case anyRunning:
		return "RUNNING"
	case anyFailed:
		return "FAILED"
	case anyTimedOut:
		return "TIMEDOUT"
	case anyInterrupted:
		return "INTERRUPTED"
	case allPassed:
		return "PASSED"
	default:
		return "UNKNOWN"
	}
}

func loadRunExecution(tx *gorm.DB, runID, executionID string) (*m.RunExecution, error) {
	var execution m.RunExecution
	err := tx.Where("run_id = ? AND id = ?", runID, executionID).First(&execution).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("load run execution: %w", err)
	}
	return &execution, nil
}

func cloneTimePtr(input *time.Time) *time.Time {
	if input == nil {
		return nil
	}
	t := *input
	return &t
}

func cloneInt32Ptr(input *int32) *int32 {
	if input == nil {
		return nil
	}
	v := *input
	return &v
}

func cloneInt64Ptr(input *int64) *int64 {
	if input == nil {
		return nil
	}
	v := *input
	return &v
}
