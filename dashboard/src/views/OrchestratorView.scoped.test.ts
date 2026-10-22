import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { JobRunProgressResponse, JobRunResponse, StrategyConfig } from '../api/types'
import OrchestratorView from './OrchestratorView.vue'

const jobMocks = vi.hoisted(() => ({
  startJobRun: vi.fn(),
  getJobRunProgress: vi.fn(),
  getJobRunSummary: vi.fn(),
  listJobRuns: vi.fn(),
  cancelJobRun: vi.fn(),
}))
const strategyMocks = vi.hoisted(() => ({ getStrategies: vi.fn() }))

vi.mock('../api/job', () => jobMocks)
vi.mock('../api/strategies', () => strategyMocks)
vi.mock('../api/selections', () => ({
  listSignalSelections: vi.fn().mockResolvedValue([]),
  latestTournament: (rows: unknown[]) => rows,
}))

function config(variantId: string, mode: StrategyConfig['mode'] = 'SHADOW'): StrategyConfig {
  return {
    id: 1,
    variantId,
    version: 1,
    strategyType: 'RSI',
    params: {},
    overlays: {},
    paramsHash: 'h',
    mode,
    paperCapital: 1000,
    current: true,
    notes: null,
    createdAt: '2026-09-21T00:00:00Z',
  }
}

const runningRun: JobRunResponse = {
  runId: 'run-1',
  triggerType: 'MANUAL',
  status: 'RUNNING',
  startedAt: '2026-09-21T10:00:00',
  completedAt: null,
  symbolsCount: 1,
  completedCount: 0,
  failedCount: 0,
  errorMessage: null,
}

function stageRow(over: Record<string, unknown>) {
  return {
    symbol: 'TCS',
    stageName: 'SIGNAL',
    status: 'COMPLETED',
    startedAt: '2026-09-21T10:00:00',
    completedAt: null,
    durationMs: 3,
    errorMessage: null,
    resultSummary: null,
    details: null,
    ...over,
  }
}

function progress(status: string, stages: unknown[], runId = 'run-1'): JobRunProgressResponse {
  return {
    runId,
    status,
    totalSymbols: 1,
    completedSymbols: 1,
    failedSymbols: 0,
    startedAt: '2026-09-21T10:00:00',
    stages,
  } as JobRunProgressResponse
}

function appError(message: string, kind: string, status: number) {
  return Object.assign(new Error(message), {
    name: 'AppError',
    kind,
    status,
    outcomeUnknown: false,
    retryable: false,
  })
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
        LoadingSpinner: true,
        StageIcon: { props: ['status'], template: '<i :data-stage-status="status" />' },
        StatusBadge: {
          props: ['status', 'label'],
          template: '<span data-testid="run-status">{{ label || status }}</span>',
        },
      },
    },
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  jobMocks.listJobRuns.mockResolvedValue([])
  jobMocks.getJobRunProgress.mockResolvedValue(progress('COMPLETED', []))
  jobMocks.getJobRunSummary.mockRejectedValue(new Error('no summary'))
  strategyMocks.getStrategies.mockResolvedValue([
    config('ALPHA'),
    config('BETA'),
    config('OLD', 'OFF'),
  ])
})

