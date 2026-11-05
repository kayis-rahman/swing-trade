package strategy

import (
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestNewDailyEquityPoint(t *testing.T) {
	one, _ := coredecimal.Parse("1")
	d, err := NewDailyEquityPoint(time.Now(), one)
	if err != nil {
		t.Fatal(err)
	}
	_ = d
	if _, err := NewDailyEquityPoint(time.Time{}, one); err == nil {
		t.Fatal("expected error")
	}
}
