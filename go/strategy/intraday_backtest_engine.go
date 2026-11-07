package strategy

import (
	"fmt"
	"sort"
	"strings"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
	"github.com/kayis-rahman/swing-trade/go/data"
)

const (
	intradayLastEntryBarMinutes    = 15 * 60
	intradaySquareOffBarMinutes    = 15*60 + 15
	intradaySquareOffCutoffMinutes = 15*60 + 20
)

type IntradayBacktestEngine struct {
	isTradingSession func(time.Time) bool
	costs            IntradayCostModel
	config           IntradayBacktestConfig
}

func NewIntradayBacktestEngine(isTradingSession func(time.Time) bool, costs IntradayCostModel, config IntradayBacktestConfig) *IntradayBacktestEngine {
	if isTradingSession == nil {
		isTradingSession = func(time.Time) bool { return true }
	}
	return &IntradayBacktestEngine{isTradingSession: isTradingSession, costs: costs, config: config}
}

func DefaultIntradayBacktestParams() StrategyParamsView {
	return StrategyParamsViewOf(map[string]any{})
}

func (e *IntradayBacktestEngine) Run(symbol string, input []IntradayBar, strategy SignalStrategy, params StrategyParamsView) (IntradayBacktestResult, error) {
	return e.run(symbol, input, strategy, params, true)
}

func (e *IntradayBacktestEngine) RunAsOf(symbol string, input []IntradayBar, strategy SignalStrategy, params StrategyParamsView, sessionDate time.Time, asOf time.Time) (IntradayBacktestResult, error) {
	if input == nil || sessionDate.IsZero() || asOf.IsZero() {
		return IntradayBacktestResult{}, fmt.Errorf("As-of time must be an expected session bar start")
	}
	ist := time.FixedZone("IST", 5*60*60+30*60)
	asOfLocal := asOf.In(ist)
	if asOfLocal.Second() != 0 || asOfLocal.Nanosecond() != 0 {
		return IntradayBacktestResult{}, fmt.Errorf("As-of time must be an expected session bar start")
	}
	asOfMin := asOfLocal.Hour()*60 + asOfLocal.Minute()
	starts := data.ExpectedNSESessionBarStarts(sessionDate, e.isTradingSession(sessionDate))
	found := false
	for _, s := range starts {
		si := s.In(ist)
		if si.Hour()*60+si.Minute() == asOfMin {
			found = true
			break
		}
	}
	if !found {
		return IntradayBacktestResult{}, fmt.Errorf("As-of time must be an expected session bar start")
	}
	var visible []IntradayBar
	hasAsOf := false
	for _, bar := range input {
		if bar.SessionDate().UTC().Format("2006-01-02") != sessionDate.UTC().Format("2006-01-02") {
			continue
		}
		if bar.SessionTimeMinutes() > asOfMin {
			continue
		}
		if !barStartMatches(starts, bar) {
			continue
		}
		if bar.SessionTimeMinutes() == asOfMin {
			hasAsOf = true
			b, err := NewIntradayBar(bar.Timestamp, bar.Open, bar.Open, bar.Open, bar.Open, 0)
			if err != nil {
				return IntradayBacktestResult{}, err
			}
			visible = append(visible, b)
		} else {
			visible = append(visible, bar)
		}
	}
	if !hasAsOf {
		return IntradayBacktestResult{}, fmt.Errorf("The selected as-of bar is missing")
	}
	return e.run(symbol, visible, strategy, params, false)
}

