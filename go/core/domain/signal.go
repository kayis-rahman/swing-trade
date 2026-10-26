// Package domain hosts the shared com.swingtrade.domain model types that the
// strategy, data, and llm modules all consume.
package domain

// SignalType mirrors Java's Signal.SignalType enum. The string form matches
// Java's enum name serialization for the signals table.
type SignalType string

// Signal type values matching Java's enum constants.
const (
	SignalTypeBuy  SignalType = "BUY"
	SignalTypeSell SignalType = "SELL"
	SignalTypeHold SignalType = "HOLD"
)

// Description mirrors SignalType.getDescription().
func (signalType SignalType) Description() string {
	switch signalType {
	case SignalTypeBuy:
		return "Buy Signal"
	case SignalTypeSell:
		return "Sell Signal"
	case SignalTypeHold:
		return "Hold"
	default:
		return ""
	}
}
