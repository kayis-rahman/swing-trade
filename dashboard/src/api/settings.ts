import { apiRequest } from './shared'

export async function getSettings(): Promise<{ selectedBroker: string }> {
  return apiRequest<{ selectedBroker: string }>('/settings', { responseContract: 'envelope' })
}

export async function setBroker(broker: string): Promise<{ selectedBroker: string }> {
  return apiRequest<{ selectedBroker: string }>('/settings/broker', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ broker }),
    responseContract: 'envelope',
  })
}

export async function getLlmSettings(): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/llm', { responseContract: 'envelope' })
}

export async function setLlmSettings(
  settings: Record<string, string>
): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/llm', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
    responseContract: 'envelope',
  })
}

export async function getDiscordSettings(): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/discord', { responseContract: 'envelope' })
}

export async function setDiscordSettings(
  settings: Record<string, string>
): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/discord', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
    responseContract: 'envelope',
  })
}

export async function testDiscordWebhook(): Promise<{ success: boolean }> {
  return apiRequest<{ success: boolean }>('/settings/test/discord', {
    method: 'POST',
    responseContract: 'envelope',
  })
}

export async function testPiConnection(): Promise<{ success: boolean; message: string }> {
  return apiRequest<{ success: boolean; message: string }>('/settings/test/pi', {
    method: 'POST',
    responseContract: 'envelope',
  })
}

export async function testOpenAiConnection(): Promise<{ success: boolean; message: string }> {
  return apiRequest<{ success: boolean; message: string }>('/settings/test/openai', {
    method: 'POST',
    responseContract: 'envelope',
  })
}

export async function testOllamaConnection(): Promise<{ success: boolean; message: string }> {
  return apiRequest<{ success: boolean; message: string }>('/settings/test/ollama', {
    method: 'POST',
    timeoutMs: 130_000,
    responseContract: 'envelope',
  })
}

export async function testPiAgentConnection(): Promise<{ success: boolean; message: string }> {
  return apiRequest<{ success: boolean; message: string }>('/settings/test/pi-agent', {
    method: 'POST',
    timeoutMs: 200_000,
    responseContract: 'envelope',
  })
}

export async function getGpuHubSettings(): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/gpuhub', { responseContract: 'envelope' })
}

export async function setGpuHubSettings(
  settings: Record<string, string>
): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/gpuhub', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
    responseContract: 'envelope',
  })
}

export async function startPiServer(): Promise<{
  success: boolean
  running: boolean
  message: string
}> {
  return apiRequest<{ success: boolean; running: boolean; message: string }>('/settings/pi/start', {
    method: 'POST',
    responseContract: 'envelope',
  })
}

export async function stopPiServer(): Promise<{
  success: boolean
  running: boolean
  message: string
}> {
  return apiRequest<{ success: boolean; running: boolean; message: string }>('/settings/pi/stop', {
    method: 'POST',
    responseContract: 'envelope',
  })
}

export async function getPiServerStatus(): Promise<{
  success: boolean
  running: boolean
  message: string
}> {
  return apiRequest<{ success: boolean; running: boolean; message: string }>(
    '/settings/pi/status',
    {
      responseContract: 'envelope',
    }
  )
}

export async function getTradingSettings(): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/trading', { responseContract: 'envelope' })
}

export async function setTradingSettings(
  settings: Record<string, string>
): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/trading', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
    responseContract: 'envelope',
  })
}

export async function getScanningSettings(): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/scanning', { responseContract: 'envelope' })
}

export async function setScanningSettings(
  settings: Record<string, string>
): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/scanning', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
    responseContract: 'envelope',
  })
}

export async function saveAllSettings(body: {
  broker?: string
  llm?: Record<string, string>
  discord?: Record<string, string>
  trading?: Record<string, string>
  scanning?: Record<string, string>
}): Promise<Record<string, string>> {
  return apiRequest<Record<string, string>>('/settings/save', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    responseContract: 'envelope',
  })
}
