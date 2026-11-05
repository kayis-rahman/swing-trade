package strategy

import (
	"fmt"
	"strings"
)

// ExcludedSymbol mirrors Java record.
type ExcludedSymbol struct {
	Symbol string
	Reason string
}

// NewExcludedSymbol mirrors the Java record constructor.
func NewExcludedSymbol(symbol, reason string) (ExcludedSymbol, error) {
	if strings.TrimSpace(symbol) == "" || strings.TrimSpace(reason) == "" {
		return ExcludedSymbol{}, fmt.Errorf("symbol and reason must be non-blank")
	}
	return ExcludedSymbol{Symbol: symbol, Reason: reason}, nil
}
