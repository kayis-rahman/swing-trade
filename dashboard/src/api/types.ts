export interface PortfolioSummary {
  totalValue: number
  totalPnl: number
  totalPnlPercent: number
  winRate: number
  totalTrades: number
  averageWin: number
  averageLoss: number
  profitFactor: number
  maxDrawdown: number
  sharpeRatio: number
}

export interface MarketOverview {
  totalPositions: number
  openPositions: number
  todayPnl: number
  todayPnlPercent: number
  todayPnLSource?: 'DAILY' | 'FALLBACK_TOTAL'
}

export interface Position {
  id: string
  brokerType?: string
  symbol: string
  entryPrice: number
  currentPrice: number
  quantity: number
  status: 'OPEN' | 'CLOSED' | 'STOPPED' | 'TARGET_HIT'
  pnl: number
  pnlPercent: number
  entryDate: string
  exitDate?: string
  stopLoss?: number
  target?: number
  entryReason?: string
  positionId?: string
  brokerPositionId?: string
  exchange?: string
  direction?: 'LONG' | 'SHORT'
  averagePrice?: number
  realizedPnL?: number
  marginUtilized?: number
  totalValue?: number
  entryTime?: string
  exitTime?: string
  exitReason?: string
}

export interface Signal {
  id: string
  symbol: string
  direction: 'BUY' | 'SELL' | 'HOLD'
  confidence: number
  reason: string
  entryPrice: number
  stopLoss: number
  target: number
  riskReward: number
  timestamp: string
  status: 'PENDING' | 'ACTIVE' | 'EXECUTED' | 'EXPIRED'
  strategy?: string
  // Backend gap: SignalController does not currently expose strategy_version in its DTO
  // (plan §4.2 migration adds the column, but no controller/DTO surfaces it yet), so this
  // is always undefined until that endpoint is extended.
  strategyVersion?: number
  indicators?: string[]
  sentimentScore?: string
  sentimentReasoning?: string
}

export interface EquityPoint {
  date: string
  value: number
}

export interface HealthStatus {
  status: string
  components: Record<string, HealthComponent>
}

export interface RiskSummary {
  totalExposure: number
  availableCapital: number
  usedCapital: number
  stopLossExposure: number
  numberOfPositions: number
  sectorExposure?: Record<string, number>
}

export interface HealthComponent {
  name: string
  status: string
  description: string
  details?: Record<string, unknown>
}

export interface CandidateScanRun {
  runId: string
  status: 'RUNNING' | 'PAUSED' | 'COMPLETED' | 'CANCELLED'
  totalSymbols: number
  scanScope?: 'WATCHLIST' | 'NSE_BROAD' | 'ACTIVE_STOCKS'
  scanTrigger?: 'MANUAL' | 'SCHEDULED'
  completedSymbols: number
  failedSymbols: number
  qualifiedSymbols: number
  startedAt: string
  completedAt?: string
  errorMessage?: string
  orchestrationStatus?: 'PENDING' | 'NOT_REQUIRED' | 'STARTED' | 'FAILED'
  orchestrationJobRunId?: string
  orchestrationError?: string
}

export interface CandidateScanResult {
  runId: string
  symbol: string
  dataStatus: 'READY' | 'INSUFFICIENT' | 'ERROR'
  candleCount: number
  signalType?: 'BUY' | 'SELL' | 'HOLD'
  strategyBuyCount?: number
  strategyEvaluationCount?: number
  strategyOutcomes?: CandidateStrategyOutcome[]
  totalTrades?: number
  winRate?: number
  totalReturn?: number
  maxDrawdownPct?: number
  qualified: boolean
  activated: boolean
  reason?: string
  errorMessage?: string
  createdAt: string
  sourceOutcome?:
    | 'EXISTING_HISTORY'
    | 'DATA_RECEIVED'
    | 'NO_USABLE_DATA'
    | 'INVALID_ROWS_REJECTED'
    | 'TRANSIENT_SOURCE_FAILURE'
  invalidRows?: number
  firstAvailableDate?: string
  lastAvailableDate?: string
  retryAfter?: string
  oosStartDate?: string
  oosEndDate?: string
  oosTotalTrades?: number
  oosWinRate?: number
  oosTotalReturn?: number
}

