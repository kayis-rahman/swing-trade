import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AppError } from '../errors/appError'
import type { IntradayPaperSessionResult } from '../api/intraday'

const apiMocks = vi.hoisted(() => ({ runIntradayPaperSession: vi.fn() }))
vi.mock('../api/intraday', () => apiMocks)

function completedSession(): IntradayPaperSessionResult {
  const entryTime = '2026-10-09T10:00:00+05:30[Asia/Kolkata]'
  const exitTime = '2026-10-09T10:00:00+05:30[Asia/Kolkata]'
  return {
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
    orders: [
      {
        orderId: 'run-1-order-1',
        symbol: 'TCS',
        side: 'BUY',
        filledAt: entryTime,
        fillPrice: 102.051,
        quantity: 10,
        status: 'FILLED',
        executionMode: 'PAPER_SIMULATION',
        fillSource: 'NEXT_BAR_OPEN',
      },
      {
        orderId: 'run-1-order-2',
        symbol: 'TCS',
        side: 'SELL',
        filledAt: exitTime,
        fillPrice: 104.03997399,
        quantity: 10,
        status: 'FILLED',
        executionMode: 'PAPER_SIMULATION',
        fillSource: 'TARGET_HIT',
      },
    ],
    positions: [
      {
        positionId: 'run-1-position-1',
        symbol: 'TCS',
        entryOrderId: 'run-1-order-1',
        exitOrderId: 'run-1-order-2',
        entryTime,
        exitTime,
        entryPrice: 102.051,
        exitPrice: 104.03997399,
        quantity: 10,
        status: 'CLOSED',
        costs: 47.57,
        realizedPnl: -27.6802601,
        exitReason: 'TARGET_HIT',
        carriedOvernight: false,
      },
    ],
    trades: [
      {
        positionId: 'run-1-position-1',
        entryOrderId: 'run-1-order-1',
        exitOrderId: 'run-1-order-2',
        strategy: 'INTRADAY_ORB',
        entryTime,
        exitTime,
        entryPrice: 102.051,
        exitPrice: 104.03997399,
        quantity: 10,
        costs: 47.57,
        pnl: -27.6802601,
        exitReason: 'TARGET_HIT',
      },
    ],
  }
}

function buttonByText(wrapper: VueWrapper, text: RegExp) {
  const button = wrapper.findAll('button').find((candidate) => text.test(candidate.text()))
  expect(button, `Expected button matching ${text}`).toBeDefined()
  return button!
}

async function mountView() {
  const View = (await import('./IntradayView.vue')).default
  return mount(View)
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

describe('IntradayView paper cockpit', () => {
  beforeEach(() => {
    apiMocks.runIntradayPaperSession.mockReset()
  })

  it('shows a neutral empty state before a session is run', async () => {
    const wrapper = await mountView()

    expect(wrapper.text()).toContain('No session loaded')
    expect(apiMocks.runIntradayPaperSession).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('loads the seeded completed session and shows net costs, position state, and linked audit IDs', async () => {
    apiMocks.runIntradayPaperSession.mockResolvedValue(completedSession())
    const wrapper = await mountView()

    await buttonByText(wrapper, /load seeded session/i).trigger('click')
    await flushPromises()

    expect(apiMocks.runIntradayPaperSession).toHaveBeenCalledWith({
      symbol: 'TCS',
      sessionDate: '2026-10-09',
      strategyType: 'INTRADAY_ORB',
      riskConfig: {
        initialCapital: 10_000,
        quantity: 10,
        maxTradesPerDay: 1,
        maxDailyLoss: 1_000,
      },
    })
    expect(wrapper.text()).toContain('COMPLETED')
    expect(wrapper.text()).toContain('25 / 25 bars')
    expect(wrapper.text()).toContain('-₹27.68')
    expect(wrapper.text()).toContain('₹47.57')
    expect(wrapper.text()).toContain('No open paper positions')
    expect(wrapper.text()).toContain('TARGET HIT')
    expect(wrapper.text()).toContain('P run-1-position-1')
    expect(wrapper.text()).toContain('E run-1-order-1')
    expect(wrapper.text()).toContain('X run-1-order-2')
    expect(wrapper.text()).toContain('No broker route')

    wrapper.unmount()
  })

  it('keeps incomplete data explicit and does not show a paper ledger', async () => {
    apiMocks.runIntradayPaperSession.mockResolvedValue({
      ...completedSession(),
      status: 'INCOMPLETE_DATA',
      receivedBars: 24,
      orders: [],
      positions: [],
      trades: [],
      openPositionsAtEnd: 0,
      maxConcurrentPositions: 0,
    })
    const wrapper = await mountView()

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('Incomplete session — no paper ledger created')
    expect(wrapper.text()).toContain('24 / 25 bars')
    expect(wrapper.text()).toContain('No simulated orders.')
    expect(wrapper.text()).not.toContain('TARGET HIT')

    wrapper.unmount()
  })

  it('renders loading state while the replay is pending and disables another submission', async () => {
    const pending = deferred<IntradayPaperSessionResult>()
    apiMocks.runIntradayPaperSession.mockReturnValueOnce(pending.promise)
    const wrapper = await mountView()

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('Replaying the complete NSE session…')
    expect(buttonByText(wrapper, /replaying/i).attributes('disabled')).toBeDefined()

    pending.resolve(completedSession())
    await flushPromises()
    expect(wrapper.text()).toContain('COMPLETED')

    wrapper.unmount()
  })

  it('shows a retryable API error and recovers on retry', async () => {
    apiMocks.runIntradayPaperSession
      .mockRejectedValueOnce(
        new AppError({ kind: 'network', message: 'Backend unavailable', retryable: true })
      )
      .mockResolvedValueOnce(completedSession())
    const wrapper = await mountView()

    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('Can’t reach the backend')

    await buttonByText(wrapper, /retry/i).trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('COMPLETED')
    expect(apiMocks.runIntradayPaperSession).toHaveBeenCalledTimes(2)

    wrapper.unmount()
  })
})
