package strategy

import (
	"fmt"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
)

// PortfolioBacktestEngineSimulate runs the shared-capital portfolio backtest.
func PortfolioBacktestEngineSimulate(symbolResults []BacktestResult, config PortfolioBacktestConfig, evaluationStart, evaluationEnd time.Time, optional ...any) (PortfolioBacktestResult, error) { //nolint:funlen
	marketData := map[string][]domain.OhlcvCandle{}
	sectors := map[string]string{}
	var benchmark *domain.BenchmarkCandleSeries
	var strategyVariantID *string
	for _, arg := range optional {
		switch v := arg.(type) {
		case nil:
			// ignore
		case map[string][]domain.OhlcvCandle:
			marketData = v
		case map[string]string:
			sectors = v
		case *domain.BenchmarkCandleSeries:
			benchmark = v
		case domain.BenchmarkCandleSeries:
			bv := v
			benchmark = &bv
		case *string:
			strategyVariantID = v
		case string:
			sv := v
			strategyVariantID = &sv
		}
	}
	if evaluationStart.IsZero() || evaluationEnd.IsZero() || evaluationStart.After(evaluationEnd) {
		return PortfolioBacktestResult{}, fmt.Errorf("Portfolio inputs must be non-null and the window must be ordered") //nolint:staticcheck // exact Java message
	}
	if config.InitialCapital.Cmp(coredecimal.Decimal{}) <= 0 || config.MaxConcurrentPositions <= 0 {
		return PortfolioBacktestResult{}, fmt.Errorf("Portfolio capital and max positions must be positive") //nolint:staticcheck // exact Java message
	}
	if config.RiskManagementPolicy == nil {
		config.RiskManagementPolicy = domain.RiskManagementPolicyNone()
	}
	if config.PortfolioExposurePolicy == nil {
		config.PortfolioExposurePolicy = domain.PortfolioExposurePolicyNone()
	}
	if marketData == nil {
		marketData = map[string][]domain.OhlcvCandle{}
	}
	if sectors == nil {
		sectors = map[string]string{}
	}

	// candidate collection
	var candidates []BacktestTrade
	for _, result := range symbolResults {
		for _, trade := range result.Trades {
			if trade.EntryDate.Before(evaluationStart) || trade.ExitDate.After(evaluationEnd) {
				continue
			}
			candidates = append(candidates, trade)
		}
	}
	// sort
	// simple insertion sort by entry date, symbol, exit date
	for i := 1; i < len(candidates); i++ {
		j := i
		for j > 0 {
			a, b := candidates[j-1], candidates[j]
			if a.EntryDate.After(b.EntryDate) ||
				(a.EntryDate.Equal(b.EntryDate) && a.Symbol > b.Symbol) ||
				(a.EntryDate.Equal(b.EntryDate) && a.Symbol == b.Symbol && a.ExitDate.After(b.ExitDate)) {
				candidates[j-1], candidates[j] = candidates[j], candidates[j-1]
				j--
			} else {
				break
			}
		}
	}

	entriesByDate := map[time.Time][]BacktestTrade{}
	exitsByDate := map[time.Time][]BacktestTrade{}
	for _, c := range candidates {
		entriesByDate[c.EntryDate] = append(entriesByDate[c.EntryDate], c)
		exitsByDate[c.ExitDate] = append(exitsByDate[c.ExitDate], c)
	}

	accepted := []BacktestTrade{}
	open := map[string]BacktestTrade{}
	highestClose := map[string]coredecimal.Decimal{}
	unsettledByDate := map[time.Time]coredecimal.Decimal{}
	observationDates := portfolioObservationDates(marketData, evaluationStart, evaluationEnd)
	nextTradingDate := map[time.Time]time.Time{}
	if len(marketData) > 0 {
		nextTradingDate = portfolioNextTradingDates(observationDates)
	}
	cash := config.InitialCapital.Money()
	equityCurve := []PortfolioEquityPoint{}
	rejected := 0
	rejectionReasons := []string{}

	for _, date := range observationDates {
		if settled, ok := unsettledByDate[date]; ok {
			cash = cash.Add(settled)
			delete(unsettledByDate, date)
		}

		// risk policy stops
		for _, held := range openList(open) {
			candle := portfolioCandleOn(marketData, held.Symbol, date)
			if candle == nil {
				continue
			}
			hc, ok := highestClose[held.Symbol]
			if !ok {
				hc = held.EntryPrice
			}
			ctx := domain.RiskManagementContext{
				EntryPrice:            held.EntryPrice,
				InitialStop:           held.StopLoss,
				Target:                held.Target,
				CurrentClose:          candle.Close,
				CurrentLow:            candle.Low,
				CurrentHigh:           candle.High,
				HighestCloseBeforeBar: hc,
				HoldingDays:           int(date.Sub(held.EntryDate).Hours() / 24),
			}
			decision := config.RiskManagementPolicy(ctx)
			if decision.Exit {
				delete(open, held.Symbol)
				managed := portfolioManagedExit(held, date, decision.StopPrice, decision.Reason)
				settleDate := portfolioNextSettlementDate(date, nextTradingDate)
				proceeds := portfolioEntryNotional(managed).Add(managed.PnL)
				unsettledByDate[settleDate] = unsettledByDate[settleDate].Add(proceeds)
				portfolioReplaceAccepted(accepted, held, managed)
			}
		}

		// exits
		for _, trade := range exitsByDate[date] {
			held, ok := open[trade.Symbol]
			if !ok {
				continue
			}
			delete(open, trade.Symbol)
			proceeds := portfolioEntryNotional(held).Add(held.PnL)
			settleDate := portfolioNextSettlementDate(date, nextTradingDate)
			unsettledByDate[settleDate] = unsettledByDate[settleDate].Add(proceeds)
		}

		// update highest closes
		for _, held := range openList(open) {
			candle := portfolioCandleOn(marketData, held.Symbol, date)
			if candle != nil {
				hc := candle.Close
				if existing, ok := highestClose[held.Symbol]; ok && existing.Cmp(hc) > 0 {
					hc = existing
				}
				highestClose[held.Symbol] = hc
			}
		}

		// entries
		for _, candidate := range entriesByDate[date] {
			notional := portfolioEntryNotional(candidate)
			baseRejection := ""
			openPosition := false
			if _, ok := open[candidate.Symbol]; ok {
				openPosition = true
			}
			switch {
			case openPosition:
				baseRejection = "OPEN_POSITION"
			case len(open) >= config.MaxConcurrentPositions:
				baseRejection = "MAX_CONCURRENT_POSITIONS"
			case notional.Cmp(coredecimal.Decimal{}) <= 0:
				baseRejection = "INVALID_NOTIONAL"
			case notional.Cmp(cash) > 0:
				baseRejection = "INSUFFICIENT_CAPITAL"
			}
			exposureAccepted := true
			exposureReason := ""
			if baseRejection == "" {
				// build open holdings
				holdings := []domain.PortfolioExposureContextHolding{}
				for _, h := range openList(open) {
					var sector *string
					if sec, ok := sectors[h.Symbol]; ok {
						v := sec
						sector = &v
					}
					holdings = append(holdings, domain.PortfolioExposureContextHolding{
						Symbol:   h.Symbol,
						Sector:   sector,
						Notional: portfolioEntryNotional(h),
					})
				}
				var candidateSector *string
				if sec, ok := sectors[candidate.Symbol]; ok {
					s := sec
					candidateSector = &s
				}
				// closing prices float (sorted by date to match Java)
				closingPrices := map[string][]float64{}
				for sym, candles := range marketData {
					sorted := append([]domain.OhlcvCandle(nil), candles...)
					// insertion sort by date
					for i := 1; i < len(sorted); i++ {
						j := i
						for j > 0 && sorted[j-1].Date.After(sorted[j].Date) {
							sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
							j--
						}
					}
					var prices []float64
					for _, c := range sorted {
						prices = append(prices, decimalToFloat64(c.Close))
					}
					closingPrices[sym] = prices
				}
				decision := config.PortfolioExposurePolicy(domain.PortfolioExposureContext{
					CandidateSymbol:   candidate.Symbol,
					CandidateSector:   candidateSector,
					CandidateNotional: notional,
					InitialCapital:    config.InitialCapital,
					OpenHoldings:      holdings,
					ClosingPrices:     closingPrices,
				})
				if !decision.Accepted {
					exposureAccepted = false
					exposureReason = decision.Reason
				}
			}
			if baseRejection != "" || !exposureAccepted {
				rejected++
				if baseRejection != "" {
					rejectionReasons = append(rejectionReasons, baseRejection)
				} else {
					rejectionReasons = append(rejectionReasons, exposureReason)
				}
				continue
			}
			open[candidate.Symbol] = candidate
			accepted = append(accepted, candidate)
			cash = cash.Sub(notional)
		}

		// equity point
		unsettled := coredecimal.Decimal{}
		for _, v := range unsettledByDate {
			unsettled = unsettled.Add(v)
		}
		positionValue := coredecimal.Decimal{}
		for _, held := range openList(open) {
			positionValue = positionValue.Add(portfolioMarketValue(held, date, marketData))
		}
		equityCurvePoint := PortfolioEquityPoint{
			Date:                date,
			Equity:              cash.Add(unsettled).Add(positionValue),
			SettledCash:         cash,
			UnsettledProceeds:   unsettled,
			PositionMarketValue: positionValue,
		}
		equityCurve = append(equityCurve, equityCurvePoint)
	}

	finalCapital := config.InitialCapital
	if len(equityCurve) > 0 {
		finalCapital = equityCurve[len(equityCurve)-1].Equity
	}
	winners := 0
	for _, t := range accepted {
		if t.PnL.Cmp(coredecimal.Decimal{}) > 0 {
			winners++
		}
	}
	equityValues := make([]coredecimal.Decimal, len(equityCurve))
	for i, p := range equityCurve {
		equityValues[i] = p.Equity
	}
	maxDrawdown := MaxDrawdownPct(equityValues)
	cagr := CagrPct(config.InitialCapital, finalCapital, evaluationStart, evaluationEnd)
	totalReturn := TotalReturnPct(config.InitialCapital, finalCapital)
	benchmarkComparison := domain.UnavailableBenchmark(decimalToFloat64(totalReturn))
	if benchmark != nil {
		benchmarkComparison = portfolioBenchmarkComparison(*benchmark, decimalToFloat64(totalReturn))
	}
	return PortfolioBacktestResult{
		EvaluationStart:     evaluationStart,
		EvaluationEnd:       evaluationEnd,
		InitialCapital:      config.InitialCapital,
		FinalCapital:        finalCapital,
		TotalReturn:         totalReturn,
		MaxDrawdownPct:      maxDrawdown,
		SharpeRatio:         SharpeRatio(equityValues),
		CAGRPct:             cagr,
		SortinoRatio:        SortinoRatio(equityValues),
		CalmarRatio:         CalmarRatio(cagr, maxDrawdown),
		TotalTrades:         len(accepted),
		WinningTrades:       winners,
		RejectedTrades:      rejected,
		Trades:              accepted,
		EquityCurve:         equityCurve,
		RejectionReasons:    rejectionReasons,
		BenchmarkComparison: benchmarkComparison,
		StrategyVariantID:   strategyVariantID,
	}, nil
}

