import { expect, test, type APIRequestContext, type APIResponse } from '@playwright/test'

// Run with an API whose datasource points at a disposable database. This suite
// deliberately does not call broker auth, LLM inference, backfill, or scan
// endpoints: those cross an external boundary or launch long-running work.
const API = process.env.AUDIT_API_URL ?? 'http://127.0.0.1:8092'
const uniqueHoliday = '2099-12-31'

function requireAuditApi(): void {
  test.skip(!process.env.AUDIT_API_URL, 'Requires an explicitly configured disposable audit API')
}

async function json(response: APIResponse) {
  return response.json() as Promise<Record<string, unknown>>
}

async function requestGet(request: APIRequestContext, path: string) {
  return request.get(`${API}${path}`, { headers: { Accept: 'application/json' } })
}

async function requestJson(
  request: APIRequestContext,
  path: string,
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  body?: unknown
) {
  return request.fetch(`${API}${path}`, {
    method,
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
    data: body,
  })
}

test.describe.configure({ mode: 'serial' })

test('empty isolated signal workflows complete without provider or model calls', async ({
  request,
}) => {
  requireAuditApi()
  const watchlist = await requestGet(request, '/api/watchlist')
  expect(watchlist.status()).toBe(200)
  const watchlistBody = await json(watchlist)
  const activeSymbols = watchlistBody.data as Array<Record<string, unknown>>
  for (const entry of activeSymbols) {
    const deactivated = await requestJson(
      request,
      `/api/watchlist/${entry.symbol}/toggle?activate=false`,
      'PATCH'
    )
    expect(deactivated.status()).toBe(200)
  }

  const scan = await requestJson(request, '/api/signals/scan', 'POST')
  expect(scan.status()).toBe(200)
  expect((await json(scan)).signalsFound).toBe(0)

  const generated = await requestJson(request, '/api/signals/generate-all', 'POST')
  expect(generated.status()).toBe(200)
  expect(await json(generated)).toMatchObject({ signals: [], skipped: [] })

  const stream = await requestJson(request, '/api/signals/generate-all/stream', 'POST')
  expect(stream.status()).toBe(200)
  expect(stream.headers()['content-type']).toContain('text/event-stream')
  const streamBody = await stream.text()
  expect(streamBody).toContain('COMPLETE')
  expect(streamBody).toContain('"status":"DONE"')

  const noHistory = await requestJson(
    request,
    '/api/signals/price-action/AUDITEMPTY/generate',
    'POST'
  )
  expect(noHistory.status()).toBe(404)
})

test('safe disconnected integrations and background job controls return expected states', async ({
  request,
}) => {
  requireAuditApi()

  const missingAuthCode = await requestJson(request, '/api/fyers/auth', 'POST', {})
  expect(missingAuthCode.status()).toBe(400)

  const logout = await requestJson(request, '/api/fyers/logout', 'POST')
  expect(logout.status()).toBe(200)
  expect((await json(logout)).connected).toBe(false)

  const missingSignalRequest = await requestJson(request, '/api/signals/generate', 'POST', {})
  expect(missingSignalRequest.status()).toBe(400)

  const missingJob = await requestJson(
    request,
    '/api/job/runs/00000000-0000-0000-0000-000000000000/cancel',
    'POST'
  )
  expect(missingJob.status()).toBe(200)
  expect((await json(missingJob)).message).toBe('Run cancelled')

  const evaluation = await requestJson(request, '/api/sentiment/evaluate/trigger', 'POST')
  expect(evaluation.status()).toBe(200)
  const result = (await json(evaluation)).data as Record<string, unknown>
  expect(result.lastStatus).toBe('OK')
  expect(result.lastCount).toBe(0)
})

