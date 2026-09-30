import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AppError } from '../errors/appError'

const storeMocks = vi.hoisted(() => {
  const state = {
    selectedBroker: 'fyers',
    tradingConfig: {
      mode: 'paper',
      maxPositionSize: 10,
      stopLoss: 5,
      takeProfit: 15,
      allocationPerPosition: 100000,
    },
    llmSettings: {
      llmBackend: 'pi_ssh',
      llmBaseUrl: '',
      openaiBaseUrl: '',
      openaiModel: '',
      openaiApiKey: '',
      ollamaBaseUrl: '',
      ollamaModel: '',
      ollamaApiKey: '',
      llamacppModel: '',
      pdfBaseUrl: '',
      pdfModel: '',
    },
    discordSettings: { webhookUrl: '', enabled: false },
  }

  return {
    getSettings: vi.fn(() => state),
    loadSettings: vi.fn(),
    saveSettings: vi.fn(),
    saveLlmSettings: vi.fn(),
    saveDiscordSettings: vi.fn(),
  }
})

const apiMocks = vi.hoisted(() => ({
  getFyersLoginUrl: vi.fn(),
  getFyersStatus: vi.fn(),
  fyersAuthCode: vi.fn(),
  fyersLogout: vi.fn(),
  testDiscordWebhook: vi.fn(),
  testPiConnection: vi.fn(),
  testOpenAiConnection: vi.fn(),
  testOllamaConnection: vi.fn(),
  startPiServer: vi.fn(),
  stopPiServer: vi.fn(),
  getPiServerStatus: vi.fn(),
  getHealthStatus: vi.fn(),
}))

vi.mock('../stores/settings', () => ({
  ...storeMocks,
  brokerLabels: { fyers: 'Fyers', upstox: 'Upstox', yahoo: 'Yahoo Finance', none: 'Paper Only' },
}))
vi.mock('../api/client', () => apiMocks)
vi.mock('../api/fyers', () => ({
  getFyersLoginUrl: apiMocks.getFyersLoginUrl,
  getFyersStatus: apiMocks.getFyersStatus,
  fyersAuthCode: apiMocks.fyersAuthCode,
  fyersLogout: apiMocks.fyersLogout,
}))
vi.mock('../api/settings', () => ({
  testDiscordWebhook: apiMocks.testDiscordWebhook,
  testPiConnection: apiMocks.testPiConnection,
  testOpenAiConnection: apiMocks.testOpenAiConnection,
  testOllamaConnection: apiMocks.testOllamaConnection,
  startPiServer: apiMocks.startPiServer,
  stopPiServer: apiMocks.stopPiServer,
  getPiServerStatus: apiMocks.getPiServerStatus,
}))
vi.mock('../api/health', () => ({ getHealthStatus: apiMocks.getHealthStatus }))

function cardWithHeading(wrapper: VueWrapper, heading: string): HTMLElement {
  const headingWrapper = wrapper
    .findAll('h2, h3')
    .find((candidate) => candidate.text().trim() === heading)
  expect(headingWrapper, `Expected heading "${heading}"`).toBeDefined()
  const card = headingWrapper!.element.closest('.card-panel')
  expect(card).not.toBeNull()
  return card as HTMLElement
}

async function mountSettings() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/settings', component: { template: '<div />' } }],
  })
  await router.push('/settings')
  await router.isReady()

  const SettingsView = (await import('./SettingsView.vue')).default
  return mount(SettingsView, {
    global: {
      plugins: [router],
      stubs: {
        LoadingSpinner: { template: '<div>Checking system...</div>' },
        Toast: {
          props: ['message', 'type'],
          template: '<div role="status" :data-type="type">{{ message }}</div>',
        },
      },
    },
  })
}

