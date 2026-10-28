package strategy

import (
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// StrategyParamsView mirrors Java's resolved strategy parameter map access.
//
//nolint:revive // name mirrors Java's StrategyParamsView
type StrategyParamsView struct {
	values map[string]any
}

// StrategyParamsViewOf builds a view over resolved parameter values.
//
//nolint:revive // name mirrors Java's StrategyParamsView.of
func StrategyParamsViewOf(values map[string]any) StrategyParamsView {
	if values == nil {
		values = map[string]any{}
	}
	return StrategyParamsView{values: values}
}

// GetInt reads an INT parameter, treating a missing or mistyped key as zero.
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

// GetDecimal reads a DECIMAL parameter, treating a missing key as zero.
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

// GetString mirrors Java's StrategyParamsView.getString: ENUM values stay
// strings, and a missing key reads as the empty string.
func (v StrategyParamsView) GetString(name string) string {
	if value, ok := v.values[name].(string); ok {
		return value
	}
	return ""
}

// GetBool mirrors Java's StrategyParamsView.getBoolean: only a real bool
// reads as true, and a missing key reads as false.
func (v StrategyParamsView) GetBool(name string) bool {
	if value, ok := v.values[name].(bool); ok {
		return value
	}
	return false
}