test('isolated CRUD and state transitions complete without unexpected errors', async ({
  request,
}) => {
  requireAuditApi()
  const health = await requestGet(request, '/api/health')
  expect(health.status()).toBe(200)
  expect((await json(health)).status).toBe('UP')

  const symbol = 'AUDITX'
  const variant = `audit-${Date.now()}`
  const added = await requestJson(
    request,
    `/api/watchlist?symbol=${symbol}&name=Audit%20Symbol&exchange=NSE`,
    'POST'
  )
  expect(added.status()).toBe(200)
  expect(((await json(added)).data as Record<string, unknown>).symbol).toBe(symbol)

  const duplicate = await requestJson(request, `/api/watchlist?symbol=${symbol}`, 'POST')
  expect(duplicate.status()).toBe(400)

  const deactivated = await requestJson(
    request,
    `/api/watchlist/${symbol}/toggle?activate=false`,
    'PATCH'
  )
  expect(deactivated.status()).toBe(200)
  const inactive = await requestGet(request, '/api/watchlist/all')
  const inactiveRows = (await json(inactive)).data as Array<Record<string, unknown>>
  expect(inactiveRows.some((entry) => entry.symbol === symbol && entry.isActive === false)).toBe(
    true
  )

  const reactivated = await requestJson(
    request,
    `/api/watchlist/${symbol}/toggle?activate=true`,
    'PATCH'
  )
  expect(reactivated.status()).toBe(200)
  const removed = await requestJson(request, `/api/watchlist/${symbol}`, 'DELETE')
  expect(removed.status()).toBe(200)

  const positionSymbol = 'AUDITP'
  const createdPosition = await requestJson(request, '/api/positions', 'POST', {
    symbol: positionSymbol,
    quantity: 1,
    direction: 'LONG',
    orderType: 'MARKET',
    price: 100,
    stopPrice: 90,
    target: 120,
    entryReason: 'isolated Playwright paper-trade audit',
  })
  expect(createdPosition.status()).toBe(200)
  expect((await json(createdPosition)).symbol).toBe(positionSymbol)
  const openPosition = await requestGet(request, `/api/positions/${positionSymbol}`)
  expect(openPosition.status()).toBe(200)
  const closedPosition = await requestJson(
    request,
    `/api/positions/${positionSymbol}/close`,
    'POST',
    { exitReason: 'isolated audit complete' }
  )
  expect(closedPosition.status()).toBe(200)
  expect((await json(closedPosition)).status).toBe('CLOSED')
  const persistedClosedPosition = await requestGet(request, `/api/positions/${positionSymbol}`)
  expect(persistedClosedPosition.status()).toBe(200)
  expect((await json(persistedClosedPosition)).status).toBe('CLOSED')

  const holiday = await requestJson(
    request,
    `/api/holidays?date=${uniqueHoliday}&occasion=Audit%20Holiday&type=FULL`,
    'POST'
  )
  expect(holiday.status()).toBe(200)
  const deletedHoliday = await requestJson(request, `/api/holidays/${uniqueHoliday}`, 'DELETE')
  expect(deletedHoliday.status()).toBe(200)

  const created = await requestJson(request, '/api/strategy-configs', 'POST', {
    variantId: variant,
    version: 1,
    strategyType: 'BREAKOUT',
    params: {},
    overlays: {},
    mode: 'SHADOW',
    paperCapital: 100000,
    notes: 'isolated Playwright audit',
  })
  expect(created.status()).toBe(201)
  expect(((await json(created)).data as Record<string, unknown>).variantId).toBe(variant)

  const current = await requestGet(request, `/api/strategy-configs/${variant}`)
  expect(current.status()).toBe(200)
  const updated = await requestJson(request, `/api/strategy-configs/${variant}`, 'PUT', {
    variantId: variant,
    version: 2,
    strategyType: 'BREAKOUT',
    params: { audit: true },
    overlays: {},
    mode: 'SHADOW',
    paperCapital: 100000,
    notes: 'isolated Playwright update audit',
  })
  expect(updated.status()).toBe(200)
  expect(((await json(updated)).data as Record<string, unknown>).version).toBe(2)
  const version = await requestGet(request, `/api/strategy-configs/${variant}/versions/2`)
  expect(version.status()).toBe(200)
  const mode = await requestJson(request, `/api/strategy-configs/${variant}/mode`, 'PUT', {
    mode: 'OFF',
    notes: 'isolated audit transition',
  })
  expect(mode.status()).toBe(200)
  const versions = await requestGet(request, `/api/strategy-configs/${variant}/versions`)
  expect(versions.status()).toBe(200)
  const deletedConfig = await requestJson(request, `/api/strategy-configs/${variant}`, 'DELETE')
  expect(deletedConfig.status()).toBe(200)

  const settings = [
    ['/api/settings/llm', { 'llm.backend': 'pi_ssh' }],
    ['/api/settings/gpuhub', {}],
    ['/api/settings/discord', { 'discord.webhook.url': '', 'discord.webhook.enabled': 'false' }],
    [
      '/api/settings/trading',
      {
        'trading.mode': 'paper',
        'trading.max_position_size': '10',
        'trading.stop_loss': '5',
        'trading.take_profit': '15',
        'trading.allocation_per_position': '100000',
        'trading.initial_capital': '500000',
      },
    ],
    ['/api/settings/scanning', { 'candidate-scan.max-concurrent': '3' }],
  ] as const
  for (const [path, body] of settings) {
    const response = await requestJson(request, path, 'PUT', body)
    expect(response.status(), path).toBe(200)
  }

  const killOn = await requestJson(request, '/api/admin/kill-switch', 'POST', {
    enabled: true,
    reason: 'isolated Playwright audit',
  })
  expect(killOn.status()).toBe(200)
  expect(((await json(killOn)).data as Record<string, unknown>).active).toBe(true)
  const killOff = await requestJson(request, '/api/admin/kill-switch/toggle?enable=false', 'POST')
  expect(killOff.status()).toBe(200)
  expect(((await json(killOff)).data as Record<string, unknown>).active).toBe(false)

  const broker = await requestJson(request, '/api/settings/broker', 'POST', { broker: 'yahoo' })
  expect(broker.status()).toBe(200)
  expect(((await json(broker)).data as Record<string, unknown>).selectedBroker).toBe('yahoo')

  const deletedSignals = await requestJson(request, '/api/signals', 'DELETE')
  expect(deletedSignals.status()).toBe(200)
  expect((await json(deletedSignals)).cleared).toBe(0)
  const deletedSymbolSignals = await requestJson(request, '/api/signals/AUDITX', 'DELETE')
  expect(deletedSymbolSignals.status()).toBe(200)
  expect((await json(deletedSymbolSignals)).cleared).toBe(0)
})

