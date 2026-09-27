import { test, expect } from '@playwright/test'

test('orchestrator page — full feature check', async ({ page }) => {
  const pageErrors: string[] = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  // These shell reads are covered by the live OpenAPI sweep. Isolating them
  // keeps this feature check focused on orchestrator data and avoids spending
  // the local API's shared rate budget on repeated navigation.
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

  await page.goto('http://localhost:3003/orchestrator')
  await page.waitForLoadState('networkidle')
  await expect(page.locator('h1:has-text("Job Orchestrator")')).toBeVisible()
  await expect(page.locator('table tbody tr').first()).toBeVisible()

  const errorBanner = page.locator('text=AN UNEXPECTED ERROR OCCURRED')
  await expect(errorBanner).not.toBeVisible()
  await expect(page.locator('button:has-text("Run")')).toBeVisible()
  await expect(page.locator('button:has-text("History")')).toBeVisible()

  const tableRows = await page.locator('table tbody tr').all()
  expect(tableRows.length).toBeGreaterThan(0)

  for (const [index, row] of tableRows.entries()) {
    const cells = await row.locator('td').all()
    let svgCount = 0
    for (let j = 1; j <= 7; j++) {
      if (j < cells.length) {
        svgCount += await cells[j].locator('svg').count()
      }
    }
    if (index < 3) {
      expect(svgCount).toBeGreaterThan(0)
    }
  }

  const firstRow = page.locator('table tbody tr').first()
  await firstRow.click()

  const detailCards = page.locator('.grid .rounded-md.border.bg-bg-surface.p-3')
  await expect(detailCards).toHaveCount(7)

  await firstRow.click()
  await expect(detailCards).toHaveCount(0)
  expect(pageErrors).toEqual([])
})
