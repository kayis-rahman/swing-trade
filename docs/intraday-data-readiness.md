# Intraday data readiness (I0)

Run the Yahoo-only coverage sampler from `backend/` with a confirmed NSE-session date range:

```bash
./gradlew :data:intradayDataReadiness --args='--from 2026-10-05 --to 2026-10-07 --output-dir /tmp/intraday-readiness-yahoo'
```

The fixed sample is `RELIANCE`, `TCS`, `INFY`, `HDFCBANK`, and `ICICIBANK`; the command requests daily (`D`) and 15-minute (`15m`) candles from Yahoo's public chart API. It writes a per-symbol/session CSV ledger and aggregate Markdown report to the requested output directory. The default date range is the same short sample window and the default output is under the system temporary directory.

Expected counts are one daily bar or 25 15-minute bars per session. The ledger distinguishes client-accepted returned bars, session-valid, missing, and invalid-timestamp bars; missing-bar rate is missing expected bars divided by requested bars. Returned counts exclude candles already rejected by `YahooFinanceClient` for null/zero OHLCV or dates outside the request; raw provider payloads are not retained. Session timestamps are checked against the NSE regular-session grid (09:15 through 15:15 IST in 15-minute increments). The standalone command treats weekdays as expected sessions and does not connect to the app's NSE holiday database: select a window of confirmed exchange sessions or holiday dates will be counted as missing.

The sampler does not authenticate to Fyers or measure its availability because this MVP check is public-Yahoo-only and Fyers requires credentials. It makes no broker calls or orders. Yahoo request throttling and retry behavior are documented in the [Yahoo Finance API reference](yahoo-finance-api.md). The command does not deliberately probe rate limits. Its report includes actual HTTP attempt count, retry count, minimum spacing between HTTP attempts, and observed 429 and other non-2xx response counts for the run; counters are zero when none were observed. Offline deterministic cases belong in test sources and are not provider evidence.
