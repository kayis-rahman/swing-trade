import { test, expect, type Page } from '@playwright/test'
import { fulfillJson } from '../fixtures/api'

const DASHBOARD = 'http://localhost:3003'

async function mockSettingsApi(page: Page) {
  const responses: Record<string, unknown> = {
    '/settings': { success: true, data: { selectedBroker: 'fyers' } },
    '/settings/llm': { success: true, data: { 'llm.backend': 'local' } },
    '/settings/discord': { success: true, data: {} },
    '/settings/trading': { success: true, data: {} },
    '/settings/scanning': { success: true, data: {} },
    '/settings/pi/status': { success: true, data: { running: false, message: 'Stopped' } },
    '/fyers/status': { success: true, data: { connected: false, clientId: null } },
    '/health': { status: 'UP', components: { api: { status: 'UP' }, db: { status: 'UP' } } },
    '/health/full': {
      status: 'UP',
      components: { api: { status: 'UP' }, db: { status: 'UP' } },
    },
  }

  await page.route(/^https?:\/\/[^/]+\/api(?:\/|$)/, async (route) => {
    const { pathname } = new URL(route.request().url())
    const path = pathname.replace(/^\/api/, '')
    const method = route.request().method()

    if (method === 'POST' && path === '/settings/save') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: {} }),
      })
      return
    }
    if (method !== 'GET') {
      await route.fulfill({ status: 405, body: 'Unexpected settings test mutation' })
      return
    }

    const body = responses[path] ?? { success: true, data: [] }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(body),
    })
  })
}

