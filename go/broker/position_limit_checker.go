package broker

import (
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// PositionLimitChecker enforces concurrent position and per-position capital limits.
type PositionLimitChecker struct {
	MaxConcurrentPositions int
	MaxCapitalPerPosition  coredecimal.Decimal
}

// NewPositionLimitChecker implements the corresponding risk operation.
func NewPositionLimitChecker(maxPositions int, maxPerPosition coredecimal.Decimal) *PositionLimitChecker {
	return &PositionLimitChecker{
		MaxConcurrentPositions: maxPositions,
		MaxCapitalPerPosition:  maxPerPosition,
	}
}

// CanAddPosition implements the corresponding risk operation.
func (p *PositionLimitChecker) CanAddPosition(currentCount int, estimatedValue coredecimal.Decimal) *RiskCheckResult {
	result := NewRiskCheckResult(true)
	result.CheckType = "POSITION_LIMIT"
	if currentCount >= p.MaxConcurrentPositions {
		result.AddError("Maximum concurrent positions limit reached: %d / %d", currentCount, p.MaxConcurrentPositions)
		return result
	}
	if estimatedValue.Cmp(coredecimal.Decimal{}) > 0 && estimatedValue.Cmp(p.MaxCapitalPerPosition) > 0 {
		result.AddError("Position value exceeds maximum allowed: %s > %s", estimatedValue, p.MaxCapitalPerPosition)
		return result
	}
	result.AddInfo("Position limit check passed. Current positions: %d/%d, Estimated position value: %s (max: %s)", currentCount, p.MaxConcurrentPositions, estimatedValue, p.MaxCapitalPerPosition)
	return result
}

// GetMaxConcurrentPositions implements the corresponding risk operation.
func (p *PositionLimitChecker) GetMaxConcurrentPositions() int { return p.MaxConcurrentPositions }

// GetMaxCapitalPerPosition implements the corresponding risk operation.
func (p *PositionLimitChecker) GetMaxCapitalPerPosition() coredecimal.Decimal {
	return p.MaxCapitalPerPosition
}

// GetRemainingPositionCapacity implements the corresponding risk operation.
func (p *PositionLimitChecker) GetRemainingPositionCapacity(currentCount int) int {
	if currentCount >= p.MaxConcurrentPositions {
		return 0
	}
	return p.MaxConcurrentPositions - currentCount
}
