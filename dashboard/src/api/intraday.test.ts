import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { runIntradayPaperSession } from './intraday'

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('runIntradayPaperSession', () => {
  it('posts the one-session paper request and returns the linked ledger', async () => {
    const request = {
      symbol: 'TCS',
      sessionDate: '2026-10-09',
      strategyType: 'INTRADAY_ORB',
      riskConfig: {
        initialCapital: 10000,
        quantity: 10,
        maxTradesPerDay: 1,
        maxDailyLoss: 1000,
      },
    }
    const result = {
      runId: 'run-1',
      executionMode: 'PAPER_SIMULATION',
      status: 'COMPLETED',
      symbol: 'TCS',
      sessionDate: '2026-10-09',
      expectedBars: 25,
      receivedBars: 25,
      strategy: 'INTRADAY_ORB',
      killSwitchActive: false,
      maxConcurrentPositions: 1,
      openPositionsAtEnd: 0,
      liveBrokerCalls: 0,
      initialCapital: 10000,
      finalCapital: 9972.3197399,
      netPnl: -27.6802601,
      orders: [],
      positions: [],
      trades: [],
    }
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify(result), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      })
    )

    await expect(runIntradayPaperSession(request)).resolves.toEqual(result)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/intraday/paper/sessions',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(request),
      })
    )
  })
})
