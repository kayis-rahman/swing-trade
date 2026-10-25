package data

import (
	"encoding/json"
	"fmt"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

type fyersHistoryResponse struct {
	Status  string              `json:"s"`
	Message string              `json:"message"`
	Candles [][]json.RawMessage `json:"candles"`
}

// ParseFyersHistory parses Fyers v3 history bars into the shared candle model.
func ParseFyersHistory(payload []byte, symbol, interval string) ([]YahooCandle, error) {
	var response fyersHistoryResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("decode Fyers history response: %w", err)
	}
	if response.Status == "error" {
		return []YahooCandle{}, nil
	}
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return nil, fmt.Errorf("load NSE timezone: %w", err)
	}
	candles := make([]YahooCandle, 0, len(response.Candles))
	for index, row := range response.Candles {
		if len(row) < 6 {
			continue
		}
		candle, valid, err := parseFyersBar(row, symbol, interval, ist)
		if err != nil {
			return nil, fmt.Errorf("parse Fyers candle at index %d: %w", index, err)
		}
		if valid {
			candles = append(candles, candle)
		}
	}
	return candles, nil
}

func parseFyersBar(row []json.RawMessage, symbol, interval string, ist *time.Location) (YahooCandle, bool, error) {
	var epoch int64
	if err := json.Unmarshal(row[0], &epoch); err != nil {
		return YahooCandle{}, false, fmt.Errorf("timestamp: %w", err)
	}
	bar := time.Unix(epoch, 0).In(ist)
	date := time.Date(bar.Year(), bar.Month(), bar.Day(), 0, 0, 0, 0, ist)
	prices := make([]coredecimal.Decimal, 4)
	for i := range prices {
		var value json.Number
		if err := json.Unmarshal(row[i+1], &value); err != nil {
			return YahooCandle{}, false, fmt.Errorf("price at column %d: %w", i+1, err)
		}
		price, err := coredecimal.Parse(value.String())
		if err != nil {
			return YahooCandle{}, false, fmt.Errorf("price at column %d: %w", i+1, err)
		}
		prices[i] = price
	}
	var volume int64
	if err := json.Unmarshal(row[5], &volume); err != nil {
		return YahooCandle{}, false, fmt.Errorf("volume: %w", err)
	}
	if volume <= 0 {
		return YahooCandle{}, false, nil
	}
	barTime := date
	if interval != "D" {
		barTime = bar
	}
	return YahooCandle{Symbol: symbol, Timeframe: storageTimeframe(interval), Date: date, BarTime: barTime, Open: prices[0], High: prices[1], Low: prices[2], Close: prices[3], AdjustedClose: prices[3], Volume: volume}, true, nil
}
