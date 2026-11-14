package strategy

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// RunSignalBacktest simulates a configured SignalStrategy over chronological
// candles. It mirrors BacktestEngine's configured-strategy path, including
// next-bar-open entries, risk-budget sizing, costs, mark-to-market equity, and
// final-bar liquidation.
func RunSignalBacktest(symbol string, candles []domain.OhlcvCandle, config BacktestConfig, strategy SignalStrategy, params StrategyParamsView) (BacktestResult, error) { //nolint:funlen
	if strings.TrimSpace(symbol) == "" {
		return BacktestResult{}, fmt.Errorf("Symbol cannot be null or blank") //nolint:staticcheck // exact Java message
	}
	if strategy == nil {
		return BacktestResult{}, fmt.Errorf("Config, strategy and params cannot be null") //nolint:staticcheck // exact Java message
	}
	adjustedCandles := make([]domain.OhlcvCandle, len(candles))
	for i, candle := range candles {
		adjustedCandles[i] = candle.AdjustedForAnalysis()
	}
	candles = adjustedCandles
	context, err := NewMarketContext(symbol, candles)
	if err != nil {
		return BacktestResult{}, err
	}
	first, last := strategy.WarmupBars(params), len(candles)-1
	if first < 0 {
		first = 0
	}
	if first >= last {
		return BacktestResult{}, fmt.Errorf("Insufficient candles for strategy warm-up") //nolint:staticcheck // exact Java message
	}

	capital := config.InitialCapital.Money()
	trades := make([]BacktestTrade, 0)
	curve := make([]coredecimal.Decimal, 0, last-first+1)
	var open *OpenPosition
	for i := first; i <= last; i++ {
		close := context.View(i).Close()
		if open != nil {
			current := open.AdvanceHighWaterMark(close)
			decision, decisionErr := strategy.EvaluateExit(context, i, current, params)
			if decisionErr != nil {
				return BacktestResult{}, fmt.Errorf("evaluate signal exit at bar %d: %w", i, decisionErr)
			}
			if decision.Exit {
				if decision.ExitPrice == nil {
					return BacktestResult{}, fmt.Errorf("exiting decision has no exit price")
				}
				trade := closeSignalTrade(symbol, current, *decision.ExitPrice, candles[i].Date, i, decision.Reason, config)
				trades = append(trades, trade)
				capital = capital.Add(trade.PnL)
				open = nil
			} else {
				open = &current
			}
		}
		curve = append(curve, markToMarketSignal(capital, open, close))
		if open == nil && i < last {
			decision, decisionErr := strategy.EvaluateEntry(context, i, params)
			if decisionErr != nil {
				return BacktestResult{}, fmt.Errorf("evaluate signal entry at bar %d: %w", i, decisionErr)
			}
			if decision.Type != domain.SignalTypeBuy || decision.SuggestedStop == nil || decision.SuggestedTarget == nil {
				continue
			}
			entry := context.View(i + 1).Open().Mul(onePlus(config.SlippagePct))
			stop, target := *decision.SuggestedStop, *decision.SuggestedTarget
			risk := entry.Sub(stop)
			if risk.Cmp(coredecimal.Decimal{}) <= 0 || target.Cmp(entry) <= 0 {
				continue
			}
			quantity := riskBudgetShares(capital, config.RiskPerTradePct, risk)
			if quantity <= 0 {
				continue
			}
			position, positionErr := OpenPositionAt(i+1, candles[i+1].Date, entry, stop, target, quantity)
			if positionErr != nil {
				return BacktestResult{}, positionErr
			}
			open = &position
		}
	}
	if open != nil {
		index := last
		trade := closeSignalTrade(symbol, *open, candles[index].AdjustedForAnalysis().Close, candles[index].Date, index, ExitReasonTimeStop, config)
		trades = append(trades, trade)
		capital = capital.Add(trade.PnL)
	}
	curve = append(curve, capital)
	return buildSignalBacktestResult(symbol, trades, curve, capital, config, candles[first], candles[last]), nil
}

