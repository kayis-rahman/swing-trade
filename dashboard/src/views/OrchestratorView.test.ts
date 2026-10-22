import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { JobRunProgressResponse, JobRunResponse } from '../api/types'
import OrchestratorView from './OrchestratorView.vue'

const apiMocks = vi.hoisted(() => ({
  startJobRun: vi.fn(),
  getJobRunProgress: vi.fn(),
  listJobRuns: vi.fn(),
  cancelJobRun: vi.fn(),
  getJobRunSummary: vi.fn().mockRejectedValue(new Error('no summary')),
}))
const strategyMocks = vi.hoisted(() => ({ getStrategies: vi.fn() }))

vi.mock('../api/job', () => apiMocks)
vi.mock('../api/strategies', () => strategyMocks)
vi.mock('../api/selections', () => ({
  listSignalSelections: vi.fn().mockResolvedValue([]),
  latestTournament: (rows: unknown[]) => rows,
}))

const activeRun: JobRunResponse = {
  runId: 'run-123',
  triggerType: 'MANUAL',
  status: 'RUNNING',
  startedAt: '2026-08-25T10:00:00',
  completedAt: null,
  symbolsCount: 2,
  completedCount: 0,
  failedCount: 0,
  errorMessage: null,
}

const runningProgress: JobRunProgressResponse = {
  runId: activeRun.runId,
  status: 'RUNNING',
  totalSymbols: 2,
  completedSymbols: 0,
  failedSymbols: 0,
  startedAt: activeRun.startedAt,
  stages: [],
}

const orderedStages = [
  'DATA_FETCH',
  'SIGNAL',
  'BACKTEST',
  'NEWS',
  'SENTIMENT',
  'LLM_ANALYSIS',
  'PAPER_TRADE',
] as const

function appError(
  message: string,
  options: { kind?: string; outcomeUnknown?: boolean } = {}
): Error & { kind: string; outcomeUnknown: boolean; retryable: boolean } {
  return Object.assign(new Error(message), {
    name: 'AppError',
    kind: options.kind ?? 'validation',
    outcomeUnknown: options.outcomeUnknown ?? false,
    retryable: false,
  })
}

function confirmed<T extends object>(data: T): T & { success: true; data: T } {
  const value = (Array.isArray(data) ? [...data] : { ...data }) as T
  return Object.assign(value, { success: true as const, data })
}

function mountView(): VueWrapper {
  return mount(OrchestratorView, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
        ErrorBoundary: {
          props: ['error'],
          template: '<div><slot /><slot v-if="error" name="error" /></div>',
        },
        LoadingSpinner: { template: '<div>Loading orchestrator data...</div>' },
        StageIcon: true,
        StatusBadge: {
          props: ['status', 'label'],
          template: '<span data-testid="run-status">{{ label || status }}</span>',
        },
      },
    },
  })
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((resolver) => {
    resolve = resolver
  })
  return { promise, resolve }
}

function button(wrapper: VueWrapper, label: string) {
  const match = wrapper.findAll('button').find((candidate) => candidate.text().trim() === label)
  if (!match) throw new Error(`Button not found: ${label}`)
  return match
}

beforeEach(() => {
  vi.clearAllMocks()
  apiMocks.listJobRuns.mockResolvedValue(confirmed<JobRunResponse[]>([]))
  apiMocks.getJobRunProgress.mockResolvedValue(confirmed(runningProgress))
  apiMocks.cancelJobRun.mockResolvedValue({ success: true })
  apiMocks.getJobRunSummary.mockRejectedValue(new Error('no summary'))
  strategyMocks.getStrategies.mockResolvedValue([])
})

afterEach(() => {
  vi.useRealTimers()
})

