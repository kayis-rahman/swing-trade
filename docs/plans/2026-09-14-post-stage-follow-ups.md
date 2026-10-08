# Post-stage follow-up plan — 2026-09-14

## Purpose

Close the three follow-ups identified after the stage deployment without
reopening completed deployment work:

1. reconcile the stale dashboard equity-curve status;
2. split the oversized core `Position` model safely across module boundaries;
3. make the SELL close-out integration test runnable in an environment with a
   Docker-capable environment.

The stage deployment recorded in
`docs/plans/2026-09-02-stage-deploy-progress.md` remains complete. None of the
work below should mutate the stage database or trigger a stage deployment until
its own verification is green.

## Source-verified starting point

- `DashboardView.vue` already fetches `getEquityCurve("1M")` and renders a
  compact equity sparkline. The corresponding item in `docs/status.md` is
  stale, but there is no focused assertion for the populated and empty chart
  states yet.
- `com.swingtrade.domain.Position` is a 23-component record containing identity,
  entry, risk, valuation, lifecycle, broker, and order data. It is constructed
  directly in core, data, broker, API tests, and broker tests, so changing its
  canonical constructor is a cross-module migration.
- `backend/api/src/main/java/com/swingtrade/api/dto/Position.java` has no
  production consumers. `PositionResponse` is the active API contract.
- `SignalPipelineSellExitIntegrationTest` already covers the real Spring/DB
  close-out path, including exit reason, exit price, trade audit, and persisted
  SELL signal. Its blocker is the repository's remote `pi-node` Docker context,
  not missing test behavior.

## Non-goals

- Do not change REST response shapes or dashboard TypeScript position types.
- Do not change the `positions` table or add a Flyway migration solely for the
  domain refactor.
- Do not change entry, exit, P&L, risk, or broker behavior while reshaping the
  model.
- Do not run integration fixtures against development or stage PostgreSQL.
- Do not make the optional integration workflow a deployment gate unless that
  policy is chosen separately.

## Phase 1 — Close the stale equity-curve item

**Status: complete.** The dashboard now has an isolated equity error/retry
state, and focused tests cover populated, empty, and failed requests.

### Changes

1. Add focused `DashboardView` tests for:
   - two or more equity points render the `aria-label="One month equity curve"`
     SVG and a non-empty path;
   - fewer than two points render the existing empty-state copy;
   - an equity request failure does not hide the other independently loaded
     dashboard sections.
2. If error visibility for the equity request is currently inadequate, give the
   sparkline its own lightweight inline error state and retry using the existing
   dashboard error patterns. Do not fail the entire dashboard refresh.
3. Run the focused dashboard test, then the dashboard typecheck and unit suite.
4. Only after those checks pass, update `docs/status.md` to state that the
   one-month sparkline exists on `/` and the full range-selectable chart remains
   on `/portfolio`.

### Acceptance criteria

- The dashboard displays a real one-month equity path when at least two points
  are returned.
- Empty and failed responses are distinguishable and do not break unrelated
  cards.
- The status document matches the verified implementation.

### Verification

```bash
cd dashboard
yarn test:run DashboardView.errorStates.test.ts
yarn typecheck
yarn test:run
```

## Phase 2 — Decompose the core Position model

**Status: implementation complete for the planned first migration slice.** The
four immutable value records are in place, production adapters use grouped
construction or named transition methods, and the unused API DTO is removed.
The compatibility constructor remains temporarily for existing test fixtures;
removing it is a follow-up cleanup after those fixtures are migrated.

This phase should be delivered independently from Phase 3. Keep `Position` as
the aggregate root and group related data into immutable value records rather
than spreading lifecycle rules across services.

### Target model

- `PositionEntry` — symbol, entry price/date/time, quantity, average price,
  direction, exchange, and entry reason.
- `PositionRisk` — stop loss, target, and margin utilized.
- `PositionValuation` — current price, unrealized P&L, and realized P&L.
- `PositionExit` — exit time and exit reason; absent while open.
- `Position` retains database ID, broker type, internal/broker position IDs,
  status, associated orders, and the four grouped values above.

Keep money as `BigDecimal`. Constructors must preserve today's defaults:
`PAPER`, NSE, LONG, entry price as average/current price where applicable, and
zero P&L/margin values.

### Migration sequence

1. Characterize current behavior in `PositionTest` before changing the model:
   defaulting, long/short P&L, zero cost basis, open/closed state, and
   `createWithRisk` calculations.
2. Add the value records in `core` with validation limited to invariants already
   enforced today. Avoid introducing new rejection behavior during a structural
   refactor.
3. Replace the long `Position.of(...)` parameter list with named factories and
   transition methods, for example:
   - `Position.openPaper(...)` or a request object for new positions;
   - `withValuation(...)` for market-price/P&L refreshes;
   - `close(...)` for lifecycle completion;
   - a persistence rehydration factory used only by `data`.
4. Migrate production callers in dependency order:
   - `core` tests and factories;
   - `data/PositionEntity.fromDomain()` and `toDomain()`;
   - broker managers, engines, state services, and risk checks;
   - API services and response mapping.
5. Migrate test fixtures to small named builders/factories so tests no longer
   repeat 23 positional arguments. Do not expose a mutable builder in production
   domain code solely for test convenience.
