import { reactive } from 'vue'
import {
  getSettings as getBrokerSettings,
  getLlmSettings,
  setLlmSettings,
  getDiscordSettings,
  setDiscordSettings,
  getTradingSettings,
  setTradingSettings,
  getScanningSettings,
  setScanningSettings,
  saveAllSettings,
} from '../api/client'

interface TradingConfig {
  mode: 'paper' | 'live'
  maxPositionSize: number
  stopLoss: number
  takeProfit: number
  allocationPerPosition: number
  initialCapital: number
}

interface ScanningConfig {
  maxConcurrent: number
}

type LlmBackend = 'local' | 'pi_ssh' | 'openai' | 'ollama' | 'laya' | 'pi_agent' | 'mlx'

interface LlmSettings {
  llmBackend: LlmBackend
  llmBaseUrl: string
  openaiBaseUrl: string
  openaiModel: string
  openaiApiKey: string
  ollamaBaseUrl: string
  ollamaModel: string
  ollamaApiKey: string
  layaBaseUrl: string
  layaModel: string
  piAgentProvider: string
  piAgentModel: string
  llamacppModel: string
  mlxModel: string
  pdfBaseUrl: string
  pdfModel: string
}

interface DiscordSettings {
  webhookUrl: string
  enabled: boolean
}

interface SettingsState {
  selectedBroker: 'fyers' | 'upstox' | 'yahoo' | 'none'
  tradingConfig: TradingConfig
  scanningConfig: ScanningConfig
  llmSettings: LlmSettings
  discordSettings: DiscordSettings
}

const defaults: SettingsState = {
  // Match the backend's broker default so the shell never probes Fyers before
  // persisted settings have finished loading.
  selectedBroker: 'yahoo',
  tradingConfig: {
    mode: 'paper',
    maxPositionSize: 10,
    stopLoss: 5,
    takeProfit: 15,
    allocationPerPosition: 100000,
    initialCapital: 500000,
  },
  scanningConfig: { maxConcurrent: 3 },
  llmSettings: {
    llmBackend: 'local',
    llmBaseUrl: '',
    openaiBaseUrl: '',
    openaiModel: '',
    openaiApiKey: '',
    piAgentProvider: 'openai-codex',
    piAgentModel: 'gpt-5.6-luna',
    llamacppModel: '',
    mlxModel: 'Qwen/Qwen2.5-3B-Instruct',
    pdfBaseUrl: '',
    pdfModel: '',
  },
  discordSettings: {
    webhookUrl: '',
    enabled: false,
  },
}

const supportedLlmBackends = new Set<LlmBackend>([
  'local',
  'pi_ssh',
  'openai',
  'ollama',
  'laya',
  'pi_agent',
  'mlx',
])

function createDefaultState(): SettingsState {
  return {
    selectedBroker: defaults.selectedBroker,
    tradingConfig: { ...defaults.tradingConfig },
    scanningConfig: { ...defaults.scanningConfig },
    llmSettings: { ...defaults.llmSettings },
    discordSettings: { ...defaults.discordSettings },
  }
}

function parseLlmBackend(raw: string | undefined): LlmBackend {
  // Normalize the legacy persisted value without exposing it as a selectable backend.
  if (raw === 'gpuhub') return 'openai'
  return supportedLlmBackends.has(raw as LlmBackend)
    ? (raw as LlmBackend)
    : defaults.llmSettings.llmBackend
}

function applySettings(target: SettingsState, source: SettingsState) {
  target.selectedBroker = source.selectedBroker
  Object.assign(target.tradingConfig, source.tradingConfig)
  Object.assign(target.llmSettings, source.llmSettings)
  Object.assign(target.discordSettings, source.discordSettings)
}

function parseTradingConfig(raw: Record<string, string>): TradingConfig {
  return {
    mode: (raw['trading.mode'] || 'paper') as 'paper' | 'live',
    maxPositionSize: parseInt(raw['trading.max_position_size'] || '10', 10),
    stopLoss: parseInt(raw['trading.stop_loss'] || '5', 10),
    takeProfit: parseInt(raw['trading.take_profit'] || '15', 10),
    allocationPerPosition: parseInt(raw['trading.allocation_per_position'] || '100000', 10),
    initialCapital: parseInt(raw['trading.initial_capital'] || '500000', 10),
  }
}

function parseScanningConfig(raw: Record<string, string>): ScanningConfig {
  const value = Number.parseInt(raw['candidate-scan.max-concurrent'] || '3', 10)
  return { maxConcurrent: Number.isFinite(value) ? Math.min(12, Math.max(1, value)) : 3 }
}

