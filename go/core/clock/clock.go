package clock

import "time"

// Clock is the wall-clock port used by runtime code and deterministic tests.
type Clock interface {
	// Now returns the current wall-clock time.
	Now() time.Time
}

// System is the production wall clock in Asia/Kolkata.
type System struct{}

// Now returns the current time in the fixed Asia/Kolkata timezone.
func (System) Now() time.Time {
	const offset = 5*60*60 + 30*60
	return time.Now().In(time.FixedZone("Asia/Kolkata", offset))
}