describe('SettingsView — unavailable and unconfirmed states', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    storeMocks.loadSettings.mockResolvedValue(['LLM settings'])
    storeMocks.saveSettings.mockResolvedValue(false)
    storeMocks.saveLlmSettings.mockResolvedValue(false)
    storeMocks.saveDiscordSettings.mockResolvedValue(false)
    apiMocks.getFyersStatus.mockRejectedValue(
      new AppError({ kind: 'network', message: 'Fyers status unavailable', retryable: true })
    )
    apiMocks.getPiServerStatus.mockRejectedValue(
      new AppError({ kind: 'network', message: 'Pi status unavailable', retryable: true })
    )
    apiMocks.getHealthStatus.mockRejectedValue(
      new AppError({ kind: 'network', message: 'Health status unavailable', retryable: true })
    )
  })

  it('does not present failed status checks as disconnected, stopped, or indefinitely checking', async () => {
    const wrapper = await mountSettings()
    await flushPromises()

    const brokerCard = cardWithHeading(wrapper, 'Broker Connection')
    expect(brokerCard.textContent).toMatch(/status unavailable/i)
    expect(brokerCard.textContent).not.toContain('Disconnected')

    await wrapper.get('[aria-label="AI/LLM"]').trigger('click')
    const piHeading = wrapper
      .findAll('h3')
      .find((candidate) => candidate.text().trim() === 'Pi SSH LLM (llama.cpp)')
    expect(piHeading).toBeDefined()
    const piSection = piHeading!.element.closest('.space-y-4') as HTMLElement
    expect(piSection.textContent).toMatch(/status unavailable/i)
    expect(piSection.textContent).not.toContain('Stopped')

    await wrapper.get('[aria-label="Health"]').trigger('click')
    const healthCard = cardWithHeading(wrapper, 'System Health')
    expect(healthCard.textContent).toMatch(/status unavailable/i)
    expect(healthCard.textContent).not.toContain('Checking system...')

    wrapper.unmount()
  })

  it('labels failed settings as unconfirmed defaults and prevents saving them', async () => {
    const wrapper = await mountSettings()
    await flushPromises()

    expect(wrapper.text()).toMatch(/LLM settings/i)
    expect(wrapper.text()).toMatch(/unconfirmed defaults/i)

    const saveButton = wrapper
      .findAll('button')
      .find((candidate) => candidate.text().includes('Save All Settings'))
    expect(saveButton).toBeDefined()
    expect(saveButton!.attributes('disabled')).toBeDefined()
    await saveButton!.trigger('click')
    expect(storeMocks.saveSettings).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})

describe('SettingsView — Fyers not-configured state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    storeMocks.loadSettings.mockResolvedValue([])
    apiMocks.getFyersStatus.mockResolvedValue({ connected: false, clientId: '' })
    apiMocks.getPiServerStatus.mockResolvedValue({ running: false })
    apiMocks.getHealthStatus.mockResolvedValue({ status: 'UP' })
  })

  async function clickConnectFyers(wrapper: VueWrapper) {
    const connectButton = wrapper
      .findAll('button')
      .find((candidate) => candidate.text().includes('Connect Fyers Account'))
    expect(connectButton, 'Expected the Connect Fyers Account button').toBeDefined()
    await connectButton!.trigger('click')
    await flushPromises()
  }

  it('reports not-configured instead of authentication-failed when the backend lacks Fyers credentials', async () => {
    // Reproduction of the staging issue: GET /api/fyers/login → 400 with code
    // FYERS_NOT_CONFIGURED must not surface as "Fyers authentication failed".
    apiMocks.getFyersLoginUrl.mockRejectedValue(
      new AppError({
        kind: 'validation',
        status: 400,
        code: 'FYERS_NOT_CONFIGURED',
        message: 'FYERS_CLIENT_ID not configured',
      })
    )

    const wrapper = await mountSettings()
    await flushPromises()
    await clickConnectFyers(wrapper)

    expect(wrapper.text()).toMatch(/Fyers is not configured on this environment/i)
    expect(wrapper.text()).not.toContain('Fyers authentication failed')

    wrapper.unmount()
  })

  it('detects not-configured from the error message when the backend sends no code', async () => {
    // Backends predating the code field only return the 400 message.
    apiMocks.getFyersLoginUrl.mockRejectedValue(
      new AppError({
        kind: 'validation',
        status: 400,
        message: 'FYERS_CLIENT_ID not configured',
      })
    )

    const wrapper = await mountSettings()
    await flushPromises()
    await clickConnectFyers(wrapper)

    expect(wrapper.text()).toMatch(/Fyers is not configured on this environment/i)
    expect(wrapper.text()).not.toContain('Fyers authentication failed')

    wrapper.unmount()
  })

  it('keeps the authentication-failed message for genuine failures', async () => {
    // Guard against over-correction: unrelated errors must keep the original message.
    apiMocks.getFyersLoginUrl.mockRejectedValue(
      new AppError({ kind: 'server', status: 500, message: 'Backend exploded' })
    )

    const wrapper = await mountSettings()
    await flushPromises()
    await clickConnectFyers(wrapper)

    expect(wrapper.text()).toContain('Fyers authentication failed. Please try again.')
    expect(wrapper.text()).not.toMatch(/not configured on this environment/i)

    wrapper.unmount()
  })
})
