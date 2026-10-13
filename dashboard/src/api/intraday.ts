import { apiRequest } from './shared'

export interface IntradayPaperSessionRequest {
  symbol: string
  sessionDate: string
  asOfBarTime?: string
  strategyType: string
  riskConfig?: {
    initialCapital: number
    quantity: number
    maxTradesPerDay: number
    maxDailyLoss: number
  }
}

export interface IntradayPaperOrder {
  orderId: string
  symbol: string
  side: 'BUY' | 'SELL'
  filledAt: string
  fillPrice: number
  quantity: number
  status: string
  executionMode: string
  fillSource: string
}

export interface IntradayPaperPosition {
  positionId: string
  symbol: string
  entryOrderId: string
  exitOrderId: string | null
  entryTime: string
  exitTime: string | null
  entryPrice: number
  exitPrice: number | null
  currentPrice: number
  quantity: number
  status: 'OPEN' | 'CLOSED'
  costs: number
  realizedPnl: number
  unrealizedPnl: number
  exitReason: string
  carriedOvernight: boolean
}

export interface IntradayPaperTrade {
  positionId: string
  entryOrderId: string
  exitOrderId: string
  strategy: string
  entryTime: string
  exitTime: string
  entryPrice: number
  exitPrice: number
  quantity: number
  costs: number
  pnl: number
  exitReason: string
}

export interface IntradayPaperSessionResult {
  runId: string
  executionMode: string
  status: 'COMPLETED' | 'INCOMPLETE_DATA' | 'BLOCKED_KILL_SWITCH'
  symbol: string
  sessionDate: string
  expectedBars: number
  receivedBars: number
  strategy: string
  killSwitchActive: boolean
  maxConcurrentPositions: number
  openPositionsAtEnd: number
  liveBrokerCalls: number
  initialCapital: number
  finalCapital: number
  netPnl: number
  orders: IntradayPaperOrder[]
  positions: IntradayPaperPosition[]
  trades: IntradayPaperTrade[]
}

export function getIntradayPaperSession(runId: string): Promise<IntradayPaperSessionResult> {
  return apiRequest<IntradayPaperSessionResult>(
    `/intraday/paper/sessions/${encodeURIComponent(runId)}`,
    { responseContract: 'direct' }
  )
}

export function runIntradayPaperSession(
  request: IntradayPaperSessionRequest
): Promise<IntradayPaperSessionResult> {
  return apiRequest<IntradayPaperSessionResult>('/intraday/paper/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
    responseContract: 'direct',
  })
}

export interface IntradayPaperScanRequest {
  sessionDate: string
  asOfBarTime: string
  riskConfig?: IntradayPaperSessionRequest['riskConfig']
}

export interface IntradayPaperScanRow {
  symbol: string
  strategy: string
  status: 'CANDIDATE' | 'NO_OPEN_SETUP' | 'INCOMPLETE_DATA' | 'BLOCKED_KILL_SWITCH'
  expectedBars: number
  receivedBars: number
  completedTrades: number
  netPnl: number
  openEntryPrice: number | null
  openEntryTime: string | null
}

export interface IntradayPaperScanResult {
  mode: 'PAPER_SCAN_AS_OF'
  sessionDate: string
  asOfBarTime: string
  universe: 'ACTIVE_NSE_WATCHLIST'
  watchlistSize: number
  scannedSymbols: number
  candidateCount: number
  liveBrokerCalls: 0
  excludedSymbols: string[]
  results: IntradayPaperScanRow[]
}

export function scanIntradayPaperWatchlist(
  request: IntradayPaperScanRequest
): Promise<IntradayPaperScanResult> {
  return apiRequest<IntradayPaperScanResult>('/intraday/paper/scans', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
    responseContract: 'direct',
  })
}