func portfolioBenchmarkComparison(series domain.BenchmarkCandleSeries, totalReturn float64) domain.BenchmarkComparison {
	candles := series.Candles
	if len(candles) < 2 {
		return domain.UnavailableBenchmark(totalReturn)
	}
	first := candles[0]
	last := candles[len(candles)-1]
	if first.Close.Cmp(coredecimal.Decimal{}) <= 0 || last.Close.Cmp(coredecimal.Decimal{}) <= 0 {
		return domain.UnavailableBenchmark(totalReturn)
	}
	returnPct := (decimalToFloat64(last.Close) - decimalToFloat64(first.Close)) / decimalToFloat64(first.Close) * 100.0
	return domain.Nifty50PriceBenchmark(totalReturn, returnPct)
}

func openList(open map[string]BacktestTrade) []BacktestTrade {
	out := []BacktestTrade{}
	for _, v := range open {
		out = append(out, v)
	}
	return out
}

func portfolioEntryNotional(trade BacktestTrade) coredecimal.Decimal {
	q, _ := coredecimal.Parse(fmt.Sprintf("%d", trade.Quantity))
	return trade.EntryPrice.Mul(q).Money()
}

func portfolioCandleOn(marketData map[string][]domain.OhlcvCandle, symbol string, date time.Time) *domain.OhlcvCandle {
	candles, ok := marketData[symbol]
	if !ok {
		return nil
	}
	for i := range candles {
		if candles[i].Date.Equal(date) {
			c := candles[i]
			return &c
		}
	}
	return nil
}

