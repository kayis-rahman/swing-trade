import type { JobRunStageResponse, JobStageStrategyResult } from '../api/types'

export interface MatrixVariant {
  variantId: string
  version: number
}

export interface StrategyMatrixModel {
  symbols: string[]
  variants: MatrixVariant[]
  cell: (symbol: string, variantId: string) => JobStageStrategyResult | undefined
}

export type MatrixCellKind = 'signal' | 'no-signal' | 'skipped' | 'error' | 'none'

const NO_SIGNAL = new Set(['', 'HOLD', 'NONE', 'NO_SIGNAL'])

/** Pivots SIGNAL-stage `details.strategies` into a symbol x variant lookup. */
export function buildStrategyMatrix(rows: JobRunStageResponse[]): StrategyMatrixModel {
  const cells = new Map<string, JobStageStrategyResult>()
  const symbols = new Set<string>()
  const variants = new Map<string, MatrixVariant>()

  for (const row of rows) {
    if (row.stageName !== 'SIGNAL') continue
    for (const result of row.details?.strategies ?? []) {
      symbols.add(row.symbol)
      cells.set(`${row.symbol}::${result.variantId}`, result)
      const known = variants.get(result.variantId)
      if (!known || result.version > known.version) {
        variants.set(result.variantId, { variantId: result.variantId, version: result.version })
      }
    }
  }

  return {
    symbols: [...symbols].sort(),
    variants: [...variants.values()].sort((a, b) => a.variantId.localeCompare(b.variantId)),
    cell: (symbol, variantId) => cells.get(`${symbol}::${variantId}`),
  }
}

export function matrixCellKind(cell: JobStageStrategyResult | undefined): MatrixCellKind {
  if (!cell) return 'none'
  if (cell.outcome === 'ERROR') return 'error'
  if (cell.outcome === 'SKIPPED') return 'skipped'
  if (cell.signal === true) return 'signal'
  if (cell.signal === false) return 'no-signal'
  const signal = (cell.signal ?? '').toUpperCase()
  return NO_SIGNAL.has(signal) ? 'no-signal' : 'signal'
}

export interface RunWarnings {
  degraded: number
  skipped: number
  messages: string[]
}

export function collectRunWarnings(rows: JobRunStageResponse[]): RunWarnings {
  const messages: string[] = []
  let degraded = 0
  let skipped = 0
  for (const row of rows) {
    if (row.status === 'DEGRADED') degraded += 1
    if (row.status === 'SKIPPED') skipped += 1
    const label = `${row.symbol} ${row.stageName}`
    if (row.status === 'DEGRADED' && row.details?.reason) {
      messages.push(`${label}: ${row.details.reason}`)
    }
    for (const warning of row.details?.warnings ?? []) messages.push(`${label}: ${warning}`)
  }
  return { degraded, skipped, messages }
}
