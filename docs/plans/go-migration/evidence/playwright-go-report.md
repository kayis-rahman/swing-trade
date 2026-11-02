# Playwright against Go API - run report

Run date: 2026-11-02. Branch: `fm/swing-trade-go-playwright-on-go-v1`.
Stack: disposable PostgreSQL 16 on 127.0.0.1:55433 (go_port_test), Go API on 127.0.0.1:8081, Vite dashboard on 127.0.0.1:3003 (proxies /api to Go).
Stats from Playwright: expected=116, unexpected=20, skipped=7, total=143.

## Failure categorisation

| Spec | Test | Status | Category | Error (truncated) |
|---|---|---|---|---|
| api-openapi-audit.spec.ts | all safe OpenAPI GET routes return no server or rate-limit errors | fail | missing slice: Go does not serve /api-docs (OpenAPI) endpoint | SyntaxError: Unexpected non-whitespace character after JSON at position 4 (line  |
| views/multi-strategy.live.spec.ts | orchestrator shows the signal tournament panel | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/multi-strategy.live.spec.ts | signals can be grouped by symbol | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/multi-strategy.live.spec.ts | positions has a portfolio switcher with shadow books | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoContainText[2m([22m[32me |
| views/multi-strategy.live.spec.ts | symbol detail shows the strategy matrix | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/multi-strategy.live.spec.ts | strategy report renders leaderboard and arbitration rules | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/multi-strategy.live.spec.ts | home shows the strategy board | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/multi-strategy.live.spec.ts | backtest exposes run and saved-report controls | fail | other | Error: [2mexpect([22m[31mreceived[39m[2m).[22mtoEqual[2m([22m[32mexpect |
| views/multi-strategy.live.spec.ts | strategies page still loads | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/orchestrator-debug.spec.ts | orchestrator page — full feature check | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/stage-sanity.spec.ts | positions page loads | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/stage-sanity.spec.ts | sentiment page loads | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |
| views/stage-sanity.spec.ts | health endpoint returns UP | fail | other | Error: [2mexpect([22m[31mreceived[39m[2m).[22mtoBe[2m([22m[32mexpected |
| views/stage-sanity.spec.ts | metrics endpoint returns Prometheus format | fail | other | Error: [2mexpect([22m[31mreceived[39m[2m).[22mtoBe[2m([22m[32mexpected |
| views/stage-sanity.spec.ts | positions stats API returns valid JSON | fail | other | Error: [2mexpect([22m[31mreceived[39m[2m).[22mtoBe[2m([22m[32mexpected |
| views/stage-sanity.spec.ts | signals API returns array | fail | other | Error: [2mexpect([22m[31mreceived[39m[2m).[22mtoBe[2m([22m[32mexpected |
| views/stage-sanity.spec.ts | positions API returns a paginated response | fail | other | Error: [2mexpect([22m[31mreceived[39m[2m).[22mtoBe[2m([22m[32mexpected |
| views/stage-sanity.spec.ts | positions performance API returns trade stats | fail | other | Error: [2mexpect([22m[31mreceived[39m[2m).[22mtoBe[2m([22m[32mexpected |
| views/stage-sanity.spec.ts | watchlist API returns data | fail | other | Error: [2mexpect([22m[31mreceived[39m[2m).[22mtoBe[2m([22m[32mexpected |
| views/stage-sanity.spec.ts | sidebar navigation works | fail | other | Error: [2mexpect([22m[31mlocator[39m[2m).[22mtoBeVisible[2m([22m[2m)[2 |

## Per-test results

| Spec | Test | Result |
|---|---|---|
| api-isolated-audit.spec.ts | empty isolated signal workflows complete without provider or model calls | skipped |
| api-isolated-audit.spec.ts | isolated CRUD and state transitions complete without unexpected errors | skipped |
| api-isolated-audit.spec.ts | isolated validation and safe no-op branches return client responses | skipped |
| api-isolated-audit.spec.ts | remaining write routes reject incomplete or nonexistent requests without 5xx | skipped |
| api-isolated-audit.spec.ts | safe disconnected integrations and background job controls return expected states | skipped |
| api-openapi-audit.spec.ts | all OpenAPI write routes accept an OPTIONS preflight | skipped |
| api-openapi-audit.spec.ts | all safe OpenAPI GET routes return no server or rate-limit errors | unexpected |
| behaviors/functional-edge-cases.spec.ts | data ingestion range mode validates missing and reversed dates before submission | expected |
| behaviors/functional-edge-cases.spec.ts | positions form keeps required fields enforced and exposes limit price only for limit orders | expected |
| behaviors/functional-edge-cases.spec.ts | sentiment history shows an empty state for a symbol with no history | expected |
| behaviors/functional-edge-cases.spec.ts | watchlist add normalizes symbol and clears the form after confirmation | expected |
| behaviors/functional-edge-cases.spec.ts | watchlist cancel leaves the form closed and does not submit | expected |
| behaviors/functional-edge-cases.spec.ts | watchlist removal honors a cancelled confirmation | expected |
| behaviors/functional-edge-cases.spec.ts | watchlist toggle updates active state | expected |
| behaviors/paper-trading-behavior.spec.ts | dashboard page shows metrics cards | expected |
| behaviors/paper-trading-behavior.spec.ts | portfolio page loads with performance data | expected |
| behaviors/paper-trading-behavior.spec.ts | positions page renders with data or empty state | expected |
| behaviors/paper-trading-behavior.spec.ts | positions page renders without JS errors | expected |
| errors/error-management.spec.ts | HTTP 200 success:false keeps the position form open and preserves input | expected |
| errors/error-management.spec.ts | backend availability banner reports an unreachable service and recovers immediately | expected |
| errors/error-management.spec.ts | initial position load failure shows an accessible retry alert, not an empty state | expected |
| go-read-parity.spec.ts | Go read endpoints match Java responses on the shared local dataset | skipped |
| stage-smoke.spec.ts | loads and navigates read-only views without frontend exceptions | expected |
| views/intraday-view.spec.ts | scanner candidate handoff selects the returned cutoff in the replay form | expected |
| views/multi-strategy-ui.spec.ts | orchestrator: scoped start posts filters and matrix renders degraded state | expected |
| views/multi-strategy-ui.spec.ts | strategies: champion warning, mode toggle and inline validation error | expected |
| views/multi-strategy.live.spec.ts | backtest exposes run and saved-report controls | unexpected |
| views/multi-strategy.live.spec.ts | home shows the strategy board | unexpected |
| views/multi-strategy.live.spec.ts | orchestrator shows the signal tournament panel | unexpected |
| views/multi-strategy.live.spec.ts | positions has a portfolio switcher with shadow books | unexpected |
| views/multi-strategy.live.spec.ts | signals can be grouped by symbol | unexpected |
| views/multi-strategy.live.spec.ts | strategies page still loads | unexpected |
| views/multi-strategy.live.spec.ts | strategy report renders leaderboard and arbitration rules | unexpected |
| views/multi-strategy.live.spec.ts | symbol detail shows the strategy matrix | unexpected |
| views/orchestrator-debug.spec.ts | orchestrator page — full feature check | unexpected |
| views/page-behavior.spec.ts | Data Ingestion route renders its expected shell | expected |
| views/page-behavior.spec.ts | Portfolio at a glance route renders its expected shell | expected |
| views/page-behavior.spec.ts | Portfolio route renders its expected shell | expected |
| views/page-behavior.spec.ts | Positions route renders its expected shell | expected |
| views/page-behavior.spec.ts | Settings route renders its expected shell | expected |
| views/page-behavior.spec.ts | Signals route renders its expected shell | expected |
| views/page-behavior.spec.ts | Watchlist route renders its expected shell | expected |
| views/positions-view.spec.ts | Close Position modal opens when clicking Close button | expected |
| views/positions-view.spec.ts | Close Position modal shows P&L summary | expected |
| views/positions-view.spec.ts | Close button only visible for OPEN positions | expected |
| views/positions-view.spec.ts | New Position modal form fields are functional | expected |
| views/positions-view.spec.ts | New Position modal opens on button click | expected |
| views/positions-view.spec.ts | New Position modal respects order type selection | expected |
| views/positions-view.spec.ts | P&L displays with color (green for profit, red for loss) | expected |
| views/positions-view.spec.ts | empty state displays when no positions exist | expected |
| views/positions-view.spec.ts | filter buttons are clickable and toggle active state | expected |
| views/positions-view.spec.ts | modal closes on Cancel button click | expected |
| views/positions-view.spec.ts | modal closes on backdrop click | expected |
| views/positions-view.spec.ts | navigation to positions page works from URL directly | expected |
| views/positions-view.spec.ts | no JavaScript errors on page load | expected |
| views/positions-view.spec.ts | page header renders | expected |
| views/positions-view.spec.ts | refresh button triggers re-fetch | expected |
| views/positions-view.spec.ts | renders filter controls | expected |
| views/positions-view.spec.ts | rows have hover state | expected |
| views/positions-view.spec.ts | search + filter combination works | expected |
| views/positions-view.spec.ts | search input filters by symbol | expected |
| views/positions-view.spec.ts | shows loading state then resolves | expected |
| views/positions-view.spec.ts | status badges show color-coded classes | expected |
| views/positions-view.spec.ts | table is scrollable horizontally | expected |
| views/positions-view.spec.ts | table renders with correct column headers | expected |
| views/positions-view.spec.ts | table rows show position data when positions exist | expected |
| views/settings-view.spec.ts | All sections are card-panel styled | expected |
| views/settings-view.spec.ts | Auth error banner renders | expected |
| views/settings-view.spec.ts | Auth success banner renders | expected |
| views/settings-view.spec.ts | Discord enabled label is visible | expected |
| views/settings-view.spec.ts | Discord notification settings render | expected |
| views/settings-view.spec.ts | Discord test saves its settings and never contacts the external webhook in E2E | expected |
| views/settings-view.spec.ts | Discord toggle is clickable | expected |
| views/settings-view.spec.ts | Fyers auth code input appears with connect button | expected |
| views/settings-view.spec.ts | Fyers connection status indicator renders | expected |
| views/settings-view.spec.ts | Fyers controls match its disconnected status | expected |
| views/settings-view.spec.ts | Fyers controls reflect the current connection status | expected |
| views/settings-view.spec.ts | Health status badges show color-coded states | expected |
| views/settings-view.spec.ts | LLM section has model name helper text | expected |
| views/settings-view.spec.ts | Local llama.cpp backend reveals its configuration | expected |
| views/settings-view.spec.ts | PDF Extraction section renders | expected |
| views/settings-view.spec.ts | Save All Settings button renders | expected |
| views/settings-view.spec.ts | System Health section renders with component statuses | expected |
| views/settings-view.spec.ts | Trading Configuration inputs accept numeric values | expected |
| views/settings-view.spec.ts | Trading Configuration section renders | expected |
| views/settings-view.spec.ts | Trading mode select has correct options | expected |
| views/settings-view.spec.ts | broker selection buttons render | expected |
| views/settings-view.spec.ts | broker selection highlights active broker | expected |
| views/settings-view.spec.ts | broker selection switches displayed content | expected |
| views/settings-view.spec.ts | navigation to settings page works from URL directly | expected |
| views/settings-view.spec.ts | no JavaScript errors on page load | expected |
| views/settings-view.spec.ts | page header renders | expected |
| views/settings-view.spec.ts | renders all section headers | expected |
| views/settings-view.spec.ts | saving submits settings and shows loading then success | expected |
| views/settings-view.spec.ts | shows loading state then resolves | expected |
| views/signal-view-check.spec.ts | signals view loads | expected |
| views/signals-selection.spec.ts | execute button appears when signals are selected | expected |
| views/signals-selection.spec.ts | filter buttons render correctly | expected |
| views/signals-selection.spec.ts | select all checkbox renders in filter area | expected |
| views/signals-selection.spec.ts | signals page renders without JS errors | expected |
| views/signals-view.spec.ts | clear all deletes signals only after confirmation | expected |
| views/signals-view.spec.ts | clear selected removes the chosen signal after the API confirms it | expected |
| views/signals-view.spec.ts | direction filter narrows displayed signals | expected |
| views/signals-view.spec.ts | execute button shows when signals selected | expected |
| views/signals-view.spec.ts | execution results toast appears after execute | expected |
| views/signals-view.spec.ts | generate all button renders | expected |
| views/signals-view.spec.ts | no signals matching filter shows when filter eliminates all | expected |
| views/signals-view.spec.ts | page header renders | expected |
| views/signals-view.spec.ts | page loads without JS errors | expected |
| views/signals-view.spec.ts | refresh button renders | expected |
| views/signals-view.spec.ts | renders direction filter buttons | expected |
| views/signals-view.spec.ts | renders signal cards when signals exist | expected |
| views/signals-view.spec.ts | renders status filter buttons | expected |
| views/signals-view.spec.ts | select all checkbox renders | expected |
| views/signals-view.spec.ts | selection count displays when signals selected | expected |
| views/signals-view.spec.ts | shows loading state on initial load | expected |
| views/signals-view.spec.ts | signal card renders confidence bar | expected |
| views/signals-view.spec.ts | signal card renders direction badge | expected |
| views/signals-view.spec.ts | signal card renders price data | expected |
| views/signals-view.spec.ts | signal card renders risk:reward | expected |
| views/signals-view.spec.ts | signal card renders sentiment badge when sentiment present | expected |
| views/signals-view.spec.ts | signal card renders sentiment reasoning when present | expected |
| views/signals-view.spec.ts | signal card renders strategy label when strategy present | expected |
| views/signals-view.spec.ts | signal card renders technical reasoning | expected |
| views/signals-view.spec.ts | status filter narrows displayed signals | expected |
| views/stage-sanity.spec.ts | backtest page loads | expected |
| views/stage-sanity.spec.ts | dashboard loads with title and shell | expected |
| views/stage-sanity.spec.ts | health endpoint returns UP | unexpected |
| views/stage-sanity.spec.ts | metrics endpoint returns Prometheus format | unexpected |
| views/stage-sanity.spec.ts | monitoring page loads | expected |
| views/stage-sanity.spec.ts | portfolio page loads | expected |
| views/stage-sanity.spec.ts | positions API returns a paginated response | unexpected |
| views/stage-sanity.spec.ts | positions page loads | unexpected |
| views/stage-sanity.spec.ts | positions performance API returns trade stats | unexpected |
| views/stage-sanity.spec.ts | positions stats API returns valid JSON | unexpected |
| views/stage-sanity.spec.ts | sentiment page loads | unexpected |
| views/stage-sanity.spec.ts | settings page loads | expected |
| views/stage-sanity.spec.ts | sidebar navigation works | unexpected |
| views/stage-sanity.spec.ts | signals API returns array | unexpected |
| views/stage-sanity.spec.ts | signals page loads | expected |
| views/stage-sanity.spec.ts | unknown route shows 404 | expected |
| views/stage-sanity.spec.ts | watchlist API returns data | unexpected |
| views/stage-sanity.spec.ts | watchlist page loads | expected |
