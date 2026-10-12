import { API_BASE_URL } from './config'
import { apiRequest } from './shared'
import type {
  CandidateScanLogEvent,
  CandidateScanResultPage,
  CandidateScanRun,
  CandidateScanSettings,
} from './types'

export function getCandidateScanSettings(): Promise<CandidateScanSettings> {
  return apiRequest<CandidateScanSettings>('/candidate-scans/settings', {
    responseContract: 'direct',
  })
}

export function updateCandidateScanSettings(
  settings: Partial<CandidateScanSettings>
): Promise<CandidateScanSettings> {
  return apiRequest<CandidateScanSettings>('/candidate-scans/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
    responseContract: 'direct',
  })
}

export function startCandidateScan(): Promise<CandidateScanRun> {
  return apiRequest<CandidateScanRun>('/candidate-scans', {
    method: 'POST',
    responseContract: 'direct',
  })
}

export function getCandidateScan(runId: string): Promise<CandidateScanRun> {
  return apiRequest<CandidateScanRun>(`/candidate-scans/${runId}`, {
    responseContract: 'direct',
  })
}

export function getCandidateScanResults(
  runId: string,
  offset = 0,
  limit = 10,
  symbol = '',
  signalType = ''
): Promise<CandidateScanResultPage> {
  const params = new URLSearchParams({ offset: String(offset), limit: String(limit) })
  if (symbol.trim()) params.set('symbol', symbol.trim())
  if (signalType) params.set('signalType', signalType)
  return apiRequest<CandidateScanResultPage>(
    `/candidate-scans/${runId}/results?${params.toString()}`,
    { responseContract: 'direct' }
  )
}

export function startCandidateAnalysis(
  runId: string,
  scope: 'NONE' | 'WATCHLIST' | 'QUALIFIED'
): Promise<CandidateScanRun> {
  return apiRequest<CandidateScanRun>(`/candidate-scans/${runId}/analysis`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scope }),
    responseContract: 'direct',
  })
}

export function cancelCandidateScan(runId: string): Promise<CandidateScanRun> {
  return apiRequest<CandidateScanRun>(`/candidate-scans/${runId}/cancel`, {
    method: 'POST',
    responseContract: 'direct',
  })
}

export function pauseCandidateScan(runId: string): Promise<CandidateScanRun> {
  return apiRequest<CandidateScanRun>(`/candidate-scans/${runId}/pause`, {
    method: 'POST',
    responseContract: 'direct',
  })
}

export function resumeCandidateScan(runId: string): Promise<CandidateScanRun> {
  return apiRequest<CandidateScanRun>(`/candidate-scans/${runId}/resume`, {
    method: 'POST',
    responseContract: 'direct',
  })
}

export function getCandidateScanHistory(): Promise<CandidateScanRun[]> {
  return apiRequest<CandidateScanRun[]>('/candidate-scans', {
    responseContract: 'direct',
  })
}

export function openCandidateScanStream(
  runId: string,
  onLog: (event: CandidateScanLogEvent) => void,
  onError?: () => void
): EventSource | null {
  if (typeof EventSource === 'undefined') return null
  const source = new EventSource(`${API_BASE_URL}/candidate-scans/${runId}/stream`)
  source.addEventListener('log', (event) => {
    try {
      onLog(JSON.parse((event as MessageEvent).data) as CandidateScanLogEvent)
    } catch {
      onError?.()
    }
  })
  source.onerror = () => onError?.()
  return source
}
