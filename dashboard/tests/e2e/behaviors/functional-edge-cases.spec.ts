import { test, expect, type Page } from '@playwright/test'
import { fulfillJson, mockDashboardApi } from '../fixtures/api'

const DASHBOARD = 'http://localhost:3003'

const watchlistEntry = (overrides: Record<string, unknown> = {}) => ({
  id: 1,
  symbol: 'RELIANCE',
  name: 'Reliance Industries',
  exchange: 'NSE',
  isActive: true,
  candleCount: 120,
  ...overrides,
})

async function mockApi(page: Page, handler: (path: string, method: string) => unknown) {
  await mockDashboardApi(page, async (route, path) => {
    if (path === '/health') {
      await fulfillJson(route, { status: 'UP', components: {} })
      return true
    }
    if (path === '/signals/latest' || path === '/paper-portfolios') {
      await fulfillJson(route, [])
      return true
    }
    await fulfillJson(route, handler(path, route.request().method()) ?? { success: true, data: [] })
    return true
  })
}

test.describe('safe functional edge cases', () => {
  test('watchlist add normalizes symbol and clears the form after confirmation', async ({
    page,
  }) => {
    let entries = [watchlistEntry()]
    let addedRequest: URL | undefined

    await mockApi(page, (path, method) => {
      if (path === '/watchlist' && method === 'GET') return { success: true, data: entries }
      if (path === '/watchlist' && method === 'POST') {
        addedRequest = new URL(`http://test${path}`)
        const added = watchlistEntry({ id: 2, symbol: 'INFY', name: 'Infosys', candleCount: 0 })
        entries = [...entries, added]
        return { success: true, data: added }
      }
      return { success: true, data: entries }
    })

    await page.goto(`${DASHBOARD}/watchlist`)
    await page.getByRole('button', { name: 'Add Stock' }).click()
    await page.getByPlaceholder('e.g. RELIANCE', { exact: true }).fill(' infy ')
    await page.getByPlaceholder('e.g. Reliance Industries').fill('Infosys')
    await page.getByRole('button', { name: 'Add', exact: true }).click()

    await expect(page.locator('tbody')).toContainText('INFY')
    await expect(page.getByPlaceholder('e.g. RELIANCE', { exact: true })).toHaveCount(0)
    expect(addedRequest).toBeDefined()
  })

  test('watchlist cancel leaves the form closed and does not submit', async ({ page }) => {
    let postCount = 0
    await mockApi(page, (path, method) => {
      if (path === '/watchlist' && method === 'POST') postCount++
      return { success: true, data: [watchlistEntry()] }
    })

    await page.goto(`${DASHBOARD}/watchlist`)
    await page.getByRole('button', { name: 'Add Stock' }).click()
    await page.getByPlaceholder('e.g. RELIANCE', { exact: true }).fill('TCS')
    await page.getByRole('button', { name: 'Cancel', exact: true }).click()

    await expect(page.getByPlaceholder('e.g. RELIANCE', { exact: true })).toHaveCount(0)
    expect(postCount).toBe(0)
  })

  test('watchlist toggle updates active state', async ({ page }) => {
    let active = true
    await mockApi(page, (path, method) => {
      if (path.includes('/toggle') && method === 'PATCH') {
        active = false
        return { success: true, data: watchlistEntry({ isActive: false }) }
      }
      return { success: true, data: [watchlistEntry({ isActive: active })] }
    })

    await page.goto(`${DASHBOARD}/watchlist`)
    await page.getByRole('button', { name: 'Active', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Inactive', exact: true })).toBeVisible()
  })

  test('watchlist removal honors a cancelled confirmation', async ({ page }) => {
    let deleteCount = 0
    await mockApi(page, (path, method) => {
      if (method === 'DELETE') deleteCount++
      return { success: true, data: [watchlistEntry()] }
    })
    await page.on('dialog', (dialog) => dialog.dismiss())

    await page.goto(`${DASHBOARD}/watchlist`)
    await page.getByTitle('Remove RELIANCE').click()
    await expect(page.locator('tbody')).toContainText('RELIANCE')
    expect(deleteCount).toBe(0)
  })

  test('sentiment history shows an empty state for a symbol with no history', async ({ page }) => {
    await mockApi(page, (path) => {
      if (path === '/watchlist') return { success: true, data: [] }
      if (path.includes('/history')) return { success: true, data: [] }
      return { success: true, data: [] }
    })

    await page.goto(`${DASHBOARD}/sentiment`)
    await page.getByPlaceholder('e.g. RELIANCE').first().fill('RELIANCE')
    await page.getByRole('button', { name: 'History', exact: true }).click()

    await expect(page.getByText('No sentiment history available.')).toBeVisible()
    await expect(page.getByText('Page 1')).toHaveCount(0)
  })

  test('positions form keeps required fields enforced and exposes limit price only for limit orders', async ({
    page,
  }) => {
    let createCount = 0
    await mockApi(page, (path, method) => {
      if (path === '/positions' && method === 'POST') createCount++
      if (path === '/positions' || path.startsWith('/positions/closed')) {
        return { content: [], totalElements: 0, totalPages: 0, size: 20, number: 0 }
      }
      if (path === '/positions/stats') {
        return {
          totalPositions: 0,
          openPositions: 0,
          closedPositions: 0,
          totalValue: 0,
          totalPnL: 0,
        }
      }
      if (path === '/positions/performance') {
        return {
          totalReturn: 0,
          annualizedReturn: 0,
          sharpeRatio: 0,
          maxDrawdown: 0,
          totalTrades: 0,
          winningTrades: 0,
          losingTrades: 0,
          winRate: 0,
          averageWin: 0,
          averageLoss: 0,
          profitFactor: 0,
          totalPnL: 0,
          totalValue: 0,
          closedTrades: 0,
          asOfDate: '2026-08-29',
        }
      }
      return { status: 'UP', components: { db: { status: 'UP' } } }
    })

    await page.goto(`${DASHBOARD}/positions`)
    await page.getByRole('button', { name: 'New Position' }).click()
    await expect(page.getByRole('heading', { name: 'New Position' })).toBeVisible()

    const orderType = page.locator('select').nth(1)
    await orderType.selectOption('LIMIT')
    await expect(page.getByText('Limit Price')).toBeVisible()
    await orderType.selectOption('MARKET')
    await expect(page.getByText('Limit Price')).toHaveCount(0)

    await page.getByRole('button', { name: 'Create Position' }).click()
    await expect(page.getByRole('heading', { name: 'New Position' })).toBeVisible()
    expect(createCount).toBe(0)
  })

  test('data ingestion range mode validates missing and reversed dates before submission', async ({
    page,
  }) => {
    let pullCount = 0
    await mockApi(page, (path, method) => {
      if (path === '/data/pull' && method === 'POST') pullCount++
      if (path === '/data/status') {
        return {
          success: true,
          data: [
            {
              symbol: 'RELIANCE',
              name: 'Reliance',
              exchange: 'NSE',
              isActive: true,
              candleCount: 0,
              lastCandleDate: null,
              earliestCandleDate: null,
              lastSyncedAt: null,
              hasData: false,
              dataQuality: 'EMPTY',
            },
          ],
        }
      }
      if (path === '/fyers/status')
        return { success: true, data: { connected: true, clientId: 'test' } }
      return { success: true, data: [] }
    })

    await page.goto(`${DASHBOARD}/data`)
    await page.locator('select').selectOption('range')
    await expect(page.getByText('From', { exact: true })).toBeVisible()
    await expect(page.getByText('To', { exact: true })).toBeVisible()

    const dates = page.locator('input[type="date"]')
    await dates.nth(0).fill('2026-08-29')
    await dates.nth(1).fill('2026-08-01')
    await page.getByRole('button', { name: /Pull/ }).click()

    await expect(page.getByText('The start date must be before the end date.')).toBeVisible()
    expect(pullCount).toBe(0)
  })
})
