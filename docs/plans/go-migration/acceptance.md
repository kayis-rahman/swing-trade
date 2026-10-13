# Acceptance criteria (to retire Java)

1. All dashboard-consumed endpoints match goldens (status, body, headers, SSE sequence); Playwright suite green unchanged.
2. Decimal paths identical over full watchlist history; backtest and portfolio backtest trade lists identical.
3. Paper ledger shadow equality across the agreed window including drills; released binary contains no live order path.
4. All schedulers owned by Go with no duplicate or missed runs over two weekly cycles.
5. Prometheus series and alerts unchanged (or deliberately migrated with Grafana JSON updated); JSON logs carry traceId, symbol, module.
6. Pi memory and startup meet the Phase 0 target (beat the native-image baseline).
7. Every Java test is mapped or dropped with a reason; ported code meets the 80% per-package gate; all scanners green.
8. Documentation, agent instructions and runbooks updated and verified; rollback drill executed.
