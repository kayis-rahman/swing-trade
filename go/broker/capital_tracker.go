package broker

import (
	"math/big"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// CapitalTracker enforces capital limits.
type CapitalTracker struct {
	InitialCapital        coredecimal.Decimal
	MaxPositions          int
	MaxCapitalPerPosition coredecimal.Decimal
	MaxTotalExposure      coredecimal.Decimal
}

// NewCapitalTracker implements the corresponding risk operation.
func NewCapitalTracker(initial coredecimal.Decimal, maxPositions int, maxPerPosition coredecimal.Decimal) *CapitalTracker {
	units := initial.Units()
	num := new(big.Int).Mul(units, big.NewInt(8))
	den := big.NewInt(10)
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	if new(big.Int).Mul(r, big.NewInt(2)).Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	ct := &CapitalTracker{
		InitialCapital:        initial,
		MaxPositions:          maxPositions,
		MaxCapitalPerPosition: maxPerPosition,
		MaxTotalExposure:      coredecimal.FromUnits(q),
	}
	return ct
}

func (c *CapitalTracker) calculateTotalDeployed(positions []Position) coredecimal.Decimal {
	total := coredecimal.Decimal{}
	for i := range positions {
		total = total.Add(positions[i].PositionValue())
	}
	return total
}

// EnforceLimits implements the corresponding risk operation.
func (c *CapitalTracker) EnforceLimits(positions []Position, orderValue coredecimal.Decimal) *RiskCheckResult {
	if positions == nil {
		positions = []Position{}
	}
	result := NewRiskCheckResult(true)
	result.CheckType = "CAPITAL_LIMITS"
	if len(positions) >= c.MaxPositions {
		result.AddError("Maximum concurrent positions limit reached: %d / %d", len(positions), c.MaxPositions)
		result.AddInfo("Cannot open more than %d positions. Close %d positions before placing new order.", c.MaxPositions, len(positions)-c.MaxPositions+1)
		return result
	}
	totalWith := c.calculateTotalDeployed(positions).Add(orderValue)
	if orderValue.Cmp(c.MaxCapitalPerPosition) > 0 {
		result.AddError("Order value exceeds maximum capital per position: %s / %s", orderValue, c.MaxCapitalPerPosition)
		result.AddInfo("Maximum capital per position is %s (20%% of initial capital %s)", c.MaxCapitalPerPosition, c.InitialCapital)
		return result
	}
	if totalWith.Cmp(c.MaxTotalExposure) > 0 {
		result.AddError("Total exposure would exceed maximum: %s / %s", totalWith, c.MaxTotalExposure)
		result.AddInfo("Maximum total exposure is %s (80%% of initial capital %s)", c.MaxTotalExposure, c.InitialCapital)
		result.AddInfo("Current deployed: %s, order value: %s", c.calculateTotalDeployed(positions), orderValue)
		return result
	}
	result.AddInfo("Capital limits validated: position %d / %d, total exposure %s / %s", len(positions)+1, c.MaxPositions, totalWith, c.MaxTotalExposure)
	return result
}

// CanAddPosition implements the corresponding risk operation.
func (c *CapitalTracker) CanAddPosition(positions []Position, orderValue coredecimal.Decimal) bool {
	if positions == nil {
		return false
	}
	return c.EnforceLimits(positions, orderValue).IsPassed()
}

// GetCurrentPositionCount implements the corresponding risk operation.
func (c *CapitalTracker) GetCurrentPositionCount(positions []Position) int {
	if positions == nil {
		return 0
	}
	return len(positions)
}

// GetRemainingPositionCapacity implements the corresponding risk operation.
func (c *CapitalTracker) GetRemainingPositionCapacity(positions []Position) int {
	cur := c.GetCurrentPositionCount(positions)
	if cur >= c.MaxPositions {
		return 0
	}
	return c.MaxPositions - cur
}

// GetDeployedCapital implements the corresponding risk operation.
func (c *CapitalTracker) GetDeployedCapital(positions []Position) coredecimal.Decimal {
	return c.calculateTotalDeployed(positions)
}

// GetAvailableCapital implements the corresponding risk operation.
func (c *CapitalTracker) GetAvailableCapital(positions []Position) coredecimal.Decimal {
	return c.InitialCapital.Sub(c.calculateTotalDeployed(positions))
}

// GetRemainingCapitalPerPosition implements the corresponding risk operation.
func (c *CapitalTracker) GetRemainingCapitalPerPosition(_ []Position) coredecimal.Decimal {
	return c.MaxCapitalPerPosition
}

// GetInitialCapital implements the corresponding risk operation.
func (c *CapitalTracker) GetInitialCapital() coredecimal.Decimal { return c.InitialCapital }

// GetMaxPositions implements the corresponding risk operation.
func (c *CapitalTracker) GetMaxPositions() int { return c.MaxPositions }

// GetMaxCapitalPerPosition implements the corresponding risk operation.
func (c *CapitalTracker) GetMaxCapitalPerPosition() coredecimal.Decimal {
	return c.MaxCapitalPerPosition
}