func closeSignalTrade(symbol string, open OpenPosition, exitPrice coredecimal.Decimal, exitDate time.Time, exitIndex int, reason ExitReason, config BacktestConfig) BacktestTrade {
	filledExit := exitPrice.Mul(oneMinus(config.SlippagePct))
	quantity := coredecimalParseInt(open.Quantity)
	grossPnL := filledExit.Sub(open.EntryPrice).Mul(quantity)
	costs := ZerodhaDeliveryCostModel()(open.EntryPrice, filledExit, open.Quantity, config.BrokeragePerTrade)
	pnl := grossPnL.Sub(costs).Money()
	entryCost := open.EntryPrice.Mul(quantity)
	return BacktestTrade{
		Symbol: symbol, EntryDate: open.EntryDate, ExitDate: exitDate,
		EntryPrice: open.EntryPrice, ExitPrice: filledExit, StopLoss: open.StopLoss,
		Target: open.Target, Quantity: open.Quantity, ExitReason: reason,
		PnL: pnl, PnLPct: coredecimal.PercentOf(grossPnL.Sub(costs), entryCost),
		HoldingDays: exitIndex - open.EntryIndex,
	}
}

func markToMarketSignal(capital coredecimal.Decimal, open *OpenPosition, close coredecimal.Decimal) coredecimal.Decimal {
	if open == nil {
		return capital
	}
	unrealized := close.Sub(open.EntryPrice).Mul(coredecimalParseInt(open.Quantity))
	return capital.Add(unrealized).Money()
}

func buildSignalBacktestResult(symbol string, trades []BacktestTrade, curve []coredecimal.Decimal, finalCapital coredecimal.Decimal, config BacktestConfig, start, end domain.OhlcvCandle) BacktestResult {
	wins, losses := 0, 0
	gainTotal, lossTotal := coredecimal.Decimal{}, coredecimal.Decimal{}
	for _, trade := range trades {
		if trade.PnL.Cmp(coredecimal.Decimal{}) > 0 {
			wins++
			gainTotal = gainTotal.Add(trade.PnLPct)
		} else {
			losses++
			lossTotal = lossTotal.Add(trade.PnLPct)
		}
	}
	avgGain, avgLoss := coredecimal.Decimal{}, coredecimal.Decimal{}
	if wins > 0 {
		avgGain = decimalQuotientRounded(gainTotal, coredecimalParseInt(wins), coredecimal.PercentScale)
	}
	if losses > 0 {
		avgLoss = absDecimal(decimalQuotientRounded(lossTotal, coredecimalParseInt(losses), coredecimal.PercentScale))
	}
	winRate := 0.0
	if len(trades) > 0 {
		winRate = float64(wins) / float64(len(trades)) * 100
	}
	initial := config.InitialCapital.Money()
	totalReturn := coredecimal.PercentOf(finalCapital.Sub(initial), initial)
	cagr := CagrPct(initial, finalCapital, start.Date, end.Date)
	drawdown := MaxDrawdownPct(curve)
	winRatio := winRate / 100
	expectancy := winRatio*decimalToFloat64(avgGain) - (1-winRatio)*decimalToFloat64(avgLoss)
	return BacktestResult{
		Symbol: symbol, TotalTrades: len(trades), WinningTrades: wins, LosingTrades: losses,
		WinRate: winRate, AvgGainPct: avgGain, AvgLossPct: avgLoss, MaxDrawdownPct: drawdown,
		SharpeRatio: SharpeRatio(curve), TotalReturn: totalReturn, Expectancy: expectancy,
		Trades: append([]BacktestTrade(nil), trades...), CAGRPct: cagr,
		SortinoRatio: SortinoRatio(curve), CalmarRatio: CalmarRatio(cagr, drawdown),
		BenchmarkComparison: BuyAndHoldComparison(totalReturn, start.AdjustedForAnalysis().Close, end.AdjustedForAnalysis().Close),
	}
}

func riskBudgetShares(capital, riskPct, riskPerShare coredecimal.Decimal) int {
	riskBudget := new(big.Rat).Mul(ratOf(capital), ratOf(riskPct))
	shares := new(big.Rat).Quo(riskBudget, ratOf(riskPerShare))
	if shares.Sign() <= 0 {
		return 0
	}
	whole := new(big.Int).Quo(shares.Num(), shares.Denom())
	if whole.Cmp(big.NewInt(int64(coredecimal.Int32Max))) > 0 {
		return int(coredecimal.Int32Max)
	}
	return int(whole.Int64())
}

func onePlus(value coredecimal.Decimal) coredecimal.Decimal  { return oneDecimal().Add(value) }
func oneMinus(value coredecimal.Decimal) coredecimal.Decimal { return oneDecimal().Sub(value) }
