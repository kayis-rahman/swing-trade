package strategy

import (
	"testing"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestZerodhaDeliveryCostModelRoundTrip(t *testing.T) {
	model := ZerodhaDeliveryCostModel()
	entry, _ := coredecimal.Parse("100")
	exit, _ := coredecimal.Parse("110")
	brokerage, _ := coredecimal.Parse("20")
	got := model(entry, exit, 10, brokerage)
	if got.String() != "41.8600" {
		t.Fatalf("expected 41.86, got %s", got.String())
	}
}