func portfolioMarketValue(trade BacktestTrade, date time.Time, marketData map[string][]domain.OhlcvCandle) coredecimal.Decimal {
	candles, ok := marketData[trade.Symbol]
	if !ok || len(candles) == 0 {
		return portfolioEntryNotional(trade)
	}
	for _, c := range candles {
		if c.Date.Equal(date) {
			return coredecimal.Decimal{}.Add(c.Close).Mul(coredecimalParseInt(trade.Quantity)).Money()
		}
	}
	return portfolioEntryNotional(trade)
}

func portfolioManagedExit(original BacktestTrade, exitDate time.Time, exitPrice coredecimal.Decimal, reason string) BacktestTrade {
	pnl := original.PnL.Add(exitPrice.Sub(original.ExitPrice).Mul(coredecimalParseInt(original.Quantity)))
	exitReason := ExitReasonBreakevenStop
	if reason == "TRAILING_STOP" {
		exitReason = ExitReasonTrailingStop
	}
	pnlPct := coredecimal.Decimal{}
	if portfolioEntryNotional(original).Cmp(coredecimal.Decimal{}) != 0 {
		pnlPct = coredecimal.PercentOf(pnl, portfolioEntryNotional(original))
	}
	return BacktestTrade{
		Symbol:      original.Symbol,
		EntryDate:   original.EntryDate,
		ExitDate:    exitDate,
		EntryPrice:  original.EntryPrice,
		ExitPrice:   exitPrice,
		StopLoss:    original.StopLoss,
		Target:      original.Target,
		Quantity:    original.Quantity,
		ExitReason:  exitReason,
		PnL:         pnl,
		PnLPct:      pnlPct,
		HoldingDays: int(exitDate.Sub(original.EntryDate).Hours() / 24),
	}
}