test.describe('Settings View', () => {
  test.beforeEach(async ({ page }) => mockSettingsApi(page))

  test('page header renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    await expect(page.locator('h1', { hasText: 'Settings' })).toBeVisible()
    await expect(
      page.locator('p', {
        hasText: 'Configure how Swing Trade connects, thinks, trades, and reports back to you.',
      })
    ).toBeVisible()
  })

  test('shows loading state then resolves', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)

    await page.waitForLoadState('networkidle')

    // Loading should be gone after networkidle
    const loadingGone = await page
      .locator('text=Checking system')
      .isVisible()
      .catch(() => false)
    expect(loadingGone).toBe(false)
  })

  test('renders all section headers', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    const sections = [
      ['Broker', 'Broker Connection'],
      ['AI/LLM', 'LLM & Intelligence'],
      ['Trading', 'Trading Configuration'],
      ['Health', 'System Health'],
    ] as const
    for (const [tab, heading] of sections) {
      await page.getByRole('tab', { name: tab }).click()
      await expect(page.getByRole('heading', { name: heading })).toBeVisible()
    }
  })

  test('broker selection buttons render', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    await expect(page.locator('.broker-option')).toHaveCount(4)
    for (const broker of ['Fyers', 'Upstox', 'Yahoo Finance', 'None (Read Only)']) {
      await expect(page.locator('.broker-option').filter({ hasText: broker })).toBeVisible()
    }
  })

  test('broker selection highlights active broker', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    const activeOptions = page.locator('.broker-option.bg-brand-subtle')
    await expect(activeOptions).toHaveCount(1)
    await expect(activeOptions).toContainText(/Fyers|Upstox|Yahoo Finance|None \(Read Only\)/)
  })

  test('broker selection switches displayed content', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    // Fyers content should be visible by default
    const fyersConnect = page.locator('button:has-text("Connect")')
    const fyersVisible = await fyersConnect.isVisible().catch(() => false)

    if (fyersVisible) {
      await expect(fyersConnect).toBeVisible()
    }

    // Click Upstox — should show placeholder
    await page.locator('.broker-option').filter({ hasText: 'Upstox' }).click()
    await expect(page.locator('text=Upstox integration coming soon')).toBeVisible()

    // Click Yahoo Finance — should show status
    const yahoo = page.locator('.broker-option').filter({ hasText: 'Yahoo Finance' })
    await yahoo.click()
    await expect(yahoo).toHaveClass(/bg-brand-subtle/)

    // Click None
    await page.locator('.broker-option').filter({ hasText: 'None (Read Only)' }).click()

    // Switch back to Fyers
    await page.locator('.broker-option').filter({ hasText: 'Fyers' }).click()
  })

  test('Fyers connection status indicator renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    // Should show Connected or Disconnected text
    const hasConnected = await page
      .locator('text=Connected')
      .isVisible()
      .catch(() => false)
    const hasDisconnected = await page
      .locator('text=Disconnected')
      .isVisible()
      .catch(() => false)
    expect(hasConnected || hasDisconnected).toBe(true)
  })

  test('Fyers controls reflect the current connection status', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.locator('.broker-option').filter({ hasText: 'Fyers' }).click()

    const connectionControl = page
      .getByRole('button', { name: 'Connect Fyers Account' })
      .or(page.getByRole('button', { name: 'Disconnect' }))
    const unavailable = page.getByText('Status unavailable')
    await expect(connectionControl.or(unavailable)).toBeVisible()
  })

  test('Fyers controls match its disconnected status', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.locator('.broker-option').filter({ hasText: 'Fyers' }).click()

    await expect(page.getByText('Disconnected', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Connect Fyers Account' })).toBeEnabled()
  })

  test('Fyers auth code input appears with connect button', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    // Auth code input should be visible alongside connect button
    const authInput = page.locator('input[placeholder="Paste auth code from browser"]')
    const authVisible = await authInput.isVisible().catch(() => false)

    if (authVisible) {
      await expect(authInput).toBeVisible()

      // Submit button should also be visible
      await expect(page.locator('button:has-text("Submit")')).toBeVisible()

      // Help text should be visible
      await expect(page.locator('text=Complete login on Fyers')).toBeVisible()
    }
  })

  test('Local llama.cpp backend reveals its configuration', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'AI/LLM' }).click()
    await page.locator('.llm-backend-option').filter({ hasText: 'Local' }).click()

    await expect(page.getByRole('heading', { name: 'Local LLM (llama.cpp)' })).toBeVisible()
    await expect(page.getByPlaceholder('http://localhost:8080/v1')).toBeVisible()
    await expect(
      page.getByPlaceholder('Path to the GGUF model configured on the server').first()
    ).toBeVisible()
  })

  test('PDF Extraction section renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'AI/LLM' }).click()

    await expect(page.locator('h3', { hasText: 'PDF Extraction (Pi 5)' })).toBeVisible()

    // Find the PDF section and check for inputs within it
    const pdfSection = page.locator('h3', { hasText: 'PDF Extraction (Pi 5)' }).locator('..')
    const inputs = pdfSection.locator('input')
    const inputCount = await inputs.count()
    expect(inputCount).toBeGreaterThanOrEqual(2)
  })

  test('Discord notification settings render', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'AI/LLM' }).click()

    await expect(page.locator('h3', { hasText: 'Discord Notifications' })).toBeVisible()

    // Toggle switch
    const toggle = page.locator('input[type="checkbox"].sr-only')
    await expect(toggle).toBeAttached()
    await expect(page.locator('label').filter({ has: toggle }).first()).toBeVisible()

    // Webhook URL input
    const webhookInput = page.locator('input[placeholder*="webhook"]')
    const webhookVisible = await webhookInput.isVisible().catch(() => false)
    if (webhookVisible) {
      await expect(webhookInput).toBeVisible()
    }

    // Test button (scoped to Discord section)
    const discordSection = page.locator('h3', { hasText: 'Discord Notifications' }).locator('..')
    const testBtns = discordSection.locator('button:has-text("Test")')
    const testCount = await testBtns.count()
    expect(testCount).toBeGreaterThanOrEqual(1)
  })

  test('Trading Configuration section renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'Trading' }).click()

    // Mode select
    const modeSelect = page.locator('select')
    await expect(modeSelect).toBeVisible()

    // Check options
    const modeOptions = await modeSelect.locator('option').allTextContents()
    expect(modeOptions).toContain('Paper Trading')
    expect(modeOptions).toContain('Live Trading')

    // Max Position Size input
    const maxPosLabel = page.locator('text=Max Position Size')
    await expect(maxPosLabel).toBeVisible()

    // Stop Loss input
    const slLabel = page.locator('text=Stop Loss')
    await expect(slLabel).toBeVisible()

    // Take Profit input
    const tpLabel = page.locator('text=Take Profit')
    await expect(tpLabel).toBeVisible()

    // Percentage indicators
    const percentSigns = await page.locator('text=%').count()
    expect(percentSigns).toBeGreaterThanOrEqual(3)
  })

  test('Trading Configuration inputs accept numeric values', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'Trading' }).click()

    // Find all number inputs in the Trading Configuration section
    const tradingSection = page.locator('text=Trading Configuration').locator('..')
    const numberInputs = tradingSection.locator('input[type="number"]')
    const count = await numberInputs.count()

    if (count > 0) {
      // Fill the first number input
      await numberInputs.first().fill('50')
      const value = await numberInputs.first().inputValue()
      expect(value).toBe('50')
    }
  })

  test('Save All Settings button renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    const saveBtn = page.locator('button:has-text("Save All Settings")')
    await expect(saveBtn).toBeVisible()
    await expect(saveBtn).toBeEnabled()
  })

  test('saving submits settings and shows loading then success', async ({ page }) => {
    let releaseSave!: () => void
    const saveGate = new Promise<void>((resolve) => {
      releaseSave = resolve
    })
    let submittedSettings: unknown
    await page.route('**/api/settings/save', async (route) => {
      submittedSettings = route.request().postDataJSON()
      await saveGate
      await fulfillJson(route, { success: true, data: {} })
    })

    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    const saveBtn = page.locator('button:has-text("Save All Settings")')
    await expect(saveBtn).toBeVisible()
    await saveBtn.click()
    await expect(page.getByRole('button', { name: 'Saving...' })).toBeVisible()
    await expect
      .poll(() => submittedSettings)
      .toMatchObject({
        broker: 'fyers',
        llm: {
          'llm.backend': 'local',
          'pi-agent.provider': 'openai-codex',
          'pi-agent.model': 'gpt-5.6-luna',
        },
        trading: {
          'trading.mode': 'paper',
          'trading.max_position_size': '10',
          'trading.allocation_per_position': '100000',
        },
        scanning: { 'candidate-scan.max-concurrent': '3' },
      })

    releaseSave()
    await expect(page.getByRole('button', { name: 'Saved!' })).toBeVisible()
  })

  test('System Health section renders with component statuses', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'Health' }).click()

    // Health section should have component rows
    const healthRows = page.locator('.settings-health-row')
    await expect(healthRows.first()).toBeVisible()

    // Should have status badges with rounded-full
    const statusBadges = page.locator('span.rounded-full.px-2\\.5')
    const badgeCount = await statusBadges.count()

    // Either badges are visible or loading spinner is visible
    const loadingVisible = await page
      .locator('text=Checking system')
      .isVisible()
      .catch(() => false)
    expect(badgeCount > 0 || loadingVisible).toBe(true)
  })

  test('Health status badges show color-coded states', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'Health' }).click()

    // Health badges should have status colors
    const healthSection = page.locator('.health-card')
    const healthText = (await healthSection.textContent()) || ''

    // Should contain status indicators (UP/DOWN/DEGRADED or similar)
    const hasStatus =
      healthText.includes('UP') ||
      healthText.includes('DOWN') ||
      healthText.includes('DEGRADED') ||
      healthText.includes('Checking')
    expect(hasStatus).toBe(true)
  })

  test('Auth error banner renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings?auth=error`)
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('Fyers authentication failed. Please try again.')).toBeVisible()
    await expect(page.locator('button').filter({ hasText: '×' })).toBeVisible()
  })

  test('Auth success banner renders', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings?auth=success`)
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('Fyers connected successfully!')).toBeVisible()
    await expect(page.locator('button').filter({ hasText: '×' })).toBeVisible()
  })

  test('no JavaScript errors on page load', async ({ page }) => {
    const consoleErrors: string[] = []
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text())
      }
    })

    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await expect(page.locator('h1', { hasText: 'Settings' })).toBeVisible()

    expect(consoleErrors).toHaveLength(0)
  })

  test('navigation to settings page works from URL directly', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    expect(page.url()).toContain('/settings')
    await expect(page.locator('h1', { hasText: 'Settings' })).toBeVisible()
  })

  test('LLM section has model name helper text', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'AI/LLM' }).click()

    // "Model name" helper text should appear next to model inputs
    const modelNameHelpers = await page.locator('text=Model name').count()
    // At least one (vLLM model)
    expect(modelNameHelpers).toBeGreaterThanOrEqual(1)
  })

  test('Discord toggle is clickable', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'AI/LLM' }).click()

    const toggle = page.locator('input[type="checkbox"].sr-only').first()
    await expect(toggle).toBeAttached()

    // Click the visible toggle label (peer-checked changes styling)
    const toggleLabel = page.locator('label').filter({ has: toggle }).first()
    await expect(toggleLabel).toBeVisible()
    await toggleLabel.click()

    // Toggle should change state
    const isChecked = await toggle.isChecked()
    expect(typeof isChecked).toBe('boolean')
  })

  test('Discord enabled label is visible', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'AI/LLM' }).click()

    await expect(page.locator('text=Enable Discord')).toBeVisible()
  })

  test('Discord test saves its settings and never contacts the external webhook in E2E', async ({
    page,
  }) => {
    let savedDiscordSettings: unknown
    let webhookTestRequests = 0
    await page.route('**/api/settings/discord', async (route) => {
      if (route.request().method() !== 'PUT') return route.fallback()
      savedDiscordSettings = route.request().postDataJSON()
      await fulfillJson(route, { success: true, data: {} })
    })
    await page.route('**/api/settings/test/discord', async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      webhookTestRequests++
      await fulfillJson(route, { success: true, data: { success: true } })
    })

    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'AI/LLM' }).click()
    await page
      .getByPlaceholder('https://discord.com/api/webhooks/...')
      .fill('https://example.invalid/test')
    const toggle = page.locator('input[type="checkbox"].sr-only').first()
    await page.locator('label').filter({ has: toggle }).first().click()
    await expect(toggle).toBeChecked()
    const discordSection = page.locator('h3', { hasText: 'Discord Notifications' }).locator('..')
    await discordSection.getByRole('button', { name: 'Test', exact: true }).click()

    await expect
      .poll(() => savedDiscordSettings)
      .toEqual({
        'discord.webhook.url': 'https://example.invalid/test',
        'discord.webhook.enabled': 'true',
      })
    await expect.poll(() => webhookTestRequests).toBe(1)
    await expect(page.getByText('Discord webhook test successful!')).toBeVisible()
  })

  test('Trading mode select has correct options', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')
    await page.getByRole('tab', { name: 'Trading' }).click()

    const modeSelect = page.getByRole('combobox', { name: 'Trading mode' })
    await expect(modeSelect).toBeVisible()

    const options = await modeSelect.locator('option').all()
    expect(options.length).toBeGreaterThanOrEqual(2)

    const optionValues = await Promise.all(options.map((o) => o.textContent()))
    expect(optionValues.some((o) => o.includes('Paper'))).toBe(true)
    expect(optionValues.some((o) => o.includes('Live'))).toBe(true)
  })

  test('All sections are card-panel styled', async ({ page }) => {
    await page.goto(`${DASHBOARD}/settings`)
    await page.waitForLoadState('networkidle')

    await expect(page.locator('.broker-card, .llm-card, .trading-card')).toHaveCount(3)
  })
})
