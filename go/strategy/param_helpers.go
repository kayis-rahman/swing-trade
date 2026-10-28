package strategy

import (
	"strconv"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// decimalPtr parses a programmer-written bound; invalid literals cannot occur.
func decimalPtr(value string) *coredecimal.Decimal {
	parsed, err := coredecimal.Parse(value)
	if err != nil {
		panic(err)
	}
	return &parsed
}

// decimalDefault parses a programmer-written default.
func decimalDefault(value string) coredecimal.Decimal {
	parsed, err := coredecimal.Parse(value)
	if err != nil {
		panic(err)
	}
	return parsed
}

// paramInt reads a resolved INT parameter, tolerating the numeric shapes
// callers pass in tests.
func paramInt(params map[string]any, name string) int {
	switch typed := params[name].(type) {
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

// paramDecimal reads a resolved DECIMAL parameter.
func paramDecimal(params map[string]any, name string) coredecimal.Decimal {
	switch typed := params[name].(type) {
	case coredecimal.Decimal:
		return typed
	case string:
		parsed, err := coredecimal.Parse(typed)
		if err != nil {
			return coredecimal.Decimal{}
		}
		return parsed
	default:
		return coredecimal.Decimal{}
	}
}

func itoa(value int) string { return strconv.Itoa(value) }

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
