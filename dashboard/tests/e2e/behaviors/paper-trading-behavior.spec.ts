import { test, expect } from '@playwright/test'

const DASHBOARD = 'http://localhost:3003'

test.describe('Paper Trading Behavior', () => {
  test('positions page renders with data or empty state', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const rows = page.locator('tbody tr')
    const count = await rows.count()

    if (count > 0) {
      // Filter out the "No positions found" placeholder row
      const dataRows = rows.filter({ hasNot: page.locator('td[colspan]') })
      const dataCount = await dataRows.count()
      if (dataCount > 0) {
        // Each row should have a symbol in the first cell
        const firstRowSymbol = await dataRows.first().locator('td:first-child').textContent()
        expect(firstRowSymbol?.trim().length).toBeGreaterThan(0)
      } else {
        await expect(page.locator('text=No positions found')).toBeVisible()
      }
    } else {
      await expect(page.locator('text=No positions found')).toBeVisible()
    }

    // No JS errors
    const bodyText = await page.locator('body').textContent()
    const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
    expect(errorMatch).toBeNull()
  })

  test('portfolio page loads with performance data', async ({ page }) => {
    await page.goto(`${DASHBOARD}/portfolio`)
    await page.waitForLoadState('networkidle')

    // Portfolio page should render without errors
    const bodyText = await page.locator('body').textContent()
    const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
    expect(errorMatch).toBeNull()
  })

  test('dashboard page shows metrics cards', async ({ page }) => {
    await page.goto(`${DASHBOARD}/`)
    await page.waitForLoadState('networkidle')

    // Dashboard should have grid layout
    const grid = page.locator('.grid').first()
    await expect(grid).toBeVisible()

    // No JS errors
    const bodyText = await page.locator('body').textContent()
    const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
    expect(errorMatch).toBeNull()
  })

  test('positions page renders without JS errors', async ({ page }) => {
    await page.goto(`${DASHBOARD}/positions`)
    await page.waitForLoadState('networkidle')

    const bodyText = await page.locator('body').textContent()
    const errorMatch = bodyText?.match(/Unexpected error|TypeError|Cannot read/)
    expect(errorMatch).toBeNull()
  })
})