test('isolated validation and safe no-op branches return client responses', async ({ request }) => {
  requireAuditApi()
  const invalidPosition = await requestJson(request, '/api/positions', 'POST', {
    symbol: '',
    quantity: 0,
    direction: 'INVALID',
    orderType: 'INVALID',
  })
  expect(invalidPosition.status()).toBe(400)

  const invalidJob = await requestJson(request, '/api/job/runs/start', 'POST', {
    symbols: ['NOT_A_SYMBOL'],
    dryRun: true,
  })
  expect(invalidJob.status()).toBe(400)

  const invalidReconcile = await requestJson(
    request,
    '/api/ingestion/reconcile?from=2026-09-20&to=2026-09-27&symbol=INFY&apply=false',
    'POST'
  )
  expect(invalidReconcile.status()).toBe(400)

  const invalidRange = await requestJson(
    request,
    '/api/ingestion/range?from=2026-09-27&to=2026-09-20',
    'POST'
  )
  expect([400, 422]).toContain(invalidRange.status())

  const cancelPull = await requestJson(request, '/api/data/pull/cancel', 'POST')
  expect(cancelPull.status()).toBe(200)

  const nonexistentBacktest = await requestJson(
    request,
    '/api/backtest/run?symbol=NOT_A_SYMBOL',
    'POST'
  )
  expect([400, 404]).toContain(nonexistentBacktest.status())
})