6. Delete the unused `api/dto/Position.java` after a repository-wide reference
   check confirms that `PositionResponse` is the only public position DTO.
7. Verify serialized API responses against controller tests to prove the REST
   contract did not change.

### Acceptance criteria

- No production call site invokes a 23-argument Position constructor or factory.
- Position state transitions remain immutable and preserve all current values.
- `PositionEntity` maps every existing column both ways; no schema migration is
  required.
- Broker and API behavior tests pass unchanged except for fixture construction.
- Module dependency direction remains unchanged and the ArchUnit boundary test
  passes.
- The dead API DTO is removed with no remaining imports or fully qualified
  references.

### Verification

Run narrow suites after each caller group, then the combined backend checks:

```bash
cd backend
./gradlew :core:test --no-daemon
./gradlew :data:test --no-daemon
./gradlew :broker:test --no-daemon
./gradlew :api:test --no-daemon
./gradlew :data:test :api:test --no-daemon
./gradlew build --no-daemon
```

## Phase 3 — Provide Docker-capable SELL integration verification

**Status: complete.** `SignalPipelineSellExitIntegrationTest` now runs and
passes green. Getting it running and green surfaced and fixed four real
pre-existing bugs (none related to Docker/Testcontainers infrastructure once
that was wired up):

1. `ErrorHandlingTestConfig` (a `@WebMvcTest`-only mock-bean config) was
   annotated `@SpringBootApplication`, so it was picked up by the full
   `@SpringBootTest` component scan and silently replaced the real
   `PositionService` bean with a mock (`allow-bean-definition-overriding=true`
   masked this). Fixed by changing it to `@TestConfiguration` and adding an
   explicit `@ComponentScan` exclude filter for `TestConfiguration` classes in
   `SwingTradeApiApplication`.
2. `PaperTradingPortfolioEntity.version` had a `= 0` default that broke Spring
   Data's new-vs-existing detection for `@Version` entities, routing the first
   write through `merge()` instead of `persist()`.
3. Same entity's `@GeneratedValue` on `id` conflicted with application code
   that always manually assigns `id = 1L` for this singleton row.
4. The SELL close-out path (`PositionManager.closePosition` →
   `PositionService.closePosition`) persisted the stale entry price instead of
   the real market close that triggered the exit — `Position.close()` carries
   forward whatever `currentPrice()` it already has rather than the exit
   price, and a JPA-context caching issue meant the re-fetch after close
   returned a stale, pre-close object. Additionally, `PositionEntity` and
   `TradeEntity` decimal columns were missing `scale = 4`, so Hibernate
   rounded persisted prices to whole numbers regardless of the above fixes.

Also removed a stale, pre-consolidation duplicate Flyway migration file
(`api/src/test/resources/db/migration/V1__swing_trade_schema.sql`, dead
scaffolding never exercised by any test) that conflicted with the
authoritative `data` module's `V1` migration once `data.jar` was on the
`integrationTest` classpath — this was blocking Spring context startup before
any of the above bugs could even be reached.

No REST response shapes, the `positions` table, or Flyway migrations changed.
No dev/stage PostgreSQL was touched — everything ran through Testcontainers.

### Decision

Keep the existing Testcontainers test. This avoids destructive use of the
shared development/stage databases and avoids relying on an SSH Docker context
that may not be Testcontainers-compatible. No GitHub Actions workflow is required.

### Changes

1. Run only:

   ```bash
   cd backend
   ./gradlew :api:integrationTest \
     --tests '*SignalPipelineSellExitIntegrationTest' --no-daemon
   ```

2. Ensure the shell environment does not inherit repository `DOCKER_HOST` or
   remote Docker context settings.
3. Preserve the Gradle XML and HTML reports from the local run when sharing a
   result.
4. Document the manual invocation and expected assertions in the SELL pipeline
   plan. Keep the existing unit/dev-stack evidence recorded as valid historical
   verification.
5. After one green run, record the commit and date in `docs/status.md`. Do not
   claim integration verification before the report is available.

If hosted CI is intentionally out of scope, run the same command in an
appropriate Docker-capable environment. Do not replace the Testcontainers test
with a script pointed at `pi-node` PostgreSQL.

### Acceptance criteria

- The existing test starts an isolated PostgreSQL 16 container.
- It proves a generated SELL closes the held position and persists the correct
  exit reason, latest exit price, closed trade record, and SELL signal.
- The test report is retained for a failed or successful run.
- No development or stage database rows are created, updated, or deleted.

## Delivery order and commits

Deliver the phases as separate reviewable changes:

1. dashboard equity verification and status correction;
2. Position value records and caller migration;
3. isolated SELL integration runner and verification record.

Phase 1 is small and closes stale documentation. Phase 2 carries the highest
regression risk and should not be mixed with infrastructure changes. Phase 3 is
independent and may be deferred without blocking paper trading or the already
completed stage deployment.

## Completion checklist

- [x] Dashboard equity states are covered and `docs/status.md` is corrected.
- [x] Position is decomposed without API, persistence, or trading-behavior drift.
- [x] Unused API Position DTO is removed.
- [x] SELL integration test has a green report.
- [x] Full relevant backend/dashboard verification is recorded before any stage
      promotion.
