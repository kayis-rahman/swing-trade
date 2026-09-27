import { describe, expect, it } from 'vitest'
import type { JobRunStageResponse } from '../api/types'
import { buildStrategyMatrix, collectRunWarnings, matrixCellKind } from './runMatrix'

function stage(over: Partial<JobRunStageResponse>): JobRunStageResponse {
  return {
    symbol: 'TCS',
    stageName: 'SIGNAL',
    status: 'COMPLETED',
    startedAt: '2026-09-21T10:00:00',
    completedAt: null,
    durationMs: 5,
    errorMessage: null,
    resultSummary: null,
    details: null,
    ...over,
  }
}

const signalRows = [
  stage({
    symbol: 'TCS',
    details: {
      strategies: [
        { variantId: 'B', version: 1, outcome: 'EVALUATED', signal: 'BUY', score: 0.8 },
        { variantId: 'A', version: 2, outcome: 'EVALUATED', signal: null },
      ],
    },
  }),
  stage({
    symbol: 'INFY',
    details: {
      strategies: [
        { variantId: 'A', version: 2, outcome: 'SKIPPED', reason: 'unknown type' },
        { variantId: 'B', version: 1, outcome: 'ERROR', reason: 'boom' },
      ],
    },
  }),
  stage({ symbol: 'INFY', stageName: 'NEWS', details: { warnings: ['x'] } }),
]

describe('buildStrategyMatrix', () => {
  it('pivots SIGNAL stage details into sorted symbols x variants', () => {
    const m = buildStrategyMatrix(signalRows)
    expect(m.symbols).toEqual(['INFY', 'TCS'])
    expect(m.variants).toEqual([
      { variantId: 'A', version: 2 },
      { variantId: 'B', version: 1 },
    ])
    expect(m.cell('TCS', 'B')?.signal).toBe('BUY')
    expect(m.cell('INFY', 'A')?.reason).toBe('unknown type')
    expect(m.cell('INFY', 'ZZ')).toBeUndefined()
  })

  it('is empty when no SIGNAL stage carries strategies', () => {
    expect(buildStrategyMatrix([stage({ stageName: 'NEWS' })]).variants).toEqual([])
  })
})

describe('matrixCellKind', () => {
  it('distinguishes signal, no-signal, skipped and error', () => {
    const m = buildStrategyMatrix(signalRows)
    expect(matrixCellKind(m.cell('TCS', 'B'))).toBe('signal')
    expect(matrixCellKind(m.cell('TCS', 'A'))).toBe('no-signal')
    expect(matrixCellKind(m.cell('INFY', 'A'))).toBe('skipped')
    expect(matrixCellKind(m.cell('INFY', 'B'))).toBe('error')
    expect(matrixCellKind(undefined)).toBe('none')
  })

  it('treats a HOLD signal as no signal', () => {
    expect(
      matrixCellKind({ variantId: 'A', version: 1, outcome: 'EVALUATED', signal: 'HOLD' })
    ).toBe('no-signal')
  })

  it('supports the boolean signal flag returned by the orchestrator API', () => {
    expect(matrixCellKind({ variantId: 'A', version: 1, outcome: 'EVALUATED', signal: true })).toBe(
      'signal'
    )
    expect(
      matrixCellKind({ variantId: 'A', version: 1, outcome: 'EVALUATED', signal: false })
    ).toBe('no-signal')
  })
})

describe('collectRunWarnings', () => {
  it('collects degraded stages and detail warnings', () => {
    const w = collectRunWarnings([
      stage({
        symbol: 'A',
        stageName: 'SENTIMENT',
        status: 'DEGRADED',
        details: { reason: 'ollama down', warnings: ['slow'] },
      }),
      stage({ symbol: 'B', stageName: 'SIGNAL', status: 'SKIPPED' }),
    ])
    expect(w.degraded).toBe(1)
    expect(w.skipped).toBe(1)
    expect(w.messages).toEqual(['A SENTIMENT: ollama down', 'A SENTIMENT: slow'])
  })
})