test('remaining write routes reject incomplete or nonexistent requests without 5xx', async ({
  request,
}) => {
  requireAuditApi()
  const cases: Array<[string, 'POST' | 'PUT' | 'PATCH' | 'DELETE', string, unknown, number[]]> = [
    ['watchlist missing symbol', 'POST', '/api/watchlist', undefined, [400]],
    ['strategy create invalid', 'POST', '/api/strategy-configs', {}, [400]],
    ['strategy update invalid', 'PUT', '/api/strategy-configs/missing', {}, [400]],
    ['strategy mode invalid', 'PUT', '/api/strategy-configs/missing/mode', {}, [400]],
    ['strategy delete missing', 'DELETE', '/api/strategy-configs/missing', undefined, [400, 404]],
    ['broker missing', 'POST', '/api/settings/broker', {}, [400]],
    ['settings save no-op', 'POST', '/api/settings/save', {}, [200]],
    ['scanning invalid', 'PUT', '/api/settings/scanning', {}, [400]],
    ['candidate settings defaults', 'PUT', '/api/candidate-scans/settings', {}, [200]],
    ['position invalid', 'POST', '/api/positions', {}, [400]],
    ['close missing position', 'POST', '/api/positions/NO_SUCH/close', {}, [400, 404]],
    ['ingestion date missing', 'POST', '/api/ingestion/date', undefined, [400]],
    ['ingestion latest missing', 'POST', '/api/ingestion/latest', undefined, [400]],
    ['ingestion backfill missing', 'POST', '/api/ingestion/backfill', undefined, [400]],
    ['incremental missing', 'POST', '/api/data/pull/incremental', undefined, [400]],
    ['holiday missing', 'POST', '/api/holidays', undefined, [400]],
    [
      'candidate cancel missing',
      'POST',
      '/api/candidate-scans/00000000-0000-0000-0000-000000000000/cancel',
      undefined,
      [404],
    ],
    [
      'candidate pause missing',
      'POST',
      '/api/candidate-scans/00000000-0000-0000-0000-000000000000/pause',
      undefined,
      [404],
    ],
    [
      'candidate resume missing',
      'POST',
      '/api/candidate-scans/00000000-0000-0000-0000-000000000000/resume',
      undefined,
      [404],
    ],
    ['backtest missing symbol', 'POST', '/api/backtest/run', undefined, [400]],
    ['tuned backtest missing symbol', 'POST', '/api/backtest/run-tune', {}, [400]],
    [
      'run-all unknown strategy',
      'POST',
      '/api/backtest/run-all?strategyName=NO_SUCH_STRATEGY',
      undefined,
      [400],
    ],
    [
      'run-all-tune unknown strategy',
      'POST',
      '/api/backtest/run-all-tune?strategyName=NO_SUCH_STRATEGY',
      {},
      [400],
    ],
    ['portfolio invalid', 'POST', '/api/backtest/portfolio', {}, [400, 404]],
    ['analysis missing symbol', 'POST', '/api/analysis/analyze', undefined, [400]],
    ['full analysis missing symbol', 'POST', '/api/analysis/run-full', undefined, [400]],
    ['admin validation invalid', 'POST', '/api/admin/data/validate', {}, [400]],
    ['kill switch disable body', 'POST', '/api/admin/kill-switch', { enabled: false }, [200]],
    [
      'watchlist toggle missing',
      'PATCH',
      '/api/watchlist/NO_SUCH/toggle?activate=true',
      undefined,
      [404],
    ],
    ['watchlist delete missing', 'DELETE', '/api/watchlist/NO_SUCH', undefined, [200, 404]],
    ['signals delete symbol empty', 'DELETE', '/api/signals/NO_SUCH', undefined, [200, 404]],
    ['holidays delete missing', 'DELETE', '/api/holidays/2099-01-01', undefined, [400]],
  ]

  for (const [name, method, path, body, expectedStatuses] of cases) {
    const response = await requestJson(request, path, method, body)
    expect(response.status(), name).toBeGreaterThanOrEqual(200)
    expect(expectedStatuses, name).toContain(response.status())
  }
})