func (e *IntradayBacktestEngine) run(symbol string, input []IntradayBar, strategy SignalStrategy, params StrategyParamsView, forceFinalSquareOff bool) (IntradayBacktestResult, error) {
	if strings.TrimSpace(symbol) == "" || input == nil || strategy == nil {
		return IntradayBacktestResult{}, fmt.Errorf("Symbol, bars, strategy and params are required")
	}
	var bars []IntradayBar
	for _, bar := range input {
		sd := bar.SessionDate()
		if !e.isTradingSession(sd) {
			continue
		}
		if !isNormalSessionMinutes(bar.SessionTimeMinutes()) {
			continue
		}
		starts := data.ExpectedNSESessionBarStarts(sd, e.isTradingSession(sd))
		if !barStartMatches(starts, bar) {
			continue
		}
		bars = append(bars, bar)
	}
	sort.Slice(bars, func(i, j int) bool { return bars[i].Timestamp.Before(bars[j].Timestamp) })
	for i := 1; i < len(bars); i++ {
		if !bars[i].Timestamp.After(bars[i-1].Timestamp) {
			return IntradayBacktestResult{}, fmt.Errorf("Bars must have unique, increasing timestamps")
		}
	}
	if len(bars) == 0 {
		return IntradayBacktestResult{
			InitialCapital: e.config.InitialCapital,
			FinalCapital:   e.config.InitialCapital,
			NetPnL:         coredecimal.Decimal{},
			Trades:         []IntradayBacktestTrade{},
		}, nil
	}
	candles := make([]domain.OhlcvCandle, 0, len(bars))
	timestamps := make([]time.Time, 0, len(bars))
	for _, b := range bars {
		candles = append(candles, b.AsCandle(symbol))
		timestamps = append(timestamps, b.Timestamp)
	}
	ctx := MarketContextOfIntraday(symbol, candles, timestamps)

	trades := []IntradayBacktestTrade{}
	tradeCounts := map[time.Time]int{}
	dayPnl := map[time.Time]coredecimal.Decimal{}
	var position *OpenPosition
	entryAt := -1
	capital := e.config.InitialCapital

	for i := 0; i < len(bars); i++ {
		bar := bars[i]
		date := bar.SessionDate()
		if position != nil && position.EntryDate.Equal(date) {
			stopHit := bar.Low.Cmp(position.StopLoss) <= 0
			targetHit := !stopHit && bar.High.Cmp(position.Target) >= 0
			barrierExit := stopHit || targetHit
			exit := ExitHold()
			if stopHit {
				ref := position.StopLoss
				if bar.Open.Cmp(ref) < 0 {
					ref = bar.Open
				}
				exit = ExitWith(ExitReasonStopLoss, ref, "Intraday stop")
			} else if targetHit {
				ref := position.Target
				if bar.Open.Cmp(ref) > 0 {
					ref = bar.Open
				}
				exit = ExitWith(ExitReasonTargetHit, ref, "Intraday target")
			} else {
				ex, err := strategy.EvaluateExit(ctx, i, *position, params)
				if err == nil {
					exit = ex
				}
			}
			cutoff := forceFinalSquareOff && bar.SessionTimeMinutes() >= intradaySquareOffCutoffMinutes
			cutoffBar := forceFinalSquareOff && bar.SessionTimeMinutes() == intradaySquareOffBarMinutes
			sessionEnd := (forceFinalSquareOff && i+1 == len(bars)) || (i+1 < len(bars) && !bars[i+1].SessionDate().Equal(date))
			if exit.Exit || cutoff || cutoffBar || sessionEnd {
				var reference coredecimal.Decimal
				var reason string
				if barrierExit {
					if exit.ExitPrice != nil {
						reference = *exit.ExitPrice
					}
					reason = string(exit.Reason)
				} else if cutoff || cutoffBar {
					reference = bar.Open
					reason = "FORCED_SQUARE_OFF"
				} else if exit.Exit && exit.ExitPrice != nil {
					reference = *exit.ExitPrice
					reason = string(exit.Reason)
				} else {
					reference = bar.Close
					reason = "FORCED_SESSION_END"
				}
				sell := e.costs.SellFill(reference)
				fees, err := e.costs.RoundTripCost(position.EntryPrice, sell, position.Quantity)
				if err != nil {
					return IntradayBacktestResult{}, err
				}
				diff := sell.Sub(position.EntryPrice)
				pnl := diff.MulInt(position.Quantity).Sub(fees)
				trade := IntradayBacktestTrade{
					Strategy:   strategy.Type(),
					EntryTime:  bars[entryAt].Timestamp,
					ExitTime:   bar.Timestamp,
					EntryPrice: position.EntryPrice,
					ExitPrice:  sell,
					Quantity:   position.Quantity,
					Costs:      fees,
					PnL:        pnl,
					ExitReason: reason,
				}
				trades = append(trades, trade)
				capital = capital.Add(pnl)
				dayPnl[date] = dayPnl[date].Add(pnl)
				tradeCounts[date]++
				position = nil
				entryAt = -1
			}
		}

		if position == nil && i+1 < len(bars) {
			dc := tradeCounts[date]
			dp := dayPnl[date]
			if dc < e.config.MaxTradesPerDay &&
				dp.Cmp(coredecimal.Decimal{}.Sub(e.config.MaxDailyLoss)) > 0 &&
				bar.SessionTimeMinutes() <= intradayLastEntryBarMinutes &&
				bars[i+1].SessionDate().Equal(date) &&
				bars[i+1].SessionTimeMinutes() < intradaySquareOffBarMinutes &&
				i+1 >= strategy.WarmupBars(params) {
				decision, err := strategy.EvaluateEntry(ctx, i, params)
				if err == nil && decision.Type == domain.SignalTypeBuy {
					fillBar := bars[i+1]
					buy := e.costs.BuyFill(fillBar.Open)
					risk := decimalProduct(buy, mustParseDecimal("0.01"))
					stop := buy.Sub(risk)
					// target = buy * 1.02
					target := decimalProduct(buy, mustParseDecimal("1.02"))
					pos, err := OpenPositionAt(i+1, date, buy, stop, target, e.config.Quantity)
					if err == nil {
						position = &pos
						entryAt = i + 1
					}
				}
			}
		}
	}

	var openEntryTime *time.Time
	if position != nil && entryAt >= 0 {
		t := bars[entryAt].Timestamp
		openEntryTime = &t
	}
	netPnl := capital.Sub(e.config.InitialCapital)
	return IntradayBacktestResult{
		InitialCapital:        e.config.InitialCapital,
		FinalCapital:          capital,
		NetPnL:                netPnl,
		Trades:                trades,
		OpenPosition:          position,
		OpenPositionEntryTime: openEntryTime,
	}, nil
}

func isNormalSessionMinutes(mins int) bool {
	start := 9*60 + 15
	end := 15*60 + 30
	return mins >= start && mins < end
}

func barStartMatches(starts []time.Time, bar IntradayBar) bool {
	barLocal := bar.Timestamp.In(nseZone)
	for _, s := range starts {
		startLocal := s.In(nseZone)
		if startLocal.Year() == barLocal.Year() &&
			startLocal.Month() == barLocal.Month() &&
			startLocal.Day() == barLocal.Day() &&
			startLocal.Hour() == barLocal.Hour() &&
			startLocal.Minute() == barLocal.Minute() &&
			startLocal.Second() == barLocal.Second() &&
			startLocal.Nanosecond() == barLocal.Nanosecond() {
			return true
		}
	}
	return false
}
