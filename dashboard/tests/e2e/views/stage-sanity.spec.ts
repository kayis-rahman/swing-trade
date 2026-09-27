import { test, expect } from '@playwright/test'

const BASE = process.env.STAGE_DASHBOARD_URL ?? 'http://localhost:3000'
const API_BASE = process.env.STAGE_API_URL ?? 'http://localhost:8080'

test.describe('Stage Dashboard — Feature Tests', () => {
  test('dashboard loads with title and shell', async ({ page }) => {
    await page.goto(`${BASE}/`)
    await expect(page).toHaveTitle(/Swing\s*Trade/i)
    await expect(page.getByRole('heading', { name: 'Portfolio at a glance' })).toBeVisible()
  })

  test('positions page loads', async ({ page }) => {
    await page.goto(`${BASE}/positions`)
    await expect(page.getByRole('heading', { name: /Positions/i })).toBeVisible()
  })

  test('signals page loads', async ({ page }) => {
    await page.goto(`${BASE}/signals`)
    await expect(page.getByRole('heading', { name: /Signals/i })).toBeVisible()
  })

  test('portfolio page loads', async ({ page }) => {
    await page.goto(`${BASE}/portfolio`)
    await expect(page.getByRole('heading', { name: 'Portfolio', exact: true })).toBeVisible()
  })

  test('watchlist page loads', async ({ page }) => {
    await page.goto(`${BASE}/watchlist`)
    await expect(page.getByRole('heading', { name: /Watchlist/i })).toBeVisible()
  })

  test('sentiment page loads', async ({ page }) => {
    await page.goto(`${BASE}/sentiment`)
    await expect(page.getByRole('heading', { name: /Sentiment/i })).toBeVisible()
  })

  test('settings page loads', async ({ page }) => {
    await page.goto(`${BASE}/settings`)
    await expect(page.getByRole('heading', { name: /Settings/i })).toBeVisible()
  })

  test('backtest page loads', async ({ page }) => {
    await page.goto(`${BASE}/backtest`)
    await expect(page.getByRole('heading', { name: 'Backtest', exact: true })).toBeVisible()
  })

  test('monitoring page loads', async ({ page }) => {
    await page.goto(`${BASE}/monitoring`)
    await expect(page.getByRole('heading', { name: /LLM Accuracy Monitoring/i })).toBeVisible()
  })

  test('health endpoint returns UP', async ({ request }) => {
    const resp = await request.get(`${BASE}/api/health`)
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(body.status).toBe('UP')
    expect(body.components.api.status).toBe('UP')
  })

  test('metrics endpoint returns Prometheus format', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/actuator/prometheus`)
    expect(resp.status()).toBe(200)
    const text = await resp.text()
    expect(text).toContain('application_')
    expect(text).toContain('jvm_')
  })

  test('positions stats API returns valid JSON', async ({ request }) => {
    const resp = await request.get(`${BASE}/api/positions/stats`)
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(body).toHaveProperty('totalPositions')
    expect(body).toHaveProperty('openPositions')
    expect(body).toHaveProperty('totalPnL')
  })

  test('signals API returns array', async ({ request }) => {
    const resp = await request.get(`${BASE}/api/signals/latest`)
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(Array.isArray(body)).toBe(true)
  })

  test('positions API returns a paginated response', async ({ request }) => {
    const resp = await request.get(`${BASE}/api/positions`)
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(Array.isArray(body.content)).toBe(true)
    expect(body).toHaveProperty('totalElements')
  })

  test('positions performance API returns trade stats', async ({ request }) => {
    const resp = await request.get(`${BASE}/api/positions/performance`)
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(body).toHaveProperty('totalTrades')
    expect(body).toHaveProperty('winRate')
    expect(body).toHaveProperty('totalReturn')
  })

  test('watchlist API returns data', async ({ request }) => {
    const resp = await request.get(`${BASE}/api/watchlist`)
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(body.success).toBe(true)
    expect(Array.isArray(body.data)).toBe(true)
    expect(body.data.length).toBeGreaterThan(0)
  })

  test('sidebar navigation works', async ({ page }) => {
    await page.goto(`${BASE}/`)
    await page
      .getByRole('link', { name: /Dashboard/i })
      .first()
      .click()
    await expect(page.getByRole('heading', { name: 'Portfolio at a glance' })).toBeVisible()
    await page
      .getByRole('link', { name: /Positions/i })
      .first()
      .click()
    await expect(page.getByRole('heading', { name: /Positions/i })).toBeVisible()
    await page
      .getByRole('link', { name: /Signals/i })
      .first()
      .click()
    await expect(page.getByRole('heading', { name: /Signals/i })).toBeVisible()
    await page
      .getByRole('link', { name: /Portfolio/i })
      .first()
      .click()
    await expect(page.getByRole('heading', { name: 'Portfolio', exact: true })).toBeVisible()
  })

  test('unknown route shows 404', async ({ page }) => {
    await page.goto(`${BASE}/nonexistent`)
    await expect(page.locator('text=PAGE NOT FOUND')).toBeVisible()
  })
})
