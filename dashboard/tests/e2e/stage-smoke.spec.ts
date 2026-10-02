import { test, expect } from '@playwright/test'

const dashboardUrl = process.env.STAGE_DASHBOARD_URL ?? 'http://localhost:3000'

test.describe('deployed stage dashboard smoke', () => {
  test('loads and navigates read-only views without frontend exceptions', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (error) => errors.push(error.message))
    await page.goto(`${dashboardUrl}/`)
    await expect(page).toHaveTitle(/Swing\s*Trade/i)
    await expect(page.getByRole('heading', { name: 'Portfolio at a glance' })).toBeVisible()

    for (const route of ['/data', '/signals', '/positions']) {
      await page.goto(`${dashboardUrl}${route}`)
      await expect(page.locator('body')).not.toContainText('Application error')
    }
    expect(errors).toEqual([])
  })
})
