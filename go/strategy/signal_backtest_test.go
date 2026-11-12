package strategy

import (
	"errors"
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

type backtestSignalStrategy struct {
	entry func(int) (Decision, error)
	exit  func(int, OpenPosition) (ExitDecision, error)
}

func (backtestSignalStrategy) Type() string                                         { return "BACKTEST_TEST" }
func (backtestSignalStrategy) ParamSchema() ParamSchema                             { return ParamSchema{} }
func (backtestSignalStrategy) RequiredIndicators(StrategyParamsView) []IndicatorKey { return nil }
func (backtestSignalStrategy) WarmupBars(StrategyParamsView) int                    { return 1 }
func (s backtestSignalStrategy) EvaluateEntry(_ MarketContext, bar int, _ StrategyParamsView) (Decision, error) {
	return s.entry(bar)
}
func (s backtestSignalStrategy) EvaluateExit(_ MarketContext, bar int, position OpenPosition, _ StrategyParamsView) (ExitDecision, error) {
	return s.exit(bar, position)
}
func (backtestSignalStrategy) CrossFieldRules() []CrossFieldRule { return nil }

func TestRunSignalBacktestUsesNextOpenAndSignalsForExit(t *testing.T) {
	candles := signalBacktestCandles(4)
	stop, _ := coredecimal.Parse("99")
	target, _ := coredecimal.Parse("105")
	exitPrice, _ := coredecimal.Parse("102")
	strategy := backtestSignalStrategy{
		entry: func(bar int) (Decision, error) {
			if bar == 1 {
				return Decision{Type: domain.SignalTypeBuy, SuggestedStop: &stop, SuggestedTarget: &target}, nil
			}
			return Decision{Type: domain.SignalTypeHold}, nil
		},
		exit: func(bar int, _ OpenPosition) (ExitDecision, error) {
			if bar == 2 {
				return ExitWith(ExitReasonSignalExit, exitPrice, "golden exit"), nil
			}
			return ExitHold(), nil
		},
	}
	config := BacktestConfigDefaults()
	config.InitialCapital, _ = coredecimal.Parse("10000")
	config.RiskPerTradePct, _ = coredecimal.Parse("0.01")
	config.SlippagePct = coredecimal.Decimal{}
	config.BrokeragePerTrade = coredecimal.Decimal{}

	got, err := RunSignalBacktest("TEST", candles, config, strategy, StrategyParamsViewOf(nil))
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalTrades != 1 || got.Trades[0].Quantity != 100 || got.Trades[0].ExitReason != ExitReasonSignalExit {
		t.Fatalf("trade = %+v, want one 100-share signal-exit trade", got.Trades)
	}
	wantEntry, _ := coredecimal.Parse("100")
	wantExit, _ := coredecimal.Parse("102")
	if got.Trades[0].EntryPrice.Cmp(wantEntry) != 0 || got.Trades[0].ExitPrice.Cmp(wantExit) != 0 {
		t.Fatalf("fill prices = %s -> %s, want 100 -> 102", got.Trades[0].EntryPrice, got.Trades[0].ExitPrice)
	}
	wantCosts := ZerodhaDeliveryCostModel()(wantEntry, wantExit, 100, coredecimal.Decimal{})
	wantPnL := coredecimal.FromUnits(coredecimal.Decimal{}.Sub(wantCosts).Units()).Add(mustParseDecimal("200")).Money()
	if got.Trades[0].PnL.Cmp(wantPnL) != 0 {
		t.Fatalf("net PnL = %s, want gross less Java Zerodha-cost result %s", got.Trades[0].PnL, wantPnL)
	}
	t.Logf("observed trade: entry=%s exit=%s reason=%s quantity=%d net_pnl=%s", got.Trades[0].EntryPrice, got.Trades[0].ExitPrice, got.Trades[0].ExitReason, got.Trades[0].Quantity, got.Trades[0].PnL)
}

func TestRunSignalBacktestLiquidatesAtFinalClose(t *testing.T) {
	candles := signalBacktestCandles(4)
	stop, _ := coredecimal.Parse("99")
	target, _ := coredecimal.Parse("105")
	strategy := backtestSignalStrategy{
		entry: func(bar int) (Decision, error) {
			if bar == 1 {
				return Decision{Type: domain.SignalTypeBuy, SuggestedStop: &stop, SuggestedTarget: &target}, nil
			}
			return Decision{Type: domain.SignalTypeHold}, nil
		},
		exit: func(int, OpenPosition) (ExitDecision, error) { return ExitHold(), nil },
	}
	config := BacktestConfigDefaults()
	config.InitialCapital, _ = coredecimal.Parse("10000")
	config.SlippagePct = coredecimal.Decimal{}
	result, err := RunSignalBacktest("TEST", candles, config, strategy, StrategyParamsViewOf(nil))
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalTrades != 1 || result.Trades[0].ExitReason != ExitReasonTimeStop || !result.Trades[0].ExitDate.Equal(candles[3].Date) {
		t.Fatalf("final liquidation = %+v, want one TIME_STOP trade on final candle", result.Trades)
	}
	t.Logf("observed final-close liquidation: date=%s exit=%s reason=%s", result.Trades[0].ExitDate.Format(time.RFC3339), result.Trades[0].ExitPrice, result.Trades[0].ExitReason)
}

func TestRunSignalBacktestReturnsStrategyErrors(t *testing.T) {
	wantErr := errors.New("strategy unavailable")
	strategy := backtestSignalStrategy{
		entry: func(int) (Decision, error) { return Decision{}, wantErr },
		exit:  func(int, OpenPosition) (ExitDecision, error) { return ExitHold(), nil },
	}
	_, err := RunSignalBacktest("TEST", signalBacktestCandles(3), BacktestConfigDefaults(), strategy, StrategyParamsViewOf(nil))
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped strategy error", err)
	}
	t.Logf("observed strategy error returned to caller: %v", err)
}

func signalBacktestCandles(count int) []domain.OhlcvCandle {
	candles := make([]domain.OhlcvCandle, count)
	for i := range candles {
		price, _ := coredecimal.Parse("100")
		date := time.Date(2024, time.January, i+1, 0, 0, 0, 0, time.UTC)
		candles[i] = domain.OhlcvCandleOf("TEST", date, price, price, price, price, 1_000)
	}
	return candles
}
