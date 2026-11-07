package strategy

import "time"

// WalkForwardFold mirrors Java's WalkForwardFold record (plan §6.3).
type WalkForwardFold struct {
	TestStart time.Time
	TestEnd   time.Time
	Result    any
}
