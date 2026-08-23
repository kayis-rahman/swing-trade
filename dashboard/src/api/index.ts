// Re-export all API functions for backward compatibility
// New code should import from domain-specific modules directly.

// Shared infrastructure
export { apiRequest, apiSseEvents, toNum } from './shared'
export type { ApiRequestOptions, ApiSseEventOptions, ResponseContract } from './shared'
export { AppError, asAppError, formatAppError, isAppError } from '../errors/appError'
export {
  CancelledError,
  MalformedResponseError,
  NetworkError,
  RuntimeAppError,
  TimeoutError,
  isErrorKind,
} from '../errors/errorClasses'
export type {
  AppErrorKind,
  AppErrorOperation,
  AppErrorOptions,
  FormatAppErrorOptions,
  FormattedAppError,
} from '../errors/appError'

// Health
export { checkHealth, getHealthStatus } from './health'

// Signals
export {
  getSignals,
  getGateEffectiveness,
  generateAllSignals,
  generateAllSignalsStream,
  clearAllSignals,
  clearSignalsForSymbol,
  getSignalsByType,
  getLatestSignalForSymbol,
  getHighConfidenceSignals,
  generatePriceActionSignal,
  triggerScan,
} from './signals'
export type {
  GateEffectivenessReport,
  GateEffectivenessSummary,
  SignalGenerationProgress,
  SignalStreamOptions,
} from './signals'

// Positions
export {
  getPortfolioSummary,
  getMarketOverview,
  getRiskSummary,
  getPositions,
  getClosedPositions,
  getTradeHistory,
  closePosition,
  executeTrade,
  getEquityCurve,
} from './positions'
export type { ExecuteTradeParams } from './positions'

// Watchlist
export {
  getWatchlist,
  addToWatchlist,
  removeFromWatchlist,
  toggleWatchlistActive,
} from './watchlist'

// Ingestion
export { getIngestionStatus, triggerDataPull, getPullProgress, cancelDataPull } from './ingestion'

// Settings
export {
  getSettings,
  setBroker,
  getLlmSettings,
  setLlmSettings,
  getDiscordSettings,
  setDiscordSettings,
  testDiscordWebhook,
  testPiConnection,
  testOpenAiConnection,
  testOllamaConnection,
  testPiAgentConnection,
  startPiServer,
  stopPiServer,
  getPiServerStatus,
  startMlxServer,
  stopMlxServer,
  getMlxServerStatus,
  testMlxConnection,
  getGpuHubSettings,
  setGpuHubSettings,
  getTradingSettings,
  setTradingSettings,
  getScanningSettings,
  setScanningSettings,
  saveAllSettings,
} from './settings'

// Sentiment
export {
  getSentimentLatest,
  getSentimentHistory,
  getAccuracyStats,
  getAccuracySummary,
  getAccuracyByWindow,
  getAccuracyByRegime,
  getAccuracyBySymbol,
  getCalibration,
  getRollingIC,
  getSignalVolume,
  getECE,
  triggerSentimentAnalysis,
  getLatestNews,
} from './sentiment'

// Backtest
export { runBacktest, runBacktestAll, listBacktestReports, getBacktestReport } from './backtest'

// Strategies
export {
  getStrategies,
  getStrategyTypes,
  changeStrategyMode,
  saveStrategyConfig,
} from './strategies'

// Analysis
export { getCompositeAnalysis, runFullAnalysis } from './analysis'
export type { AnalysisStreamOptions } from './analysis'

// Job
export { startJobRun, getJobRunProgress, getJobRunSummary, listJobRuns, cancelJobRun } from './job'

// Holidays
export { getTodayHolidayStatus, getUpcomingHolidays } from './holidays'

// Fyers
export { getFyersLoginUrl, fyersAuthCode, getFyersStatus, fyersLogout } from './fyers'

// Backfill
export { backfillSymbol } from './backfill'

// Types
export * from './types'