describe('OrchestratorView start panel', () => {
  it('lists configured strategies (excluding OFF) and starts a scoped, quick, dry run', async () => {
    jobMocks.startJobRun.mockResolvedValue(runningRun)
    const wrapper = mountView()
    await flushPromises()

    const panel = wrapper.get('[data-testid="start-panel"]')
    expect(panel.text()).toContain('ALPHA')
    expect(panel.text()).toContain('BETA')
    expect(panel.text()).not.toContain('OLD')

    await panel.get('input[value="ALPHA"]').setValue(true)
    await panel.get('[data-testid="symbol-filter"]').setValue('tcs, infy ,TCS')
    await panel.get('[data-testid="quick-run"]').setValue(true)
    await panel.get('[data-testid="dry-run"]').setValue(true)

    await wrapper.get('[aria-label="Run job orchestrator"]').trigger('click')
    await flushPromises()

    expect(jobMocks.startJobRun).toHaveBeenCalledWith('MANUAL', {
      symbols: ['TCS', 'INFY'],
      variantIds: ['ALPHA'],
      skipLlm: true,
      dryRun: true,
    })
    wrapper.unmount()
  })

  it('sends no body when the panel is left at its defaults', async () => {
    jobMocks.startJobRun.mockResolvedValue(runningRun)
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[aria-label="Run job orchestrator"]').trigger('click')
    await flushPromises()
    expect(jobMocks.startJobRun).toHaveBeenCalledWith('MANUAL', undefined)
    wrapper.unmount()
  })

  it('disables the panel and Run with an explanatory tooltip while a run is active', async () => {
    jobMocks.listJobRuns.mockResolvedValue([runningRun])
    jobMocks.getJobRunProgress.mockResolvedValue(progress('RUNNING', []))
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[aria-label="Run job orchestrator"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="run-button-wrap"]').attributes('title')).toMatch(
      /run is already active/i
    )
    expect(wrapper.get('[data-testid="quick-run"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })

  it('handles 409 by explaining an active run exists and refreshing status', async () => {
    jobMocks.startJobRun.mockRejectedValue(appError('A run is already active', 'conflict', 409))
    const wrapper = mountView()
    await flushPromises()

    jobMocks.listJobRuns.mockClear()
    jobMocks.listJobRuns.mockResolvedValue([runningRun])
    jobMocks.getJobRunProgress.mockResolvedValue(progress('RUNNING', []))
    await wrapper.get('[aria-label="Run job orchestrator"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toMatch(/already active/i)
    expect(jobMocks.listJobRuns).toHaveBeenCalled()
    expect(wrapper.get('[data-testid="run-status"]').text()).toBe('RUNNING')
    wrapper.unmount()
  })

  it('shows the server message for 400 unknown symbols/variants as an alert', async () => {
    jobMocks.startJobRun.mockRejectedValue(appError('Unknown symbols: ZZZ', 'validation', 400))
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[aria-label="Run job orchestrator"]').trigger('click')
    await flushPromises()

    const alert = wrapper.get('[role="alert"]')
    expect(alert.text()).toContain('Unknown symbols: ZZZ')
    wrapper.unmount()
  })

  it('still renders the panel when strategies fail to load', async () => {
    strategyMocks.getStrategies.mockRejectedValue(new Error('down'))
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.get('[data-testid="start-panel"]').text()).toMatch(/strategies unavailable/i)
    wrapper.unmount()
  })
})

describe('OrchestratorView run detail', () => {
  const doneRun = {
    ...runningRun,
    status: 'COMPLETED_WITH_WARNINGS',
    completedAt: '2026-09-21T10:01:00',
  }

  beforeEach(() => {
    jobMocks.listJobRuns.mockResolvedValue([doneRun])
    jobMocks.getJobRunProgress.mockResolvedValue(
      progress('COMPLETED_WITH_WARNINGS', [
        stageRow({
          details: {
            strategies: [
              { variantId: 'ALPHA', version: 1, outcome: 'EVALUATED', signal: 'BUY' },
              { variantId: 'BETA', version: 1, outcome: 'SKIPPED', reason: 'bad params' },
            ],
          },
        }),
        stageRow({
          stageName: 'SENTIMENT',
          status: 'DEGRADED',
          details: { source: 'KEYWORD_FALLBACK', reason: 'LLM returned 0 chars', warnings: [] },
        }),
        stageRow({
          stageName: 'LLM_ANALYSIS',
          details: { source: 'LLM', reason: null, warnings: [] },
        }),
      ])
    )
  })

  it('renders the per-symbol x per-strategy matrix with reasons', async () => {
    const wrapper = mountView()
    await flushPromises()

    const matrix = wrapper.get('[data-testid="strategy-matrix"]')
    expect(matrix.text()).toContain('ALPHA')
    expect(matrix.text()).toContain('BETA')
    expect(matrix.get('[data-testid="cell-TCS-ALPHA"]').attributes('data-kind')).toBe('signal')
    expect(matrix.get('[data-testid="cell-TCS-ALPHA"]').text()).toContain('BUY')
    const skipped = matrix.get('[data-testid="cell-TCS-BETA"]')
    expect(skipped.attributes('data-kind')).toBe('skipped')
    expect(skipped.text()).toContain('bad params')
    wrapper.unmount()
  })

  it('shows COMPLETED_WITH_WARNINGS, a warnings banner and DEGRADED stage state', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-testid="run-status"]').text()).toBe('COMPLETED_WITH_WARNINGS')
    const banner = wrapper.get('[data-testid="warnings-banner"]')
    expect(banner.text()).toContain('1 degraded stage')
    expect(banner.text()).toContain('LLM returned 0 chars')
    expect(wrapper.find('[data-stage-status="DEGRADED"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('shows LLM vs keyword-fallback source with the degraded reason in stage cards', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('tbody tr').trigger('click')

    const sentiment = wrapper.get('[data-testid="stage-card-SENTIMENT"]')
    expect(sentiment.text()).toContain('Keyword fallback')
    expect(sentiment.text()).toContain('LLM returned 0 chars')
    expect(wrapper.get('[data-testid="stage-card-LLM_ANALYSIS"]').text()).toContain('Source: LLM')
    wrapper.unmount()
  })

  it('uses summary breakdowns in the banner when available', async () => {
    jobMocks.getJobRunSummary.mockResolvedValue({
      runId: 'run-1',
      status: 'COMPLETED_WITH_WARNINGS',
      totalSymbols: 1,
      completedSymbols: 1,
      failedSymbols: 0,
      totalDurationMs: 10,
      stageStats: {},
      symbolDetails: [],
      degradedStages: 1,
      skippedStrategies: 1,
      degradedStageBreakdown: [{ stage: 'SENTIMENT', reason: 'ollama down', count: 1 }],
      skippedStrategyBreakdown: [
        { variantId: 'BETA', outcome: 'SKIPPED', reason: 'bad params', symbols: 1 },
      ],
    })
    const wrapper = mountView()
    await flushPromises()

    const banner = wrapper.get('[data-testid="warnings-banner"]')
    expect(banner.text()).toContain('1 skipped strategy')
    expect(banner.text()).toContain('ollama down')
    expect(banner.text()).toContain('BETA')
    wrapper.unmount()
  })
})
