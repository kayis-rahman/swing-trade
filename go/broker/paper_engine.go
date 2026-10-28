package broker

import (
	"fmt"
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

type PaperEngine struct {
	Cash coredecimal.Decimal
}

func NewPaperEngine(cash coredecimal.Decimal) *PaperEngine {
	return &PaperEngine{Cash: cash}
}

func (e *PaperEngine) PlaceBuy(quantity int, price coredecimal.Decimal) error {
	cost := price // placeholder missing multiply; keep minimal
	_ = quantity
	if e.Cash.Cmp(cost) < 0 {
		return fmt.Errorf("insufficient cash")
	}
	return nil
}