export interface CandidateStrategyOutcome {
  variantId: string
  signalType: 'BUY' | 'SELL' | 'HOLD' | 'SKIPPED'
  score?: number
  detail?: string
  backtestTotalTrades?: number
  backtestWinRate?: number
  backtestTotalReturn?: number
  oosTotalTrades?: number
  oosWinRate?: number
  oosTotalReturn?: number
  performanceStatus?: 'EVALUATED' | 'SKIPPED' | 'ERROR' | 'NOT_RUN'
}

export interface CandidateScanSettings {
  'candidate-scan.min-win-rate': string
  'candidate-scan.min-total-return': string
  'candidate-scan.max-concurrent': string
  'candidate-scan.backfill-years': string
  'candidate-scan.min-trades': string
  'candidate-scan.out-of-sample-days': string
  'candidate-scan.min-strategy-buys': string
}

export interface CandidateScanResultPage {
  items: CandidateScanResult[]
  total: number
  offset: number
  limit: number
}

export interface CandidateScanLogEvent {
  eventType:
    | 'RUN_SNAPSHOT'
    | 'RUN_STARTED'
    | 'RUN_PAUSED'
    | 'RUN_RESUMED'
    | 'STAGE_STARTED'
    | 'STAGE_COMPLETED'
    | 'SYMBOL_STARTED'
    | 'SYMBOL_COMPLETED'
    | 'SYMBOL_FAILED'
    | 'RUN_COMPLETED'
    | 'RUN_CANCELLED'
  runId: string
  symbol?: string | null
  level: 'INFO' | 'SUCCESS' | 'WARN' | 'ERROR'
  message: string
  completedSymbols: number
  totalSymbols: number
  failedSymbols: number
  qualifiedSymbols: number
  timestamp: string
}

export interface FyersStatus {
  connected: boolean
  clientId: string
}

export interface FyersLoginUrl {
  url: string
  message: string
}

export type BrokerType = 'fyers' | 'upstox' | 'yahoo' | 'none'

// ---------------------------------------------------------------------------
// Watchlist
// ---------------------------------------------------------------------------

export interface WatchlistEntry {
  id?: number
  symbol: string
  name: string
  exchange: string
  isActive: boolean
  addedAt?: string
  lastSyncedAt?: string
  candleCount?: number
}

// ---------------------------------------------------------------------------
// Data Ingestion Status
// ---------------------------------------------------------------------------

export interface IngestionStatus {
  symbol: string
  name: string
  exchange: string
  isActive: boolean
  candleCount: number
  lastCandleDate: string | null
  earliestCandleDate: string | null
  lastSyncedAt: string | null
  hasData: boolean
  dataQuality: string
}

// ---------------------------------------------------------------------------
// Data Pull Progress
// ---------------------------------------------------------------------------

export interface PullProgress {
  pullId: string
  status: string
  total: number
  completed: number
  failed: number
  currentSymbol: string
  percentComplete: number
}

// ---------------------------------------------------------------------------
// Backtest
// ---------------------------------------------------------------------------

export interface BacktestTrade {
  symbol: string
  entryDate: string
  exitDate: string
  entryPrice: number
  exitPrice: number
  stopLoss: number
  target: number
  quantity: number
  exitReason: string
  pnl: number
  pnlPct: number
  holdingDays: number
}

export interface BacktestResult {
  symbol: string
  totalTrades: number
  winningTrades: number
  losingTrades: number
  winRate: number
  avgGainPct: number
  avgLossPct: number
  maxDrawdownPct: number
  sharpeRatio: number
  totalReturn: number
  expectancy: number
  trades: BacktestTrade[]
}

export interface BacktestReportSummary {
  generatedAt: string
  symbolsBacktested: number
  top10ByWinRate: BacktestResult[]
  top10ByTotalReturn: BacktestResult[]
  overallWinRate: number
  overallSharpeRatio: number
  results: BacktestResult[]
}

// ---------------------------------------------------------------------------
// Sentiment
// ---------------------------------------------------------------------------

