package strategy

import "github.com/kayis-rahman/swing-trade/go/core/decimal"

// BacktestCostModel mirrors Java functional interface.
type BacktestCostModel func(entryPrice, exitPrice decimal.Decimal, quantity int, brokerage decimal.Decimal) decimal.Decimal
