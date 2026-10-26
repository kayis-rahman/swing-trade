package clock

import (
	"testing"
	"time"
)

type fixed struct{ value time.Time }

func (f fixed) Now() time.Time { return f.value }

func TestSystemClockUsesIndiaStandardTime(t *testing.T) {
	if got := (System{}).Now().Location().String(); got != "Asia/Kolkata" {
		t.Fatalf("System clock location = %q, want Asia/Kolkata", got)
	}
}

func TestClockPortSupportsFixedTime(t *testing.T) {
	want := time.Date(2026, 10, 17, 13, 0, 0, 0, time.UTC)
	var clock Clock = fixed{value: want}
	if got := clock.Now(); !got.Equal(want) {
		t.Fatalf("Now() = %s, want %s", got, want)
	}
}
