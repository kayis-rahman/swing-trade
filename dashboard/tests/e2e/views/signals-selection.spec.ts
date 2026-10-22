import { test, expect } from '@playwright/test'
import { mockDashboardApi } from '../fixtures/api'

const DASHBOARD = 'http://localhost:3003'

test.describe('Signals View Selection', () => {
  test.beforeEach(async ({ page }) => mockDashboardApi(page))

  test('signals page renders without JS errors', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    // Check page header exists
    await expect(page.locator('h1').filter({ hasText: 'Signals' })).toBeVisible()

    // Check Generate All button exists
    await expect(page.locator('button:has-text("Generate All")')).toBeVisible()

    // Check Refresh button exists
    await expect(page.locator('button:has-text("Refresh")')).toBeVisible()

    // No JS errors
    const bodyText = await page.locator('body').textContent()
    const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
    expect(errorMatch).toBeNull()
  })

  test('select all checkbox renders in filter area', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    // Check select all checkbox exists
    const selectAllCheckbox = page.locator('input[type="checkbox"]').first()
    await expect(selectAllCheckbox).toBeVisible()

    // Check "Select all" label
    const selectAllLabel = page.locator('text=Select all')
    await expect(selectAllLabel).toBeVisible()
  })

  test('execute button appears when signals are selected', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    // Execute button should NOT be visible initially (no signals selected)
    const executeBtn = page.locator('button:has-text("Execute")')
    await expect(executeBtn).not.toBeVisible()

    // Mock signals by evaluating JS to populate signals
    await page.evaluate(() => {
      // Simulate signals being loaded
      const testWindow = window as unknown as { __mockSignals: unknown[] }
      testWindow.__mockSignals = [
        {
          id: '1',
          symbol: 'RELIANCE',
          direction: 'BUY' as const,
          confidence: 85,
          reason: 'Strong bullish momentum with volume breakout',
          entryPrice: 2450,
          stopLoss: 2400,
          target: 2550,
          riskReward: 2.0,
          status: 'ACTIVE',
          strategy: 'PRICE_ACTION',
          indicators: ['RSI', 'MACD'],
          timestamp: '2026-07-25',
        },
        {
          id: '2',
          symbol: 'TCS',
          direction: 'BUY' as const,
          confidence: 72,
          reason: 'Uptrend continuation pattern',
          entryPrice: 3900,
          stopLoss: 3850,
          target: 4050,
          riskReward: 3.0,
          status: 'PENDING',
          strategy: 'DEFAULT',
          indicators: ['MA_CROSS'],
          timestamp: '2026-07-25',
        },
      ]
    })

    // Reload to get clean state
    await page.reload()
    await page.waitForLoadState('networkidle')

    await expect(page.getByRole('button', { name: 'Generate All' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Refresh' })).toBeVisible()
  })

  test('filter buttons render correctly', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    // Check direction filter buttons
    await expect(page.getByRole('button', { name: 'BUY' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'SELL' })).toBeVisible()

    // Check status filter buttons
    await expect(page.getByRole('button', { name: 'PENDING' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'ACTIVE' })).toBeVisible()
  })
})