interface LoadAllResult {
  state: SettingsState
  failedSections: string[]
}

function readData<T>(value: T | { success: boolean; data?: T }): T | undefined {
  if (typeof value === 'object' && value !== null && 'success' in value) {
    return value.success ? value.data : undefined
  }
  return value as T
}

async function loadAll(): Promise<LoadAllResult> {
  const state = createDefaultState()
  const failedSections: string[] = []

  // Load broker selection
  try {
    const brokerRes = await getBrokerSettings()
    const brokerData = readData(brokerRes)
    if (
      brokerData?.selectedBroker &&
      ['fyers', 'upstox', 'yahoo', 'none'].includes(brokerData.selectedBroker)
    ) {
      state.selectedBroker = brokerData.selectedBroker as SettingsState['selectedBroker']
    } else {
      // Either the response was missing entirely, or it came back without a
      // recognized selectedBroker - both mean we're falling back to the
      // in-memory default, so both must be reported like every other section.
      failedSections.push('Broker settings')
    }
  } catch {
    failedSections.push('Broker settings')
  }

  // Load LLM settings
  try {
    const llmRes = await getLlmSettings()
    const llmData = readData(llmRes)
    if (llmData) {
      // Secret fields intentionally stay blank; a blank input means "keep the configured secret".
      Object.assign(state.llmSettings, {
        llmBackend: parseLlmBackend(llmData['llm.backend']),
        llmBaseUrl: llmData['llm.base_url'] ?? state.llmSettings.llmBaseUrl,
        openaiBaseUrl: llmData['openai.base_url'] ?? state.llmSettings.openaiBaseUrl,
        openaiModel: llmData['openai.model'] ?? state.llmSettings.openaiModel,
        ollamaBaseUrl: llmData['ollama.base_url'] ?? state.llmSettings.ollamaBaseUrl,
        ollamaModel: llmData['ollama.model'] ?? state.llmSettings.ollamaModel,
        layaBaseUrl: llmData['laya.base_url'] ?? state.llmSettings.layaBaseUrl,
        layaModel: llmData['laya.model'] ?? state.llmSettings.layaModel,
        piAgentProvider: llmData['pi-agent.provider'] ?? state.llmSettings.piAgentProvider,
        piAgentModel: llmData['pi-agent.model'] ?? state.llmSettings.piAgentModel,
        llamacppModel: llmData['llamacpp.model'] ?? state.llmSettings.llamacppModel,
        mlxModel: llmData['mlx.model'] ?? state.llmSettings.mlxModel,
        pdfBaseUrl: llmData['llm.pdf.base_url'] ?? state.llmSettings.pdfBaseUrl,
        pdfModel: llmData['llm.pdf.model'] ?? state.llmSettings.pdfModel,
      })
    } else {
      failedSections.push('LLM settings')
    }
  } catch {
    failedSections.push('LLM settings')
  }

  // Load Discord settings
  try {
    const discordRes = await getDiscordSettings()
    const discordData = readData(discordRes)
    if (discordData) {
      state.discordSettings.webhookUrl =
        discordData['discord.webhook.url'] || state.discordSettings.webhookUrl
      state.discordSettings.enabled = discordData['discord.webhook.enabled'] === 'true'
    } else {
      failedSections.push('Discord settings')
    }
  } catch {
    failedSections.push('Discord settings')
  }

  // Load trading config
  try {
    const tradingRes = await getTradingSettings()
    const tradingData = readData(tradingRes)
    if (tradingData) {
      Object.assign(state.tradingConfig, parseTradingConfig(tradingData))
    } else {
      failedSections.push('Trading settings')
    }
  } catch {
    failedSections.push('Trading settings')
  }

  try {
    if (typeof getScanningSettings !== 'function') return { state, failedSections }
    const scanningRes = await getScanningSettings()
    const scanningData = readData(scanningRes)
    if (scanningData) Object.assign(state.scanningConfig, parseScanningConfig(scanningData))
    else failedSections.push('Scanning settings')
  } catch {
    failedSections.push('Scanning settings')
  }

  return { state, failedSections }
}

const state = reactive<SettingsState>(createDefaultState())
let loaded = false
let failedSections: string[] = []

export async function loadSettings(): Promise<string[]> {
  if (loaded) return []
  const { state: fresh, failedSections: failures } = await loadAll()
  applySettings(state, fresh)
  failedSections = failures
  loaded = failures.length === 0
  return failures
}

