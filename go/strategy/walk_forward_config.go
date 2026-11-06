package strategy

import (
	"fmt"
	"time"
)

// WalkForwardConfig mirrors Java record.
type WalkForwardConfig struct {
	TrainMonths   int
	TestMonths    int
	StepMonths    int
	HoldoutMonths int
	UnlockHoldout bool
}

// NewWalkForwardConfig constructs and validates a WalkForwardConfig.
func NewWalkForwardConfig(trainMonths, testMonths, stepMonths, holdoutMonths int, unlockHoldout bool) (WalkForwardConfig, error) {
	if trainMonths <= 0 || testMonths <= 0 || stepMonths <= 0 {
		return WalkForwardConfig{}, fmt.Errorf("trainMonths/testMonths/stepMonths must be positive")
	}
	if holdoutMonths < 0 {
		return WalkForwardConfig{}, fmt.Errorf("holdoutMonths cannot be negative")
	}
	return WalkForwardConfig{
		TrainMonths: trainMonths, TestMonths: testMonths, StepMonths: stepMonths,
		HoldoutMonths: holdoutMonths, UnlockHoldout: unlockHoldout,
	}, nil
}

// WalkForwardConfigDefaults returns the default WalkForwardConfig.
func WalkForwardConfigDefaults() WalkForwardConfig {
	c, _ := NewWalkForwardConfig(24, 6, 6, 6, false)
	return c
}

// HoldoutStart mirrors Java method.
func (c WalkForwardConfig) HoldoutStart(overallEnd time.Time) time.Time {
	if c.HoldoutMonths == 0 {
		return overallEnd.AddDate(0, 0, 1)
	}
	// Mirror Java's LocalDate.minusMonths: subtract calendar months and clamp
	// the day to the last valid day of the target month instead of letting the
	// date overflow into the following month like time.AddDate does.
	year, month, day := overallEnd.UTC().Date()
	totalMonths := year*12 + int(month) - 1 - c.HoldoutMonths
	newYear := totalMonths / 12
	newMonth := time.Month(totalMonths%12 + 1)
	lastDay := time.Date(newYear, newMonth+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	clamped := time.Date(newYear, newMonth, day, 0, 0, 0, 0, time.UTC)
	return clamped.AddDate(0, 0, 1)
}
