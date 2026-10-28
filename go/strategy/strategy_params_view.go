package strategy

import (

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// StrategyParamsView mirrors Java's resolved strategy parameter map access.
type StrategyParamsView struct {
	values map[string]any
}

func StrategyParamsViewOf(values map[string]any) StrategyParamsView {
	if values == nil {
		values = map[string]any{}
	}
	return StrategyParamsView{values: values}
}

func (v StrategyParamsView) GetInt(name string) int {
	switch typed := v.values[name].(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func (v StrategyParamsView) GetDecimal(name string) coredecimal.Decimal {
	switch typed := v.values[name].(type) {
	case coredecimal.Decimal:
		return typed
	case string:
		dec, _ := coredecimal.Parse(typed)
		return dec
	default:
		return coredecimal.Decimal{}
	}
}
