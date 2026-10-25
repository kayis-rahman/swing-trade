package data

import (
	"encoding/json"
	"fmt"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// YahooCandle is one accepted OHLCV bar parsed from Yahoo's chart endpoint.
type YahooCandle struct {
	Symbol        string
	Date          time.Time
	BarTime       time.Time
	Open          coredecimal.Decimal
	High          coredecimal.Decimal
	Low           coredecimal.Decimal
	Close         coredecimal.Decimal
	AdjustedClose coredecimal.Decimal
	Volume        int64
}

type yahooChartEnvelope struct {
	Chart struct {
		Result []struct {
			Timestamp []int64 `json:"timestamp"`
			Meta      struct {
				Timezone string `json:"timezone"`
			} `json:"meta"`
			Indicators struct {
				Quote []struct {
					Open   []*json.Number `json:"open"`
					High   []*json.Number `json:"high"`
					Low    []*json.Number `json:"low"`
					Close  []*json.Number `json:"close"`
					Volume []*int64       `json:"volume"`
				} `json:"quote"`
				Adjusted []struct {
					Close []*json.Number `json:"adjclose"`
				} `json:"adjclose"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

// ParseYahooChart parses Yahoo chart data using Java's date and null-bar rules.
func ParseYahooChart(payload []byte, symbol string, startDate, endDate time.Time, interval string) ([]YahooCandle, error) {
	var response yahooChartEnvelope
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("decode Yahoo chart response: %w", err)
	}
	if len(response.Chart.Result) == 0 || len(response.Chart.Result[0].Indicators.Quote) == 0 {
		return []YahooCandle{}, nil
	}
	result := response.Chart.Result[0]
	quote := result.Indicators.Quote[0]
	zone := yahooExchangeLocation(result.Meta.Timezone)
	start, end := exchangeDate(startDate.In(zone), zone), exchangeDate(endDate.In(zone), zone)
	candles := make([]YahooCandle, 0, len(result.Timestamp))
	for i, stamp := range result.Timestamp {
		if !yahooHasBar(quote, i) {
			continue
		}
		bar := time.Unix(stamp, 0).In(zone)
		date := exchangeDate(bar, zone)
		if date.Before(start) || date.After(end) || bar.Weekday() == time.Saturday || bar.Weekday() == time.Sunday {
			continue
		}
		candle, accepted, err := parseYahooBar(quote, result.Indicators.Adjusted, i, symbol, date, bar, interval)
		if err != nil {
			return nil, err
		}
		if accepted {
			candles = append(candles, candle)
		}
	}
	return candles, nil
}

func yahooExchangeLocation(name string) *time.Location {
	if name != "" {
		if location, err := time.LoadLocation(name); err == nil {
			return location
		}
	}
	return time.FixedZone("IST", 5*60*60+30*60)
}

func exchangeDate(value time.Time, zone *time.Location) time.Time {
	local := value.In(zone)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
}

func yahooHasBar(quote struct {
	Open   []*json.Number `json:"open"`
	High   []*json.Number `json:"high"`
	Low    []*json.Number `json:"low"`
	Close  []*json.Number `json:"close"`
	Volume []*int64       `json:"volume"`
}, i int) bool {
	return i < len(quote.Open) && i < len(quote.High) && i < len(quote.Low) && i < len(quote.Close) && i < len(quote.Volume)
}

func parseYahooBar(quote struct {
	Open   []*json.Number `json:"open"`
	High   []*json.Number `json:"high"`
	Low    []*json.Number `json:"low"`
	Close  []*json.Number `json:"close"`
	Volume []*int64       `json:"volume"`
}, adjusted []struct {
	Close []*json.Number `json:"adjclose"`
}, i int, symbol string, date, bar time.Time, interval string) (YahooCandle, bool, error) {
	if quote.Open[i] == nil || quote.High[i] == nil || quote.Low[i] == nil || quote.Close[i] == nil || quote.Volume[i] == nil || *quote.Volume[i] == 0 {
		return YahooCandle{}, false, nil
	}
	open, err := yahooPrice(quote.Open[i], "open", i)
	if err != nil {
		return YahooCandle{}, false, err
	}
	high, err := yahooPrice(quote.High[i], "high", i)
	if err != nil {
		return YahooCandle{}, false, err
	}
	low, err := yahooPrice(quote.Low[i], "low", i)
	if err != nil {
		return YahooCandle{}, false, err
	}
	closePrice, err := yahooPrice(quote.Close[i], "close", i)
	if err != nil {
		return YahooCandle{}, false, err
	}
	if closePrice.Cmp(coredecimal.Decimal{}) == 0 {
		return YahooCandle{}, false, nil
	}
	adjustedClose := closePrice
	if len(adjusted) > 0 && i < len(adjusted[0].Close) && adjusted[0].Close[i] != nil {
		adjustedClose, err = yahooPrice(adjusted[0].Close[i], "adjusted close", i)
		if err != nil {
			return YahooCandle{}, false, err
		}
	}
	barTime := date
	if interval != "1d" {
		barTime = bar
	}
	return YahooCandle{Symbol: symbol, Date: date, BarTime: barTime, Open: open, High: high, Low: low, Close: closePrice, AdjustedClose: adjustedClose, Volume: *quote.Volume[i]}, true, nil
}

func yahooPrice(value *json.Number, field string, index int) (coredecimal.Decimal, error) {
	price, err := coredecimal.Parse(value.String())
	if err != nil {
		return coredecimal.Decimal{}, fmt.Errorf("parse Yahoo %s at index %d: %w", field, index, err)
	}
	return price, nil
}