function createLlmPayload(): Record<string, string> {
  const settings: Record<string, string> = {
    'llm.base_url': state.llmSettings.llmBaseUrl,
    'llm.backend': state.llmSettings.llmBackend,
    'openai.base_url': state.llmSettings.openaiBaseUrl,
    'openai.model': state.llmSettings.openaiModel,
    'ollama.base_url': state.llmSettings.ollamaBaseUrl,
    'ollama.model': state.llmSettings.ollamaModel,
    'laya.base_url': state.llmSettings.layaBaseUrl,
    'laya.model': state.llmSettings.layaModel,
    'pi-agent.provider': state.llmSettings.piAgentProvider,
    'pi-agent.model': state.llmSettings.piAgentModel,
    'llamacpp.model': state.llmSettings.llamacppModel,
    'mlx.model': state.llmSettings.mlxModel,
    'llm.pdf.base_url': state.llmSettings.pdfBaseUrl,
    'llm.pdf.model': state.llmSettings.pdfModel,
  }

  const openaiApiKey = state.llmSettings.openaiApiKey.trim()
  const ollamaApiKey = state.llmSettings.ollamaApiKey.trim()
  if (openaiApiKey) settings['openai.api_key'] = openaiApiKey
  if (ollamaApiKey) settings['ollama.api_key'] = ollamaApiKey

  return settings
}

function clearSensitiveInputs() {
  state.llmSettings.openaiApiKey = ''
  state.llmSettings.ollamaApiKey = ''
}

export async function saveSettings(): Promise<boolean> {
  if (failedSections.length > 0) return false
  const body = {
    broker: state.selectedBroker,
    llm: createLlmPayload(),
    discord: {
      'discord.webhook.url': state.discordSettings.webhookUrl,
      'discord.webhook.enabled': String(state.discordSettings.enabled),
    },
    trading: {
      'trading.mode': state.tradingConfig.mode,
      'trading.max_position_size': String(state.tradingConfig.maxPositionSize),
      'trading.stop_loss': String(state.tradingConfig.stopLoss),
      'trading.take_profit': String(state.tradingConfig.takeProfit),
      'trading.allocation_per_position': String(state.tradingConfig.allocationPerPosition),
      'trading.initial_capital': String(state.tradingConfig.initialCapital),
    },
    scanning: {
      'candidate-scan.max-concurrent': String(state.scanningConfig.maxConcurrent),
    },
  }

  const res = await saveAllSettings(body)
  if (!readData(res)) {
    console.error('Failed to save settings')
    return false
  }
  clearSensitiveInputs()
  return true
}

// Keep individual save methods for toggle-on-change behavior
export async function saveLlmSettings(): Promise<boolean> {
  const res = await setLlmSettings(createLlmPayload())
  if (!readData(res)) {
    console.error('Failed to save LLM settings')
    return false
  }
  clearSensitiveInputs()
  return true
}

export async function saveDiscordSettings(): Promise<boolean> {
  const settings: Record<string, string> = {
    'discord.webhook.url': state.discordSettings.webhookUrl,
    'discord.webhook.enabled': String(state.discordSettings.enabled),
  }
  const res = await setDiscordSettings(settings)
  if (!readData(res)) {
    console.error('Failed to save Discord settings')
    return false
  }
  return true
}

export async function saveScanningConfig(): Promise<boolean> {
  const res = await setScanningSettings({
    'candidate-scan.max-concurrent': String(state.scanningConfig.maxConcurrent),
  })
  if (!readData(res)) {
    console.error('Failed to save scanning config')
    return false
  }
  return true
}

export async function saveTradingConfig(): Promise<boolean> {
  const settings: Record<string, string> = {
    'trading.mode': state.tradingConfig.mode,
    'trading.max_position_size': String(state.tradingConfig.maxPositionSize),
    'trading.stop_loss': String(state.tradingConfig.stopLoss),
    'trading.take_profit': String(state.tradingConfig.takeProfit),
    'trading.allocation_per_position': String(state.tradingConfig.allocationPerPosition),
    'trading.initial_capital': String(state.tradingConfig.initialCapital),
  }
  const res = await setTradingSettings(settings)
  if (!readData(res)) {
    console.error('Failed to save trading config')
    return false
  }
  return true
}

export function getSettings() {
  return state
}

export function resetSettings() {
  applySettings(state, createDefaultState())
  loaded = false
}

export const brokerLabels: Record<string, string> = {
  fyers: 'Fyers',
  upstox: 'Upstox',
  yahoo: 'Yahoo Finance',
  none: 'Paper Only',
}

export const brokerIcons: Record<string, string> = {
  fyers: 'FY',
  upstox: 'UX',
  yahoo: 'YF',
  none: 'PT',
}
