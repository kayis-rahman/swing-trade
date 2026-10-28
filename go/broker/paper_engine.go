package broker

import (
	"fmt"
	"math/big"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// PaperEngine is the phase 7 paper-trading placeholder: it tracks cash only.
type PaperEngine struct {
	Cash coredecimal.Decimal
}

// NewPaperEngine builds an engine with the supplied starting cash.
func NewPaperEngine(cash coredecimal.Decimal) *PaperEngine {
	return &PaperEngine{Cash: cash}
}

// PlaceBuy validates cash without mutating state; order accounting is a later
// phase 7 slice.
func (e *PaperEngine) PlaceBuy(quantity int, price coredecimal.Decimal) error {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(coredecimal.MoneyScale), nil)
	costRat := new(big.Rat).Mul(
		new(big.Rat).SetFrac(price.Units(), scale),
		new(big.Rat).SetInt64(int64(quantity)),
	)
	cost, err := coredecimal.Parse(costRat.FloatString(coredecimal.MoneyScale))
	if err != nil {
		return err
	}
	if e.Cash.Cmp(cost) < 0 {
		return fmt.Errorf("insufficient cash")
	}
	return nil
}
