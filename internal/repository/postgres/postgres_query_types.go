package postgres

import "time"

type ListRunsFilter struct {
	RunID       string
	Status      string
	ProjectName string
	Marker      string
}

type TestTrendItem struct {
	TestID    string     `json:"testId"`
	RunID     string     `json:"runId"`
	Status    string     `json:"status"`
	Duration  *int64     `json:"duration,omitempty"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type MarkerInfo struct {
	Marker string `json:"marker"`
	Count  int64  `json:"count"`
}