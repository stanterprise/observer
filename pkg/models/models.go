package models

import (
	"time"

	"gorm.io/datatypes"
)

// TestCase represents a collected test case entity.
type TestCase struct {
    ID        string             `gorm:"primaryKey;type:text"`
    Name      string             `gorm:"type:text"`
    Status    string             `gorm:"type:text"`
    Metadata  datatypes.JSONMap  `gorm:"type:jsonb"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Step represents a step inside a test case.
type Step struct {
    ID        uint      `gorm:"primaryKey"`
    TestID    string    `gorm:"index;type:text"`
    Status    string    `gorm:"type:text"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