func portfolioReplaceAccepted(accepted []BacktestTrade, original, replacement BacktestTrade) {
	for i := range accepted {
		if accepted[i] == original {
			accepted[i] = replacement
			break
		}
	}
}

func portfolioNextSettlementDate(exitDate time.Time, nextTradingDate map[time.Time]time.Time) time.Time {
	if len(nextTradingDate) > 0 {
		if v, ok := nextTradingDate[exitDate]; ok {
			return v
		}
		return exitDate.AddDate(0, 0, 1)
	}
	settlement := exitDate.AddDate(0, 0, 1)
	for settlement.Weekday() == time.Saturday || settlement.Weekday() == time.Sunday {
		settlement = settlement.AddDate(0, 0, 1)
	}
	return settlement
}

func portfolioObservationDates(marketData map[string][]domain.OhlcvCandle, start, end time.Time) []time.Time {
	if len(marketData) == 0 {
		dates := []time.Time{}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			dates = append(dates, d)
		}
		return dates
	}
	seen := map[time.Time]bool{}
	var dates []time.Time
	for _, candles := range marketData {
		for _, c := range candles {
			if (c.Date.Equal(start) || c.Date.After(start)) && (c.Date.Equal(end) || c.Date.Before(end)) {
				if !seen[c.Date] {
					seen[c.Date] = true
					dates = append(dates, c.Date)
				}
			}
		}
	}
	// sort
	for i := 1; i < len(dates); i++ {
		j := i
		for j > 0 && dates[j-1].After(dates[j]) {
			dates[j-1], dates[j] = dates[j], dates[j-1]
			j--
		}
	}
	return dates
}

func portfolioNextTradingDates(dates []time.Time) map[time.Time]time.Time {
	next := map[time.Time]time.Time{}
	for i := 0; i+1 < len(dates); i++ {
		next[dates[i]] = dates[i+1]
	}
	return next
}

func coredecimalParseInt(n int) coredecimal.Decimal {
	d, _ := coredecimal.Parse(fmt.Sprintf("%d", n))
	return d
}