describe('OrchestratorView — run control', () => {
  it('renders stages in the backend execution sequence', async () => {
    const wrapper = mountView()
    await flushPromises()

    const sequence = wrapper.get('[aria-label="Execution sequence"]')
    expect(sequence.text()).toContain('Data fetch')
    expect(sequence.text()).toContain('Signal')
    expect(sequence.text()).toContain('Backtest')
    expect(sequence.text()).toContain('News')
    expect(sequence.text()).toContain('Sentiment')
    expect(sequence.text()).toContain('LLM analysis')
    expect(sequence.text()).toContain('Paper trade')

    const labels = sequence.findAll('span').map((node) => node.text().trim())
    expect(
      labels.filter(
        (label) => label && !/^\d+$/.test(label) && label !== '→' && label !== '7 stages'
      )
    ).toEqual([
      'Data fetch',
      'Signal',
      'Backtest',
      'News',
      'Sentiment',
      'LLM analysis',
      'Paper trade',
    ])
    expect(orderedStages).toEqual([
      'DATA_FETCH',
      'SIGNAL',
      'BACKTEST',
      'NEWS',
      'SENTIMENT',
      'LLM_ANALYSIS',
      'PAPER_TRADE',
    ])
    wrapper.unmount()
  })

  it('shows an immediate starting state and prevents duplicate start requests', async () => {
    const startRequest = deferred<JobRunResponse>()
    apiMocks.startJobRun.mockReturnValue(startRequest.promise)
    const wrapper = mountView()
    await flushPromises()

    const runButton = wrapper.find('[aria-label="Run job orchestrator"]')
    await runButton.trigger('click')

    expect(apiMocks.startJobRun).toHaveBeenCalledTimes(1)
    expect(runButton.attributes('disabled')).toBeDefined()
    expect(runButton.text()).toContain('Starting...')

    await runButton.trigger('click')
    expect(apiMocks.startJobRun).toHaveBeenCalledTimes(1)

    startRequest.resolve(confirmed(activeRun))
    await flushPromises()

    expect(runButton.attributes('disabled')).toBeDefined()
    expect(runButton.text()).toContain('Running...')
    expect(wrapper.text()).toContain('Pipeline run started')
    wrapper.unmount()
  })

  it('resumes polling for a restored running job and re-enables Run when it completes', async () => {
    vi.useFakeTimers()
    apiMocks.listJobRuns.mockResolvedValue(confirmed([activeRun]))
    apiMocks.getJobRunProgress
      .mockResolvedValueOnce(confirmed(runningProgress))
      .mockResolvedValueOnce(
        confirmed({
          ...runningProgress,
          status: 'COMPLETED',
          completedSymbols: 2,
          completedAt: '2026-08-25T10:05:00',
        })
      )

    const wrapper = mountView()
    await flushPromises()

    const runButton = wrapper.find('[aria-label="Run job orchestrator"]')
    expect(runButton.text()).toContain('Running...')
    expect(runButton.attributes('disabled')).toBeDefined()

    await vi.advanceTimersByTimeAsync(3000)
    await flushPromises()

    expect(apiMocks.getJobRunProgress).toHaveBeenCalledTimes(2)
    expect(runButton.text()).toBe('Run')
    expect(runButton.attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('restores Run and does not invent a current run when start is rejected', async () => {
    apiMocks.startJobRun.mockRejectedValue(appError('Backend rejected the run'))
    const wrapper = mountView()
    await flushPromises()

    const runButton = wrapper.find('[aria-label="Run job orchestrator"]')
    await runButton.trigger('click')
    await flushPromises()

    expect(runButton.text()).toBe('Run')
    expect(runButton.attributes('disabled')).toBeUndefined()
    expect(wrapper.text()).toContain('Backend rejected the run')
    expect(wrapper.text()).toContain('No active run')
    expect(wrapper.text()).not.toContain('Pipeline run started')
    expect(wrapper.find('[data-testid="run-status"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('offers current-status refresh after an ambiguous start without retrying or inventing state', async () => {
    apiMocks.startJobRun.mockRejectedValue(
      appError('Connection closed after start request', {
        kind: 'network',
        outcomeUnknown: true,
      })
    )
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[aria-label="Run job orchestrator"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toMatch(
      /run start.*could not be confirmed|run start.*couldn.t be confirmed/i
    )
    expect(wrapper.text()).toContain('No active run')
    expect(wrapper.find('[data-testid="run-status"]').exists()).toBe(false)
    expect(
      wrapper.findAll('button').some((candidate) => /retry start/i.test(candidate.text()))
    ).toBe(false)

    apiMocks.listJobRuns.mockClear()
    apiMocks.listJobRuns.mockResolvedValue(confirmed([activeRun]))
    await button(wrapper, 'Refresh current status').trigger('click')
    await flushPromises()

    expect(apiMocks.listJobRuns).toHaveBeenCalled()
    expect(apiMocks.startJobRun).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="run-status"]').text()).toBe('RUNNING')
    wrapper.unmount()
  })

  it('retains the confirmed running state after ambiguous cancellation and refreshes status instead', async () => {
    vi.useFakeTimers()
    apiMocks.listJobRuns.mockResolvedValue(confirmed([activeRun]))
    apiMocks.cancelJobRun.mockRejectedValue(
      appError('Connection closed after cancellation request', {
        kind: 'network',
        outcomeUnknown: true,
      })
    )
    const wrapper = mountView()
    await flushPromises()
    apiMocks.getJobRunProgress.mockClear()

    await button(wrapper, 'Cancel').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="run-status"]').text()).toBe('RUNNING')
    expect(wrapper.text()).not.toContain('Run cancelled')
    expect(wrapper.text()).toMatch(
      /cancellation.*could not be confirmed|cancellation.*couldn.t be confirmed/i
    )
    expect(
      wrapper.findAll('button').some((candidate) => /retry cancel/i.test(candidate.text()))
    ).toBe(false)

    await button(wrapper, 'Refresh current status').trigger('click')
    await flushPromises()

    expect(apiMocks.getJobRunProgress).toHaveBeenCalledTimes(1)
    expect(apiMocks.cancelJobRun).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('shows confirmed cancellation with a stale warning when the status refresh fails', async () => {
    vi.useFakeTimers()
    apiMocks.listJobRuns.mockResolvedValue(confirmed([activeRun]))
    apiMocks.getJobRunProgress.mockResolvedValueOnce(confirmed(runningProgress))
    const wrapper = mountView()
    await flushPromises()
    apiMocks.getJobRunProgress.mockRejectedValue(
      appError('Current status refresh failed', { kind: 'network' })
    )

    await button(wrapper, 'Cancel').trigger('click')
    await flushPromises()

    expect(apiMocks.cancelJobRun).toHaveBeenCalledWith(activeRun.runId)
    expect(wrapper.text()).toContain('Cancellation confirmed')
    expect(wrapper.text()).toMatch(/current status.*stale|run status.*out of date|refresh failed/i)
    expect(wrapper.get('[data-testid="run-status"]').text()).toBe('RUNNING')
    expect(wrapper.text()).not.toContain('CANCELLED')
    wrapper.unmount()
  })
})
