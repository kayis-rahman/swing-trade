package strategy

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/data"
)

// Data-quality gate constants matching Java's DataQualityGate.
const (
	// maxMissingBarsFraction: expected bars are Mon-Fri calendar days, so the
	// baseline NSE holiday rate sits below this8% cutoff (0.02 rejected every
	// symbol in practice).
	maxMissingBarsFraction = 0.08
	// splitJumpThreshold is the single-bar move beyond which the unadjusted
	// split heuristic fires (Java's SPLIT_JUMP_THRESHOLD of 0.40).
	splitJumpThreshold = "0.40"
)

// EvaluateDataQuality ports Java's DataQualityGate.evaluate over a
// chronological candle window: it returns excluded=true with the reason when
// the symbol must be dropped from a run (empty history, non-positive prices,
// too many missing weekday bars, or a suspected unadjusted split jump).
// The corporate-action limitation and the weekday-only expected-bar count
// are Java's documented heuristics, preserved verbatim.
func EvaluateDataQuality(symbol string, candles []data.YahooCandle, start, end time.Time) (reason string, excluded bool) {
	if len(candles) == 0 {
		return "No candles in evaluation window", true
	}
	if reason := checkPrices(candles); reason != "" {
		return reason, true
	}
	if reason := checkMissingBars(candles, start, end); reason != "" {
		return reason, true
	}
	if reason := checkUnadjustedSplitJumps(candles); reason != "" {
		return reason, true
	}
	return "", false
}

func checkPrices(candles []data.YahooCandle) string {
	for _, candle := range candles {
		if isNonPositive(candle.Open) || isNonPositive(candle.High) ||
			isNonPositive(candle.Low) || isNonPositive(candle.Close) {
			return fmt.Sprintf("Zero or negative price on %s", candle.Date.Format("2006-01-02"))
		}
	}
	return ""
}

func isNonPositive(value coredecimal.Decimal) bool {
	zero := coredecimal.Decimal{}
	return value.Cmp(zero) <= 0
}

// checkMissingBars compares stored bars against Mon-Fri calendar days in the
// window, the same non-holiday-aware approximation Java documents.
func checkMissingBars(candles []data.YahooCandle, start, end time.Time) string {
	expectedBars := countWeekdays(start, end)
	if expectedBars == 0 {
		return ""
	}
	actualBars := 0
	for _, candle := range candles {
		date := candle.Date
		if !date.Before(start) && !date.After(end) {
			actualBars++
		}
	}
	missingFraction := 1.0 - float64(actualBars)/float64(expectedBars)
	if missingFraction > maxMissingBarsFraction {
		return fmt.Sprintf("Missing bars %.1f%% exceeds %.1f%% threshold (expected ~%d, found %d)",
			missingFraction*100.0, maxMissingBarsFraction*100.0, expectedBars, actualBars)
	}
	return ""
}

func countWeekdays(start, end time.Time) int {
	count := 0
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		weekday := day.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			count++
		}
	}
	return count
}

// checkUnadjustedSplitJumps mirrors Java's best-effort heuristic: a
// close-to-close move above40%, computed as BigDecimal.divide(previousClose,
// 6, HALF_UP) over the exact decimal strings, then rendered through the same
// double formatting.
func checkUnadjustedSplitJumps(candles []data.YahooCandle) string {
	for i := 1; i < len(candles); i++ {
		previousClose := candles[i-1].Close
		if previousClose.Cmp(coredecimal.Decimal{}) == 0 {
			continue
		}
		difference := candles[i].Close.Sub(previousClose).String()
		difference = strings.TrimPrefix(difference, "-")
		move := divideHalfUpToScale(strings.TrimSpace(difference),
			strings.TrimSpace(previousClose.String()), 6)
		if compareDecimalTextTo(move, splitJumpThreshold) > 0 {
			percent, err := strconv.ParseFloat(move, 64)
			if err != nil {
				continue
			}
			return fmt.Sprintf(
				"Suspected unadjusted split/bonus: %.1f%% single-bar move on %s (heuristic, no corporate-action feed available to confirm)",
				percent*100.0, candles[i].Date.Format("2006-01-02"))
		}
	}
	return ""
}

// divideHalfUpToScale divides two exact decimal strings and rounds half-up to
// scale places, matching BigDecimal.divide(denominator, scale, HALF_UP).
func divideHalfUpToScale(numerator, denominator string, scale int) string {
	num, numOK := new(big.Rat).SetString(numerator)
	den, denOK := new(big.Rat).SetString(denominator)
	if !numOK || !denOK || den.Sign() == 0 {
		return "0." + strings.Repeat("0", scale)
	}
	absNum := new(big.Rat).Abs(num)
	quotient := new(big.Rat).Quo(absNum, den)
	// Round half-up at the requested scale: scale the rational to an integer
	// with remainder-based half-up rounding.
	scaled := new(big.Rat).Mul(quotient, new(big.Rat).SetInt(pow10(scale)))
	numeratorInt, denominatorInt := scaled.Num(), scaled.Denom()
	whole, remainder := new(big.Int).QuoRem(numeratorInt, denominatorInt, new(big.Int))
	twiceRemainder := new(big.Int).Lsh(remainder, 1)
	if twiceRemainder.Cmp(denominatorInt) >= 0 {
		whole.Add(whole, big.NewInt(1))
	}
	negative := num.Sign() < 0
	whole.Abs(whole)
	if scale == 0 {
		if negative {
			return "-" + whole.String()
		}
		return whole.String()
	}
	factor := pow10(scale)
	wholePart, fractionPart := new(big.Int).QuoRem(whole, factor, new(big.Int))
	formatted := fmt.Sprintf("%s.%0*s", wholePart.String(), scale, fractionPart.String())
	if negative {
		return "-" + formatted
	}
	return formatted
}

func pow10(exponent int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
}

// compareDecimalTextTo orders two plain decimal strings numerically.
func compareDecimalTextTo(left, right string) int {
	leftRat, leftOK := new(big.Rat).SetString(left)
	rightRat, rightOK := new(big.Rat).SetString(right)
	if !leftOK || !rightOK {
		return 0
	}
	return leftRat.Cmp(rightRat)
}
