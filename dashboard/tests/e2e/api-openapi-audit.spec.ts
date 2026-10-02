import { expect, test, type Page } from '@playwright/test'

const excludedGets = new Set(['/api/news/{symbol}/latest', '/api/signals/combined/{symbol}'])

const jsonObject = (value: unknown): Record<string, unknown> =>
  typeof value === 'object' && value !== null ? (value as Record<string, unknown>) : {}

async function apiJson(page: Page, path: string) {
  return page.evaluate(async (requestPath: string) => {
    const response = await fetch(requestPath, { headers: { Accept: 'application/json' } })
    return { status: response.status, body: await response.text() }
  }, path)
}

function substitutePath(path: string, values: Record<string, string>): string {
  return path.replace(
    /\{([^}]+)\}/g,
    (_, name: string) => values[name] ?? '00000000-0000-0000-0000-000000000000'
  )
}

test.describe.configure({ mode: 'serial' })

test('all safe OpenAPI GET routes return no server or rate-limit errors', async ({ page }) => {
  await page.goto('/api-docs')
  const spec = JSON.parse((await page.locator('body').textContent()) || '{}') as {
    paths: Record<string, Record<string, { parameters?: Array<{ in?: string; name?: string }> }>>
  }

  const strategyResponse = await apiJson(page, '/api/strategy-configs')
  const runResponse = await apiJson(page, '/api/job/runs')
  const candidateResponse = await apiJson(page, '/api/candidate-scans')
  const strategyBody = jsonObject(JSON.parse(strategyResponse.body))
  const strategies = Array.isArray(strategyBody.data) ? strategyBody.data : []
  const runs = Array.isArray(JSON.parse(runResponse.body)) ? JSON.parse(runResponse.body) : []
  const candidates = Array.isArray(JSON.parse(candidateResponse.body))
    ? JSON.parse(candidateResponse.body)
    : []
  const firstStrategy = jsonObject(strategies[0])
  const firstRun = jsonObject(runs[0])
  const terminalCandidate = candidates
    .map(jsonObject)
    .find((candidate) => ['COMPLETED', 'CANCELLED', 'FAILED'].includes(String(candidate.status)))
  const jobRunId = String(firstRun.runId ?? '00000000-0000-0000-0000-000000000000')
  const candidateRunId = String(terminalCandidate?.runId ?? '00000000-0000-0000-0000-000000000000')
  const values: Record<string, string> = {
    variantId: String(firstStrategy.variantId ?? 'breakout-v1'),
    version: '1',
    runId: jobRunId,
    portfolioId: '1',
    symbol: 'INFY',
    type: 'BUY',
    status: 'OPEN',
    sector: 'IT',
    pullId: '00000000-0000-0000-0000-000000000000',
    filename: 'missing-report.json',
  }
  const today = new Date().toISOString().slice(0, 10)
  const from = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
  const queryValues: Record<string, string> = {
    from,
    to: today,
    fromDate: from,
    toDate: today,
    startDate: from,
    endDate: today,
    symbol: 'INFY',
    minConfidence: '0.7',
    windowDays: '30',
    page: '0',
    size: '20',
    limit: '100',
    offset: '0',
    format: 'CSV',
    type: 'BUY',
    status: 'OPEN',
    sector: 'IT',
    days: '30',
    verbose: 'false',
    variantId: values.variantId,
    mode: 'CHAMPION',
    signalType: 'BUY',
  }

  const results: Array<{ path: string; status: number }> = []
  for (const [path, operations] of Object.entries(spec.paths)) {
    const operation = operations.get
    if (!operation || excludedGets.has(path)) continue
    const requestValues = {
      ...values,
      runId: path.startsWith('/api/candidate-scans') ? candidateRunId : jobRunId,
    }
    const requestPath = substitutePath(path, requestValues)
    const params = (operation.parameters ?? [])
      .filter(
        (parameter) => parameter.in === 'query' && parameter.name && queryValues[parameter.name]
      )
      .map((parameter) => `${parameter.name}=${encodeURIComponent(queryValues[parameter.name!])}`)
    const result = await apiJson(
      page,
      `${requestPath}${params.length ? `?${params.join('&')}` : ''}`
    )
    results.push({ path, status: result.status })
    if (path === '/api/candidate-scans/{runId}/stream' && result.status === 200) {
      expect(result.body).toContain('RUN_SNAPSHOT')
    }
  }

  const unexpected = results.filter(({ path, status }) => {
    if (status >= 500 || status === 429) return true
    if (path === '/api/fyers/login') return ![200, 400].includes(status)
    if (path === '/api/fyers/callback') return status !== 200
    if (path === '/api/candidate-scans/{runId}/stream') return ![200, 404].includes(status)
    if (path === '/api/strategy-configs/{variantId}/promotion-eligibility') return status !== 400
    if (path === '/api/job/runs/{runId}/summary' || path === '/api/job/runs/{runId}/progress')
      return ![200, 404].includes(status)
    if (path === '/api/candidate-scans/{runId}' || path === '/api/candidate-scans/{runId}/results')
      return ![200, 404].includes(status)
    if (path === '/api/backtest/reports/{filename}') return status !== 400
    return status !== 200
  })
  expect(unexpected, JSON.stringify(unexpected)).toEqual([])
  expect(results).toHaveLength(85)
})

test('all OpenAPI write routes accept an OPTIONS preflight', async ({ request }) => {
  const specResponse = await request.get('/api-docs')
  expect(specResponse.status()).toBe(200)
  const spec = (await specResponse.json()) as {
    paths: Record<string, Record<string, unknown>>
  }
  const methods = ['post', 'put', 'patch', 'delete']
  const routes = Object.entries(spec.paths).flatMap(([path, operations]) =>
    methods.filter((method) => operations[method]).map((method) => ({ path, method }))
  )

  const failures: Array<{ method: string; path: string; status: number }> = []
  for (const route of routes) {
    const path = substitutePath(route.path, {
      variantId: 'missing',
      version: '1',
      runId: '00000000-0000-0000-0000-000000000000',
      portfolioId: '1',
      symbol: 'INFY',
      date: '2099-01-01',
    })
    const response = await request.fetch(path, { method: 'OPTIONS' })
    if (![200, 204].includes(response.status())) {
      failures.push({ ...route, status: response.status() })
    }
  }
  expect(failures, JSON.stringify(failures)).toEqual([])
  expect(routes).toHaveLength(65)
})