export interface SentimentResult {
  id: number
  symbol: string
  date: string
  score: 'POSITIVE' | 'NEUTRAL' | 'NEGATIVE'
  summary: string
  rawContent: string
  confidence: number
  analyzedAt: string
  redFlags: string[]
  catalysts: string[]
}

export interface SentimentAccuracyStats {
  total: number
  correct: number
  accuracy_pct: number
  by_sentiment: Record<string, number>
  by_symbol: Record<string, number>
}

export interface AccuracySummary {
  total: number
  correct: number
  accuracy_pct: number
  directional_accuracy: number
  avg_confidence: number
  by_sentiment: Record<string, number>
  by_symbol: Record<string, number>
}

export interface AccuracyByWindow {
  window: string
  total: number
  accuracy: number
  avg_return: number
}

export interface AccuracyByRegime {
  regime: string
  total: number
  accuracy: number
  avg_confidence: number
}

export interface AccuracyBySymbol {
  symbol: string
  total_signals: number
  accuracy: number
  avg_confidence: number
}

export interface CalibrationData {
  confidence_bin: number
  predicted_confidence: number
  actual_accuracy: number
  error: number
  count: number
}

export interface RollingIC {
  date: string
  spearman_ic: number
}

export interface SignalVolumeStats {
  today_count: number
  seven_day_count: number
  seven_day_avg: number
  thirty_day_avg: number
}

export interface ECEStats {
  ece: number
  bins: number
}

export interface NewsArticle {
  title: string
  link: string
  description: string
  publishedDate: string
  source: string
  rawContent: string
}

export interface CompositeAnalysis {
  symbol: string
  date: string
  compositeScore: number
  compositeSignal: 'BUY' | 'SELL' | 'HOLD'
  compositeConfidence: number
  sources: Array<{ name: string; score: number; weight: number; description: string }>
  news: {
    score: number
    summary: string
    catalysts: string[]
    redFlags: string[]
    articleCount: number
  }
  technical: {
    score: number
    signal: string
    confidence: number
    indicators: string[]
  }
  fundamentals: {
    score: number
    factors: string[]
  }
  backtest: {
    totalTrades: number
    winRate: number
    profitFactor: number
    maxDrawdown: number
    totalReturn: number
    expectancy: number
    hasEnoughData: boolean
  }
  reasoning: string
  synthesis?: SynthesisResult
}

export interface AnalysisProgress {
  stageNumber: number
  stageName: string
  status: 'running' | 'completed' | 'skipped' | 'error'
  message: string
  timestamp: string
  details?: StageDetails
  _eventType?: 'progress'
}

export interface StageDetails {
  type: string
  payload: Record<string, unknown>
}

export interface SynthesisResult {
  narrative: string
  recommendation: string
  confidence: number
  keyDrivers: string[]
  bullishFactors: string[]
  bearishFactors: string[]
  success: boolean
}

export interface FullAnalysisResult {
  composite: CompositeAnalysis | null
  progress: AnalysisProgress[]
  durationMs: number
  symbol: string
  _eventType?: 'complete'
}

// ---------------------------------------------------------------------------
// NSE Holidays
// ---------------------------------------------------------------------------

export interface NseHoliday {
  date: string
  occasion: string
  type: string
}

export interface HolidayListResponse {
  count: number
  holidays: NseHoliday[]
}

export interface TodayHolidayStatus {
  date: string
  marketClosed: boolean
  reason: Record<string, string>
}

// ---------------------------------------------------------------------------
// Strategy configuration
// ---------------------------------------------------------------------------

export type StrategyMode = 'OFF' | 'BACKTEST_ONLY' | 'SHADOW' | 'CHAMPION'

export interface StrategyConfig {
  id: number | null
  variantId: string
  version: number
  strategyType: string
  params: Record<string, unknown>
  overlays: Record<string, unknown>
  paramsHash: string
  mode: StrategyMode
  paperCapital: number
  current: boolean
  notes: string | null
  createdAt: string
}

export type PromotionEligibilityStatus = 'ELIGIBLE' | 'NOT_ELIGIBLE' | 'INSUFFICIENT_SAMPLE'

