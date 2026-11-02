package broker

import (
	"math/big"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

var marketLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return time.FixedZone("IST", 5*3600+30*60)
	}
	return loc
}()

// DailyLossCircuitBreaker halts trading if daily losses exceed threshold.
type DailyLossCircuitBreaker struct {
	ThresholdPercent coredecimal.Decimal
	InitialCapital   coredecimal.Decimal

	dailyPnL        map[time.Time]coredecimal.Decimal
	isCircuitOpen   bool
	circuitOpenTime *time.Time
	lossAtOpen      coredecimal.Decimal
}

// NewDailyLossCircuitBreaker implements the corresponding risk operation.
func NewDailyLossCircuitBreaker(threshold, initial coredecimal.Decimal) *DailyLossCircuitBreaker {
	cb := &DailyLossCircuitBreaker{
		ThresholdPercent: threshold,
		InitialCapital:   initial,
		dailyPnL:         map[time.Time]coredecimal.Decimal{},
	}
	cb.resetTracker()
	return cb
}

func (c *DailyLossCircuitBreaker) today() time.Time {
	now := time.Now().In(marketLocation)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, marketLocation)
}

func (c *DailyLossCircuitBreaker) resetTracker() {
	c.dailyPnL = map[time.Time]coredecimal.Decimal{}
	c.dailyPnL[c.today()] = coredecimal.Decimal{}
}

// IsTradingAllowed implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) IsTradingAllowed() bool {
	today := c.today()
	if _, ok := c.dailyPnL[today]; !ok {
		c.resetTracker()
	}
	return !c.isCircuitOpen
}

// GetCurrentDailyPnL implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) GetCurrentDailyPnL() coredecimal.Decimal {
	return c.dailyPnL[c.today()]
}

func (c *DailyLossCircuitBreaker) calculateLossPercentFromPnL(pnl coredecimal.Decimal) coredecimal.Decimal {
	if c.InitialCapital.Cmp(coredecimal.Decimal{}) == 0 {
		return coredecimal.Decimal{}
	}
	return coredecimal.PercentOf(pnl, c.InitialCapital)
}

// CanPlaceTrade implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) CanPlaceTrade(estimatedTradeValue, estimatedRisk coredecimal.Decimal) *RiskCheckResult {
	result := NewRiskCheckResult(true)
	result.CheckType = "DAILY_LOSS_CIRCUIT_BREAKER"
	if c.isCircuitOpen {
		result.AddError("Daily loss circuit breaker is OPEN. Trading halted for today.")
		result.AddInfo("Circuit opened at: %v, Loss: %s (%s%%)", c.circuitOpenTime, c.lossAtOpen, c.calculateLossPercentFromPnL(c.lossAtOpen))
		return result
	}
	current := c.GetCurrentDailyPnL()
	projected := current.Sub(estimatedRisk)
	thresholdAmount := percentDecimal(c.InitialCapital, c.ThresholdPercent)
	// compare projected < -thresholdAmount
	neg := coredecimal.FromUnits(new(big.Int).Neg(thresholdAmount.Units()))
	if projected.Cmp(neg) < 0 {
		result.AddError("Trade would exceed daily loss limit. Projected loss: %s (%s%%), Threshold: %s", projected, c.calculateLossPercentFromPnL(projected), thresholdAmount)
		return result
	}
	result.AddInfo("Trade allowed. Current daily P&L: %s (%s%%), Projected P&L after trade: %s (%s%%)", current, c.calculateLossPercentFromPnL(current), projected, c.calculateLossPercentFromPnL(projected))
	return result
}

// UpdateWithCurrentPositions implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) UpdateWithCurrentPositions(totalUnrealized, totalRealized coredecimal.Decimal) {
	daily := totalUnrealized.Add(totalRealized)
	c.dailyPnL[c.today()] = daily
	thresholdAmount := percentDecimal(c.InitialCapital, c.ThresholdPercent)
	neg := coredecimal.FromUnits(new(big.Int).Neg(thresholdAmount.Units()))
	if daily.Cmp(neg) < 0 {
		c.openCircuit(daily)
	}
}

func (c *DailyLossCircuitBreaker) openCircuit(loss coredecimal.Decimal) {
	c.isCircuitOpen = true
	now := time.Now().In(marketLocation)
	c.circuitOpenTime = &now
	c.lossAtOpen = loss
}

// OpenCircuit implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) OpenCircuit() { c.openCircuit(c.GetCurrentDailyPnL()) }

// CloseCircuit implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) CloseCircuit() {
	c.isCircuitOpen = false
	c.circuitOpenTime = nil
	c.lossAtOpen = coredecimal.Decimal{}
}

// IsCircuitOpen implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) IsCircuitOpen() bool { return c.isCircuitOpen }

// GetCircuitOpenTime implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) GetCircuitOpenTime() *time.Time { return c.circuitOpenTime }

// GetLossAtCircuitOpen implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) GetLossAtCircuitOpen() coredecimal.Decimal { return c.lossAtOpen }

// GetLossPercent implements the corresponding risk operation.
func (c *DailyLossCircuitBreaker) GetLossPercent() coredecimal.Decimal {
	return c.calculateLossPercentFromPnL(c.GetCurrentDailyPnL())
}
