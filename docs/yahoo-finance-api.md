# Yahoo Finance API Reference

> **Warning**: Yahoo Finance has NO official API. Yahoo killed the yfapi in 2017.
> All endpoints are undocumented, reverse-engineered, and can break without notice.
> This doc is based on the community-maintained [yahoo-finance2](https://github.com/gadicc/yahoo-finance2) library (v2.x).

## Host

```
query1.finance.yahoo.com   (primary — query2 aggressively blocks automated requests with 429)
query2.finance.yahoo.com   (fallback — works for browser clients)
```

No authentication, crumb, or cookie required for v8/chart and v1/search endpoints.
The v7/finance/quote endpoint is dead (401).

## Usage boundary

Yahoo Finance is an undocumented public service with no SLA. In this project it is intended for
local development, paper-trading experiments, and historical backfills only; production market
data should use an authenticated provider with an explicit exchange-data contract.

`YahooFinanceClient.fetchCandle(symbol, date)` requests only the requested UTC day (the end bound
is the following day because Yahoo's `period2` is exclusive), selects the matching exchange-local
timestamp, and logs a warning when the response has no usable close value.

---

### 429 Blocking

`query2.finance.yahoo.com` aggressively blocks non-browser clients with HTTP 429 (Too Many Requests).
This happens even with a browser User-Agent header. `query1` is more permissive and should be used
as the primary host for automated clients. If you get 429, switch to `query1` or add delays between
requests. The Yahoo Finance client in this project uses `query1` and applies a one-second shared
request throttle with bounded retries for 429 and 5xx responses. It also exposes a read-only
snapshot of observed HTTP attempts, retry count, minimum attempt spacing, and non-2xx response
counts for the I0 readiness report; these counters do not alter request or retry behavior.

---

## 1. Historical Chart (OHLCV)

**Endpoint**: `GET https://query2.finance.yahoo.com/v8/finance/chart/{symbol}`

**Reference**: [yahoo-finance2 chart module](https://github.com/gadicc/yahoo-finance2/blob/dev/docs/modules/chart.md)

### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `period1` | epoch seconds | Yes | — | Start of range |
| `period2` | epoch seconds | No | `now` | End of range |
| `interval` | string | No | `1d` | `1m` `2m` `5m` `15m` `30m` `60m` `90m` `1h` `1d` `5d` `1wk` `1mo` `3mo` |
| `includePrePost` | boolean | No | `true` | Include pre/post market data (not relevant for Indian stocks) |
| `events` | string | No | `div|split|earn` | Include dividends/splits/earnings data. Use `history` for OHLCV only |
| `lang` | string | No | `en-US` | Language code |
| `useYfid` | boolean | No | `true` | Use Yahoo Finance ID |

### Response Structure

```json
{
  "chart": {
    "result": [{
      "meta": {
        "symbol": "RELIANCE.NS",
        "fullExchangeName": "NSE",
        "instrumentType": "EQUITY",
        "currency": "INR",
        "longName": "Reliance Industries Limited",
        "shortName": "RELIANCE INDUSTRIES LTD",
        "regularMarketPrice": 1316.5,
        "fiftyTwoWeekHigh": 1611.8,
        "fiftyTwoWeekLow": 1253.2,
        "chartPreviousClose": 1356.3,
        "regularMarketTime": 1782381599,
        "firstTradeDate": 820467900,
        "timezone": "IST",
        "gmtoffset": 19800
      },
      "timestamp": [1704998400, 1705257600, ...],
      "indicators": {
        "quote": [{
          "open": [100.0, 2500.0, ...],
          "high": [105.0, 2510.0, ...],
          "low": [99.0, 2440.0, ...],
          "close": [104.0, 2500.0, ...],
          "volume": [5000000, 4800000, ...]
        }],
        "adjclose": [{
          "adjclose": [104.0, 2475.50, ...]
        }]
      }
    }],
    "error": null
  }
}
```

### Key Notes

- Arrays are parallel: `timestamp[i]` corresponds to `quote[0].open[i]`, `quote[0].close[i]`, `adjclose[0].adjclose[i]`
- `adjclose` array may be missing or empty for some symbols
- Pre-market placeholder candles have `volume = 0` — filter these out
- Missing/invalid data returns `null` values in arrays
- `timestamp` values are Unix epoch seconds (NOT days)
- `firstTradeDate` is also epoch seconds

---

## 2. Quote (Real-Time Price) — **DEAD**

**Endpoint**: `GET https://query2.finance.yahoo.com/v7/finance/quote?symbols={symbol}`

**Status**: **Shut off by Yahoo — returns 401 Unauthorized**

```json
{
  "finance": {
    "result": null,
    "error": {
      "code": "Unauthorized",
      "description": "User is unable to access this feature - https://bit.ly/yahoo-finance-api-feedback"
    }
  }
}
```

Yahoo killed this endpoint without notice. Both `query1` and `query2` return 401. This is part of Yahoo's broader restriction of undocumented API access.

**Replacement**: Use the chart endpoint's `meta` field (Section 1) which already provides most quote data:
`regularMarketPrice`, `regularMarketDayHigh`, `regularMarketDayLow`, `regularMarketVolume`,
`fiftyTwoWeekHigh`, `fiftyTwoWeekLow`, `chartPreviousClose`.

Or use your broker API (Fyers v3 `GetStockQuotes` or Upstox historical data) for real-time quotes.

---

## 3. Search

**Endpoint**: `GET https://query2.finance.yahoo.com/v1/finance/search?q={query}`

**Reference**: [yahoo-finance2 search module](https://github.com/gadicc/yahoo-finance2/blob/dev/docs/modules/search.md)

### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `q` | string | Yes | — | Search term (symbol, company name, keyword) |
| `lang` | string | No | `en-US` | Language code |
| `region` | string | No | `US` | Region code |
| `quotesCount` | int | No | `6` | Number of quote results |
| `newsCount` | int | No | `4` | Number of news results |
| `enableFuzzyQuery` | boolean | No | `false` | Fuzzy/partial matching |
| `enableEnhancedTrivialQuery` | boolean | No | `true` | Enhanced trivial query processing |
| `enableCb` | boolean | No | `true` | Enable callback wrapping (JSONP) |
| `enableNavLinks` | boolean | No | `true` | Enable navigation links |

### Response Structure

```json
{
  "quotes": [{
    "symbol": "RELIANCE.NS",
    "shortname": "Reliance Industries Limited",
    "quoteType": "EQUITY",
    "exchange": "NSI",
    "longname": "Reliance Industries Limited",
    "index": "quotes",
    "score": 20018.0,
    "typeDisp": "Equity",
    "isYahooFinance": true,
    "exchDisp": "NSE",
    "sector": "Energy",
    "industry": "Oil & Gas Refining & Marketing"
  },
  ...],
  "news": [...],
  "count": 13,
  "explains": [],
  "totalTime": 123,
  "timeTakenForQuotes": 50,
  "timeTakenForNews": 30,
  "timeTakenForPredefinedScreener": 10,
  "timeTakenForPathway": 0,
  "buoys": [...]
}
```

### Key Notes

- Returns both Yahoo Finance symbols AND non-Yahoo entities (Crunchbase companies)
- Filter by `isYahooFinance: true` for valid trading symbols
- `quoteType` distinguishes EQUITY, ETF, MUTUALFUND, CRYPTOCURRENCY, etc.
- `exchange` field indicates the trading venue (NSI for NSE, NYQ for NYSE, etc.)
- `exchDisp` is the display-friendly exchange name (NSE, NASDAQ, etc.)
- Additional fields: `score`, `sector`, `industry`, `typeDisp`, `longname`
- **Note**: `exchangeName` does NOT exist in the response — use `exchange` or `exchDisp`
- Useful for discovering symbols before fetching chart/quote data

---

## Response Error Format

All endpoints return errors in a consistent structure:

```json
{
  "finance": {
    "result": null,
    "error": {
      "code": "Bad Request",
      "description": "Missing required query parameter=q"
    }
  }
}
```

Or for chart:

```json
{
  "chart": {
    "result": null,
    "error": {
      "code": "NotFound",
      "description": "No data found for symbol XYZ"
    }
  }
}
```

HTTP-level errors return the status text as the error message.

---

## Comparison: Yahoo vs Our Implementation

| Feature | Yahoo API | YahooFinanceClient |
|---------|-----------|-------------------|
| Base URL | `query1.finance.yahoo.com` | `query1.finance.yahoo.com` |
| Chart endpoint | `/v8/finance/chart/{symbol}` | `/v8/finance/chart/{symbol}` |
| Quote endpoint | **DEAD (401)** | `/v7/finance/quote?symbols=X` ⚠️ |
| Search endpoint | `/v1/finance/search?q=X` | `/v1/finance/search?q=X` |
| Date format | Epoch seconds | Epoch seconds |
| adjClose parsing | `indicators.adjclose[0].adjclose` | Implemented |
| Pre-market filtering | `volume == 0` candles | Implemented |
| Meta extraction | `chart.result[0].meta` | Implemented |
| Rate limiting | Aggressive 429 on query2 | 1s minimum between requests, including retries |
| Retry/backoff | Provider-dependent | Up to 3 retries with exponential backoff for 429/5xx |
| Search filtering | `isYahooFinance` field | Filters non-Yahoo + non-EQUITY |
| Search `exchange` field | `exchange` (not `exchangeName`) | Reads `exchangeName` ⚠️ |

**Issues**:
- Automated requests can still be rate-limited even on `query1`; the client uses throttling and bounded retries
- Quote endpoint (`/v7/finance/quote`) is dead (401) — `fetchQuote`/`fetchQuotes` always return null
- Search reads `exchangeName` but API returns `exchange` — search results get null exchange

---

## Reliability Notes

From the yahoo-finance2 library:

- **Network errors**: request timeouts, no response
- **HTTP errors**: internal errors from Yahoo
- **Missing resources**: asking for data that doesn't exist
- **Validation errors**: Yahoo returns data that doesn't match expected schema
- **Delisted stocks**: queries that worked previously begin to throw errors, including historical data from before delisting
- **No SLA**: endpoints can and do change without notice
- **Blocking**: Yahoo aggressively blocks automated requests with 429 on `query2`; use `query1` as primary
- **Endpoint rot**: `/v7/finance/quote` was killed without notice (401); chart and search still work

## Alternatives for Quote Data

Since the quote endpoint is dead, use these for real-time quote data:

1. **Fyers v3 `GetStockQuotes`** — Already implemented in `FyersServiceClient`. Provides price, change, change%, high, low, prev close, volume. Requires Fyers broker account.
2. **Upstox historical candle data** — Provides OHLCV but not real-time quotes. Use `fetchCandles` for recent data.
3. **Chart endpoint `meta` field** — Already extracted from `/v8/finance/chart`. Provides price, day high/low, volume, 52w range, previous close. Missing: change, change%, avg volume.
4. **NSE/BSE official APIs** — NSE India (nseindia.com) has public APIs but aggressive anti-bot measures.

---

## References

- [yahoo-finance2 (main repo)](https://github.com/gadicc/yahoo-finance2)
- [yahoo-finance2 API docs](https://github.com/gadicc/yahoo-finance2/blob/dev/docs/README.md)
- [yahoo-finance2 chart module docs](https://github.com/gadicc/yahoo-finance2/blob/dev/docs/modules/chart.md)
- [yfinance (Python)](https://github.com/ranaroussi/yfinance) — best reference for endpoint behavior
- `YF_QUERY_HOST` default: `query2.finance.yahoo.com` (from [yahooFinanceFetch.ts](https://github.com/gadicc/yahoo-finance2/blob/dev/src/lib/yahooFinanceFetch.ts))
