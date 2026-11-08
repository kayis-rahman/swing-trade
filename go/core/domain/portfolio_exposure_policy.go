package domain

import (
	"fmt"
	"math"
	"strconv"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

type PortfolioExposureContext struct {
	CandidateSymbol    string
	CandidateSector    *string
	CandidateNotional  coredecimal.Decimal
	InitialCapital     coredecimal.Decimal
	OpenHoldings       []PortfolioExposureContextHolding
	ClosingPrices      map[string][]float64
}

type PortfolioExposureContextHolding struct {
	Symbol   string
	Sector   *string
	Notional coredecimal.Decimal
}

type PortfolioExposureDecision struct {
	Accepted bool
	Reason   string
}

func (d PortfolioExposureDecision) AcceptedDecision() bool { return d.Accepted }

func PortfolioExposureDecisionAccept() PortfolioExposureDecision {
	return PortfolioExposureDecision{Accepted: true, Reason: ""}
}

func PortfolioExposureDecisionReject(reason string) PortfolioExposureDecision {
	return PortfolioExposureDecision{Accepted: false, Reason: reason}
}

type PortfolioExposurePolicy func(ctx PortfolioExposureContext) PortfolioExposureDecision

func PortfolioExposurePolicyNone() PortfolioExposurePolicy {
	return func(ctx PortfolioExposureContext) PortfolioExposureDecision { return PortfolioExposureDecisionAccept() }
}

type SectorExposureLimit struct {
	MaxPositions     int
	MaxCapitalFraction float64
}


type CorrelationExposureLimit struct {
	LookbackDays            int
	MaxAbsoluteCorrelation  float64
}


func PortfolioExposurePolicyLimits(sectorLimit SectorExposureLimit, correlationLimit CorrelationExposureLimit) PortfolioExposurePolicy {
	if sectorLimit.MaxPositions < 0 || !math.IsNaN(sectorLimit.MaxCapitalFraction) && !math.IsInf(sectorLimit.MaxCapitalFraction, 0) ||
		sectorLimit.MaxCapitalFraction < 0 || sectorLimit.MaxCapitalFraction > 1 {
		panic("Sector exposure limits must be non-negative and bounded")
	}
	if !math.IsNaN(correlationLimit.MaxAbsoluteCorrelation) && !math.IsInf(correlationLimit.MaxAbsoluteCorrelation, 0) ||
		correlationLimit.MaxAbsoluteCorrelation < 0 || correlationLimit.MaxAbsoluteCorrelation > 1 ||
		correlationLimit.LookbackDays < 2 {
		panic("Correlation exposure limits must be bounded with a valid lookback")
	}
	return func(ctx PortfolioExposureContext) PortfolioExposureDecision {
		var sectorHoldings []PortfolioExposureContextHolding
		for _, h := range ctx.OpenHoldings {
			if ctx.CandidateSector != nil && h.Sector != nil && *ctx.CandidateSector == *h.Sector {
				sectorHoldings = append(sectorHoldings, h)
			}
		}
		if len(sectorHoldings) >= sectorLimit.MaxPositions {
			return PortfolioExposureDecisionReject("SECTOR_POSITION_LIMIT")
		}
		sectorNotional := coredecimal.Decimal{}
		for _, h := range sectorHoldings {
			sectorNotional = sectorNotional.Add(h.Notional)
		}
		frac, _ := coredecimal.Parse(strconv.FormatFloat(sectorLimit.MaxCapitalFraction, 'f', -1, 64))
		capitalCap := ctx.InitialCapital.Mul(frac).Money()
		if sectorNotional.Add(ctx.CandidateNotional).Cmp(capitalCap) > 0 {
			return PortfolioExposureDecisionReject("SECTOR_CAPITAL_LIMIT")
		}
		// correlation check
		for _, held := range ctx.OpenHoldings {
			candidateReturns := returns(ctx.ClosingPrices[ctx.CandidateSymbol], correlationLimit.LookbackDays)
			heldReturns := returns(ctx.ClosingPrices[held.Symbol], correlationLimit.LookbackDays)
			if len(candidateReturns) >= 2 && len(heldReturns) >= 2 {
				size := len(candidateReturns)
				if len(heldReturns) < size {
					size = len(heldReturns)
				}
				corr := correlation(candidateReturns[len(candidateReturns)-size:], heldReturns[len(heldReturns)-size:])
				if (!math.IsNaN(corr) && !math.IsInf(corr, 0)) && math.Abs(corr) > correlationLimit.MaxAbsoluteCorrelation {
					return PortfolioExposureDecisionReject("CORRELATION_LIMIT")
				}
			}
		}
		return PortfolioExposureDecisionAccept()
	}
}

func returns(prices []float64, lookback int) []float64 {
	if prices == nil {
		return nil
	}
	from := 0
	if len(prices) > lookback+1 {
		from = len(prices) - lookback - 1
	}
	var result []float64
	for i := from + 1; i < len(prices); i++ {
		previous := prices[i-1]
		current := prices[i]
		if previous <= 0 || current <= 0 || math.IsNaN(previous) || math.IsInf(previous, 0) || math.IsNaN(current) || math.IsInf(current, 0) {
			return []float64{}
		}
		result = append(result, current/previous-1.0)
	}
	return result
}

func correlation(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return math.NaN()
	}
	lm, rm := 0.0, 0.0
	for i := range left {
		lm += left[i]
		rm += right[i]
	}
	lm /= float64(len(left))
	rm /= float64(len(right))
	cov, lv, rv := 0.0, 0.0, 0.0
	for i := range left {
		l := left[i] - lm
		r := right[i] - rm
		cov += l * r
		lv += l * l
		rv += r * r
	}
	if lv == 0 || rv == 0 {
		return math.NaN()
	}
	return cov / math.Sqrt(lv*rv)
}

// MoneyCap stub placeholders for sector capital limit omitted intentionally; handled below.

// NewSectorExposureLimit validates inputs like Java record.
func NewSectorExposureLimit(maxPositions int, maxCapitalFraction float64) (SectorExposureLimit, error) {
	if maxPositions < 0 || !math.IsNaN(maxCapitalFraction) && !math.IsInf(maxCapitalFraction, 0) || maxCapitalFraction < 0 || maxCapitalFraction > 1 {
		return SectorExposureLimit{}, fmt.Errorf("Sector exposure limits must be non-negative and bounded")
	}
	return SectorExposureLimit{MaxPositions: maxPositions, MaxCapitalFraction: maxCapitalFraction}, nil
}

// NewCorrelationExposureLimit validates inputs like Java record.
func NewCorrelationExposureLimit(maxAbsoluteCorrelation float64, lookbackDays int) (CorrelationExposureLimit, error) {
	if !math.IsNaN(maxAbsoluteCorrelation) && !math.IsInf(maxAbsoluteCorrelation, 0) || maxAbsoluteCorrelation < 0 || maxAbsoluteCorrelation > 1 || lookbackDays < 2 {
		return CorrelationExposureLimit{}, fmt.Errorf("Correlation exposure limits must be bounded with a valid lookback")
	}
	return CorrelationExposureLimit{MaxAbsoluteCorrelation: maxAbsoluteCorrelation, LookbackDays: lookbackDays}, nil
}
