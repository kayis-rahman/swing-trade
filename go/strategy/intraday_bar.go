package strategy

import (
	"fmt"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
)

// IntradayBar mirrors Java record.
type IntradayBar struct {
	Timestamp time.Time
	Open      coredecimal.Decimal
	High      coredecimal.Decimal
	Low       coredecimal.Decimal
	Close     coredecimal.Decimal
	Volume    int64
}

var nseZone = time.FixedZone("Asia/Kolkata", 5*60*60+30*60)

func NewIntradayBar(timestamp time.Time, open, high, low, close coredecimal.Decimal, volume int64) (IntradayBar, error) {
	if timestamp.IsZero() {
		return IntradayBar{}, fmt.Errorf("Timestamp and OHLC values are required")
	}
	if high.Cmp(low) < 0 || high.Cmp(open) < 0 || high.Cmp(close) < 0 ||
		low.Cmp(open) > 0 || low.Cmp(close) > 0 || volume < 0 {
		return IntradayBar{}, fmt.Errorf("Invalid OHLCV bar")
	}
	return IntradayBar{
		Timestamp: timestamp, Open: open, High: high, Low: low, Close: close, Volume: volume,
	}, nil
}

func (b IntradayBar) SessionDate() time.Time {
	return b.Timestamp.In(nseZone)
}

func (b IntradayBar) SessionDateLocal() string {
	return b.Timestamp.In(nseZone).Format("2006-01-02")
}

func (b IntradayBar) AsCandle(symbol string) domain.OhlcvCandle {
	return domain.OhlcvCandleOf(symbol, b.Timestamp.In(nseZone).Truncate(24*time.Hour), b.Open, b.High, b.Low, b.Close, b.Volume)
}
