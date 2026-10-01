# Swing Trade System

Automated swing trading system for NSE/BSE Indian equities. Implements 1-4 week hold periods on Nifty 500 stocks using technical analysis, LLM sentiment, and paper trading.

## Architecture

```
swing-trade/
├── backend/                # Multi-module Gradle project (Spring Boot 4.1.1)
│   ├── core/               # Domain models (Stock, OhlcvCandle, Signal, Position, Trade)
│   ├── data/               # Data ingestion & storage (Upstox/Fyers/Yahoo clients, JPA, Flyway)
│   ├── strategy/           # TA with TA4j, signal generation, backtesting
│   ├── llm/                # LLM client (vLLM/OpenAI), sentiment analysis, news
│   ├── broker/             # Paper trading engine, order/position management
│   ├── gpuhub/             # GPUHub elastic deployment API client
│   └── api/                # REST endpoints, scheduled jobs
├── dashboard/              # Vue 3 + TypeScript frontend
│   ├── src/                # Components, views, router, stores
│   └── tests/              # Vitest unit + Playwright E2E tests
├── infra/                  # Docker, env files, monitoring, nginx
├── docs/                   # Project documentation
├── dev-stack.sh            # Dev stack orchestration (see also bin/swingdev)
└── bin/e2e-gate            # Local end-to-end quality gate (tests + Docker + live checks)
```

### Module Dependencies

```
api → strategy, llm, broker, gpuhub, data, core
broker → strategy, data, core
strategy → data, llm, core
llm → data, core
data → core
gpuhub → (none — standalone)
core → (none)
```

## Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Java 21 |
| Framework | Spring Boot 4.1.1 |
| Build | Gradle 9.6.1 (Kotlin DSL, multi-module) |
| Database | PostgreSQL 16 + TimescaleDB |
| TA | TA4j 0.16 |
| LLM | LangChain4j 1.18.1 + vLLM |
| Frontend | Vue 3.5 + TypeScript + Tailwind CSS |
| DB Migrations | Flyway |

## Quick Start

Agent guidance: [`AGENTS.md`](AGENTS.md) is shared by Codex and Claude Code. [`CLAUDE.md`](CLAUDE.md) contains Claude Code-specific routing and skills.

### 1. Start Infrastructure

```bash
./dev-stack.sh start
```

This runs PostgreSQL on pi-node via SSH, with the app locally on your Mac.

### 2. Build Backend

```bash
cd backend
./gradlew build
```

### 3. Run Backend

```bash
cd backend
./gradlew :api:bootRun --args='--spring.profiles.active=local,fyers'
```

### 4. Run Frontend

```bash
cd dashboard
yarn
yarn dev
```

Dashboard: http://localhost:3003
API: http://localhost:8080

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | System health and component checks |
| `/api/signals/latest` | GET | Latest signal per symbol |
| `/api/signals/symbol/{symbol}` | GET | Signals for one symbol |
| `/api/signals/generate-all` | POST | Generate price-action signals for the active watchlist |
| `/api/signals/generate-all/stream` | POST | Stream signal-generation progress |
| `/api/positions` | POST | Open a paper-trading position |
| `/api/positions` | GET | View positions |
| `/api/positions/{symbol}/close` | POST | Close position |
| `/api/positions/performance` | GET | P&L and trade performance |
| `/api/watchlist` | GET/POST | Read or add active watchlist symbols |
| `/api/data/pull` | POST | Pull historical data for active watchlist symbols |
| `/api/candidate-scans` | GET/POST | Browse or start candidate scans; qualified results can activate the pilot wishlist |
| `/api/candidate-scans/{runId}/results` | GET | Browse paginated scan results and source/OOS outcomes |
| `/api/settings/pi/status` | GET | View lazy Pi llama-server lifecycle and readiness status |

## Trading Strategy

**BUY signals** (4+ conditions): EMA 20 > EMA 50 crossover, RSI < 30, MACD positive, price above EMA, volume confirmation, positive momentum.

**SELL signals** (4+ conditions): EMA bearish crossover, RSI > 70, MACD negative, volume spike bearish, price decline.

**Risk management**: 2x ATR stop loss, 2.5x risk-reward target, configurable position sizing.

## Testing

### Quality gate (whole system, in Docker)

```bash
./dev-stack.sh gate     # or: bin/e2e-gate
```

Runs the backend and frontend suites, builds the deployment images, boots a
disposable PostgreSQL + API + dashboard stack, asserts against it over HTTP and
SQL (including a settings write verified in the database), then tears the whole
thing down. Exits 0 on pass, 1 on fail. Method and known gaps:
[`docs/infra/local-e2e-gate.md`](docs/infra/local-e2e-gate.md).

### Individual checks

```bash
# Backend — all modules (unit tests)
cd backend && ./gradlew test

# Backend — specific module
cd backend && ./gradlew :data:test

# Backend — full verification (tests + PMD + checkstyle + integration tests)
cd backend && ./gradlew check

# Backend — coverage report
cd backend && ./gradlew jacocoTestReport

# Frontend unit tests
cd dashboard && yarn test

# Frontend E2E tests
cd dashboard && yarn playwright test
```

## Configuration

Environment in `infra/env/.env` (loaded by `dev-stack.sh` / `bin/swingdev`). Spring profiles: `local` (dev), `dev`, `fyers` (broker integration), `stage`.

## Disclaimer

This software is for educational purposes only. Trading stocks involves risk of loss.
