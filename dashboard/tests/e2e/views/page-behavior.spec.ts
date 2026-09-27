import { test, expect } from '@playwright/test'

const DASHBOARD = 'http://localhost:3003'

const pages = [
  {
    path: '/',
    heading: 'Portfolio at a glance',
    marker: 'Capital, risk, performance, and execution readiness in one place.',
  },
  { path: '/positions', heading: 'Positions', marker: 'Active and closed paper trading positions' },
  {
    path: '/signals',
    heading: 'Signals',
    marker:
      'Review fresh opportunities, compare conviction, and execute only when the setup is clear.',
  },
  { path: '/portfolio', heading: 'Portfolio', marker: 'Portfolio' },
  { path: '/watchlist', heading: 'Watchlist', marker: 'Watchlist' },
  {
    path: '/data',
    heading: 'Data Ingestion',
    marker: 'Monitor data quality and pull historical market data',
  },
  {
    path: '/settings',
    heading: 'Settings',
    marker: 'Configure how Swing Trade connects, thinks, trades, and reports back to you.',
  },
]

for (const { path, heading, marker } of pages) {
  test(`${heading} route renders its expected shell`, async ({ page }) => {
    await page.goto(`${DASHBOARD}${path}`)
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveTitle(/Swing Trade/i)
    await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible()
    await expect(page.locator('body')).toContainText(marker)

    const bodyText = await page.locator('body').textContent()
    expect(bodyText).not.toMatch(/Unexpected error|TypeError|Cannot read/)
  })
}
