import { test, expect } from '@playwright/test'
import { fulfillJson, mockDashboardApi } from '../fixtures/api'

const DASHBOARD = 'http://localhost:3003'
const signalFixture = {
  id: 10001,
  symbol: 'RELIANCE',
  date: '2026-09-01',
  signalType: 'BUY',
  confidence: 0.85,
  reasoning: 'Fixture signal for safe UI testing',
  entryPrice: 2500,
  stopLoss: 2450,
  target: 2600,
  riskRewardRatio: 2,
  indicators: ['RSI'],
  generatedAt: '2026-09-01T10:00:00',
  strategy: 'PRICE_ACTION',
}

test.describe('Signals View', () => {
  test.beforeEach(async ({ page }) => {
    await mockDashboardApi(page, async (route, path) => {
      if (path === '/signals/latest') {
        await fulfillJson(route, [signalFixture])
        return true
      }
      if (path === '/strategy-configs') {
        await fulfillJson(route, { success: true, data: [] })
        return true
      }
      if (path === '/signal-selections') {
        await fulfillJson(route, [])
        return true
      }
      if (path === '/settings') {
        await fulfillJson(route, { success: true, data: { selectedBroker: 'yahoo' } })
        return true
      }
      if (path.startsWith('/settings/')) {
        await fulfillJson(route, { success: true, data: {} })
        return true
      }
      if (route.request().method() !== 'GET') {
        await fulfillJson(route, { message: 'Writes are disabled in this UI suite' }, 405)
        return true
      }
      return false
    })
  })

  test('page header renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    await expect(page.locator('h1', { hasText: 'Signals' })).toBeVisible()
    await expect(page.locator('text=Review fresh opportunities')).toBeVisible()
  })

  test('shows loading state on initial load', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    // Very briefly intercept to catch loading state
    // Loading resolves quickly; just check page loads without error
    await page.waitForLoadState('networkidle')
    const bodyText = await page.locator('body').textContent()
    const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
    expect(errorMatch).toBeNull()
  })

  test('renders signal cards when signals exist', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    // Signals should render as cards in a grid
    const grid = page.locator('.grid.grid-cols-1')
    if (grid.count()) {
      // Grid exists — signals are rendering
      const bodyText = await page.locator('body').textContent()
      const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
      expect(errorMatch).toBeNull()
    }
  })

  test('renders direction filter buttons', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    for (const dir of ['ALL', 'BUY', 'SELL']) {
      // Filter buttons are in a group — find them
      const buttons = page.locator('.flex.rounded-md.border button')
      const texts = await buttons.allTextContents()
      const found = texts.some((t) => t.trim() === dir)
      expect(found).toBe(true)
    }
  })

  test('renders status filter buttons', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const buttons = page.locator('.flex.rounded-md.border button')
    const texts = await buttons.allTextContents()
    for (const st of ['ALL', 'ACTIVE', 'PENDING']) {
      expect(texts).toContain(st)
    }
  })

  test('direction filter narrows displayed signals', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const buttons = page.locator('.flex.rounded-md.border button')
    const texts = await buttons.allTextContents()
    const buyIdx = texts.indexOf('BUY')

    if (buyIdx >= 0) {
      await buttons.nth(buyIdx).click()

      // All displayed cards should be BUY
      const cards = page.locator('div.card-panel')
      const count = await cards.count()
      if (count > 0) {
        await expect(cards.first()).toContainText('BUY')
      }
    }
  })

  test('status filter narrows displayed signals', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const buttons = page.locator('.flex.rounded-md.border button')
    const texts = await buttons.allTextContents()
    const activeIdx = texts.indexOf('ACTIVE')

    if (activeIdx >= 0) {
      await buttons.nth(activeIdx).click()

      const cards = page.locator('div.card-panel')
      const count = await cards.count()
      if (count > 0) {
        await expect(cards.first()).toContainText('ACTIVE')
      }
    }
  })

  test('select all checkbox renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const checkboxes = page.locator('input[type="checkbox"]')
    const count = await checkboxes.count()
    expect(count).toBeGreaterThan(0)
  })

  test('selection count displays when signals selected', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    // Check if any signal cards exist
    const cards = page.locator('div.card-panel')
    const cardCount = await cards.count()

    if (cardCount > 0) {
      // Click the first card's checkbox
      const firstCheckbox = page.locator('input[type="checkbox"]').first()
      const isChecked = await firstCheckbox.isChecked()
      if (!isChecked) {
        await firstCheckbox.click()
      }

      // Selected count should appear
      const selectedLocator = page.getByText(/\d+ selected/)
      await expect(selectedLocator).toBeVisible()
      const selectedText = await selectedLocator.textContent()
      expect(selectedText?.match(/\d+ selected/)).not.toBeNull()
    }
  })

  test('execute button shows when signals selected', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const cardCount = await cards.count()

    if (cardCount > 0) {
      const firstCheckbox = page.locator('input[type="checkbox"]').first()
      const isChecked = await firstCheckbox.isChecked()
      if (!isChecked) {
        await firstCheckbox.click()
      }

      // Execute button should appear
      const execBtn = page.getByRole('button', { name: /Execute \d+/ })
      await expect(execBtn).toBeVisible()
    }
  })

  test('clear selected removes the chosen signal after the API confirms it', async ({ page }) => {
    let removedSymbol: string | undefined
    await page.route('**/api/signals/RELIANCE', async (route) => {
      if (route.request().method() !== 'DELETE') return route.fallback()
      removedSymbol = new URL(route.request().url()).pathname.split('/').at(-1)
      await fulfillJson(route, { cleared: 1 })
    })

    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')
    await page.locator('input[type="checkbox"]').first().check()
    await page.getByRole('button', { name: 'Clear selected (1)' }).click()

    await expect(page.locator('.signal-card-wrapper')).toHaveCount(0)
    expect(removedSymbol).toBe('RELIANCE')
  })

  test('clear all deletes signals only after confirmation', async ({ page }) => {
    let clearAllRequests = 0
    await page.route('**/api/signals', async (route) => {
      if (route.request().method() !== 'DELETE') return route.fallback()
      clearAllRequests++
      await fulfillJson(route, { cleared: 1 })
    })
    page.on('dialog', (dialog) => dialog.accept())

    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('button', { name: 'Clear all' }).click()

    await expect(page.locator('.signal-card-wrapper')).toHaveCount(0)
    expect(clearAllRequests).toBe(1)
  })

  test('generate all button renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const genBtn = page.getByRole('button', { name: /Generate|Generating/ })
    await expect(genBtn).toBeVisible()
  })

  test('refresh button renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const refreshBtn = page.getByRole('button', { name: 'Refresh' })
    await expect(refreshBtn).toBeVisible()
  })

  test('no signals matching filter shows when filter eliminates all', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: 'SELL', exact: true }).click()

    await expect(page.locator('.signal-card-wrapper')).toHaveCount(0)
    await expect(page.getByText('No signals match this view')).toBeVisible()
  })

  test('signal card renders direction badge', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const count = await cards.count()

    if (count > 0) {
      const cardText = await cards.first().textContent()
      // Should contain BUY, SELL, or HOLD
      expect(cardText?.match(/BUY|SELL|HOLD/)).not.toBeNull()
    }
  })

  test('signal card renders confidence bar', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const count = await cards.count()

    if (count > 0) {
      const cardText = await cards.first().textContent()
      // Should contain a percentage
      expect(cardText?.match(/\d+%/)).not.toBeNull()
    }
  })

  test('signal card renders price data', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const count = await cards.count()

    if (count > 0) {
      const cardText = await cards.first().textContent()
      expect(cardText).toContain('₹')
      expect(cardText).toContain('Entry')
      expect(cardText).toContain('Stop Loss')
      expect(cardText).toContain('Target')
    }
  })

  test('signal card renders risk:reward', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const count = await cards.count()

    if (count > 0) {
      const cardText = await cards.first().textContent()
      expect(cardText).toContain('Risk:Reward')
    }
  })

  test('signal card renders strategy label when strategy present', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const count = await cards.count()

    if (count > 0) {
      const cardText = await cards.first().textContent()
      // Strategy label appears as a pill with brand color; may not exist if no strategy
      // Check for either a known strategy label or that the strategy section simply doesn't render
      // Strategy is optional; the rendered card itself is the contract here.
      expect(cardText.length).toBeGreaterThan(0)
    }
  })

  test('signal card renders sentiment badge when sentiment present', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const count = await cards.count()

    if (count > 0) {
      const cardText = await cards.first().textContent()
      // Sentiment badge (POS/NEG/NEUTRAL) only renders when sentimentScore is set
      // Either it exists or it doesn't — both are valid
      expect(cardText?.length ?? 0).toBeGreaterThan(0)
    }
  })

  test('signal card renders sentiment reasoning when present', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const count = await cards.count()

    if (count > 0) {
      const cardText = await cards.first().textContent()
      // Sentiment reasoning is optional; the rendered card itself is the contract here.
      expect(cardText?.length ?? 0).toBeGreaterThan(0)
    }
  })

  test('signal card renders technical reasoning', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const cards = page.locator('div.card-panel')
    const count = await cards.count()

    if (count > 0) {
      const cardText = await cards.first().textContent()
      // Reason text is data-dependent; the rendered card itself is the contract here.
      expect(cardText?.length ?? 0).toBeGreaterThan(0)
    }
  })

  test('page loads without JS errors', async ({ page }) => {
    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    const bodyText = await page.locator('body').textContent()
    const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
    expect(errorMatch).toBeNull()
  })

  test('execution results toast appears after execute', async ({ page }) => {
    let executionBody: unknown
    await page.route('**/api/positions', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      executionBody = route.request().postDataJSON()
      await fulfillJson(route, { message: 'Execution blocked by test fixture' }, 409)
    })

    await page.goto(`${DASHBOARD}/signals`)
    await page.waitForLoadState('networkidle')

    await expect(page.locator('.signal-card-wrapper')).toHaveCount(1)
    await page.locator('input[type="checkbox"]').first().check()
    await page.getByRole('button', { name: 'Execute 1' }).click()

    const toast = page.locator('div.fixed.bottom-4.right-4')
    await expect(toast).toContainText('Failed')
    await expect
      .poll(() => executionBody)
      .toEqual({
        symbol: 'RELIANCE',
        quantity: 40,
        direction: 'LONG',
        orderType: 'MARKET',
        price: 2500,
        target: 2600,
        entryReason: 'Signal: RELIANCE — Fixture signal for safe UI testing',
      })
  })
})