export interface PromotionConditionResult {
  name: string
  met: boolean
  actualValue: string
  threshold: string
  note: string | null
}

export interface PromotionEligibilityResponse {
  challengerVariantId: string
  championVariantId: string
  status: PromotionEligibilityStatus
  conditions: PromotionConditionResult[]
  notes: string[]
  dataLimitations: string[]
}

// ---------------------------------------------------------------------------
// Job Orchestrator
// ---------------------------------------------------------------------------

export type JobRunStatus =
  'RUNNING' | 'COMPLETED' | 'COMPLETED_WITH_WARNINGS' | 'FAILED' | 'CANCELLED'

export type JobStageStatus =
  'PENDING' | 'RUNNING' | 'COMPLETED' | 'DEGRADED' | 'SKIPPED' | 'ERROR' | 'CANCELLED'

export type StrategyOutcome = 'EVALUATED' | 'SKIPPED' | 'ERROR'

export interface JobStageStrategyResult {
  variantId: string
  version: number
  outcome: StrategyOutcome
  reason?: string | null
  score?: number | null
  /** Configured signal stage currently returns whether a signal was produced. */
  signal?: string | boolean | null
}

export interface JobStageDetails {
  source?: 'LLM' | 'KEYWORD_FALLBACK' | null
  reason?: string | null
  warnings?: string[] | null
  /** Only present on the SIGNAL stage. */
  strategies?: JobStageStrategyResult[] | null
}

export interface StartJobRunRequest {
  symbols?: string[]
  variantIds?: string[]
  stages?: string[]
  skipLlm?: boolean
  dryRun?: boolean
}

export interface JobRunResponse {
  runId: string
  triggerType: 'MANUAL' | 'SCHEDULED'
  status: JobRunStatus
  startedAt: string
  completedAt: string | null
  symbolsCount: number
  completedCount: number
  failedCount: number
  errorMessage: string | null
}

export interface JobRunStageResponse {
  symbol: string
  stageName:
    'DATA_FETCH' | 'NEWS' | 'SENTIMENT' | 'LLM_ANALYSIS' | 'SIGNAL' | 'BACKTEST' | 'PAPER_TRADE'
  status: JobStageStatus
  startedAt: string
  completedAt: string | null
  durationMs: number | null
  errorMessage: string | null
  resultSummary: string | null
  details?: JobStageDetails | null
}

export interface JobRunProgressResponse {
  runId: string
  status: string
  totalSymbols: number
  completedSymbols: number
  failedSymbols: number
  startedAt?: string
  completedAt?: string | null
  stages: JobRunStageResponse[]
}

export interface JobRunSummaryResponse {
  runId: string
  status: string
  totalSymbols: number
  completedSymbols: number
  failedSymbols: number
  totalDurationMs: number
  stageStats: Record<
    string,
    { total: number; completed: number; degraded?: number; errors: number; totalDurationMs: number }
  >
  degradedStages?: number
  skippedStrategies?: number
  degradedStageBreakdown?: Array<{ stage: string; reason: string; count: number }>
  skippedStrategyBreakdown?: Array<{
    variantId: string
    outcome: string
    reason: string
    symbols: number
  }>
  symbolDetails: Array<{
    symbol: string
    stageStatuses: Record<string, string>
    latestSignal: string | null
    sentimentScore: string | null
    backtestWinRate: number
    backtestReturn: number
    tradesExecuted: number
  }>
}

// ---------------------------------------------------------------------------
// Strategy types (params schema)
// ---------------------------------------------------------------------------

export interface StrategyParamSpec {
  name: string
  type: string
  min?: number | null
  max?: number | null
  defaultValue?: unknown
  description?: string | null
  group?: string | null
}

export interface StrategyTypeInfo {
  type: string
  paramSchema: { params: StrategyParamSpec[] }
  warmupBars: number
  requiredIndicators: string[]
}

export interface StrategyConfigRequest {
  variantId: string
  strategyType: string
  params: Record<string, unknown>
  overlays?: Record<string, unknown>
  mode: StrategyMode
  paperCapital: number
  notes?: string | null
}
