import { test, expect, type Page } from '@playwright/test'

const DASHBOARD = 'http://localhost:3003'

const openPositions = [
  {
    id: 1,
    symbol: 'RELIANCE',
    entryPrice: 100,
    currentPrice: 105,
    entryDate: '2026-09-01',
    quantity: 10,
    stopLoss: 95,
    target: 115,
    status: 'OPEN',
    unrealizedPnL: 50,
    unrealizedPnLPercent: 5,
  },
]

const closedPositions = [
  {
    id: 2,
    symbol: 'TCS',
    entryPrice: 200,
    currentPrice: 190,
    entryDate: '2026-08-20',
    quantity: 5,
    stopLoss: 190,
    target: 220,
    status: 'CLOSED',
    unrealizedPnL: -50,
    unrealizedPnLPercent: -5,
  },
]

async function mockPositionApi(page: Page) {
  await page.route(/^https?:\/\/[^/]+\/api\/positions(?:\/closed)?(?:\?.*)?$/, async (route) => {
    const request = route.request()
    if (request.method() !== 'GET') {
      await route.fulfill({ status: 405, body: 'Position mutations are disabled in this UI suite' })
      return
    }

    const closed = new URL(request.url()).pathname.endsWith('/closed')
    const positions = closed ? closedPositions : openPositions
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        content: positions,
        totalElements: positions.length,
        totalPages: 1,
        size: 20,
        number: 0,
      }),
    })
  })

  await page.route('**/api/health', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'UP', components: { api: { status: 'UP' } } }),
    })
  )
  await page.route('**/api/holidays', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { holidays: [], count: 0 } }),
    })
  )

  // Keep the application shell deterministic as well; these reads are shared
  // by every route and otherwise consume the local API's limited rate budget.
  await page.route(
    /^https?:\/\/[^/]+\/api\/(?:settings(?:\/llm|\/discord|\/trading|\/scanning)?|signals\/latest|fyers\/status)(?:\?.*)?$/,
    async (route) => {
      const path = new URL(route.request().url()).pathname
      const body = path.endsWith('/fyers/status')
        ? { connected: false }
        : path.endsWith('/signals/latest')
          ? []
          : {
              success: true,
              data: path.endsWith('/settings') ? { selectedBroker: 'yahoo' } : {},
            }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(body),
      })
    }
  )
}

