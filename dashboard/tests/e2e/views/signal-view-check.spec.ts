import { test, expect } from '@playwright/test'

test('signals view loads', async ({ page }) => {
  await page.goto('http://localhost:3003/signals')
  await page.waitForLoadState('networkidle')

  await expect(page).toHaveTitle(/Swing\s?Trade/i)
  await expect(page.getByRole('heading', { name: 'Signals', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Generate All' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Refresh' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'BUY' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'SELL' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'ACTIVE' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'PENDING' })).toBeVisible()

  const bodyText = await page.locator('body').textContent()
  expect(bodyText).not.toMatch(/Unexpected error|TypeError|Cannot read/)
})