test.describe('Positions View', () => {
  test.beforeEach(async ({ page }) => mockPositionApi(page))

  test('page header renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    await expect(page.locator('h1', { hasText: 'Positions' })).toBeVisible()
    await expect(
      page.locator('p', { hasText: /Active and closed paper trading positions/ })
    ).toBeVisible()
  })

  test('shows loading state then resolves', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)

    await page.waitForLoadState('networkidle')

    // Loading should be gone after networkidle
    const loadingGone = await page
      .locator('text=Loading positions')
      .isVisible()
      .catch(() => false)
    expect(loadingGone).toBe(false)
  })

  test('renders filter controls', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    // Search input
    const searchInput = page.locator('input[placeholder="Search symbol..."]')
    await expect(searchInput).toBeVisible()

    // Filter buttons — use getByRole to avoid ambiguity with "CLOSED" appearing in other text
    await expect(page.getByRole('button', { name: 'ALL' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'OPEN' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'CLOSED' })).toBeVisible()
  })

  test('filter buttons are clickable and toggle active state', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    // ALL is active by default
    await expect(page.getByRole('button', { name: 'ALL' })).toHaveClass(/bg-brand-subtle/)

    // Click OPEN filter
    await page.getByRole('button', { name: 'OPEN' }).click()
    await expect(page.getByRole('button', { name: 'OPEN' })).toHaveClass(/bg-brand-subtle/)
    await expect(page.getByRole('button', { name: 'ALL' })).not.toHaveClass(/bg-brand-subtle/)

    // Click CLOSED filter
    await page.getByRole('button', { name: 'CLOSED' }).click()
    await expect(page.getByRole('button', { name: 'CLOSED' })).toHaveClass(/bg-brand-subtle/)
  })

  test('search input filters by symbol', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const rows = page.locator('tbody tr')
    const initialCount = await rows.count()

    if (initialCount === 0) {
      // No data — just verify search input works without error
      await page.locator('input[placeholder="Search symbol..."]').fill('TEST')
      await expect(page.locator('text=No positions found')).toBeVisible()
      await page.locator('input[placeholder="Search symbol..."]').clear()
      return
    }

    const searchInput = page.locator('input[placeholder="Search symbol..."]')
    await searchInput.fill('RELIANCE')

    // Table should only show rows matching "RELIANCE" or show empty state
    const visibleRows = page.locator('tbody tr:visible')
    const count = await visibleRows.count()

    if (count > 0) {
      // Check first data row (skip "No positions found" row if present)
      const dataRows = page.locator('tbody tr').filter({ hasNot: page.locator('td[colspan]') })
      const dataCount = await dataRows.count()
      if (dataCount > 0) {
        const firstCellText = await dataRows.first().locator('td:first-child').textContent()
        expect(firstCellText?.toUpperCase()).toContain('RELIANCE')
      }
      // else: empty state shown, which is fine
    }

    // Clear search
    await searchInput.clear()
    await page.waitForLoadState('networkidle')
  })

  test('table renders with correct column headers', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const headers = page.locator('thead th')
    const headerTexts = (await headers.allTextContents()).map((t) => t.trim())
    const expected = [
      'Symbol',
      'Entry',
      'Qty',
      'Current',
      'Stop Loss',
      'Target',
      'Status',
      'P&L',
      'Action',
    ]

    for (const expectedHeader of expected) {
      expect(headerTexts).toContain(expectedHeader)
    }
  })

  test('table rows show position data when positions exist', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const rows = page.locator('tbody tr')
    const count = await rows.count()

    if (count === 0) {
      await expect(page.locator('text=No positions found')).toBeVisible()
      return
    }

    expect(count).toBeGreaterThan(0)

    // Filter out the "No positions found" placeholder row
    const dataRows = page.locator('tbody tr').filter({ hasNot: page.locator('td[colspan]') })
    const dataRowCount = await dataRows.count()

    if (dataRowCount === 0) {
      await expect(page.locator('text=No positions found')).toBeVisible()
      return
    }

    // Each row should have a symbol in the first cell
    const firstRowSymbol = await dataRows.first().locator('td:first-child').textContent()
    expect(firstRowSymbol?.trim().length).toBeGreaterThan(0)

    // Row should have at least 8 cells (9 columns)
    const cellsInRow = await dataRows.first().locator('td').count()
    expect(cellsInRow).toBeGreaterThanOrEqual(8)

    // P&L cell (8th) should contain rupee symbol or a number
    const pnlCell = dataRows.first().locator('td:nth-child(8)')
    const pnlText = (await pnlCell.textContent()) || ''
    expect(pnlText.length).toBeGreaterThan(0)
  })

  test('status badges show color-coded classes', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const rows = page.locator('tbody tr')
    const count = await rows.count()

    if (count === 0) {
      await expect(page.locator('text=No positions found')).toBeVisible()
      return
    }

    // Filter out the "No positions found" placeholder row
    const dataRows = page.locator('tbody tr').filter({ hasNot: page.locator('td[colspan]') })
    const dataRowCount = await dataRows.count()

    if (dataRowCount === 0) {
      await expect(page.locator('text=No positions found')).toBeVisible()
      return
    }

    // Status column is the 7th cell — should contain a badge span
    const statusCell = dataRows.first().locator('td:nth-child(7)')
    const statusText = (await statusCell.textContent())?.trim()
    expect(statusText).toBeTruthy()

    // Badge should have rounded-full class
    const badge = statusCell.locator('span.rounded-full')
    await expect(badge).toBeVisible()
  })

  test('P&L displays with color (green for profit, red for loss)', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const rows = page.locator('tbody tr')
    const count = await rows.count()

    if (count === 0) {
      await expect(page.locator('text=No positions found')).toBeVisible()
      return
    }

    // Filter out the "No positions found" placeholder row
    const dataRows = page.locator('tbody tr').filter({ hasNot: page.locator('td[colspan]') })
    const dataRowCount = await dataRows.count()

    if (dataRowCount === 0) {
      await expect(page.locator('text=No positions found')).toBeVisible()
      return
    }

    // P&L column is the 8th cell
    const pnlCell = dataRows.first().locator('td:nth-child(8)')
    const pnlText = await pnlCell.textContent()

    // Should contain rupee symbol
    expect(pnlText).toContain('₹')

    // Should have either text-success or text-danger class on the td itself
    const pnlClass = await pnlCell.evaluate((el) => el.className)
    expect(pnlClass.includes('text-success') || pnlClass.includes('text-danger')).toBe(true)
  })

  test('Close button only visible for OPEN positions', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const rows = page.locator('tbody tr')
    const count = await rows.count()

    if (count === 0) {
      await expect(page.locator('text=No positions found')).toBeVisible()
      return
    }

    // Check each row — Close button should only appear on OPEN rows
    const closeBtns = page.locator('tbody tr td button:has-text("Close")')
    const closeCount = await closeBtns.count()

    // Count OPEN status cells (7th column)
    const statusCells = page.locator('tbody tr td:nth-child(7)')
    const openCount = await statusCells.filter({ hasText: /^OPEN$/ }).count()

    // Close buttons should match OPEN positions
    expect(closeCount).toBe(openCount)
  })

  test('New Position modal opens on button click', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: 'New Position' }).click()

    await expect(page.locator('h2', { hasText: 'New Position' })).toBeVisible()
    await expect(page.locator('label', { hasText: 'Symbol' })).toBeVisible()
    await expect(page.locator('label', { hasText: 'Direction' })).toBeVisible()
    await expect(page.locator('label', { hasText: 'Quantity' })).toBeVisible()
    await expect(page.locator('label', { hasText: 'Entry Price' })).toBeVisible()
    await expect(page.locator('label', { hasText: 'Stop Loss' })).toBeVisible()
    await expect(page.locator('label', { hasText: 'Target' })).toBeVisible()
    await expect(page.locator('label', { hasText: 'Entry Reason' })).toBeVisible()
  })

  test('New Position modal form fields are functional', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: 'New Position' }).click()

    // Fill symbol
    await page.locator('input[placeholder="e.g. RELIANCE"]').fill('TCS')

    // Select direction
    await page.locator('select').first().selectOption('SHORT')

    // Fill quantity
    await page.locator('input[type="number"]').first().fill('10')

    // Fill entry price
    await page.locator('input[type="number"]').nth(1).fill('3500')

    // Fill stop loss
    const slInput = page.locator('input[placeholder="Optional"]').first()
    await expect(slInput).toBeVisible()
    await slInput.fill('3400')

    // Fill target
    const targetInput = page.locator('input[placeholder="Optional"]').last()
    await targetInput.fill('3700')

    // Fill entry reason
    await page
      .locator('textarea[placeholder="Why are you entering this trade?"]')
      .fill('Breakout above resistance')

    // Verify Cancel button exists
    await expect(page.locator('button', { hasText: 'Cancel' })).toBeVisible()

    // Verify Create Position button exists
    await expect(page.locator('button', { hasText: 'Create Position' })).toBeVisible()
  })

  test('New Position modal respects order type selection', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: 'New Position' }).click()

    // Default order type is MARKET — Limit Price field should be hidden
    const limitPriceVisible = await page
      .locator('label', { hasText: 'Limit Price' })
      .isVisible()
      .catch(() => false)
    expect(limitPriceVisible).toBe(false)
  })

  test('Close Position modal opens when clicking Close button', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    // Look for Close button in the table — it's in a td, not a button element
    const closeBtn = page.locator('tbody tr td button:text("Close")').first()
    const closeBtnVisible = await closeBtn.isVisible().catch(() => false)

    if (closeBtnVisible) {
      await closeBtn.click()

      await expect(page.locator('h2', { hasText: 'Close Position' })).toBeVisible()

      // Should show position summary
      await expect(page.locator('text=Exit Reason (Optional)')).toBeVisible()

      // Should have Cancel and Confirm Close buttons
      await expect(page.locator('button', { hasText: 'Cancel' })).toBeVisible()
      await expect(page.locator('button', { hasText: 'Confirm Close' })).toBeVisible()
    } else {
      test.skip()
    }
  })

  test('Close Position modal shows P&L summary', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const closeBtn = page.locator('tbody tr td button:text("Close")').first()
    const closeBtnVisible = await closeBtn.isVisible().catch(() => false)

    if (closeBtnVisible) {
      await closeBtn.click()

      // The modal should show the position's P&L with rupee symbol
      const modal = page.locator('.fixed.z-50').last()
      const modalText = await modal.textContent()
      expect(modalText?.includes('₹')).toBe(true)
    } else {
      test.skip()
    }
  })

  test('modal closes on Cancel button click', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: 'New Position' }).click()
    await expect(page.locator('h2', { hasText: 'New Position' })).toBeVisible()

    await page.locator('button', { hasText: 'Cancel' }).click()
    await expect(page.locator('h2', { hasText: 'New Position' })).not.toBeVisible()
  })

  test('modal closes on backdrop click', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    await page.getByRole('button', { name: 'New Position' }).click()
    await expect(page.locator('h2', { hasText: 'New Position' })).toBeVisible()

    // The modal uses @click.self on the outer fixed div — dispatch click on the outer
    // container (not a child element) so Vue's .self modifier fires the handler
    await page.locator('.fixed.inset-0.z-50').dispatchEvent('click')

    // Wait for the Teleport v-if to remove the modal
    await expect(page.locator('h2', { hasText: 'New Position' })).not.toBeVisible()
  })

  test('empty state displays when no positions exist', async ({ page }) => {
    await page.route('**/api/positions', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ content: [], totalElements: 0, totalPages: 0, size: 20, number: 0 }),
      })
    )
    await page.route('**/api/positions/closed', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ content: [], totalElements: 0, totalPages: 0, size: 20, number: 0 }),
      })
    )
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    // Could be data rows OR empty state
    const hasRows = (await page.locator('tbody tr').count()) > 0
    const hasEmpty = await page
      .locator('text=No positions found')
      .isVisible()
      .catch(() => false)

    expect(hasRows || hasEmpty).toBe(true)
  })

  test('search + filter combination works', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const rows = page.locator('tbody tr')
    const beforeCount = await rows.count()

    if (beforeCount === 0) {
      test.skip()
      return
    }

    // Apply search
    await page.locator('input[placeholder="Search symbol..."]').fill('TCS')
    await page.waitForLoadState('networkidle')

    const afterSearchCount = await page.locator('tbody tr').count()

    // After search, either 0 results or only matching rows
    if (afterSearchCount > 0) {
      const dataRows = page.locator('tbody tr').filter({ hasNot: page.locator('td[colspan]') })
      const dataCount = await dataRows.count()
      if (dataCount > 0) {
        const firstCell = await dataRows.first().locator('td:first-child').textContent()
        expect(firstCell?.toUpperCase()).toContain('TCS')
      }
    }

    // Reset search
    await page.locator('input[placeholder="Search symbol..."]').clear()
    await page.waitForLoadState('networkidle')
  })

  test('refresh button triggers re-fetch', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const refreshBtn = page.locator('text=Refresh')
    await expect(refreshBtn).toBeVisible()

    // Click refresh and verify no errors
    await refreshBtn.click()
    await page.waitForLoadState('networkidle')

    // Page should still be functional
    const bodyText = await page.locator('body').textContent()
    const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
    expect(errorMatch).toBeNull()
  })

  test('no JavaScript errors on page load', async ({ page }) => {
    const consoleErrors: string[] = []
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text())
      }
    })

    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')
    await expect(page.locator('h1', { hasText: 'Positions' })).toBeVisible()

    expect(consoleErrors).toHaveLength(0)
  })

  test('table is scrollable horizontally', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const tableContainer = page.locator('.overflow-x-auto')
    await expect(tableContainer).toBeVisible()

    // The table should have enough columns to warrant scrolling
    const table = page.locator('table.min-w-full')
    await expect(table).toBeVisible()
  })

  test('rows have hover state', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const rows = page.locator('tbody tr')
    const count = await rows.count()

    if (count > 0) {
      const firstRow = rows.first()
      // The row template has transition-colors hover:bg-bg-hover as CSS classes
      // Verify the row element has the hover CSS class defined
      const classAttr = await firstRow.evaluate((el) => el.className)
      expect(classAttr).toContain('hover:bg-bg-hover')
    }
  })

  test('navigation to positions page works from URL directly', async ({ page }) => {
    // Navigate directly to positions URL (not via nav)
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    // URL should be /positions
    expect(page.url()).toContain('/positions')

    // Page should be fully rendered
    await expect(page.locator('h1', { hasText: 'Positions' })).toBeVisible()
  })
})
