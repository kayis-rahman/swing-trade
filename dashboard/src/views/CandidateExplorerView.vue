<template>
  <div class="view-shell candidate-explorer p-4 sm:p-6 animate-fade-in">
    <header
      class="mb-6 flex flex-col gap-4 rounded-2xl border border-border-subtle p-5 sm:flex-row sm:items-end sm:justify-between sm:p-7"
    >
      <div>
        <p class="explorer-kicker">Research desk / candidate discovery</p>
        <h1 class="mt-2 font-display text-3xl font-semibold tracking-tight text-text-primary">
          Candidate Explorer
        </h1>
        <p class="mt-2 max-w-2xl text-sm leading-6 text-text-muted">
          Scan all active stock records for strategy-consensus BUY signals with a profitable
          backtest. Candidates need agreement from at least two active strategies before they join
          the pilot wishlist.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="rounded-xl border border-border-subtle px-4 py-2.5 text-sm font-semibold text-text-secondary transition-transform duration-150 ease-out hover:border-brand active:scale-[0.97]"
          @click="showSettings = !showSettings"
        >
          {{ showSettings ? 'Hide settings' : 'Configure' }}
        </button>
        <button
          v-if="run?.status === 'RUNNING'"
          class="rounded-xl border border-warning/40 px-4 py-2.5 text-sm font-semibold text-warning transition-transform duration-150 ease-out active:scale-[0.97]"
          @click="pause"
        >
          Pause scan
        </button>
        <button
          v-if="run?.status === 'RUNNING' || run?.status === 'PAUSED'"
          class="rounded-xl border border-warning/40 px-4 py-2.5 text-sm font-semibold text-warning transition-transform duration-150 ease-out active:scale-[0.97]"
          @click="cancel"
        >
          Cancel scan
        </button>
        <button
          v-if="run?.status === 'PAUSED'"
          class="rounded-xl bg-brand px-4 py-2.5 text-sm font-semibold text-brand-text transition-transform duration-150 ease-out hover:bg-brand-hover active:scale-[0.97]"
          @click="resume"
        >
          Resume scan
        </button>
        <button
          v-else
          data-test="scan-toggle"
          class="rounded-xl bg-brand px-4 py-2.5 text-sm font-semibold text-brand-text transition-transform duration-150 ease-out hover:bg-brand-hover active:scale-[0.97]"
          :disabled="loading"
          @click="start"
        >
          {{ loading ? 'Starting...' : 'Scan full universe' }}
        </button>
      </div>
    </header>

    <section v-if="showSettings" class="card-panel mb-6 overflow-hidden">
      <div class="border-b border-border-subtle px-5 py-4">
        <p class="explorer-kicker">Scan configuration</p>
        <h2 class="mt-1 text-base font-semibold text-text-primary">Gate & performance settings</h2>
      </div>
      <div v-if="settingsError" class="px-5 pt-4 text-sm text-error">{{ settingsError }}</div>
      <div class="grid gap-4 px-5 py-5 sm:grid-cols-2 xl:grid-cols-4">
        <label class="flex flex-col gap-1.5 text-sm">
          <span class="font-medium text-text-secondary">Strategies required to BUY</span>
          <input
            v-model.number="settingsForm.minStrategyBuys"
            type="number"
            min="1"
            max="12"
            step="1"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-text-primary outline-none focus:border-brand"
          />
          <span class="text-xs leading-5 text-text-muted">
            Strategy performance is persisted per variant for attribution; this table currently
            shows signal, variant, and score while qualification still uses the common gates below.
          </span>
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <span class="font-medium text-text-secondary">Min win rate (%)</span>
          <input
            v-model.number="settingsForm.minWinRate"
            type="number"
            min="0"
            max="100"
            step="1"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-text-primary outline-none focus:border-brand"
          />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <span class="font-medium text-text-secondary">Min total return (%)</span>
          <input
            v-model.number="settingsForm.minTotalReturn"
            type="number"
            step="1"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-text-primary outline-none focus:border-brand"
          />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <span class="font-medium text-text-secondary">Max concurrent workers</span>
          <input
            v-model.number="settingsForm.maxConcurrent"
            type="number"
            min="1"
            max="12"
            step="1"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-text-primary outline-none focus:border-brand"
          />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <span class="font-medium text-text-secondary">Minimum trades (each gate)</span>
          <input
            v-model.number="settingsForm.minTrades"
            type="number"
            min="1"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-text-primary outline-none focus:border-brand"
          />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <span class="font-medium text-text-secondary">Out-of-sample window (days)</span>
          <input
            v-model.number="settingsForm.oosDays"
            type="number"
            min="60"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-text-primary outline-none focus:border-brand"
          />
        </label>
        <label class="flex flex-col gap-1.5 text-sm">
          <span class="font-medium text-text-secondary">Backfill years</span>
          <input
            v-model.number="settingsForm.backfillYears"
            type="number"
            min="1"
            max="10"
            step="1"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-text-primary outline-none focus:border-brand"
          />
        </label>
      </div>
      <div class="flex items-center justify-end gap-2 border-t border-border-subtle px-5 py-3">
        <span v-if="settingsSaved" class="text-xs text-success">Saved.</span>
        <button
          class="rounded-lg bg-brand px-4 py-2 text-sm font-semibold text-brand-text transition-transform duration-150 ease-out hover:bg-brand-hover active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="settingsSaving"
          @click="saveSettings"
        >
          {{ settingsSaving ? 'Saving...' : 'Save settings' }}
        </button>
      </div>
    </section>

    <div class="mb-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
      <div
        v-for="metric in metrics"
        :key="metric.label"
        class="rounded-xl border border-border-subtle bg-bg-surface/45 p-4"
      >
        <span class="text-xs font-medium uppercase tracking-[0.12em] text-text-muted">{{
          metric.label
        }}</span>
        <strong class="mt-2 block text-xl font-semibold" :class="metric.tone">{{
          metric.value
        }}</strong>
        <small class="mt-1 block text-xs text-text-muted">{{ metric.detail }}</small>
      </div>
    </div>

    <section
      v-if="run?.status === 'RUNNING' || run?.status === 'PAUSED'"
      class="card-panel mb-4 overflow-hidden"
    >
      <div class="flex items-center justify-between border-b border-border-subtle px-5 py-4">
        <div>
          <p class="explorer-kicker">Live telemetry</p>
          <h2 class="mt-1 text-base font-semibold text-text-primary">Scan activity</h2>
        </div>
        <span class="inline-flex items-center gap-2 text-xs text-text-muted">
          <span
            class="h-1.5 w-1.5 rounded-full"
            :class="streamConnected ? 'bg-success pulse-dot' : 'bg-text-muted'"
          />
          {{
            run?.status === 'PAUSED' ? 'Paused' : streamConnected ? 'Streaming' : 'Waiting for scan'
          }}
        </span>
      </div>
      <div v-if="logs.length === 0" class="px-5 py-8 text-sm text-text-muted">
        Scan events will appear here as symbols move through data, signal, and backtest stages.
      </div>
      <ol v-else class="max-h-72 overflow-y-auto px-5 py-3 font-mono text-xs" aria-live="polite">
        <li
          v-for="(entry, index) in logs"
          :key="`${entry.timestamp}-${index}`"
          class="flex gap-3 border-b border-border-subtle/40 py-2 last:border-0"
        >
          <time class="shrink-0 text-text-muted">{{ formatLogTime(entry.timestamp) }}</time>
          <span class="shrink-0 font-semibold" :class="logLevelClass(entry.level)">{{
            entry.level
          }}</span>
          <span class="text-text-secondary"
            >{{ entry.symbol ? `${entry.symbol}: ` : '' }}{{ entry.message }}</span
          >
        </li>
      </ol>
    </section>

    <section class="card-panel overflow-hidden">
      <div
        class="flex flex-col gap-3 border-b border-border-subtle px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
      >
        <div>
          <p class="explorer-kicker">Three-stage screen</p>
          <h2 class="mt-1 text-base font-semibold text-text-primary">Signal candidates</h2>
        </div>
        <div class="flex items-center gap-2 text-xs text-text-muted">
          <span class="rounded-full bg-brand/10 px-2.5 py-1 font-semibold text-brand"
            >≥2 strategy BUYs + common performance gates</span
          >
          <span v-if="run">Updated {{ updatedAt }}</span>
        </div>
      </div>

      <div
        v-if="run"
        class="flex flex-col gap-3 border-b border-border-subtle px-5 py-4 sm:flex-row sm:items-center"
      >
        <label class="relative flex-1">
          <span class="sr-only">Filter symbols</span>
          <input
            v-model="symbolFilter"
            type="search"
            placeholder="Filter symbols…"
            class="h-10 w-full rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-sm text-text-primary outline-none transition-colors duration-150 placeholder:text-text-muted focus:border-brand"
          />
        </label>
        <label>
          <span class="sr-only">Filter signal</span>
          <select
            v-model="signalFilter"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-sm text-text-primary outline-none focus:border-brand"
          >
            <option value="">All signals</option>
            <option value="BUY">BUY</option>
            <option value="SELL">SELL</option>
            <option value="HOLD">HOLD</option>
          </select>
        </label>
        <label>
          <span class="sr-only">Rows per page</span>
          <select
            v-model.number="pageSize"
            class="h-10 rounded-lg border border-border-subtle bg-bg-surface/60 px-3 text-sm text-text-primary outline-none focus:border-brand"
          >
            <option :value="10">10 / page</option>
            <option :value="25">25 / page</option>
            <option :value="50">50 / page</option>
            <option :value="100">100 / page</option>
          </select>
        </label>
      </div>

      <div v-if="error" class="px-5 py-8 text-sm text-error">{{ error }}</div>
      <div v-else-if="!run" class="px-5 py-16 text-center">
        <div
          class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-brand/10 text-xl text-brand"
        >
          ⌕
        </div>
        <h3 class="font-semibold text-text-primary">Find the next pilot symbol</h3>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-text-muted">
          Start a scan to fetch missing history, calculate the current signal, and run the default
          strategy backtest.
        </p>
      </div>
      <div
        v-else-if="results.length === 0 && run.status === 'RUNNING'"
        class="px-5 py-16 text-center text-sm text-text-muted"
      >
        Waiting for the first symbol to finish…
      </div>
      <div v-else-if="results.length === 0" class="px-5 py-16 text-center text-sm text-text-muted">
        No scan results match the selected filters.
      </div>
      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[850px] text-left text-sm">
          <thead
            class="border-b border-border-subtle bg-bg-surface/60 text-xs uppercase tracking-[0.12em] text-text-muted"
          >
            <tr>
              <th class="px-5 py-3 font-semibold">Symbol</th>
              <th class="px-3 py-3 font-semibold">Data</th>
              <th class="px-3 py-3 font-semibold">Signal</th>
              <th class="px-3 py-3 font-semibold">Strategy votes</th>
              <th class="px-3 py-3 font-semibold">Trades</th>
              <th class="px-3 py-3 font-semibold">Win rate</th>
              <th class="px-3 py-3 font-semibold">Return</th>
              <th class="px-3 py-3 font-semibold">Decision</th>
              <th class="px-3 py-3 font-semibold">History / activation</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-subtle/70">
            <tr
              v-for="item in results"
              :key="item.symbol"
              class="transition-colors duration-150 hover:bg-bg-hover/60"
            >
              <td class="px-5 py-3.5 font-semibold text-text-primary">{{ item.symbol }}</td>
              <td class="px-3 py-3.5 text-text-muted">
                {{ item.candleCount.toLocaleString() }} candles
              </td>
              <td class="px-3 py-3.5">
                <span :class="signalClass(item.signalType)">{{ item.signalType ?? '—' }}</span>
              </td>
              <td class="px-3 py-3.5 text-text-muted">
                {{ item.strategyBuyCount ?? 0 }}/{{ item.strategyEvaluationCount ?? 0 }} BUY
                <div v-if="item.strategyOutcomes?.length" class="mt-1 space-y-0.5 text-[11px]">
                  <div v-for="outcome in item.strategyOutcomes" :key="outcome.variantId">
                    <span :class="signalClass(outcome.signalType)">{{ outcome.signalType }}</span>
                    <span class="ml-1 text-text-muted">{{ outcome.variantId }}</span>
                    <span v-if="outcome.score != null" class="ml-1 text-text-muted">
                      ({{ outcome.score.toFixed(2) }})
                    </span>
                  </div>
                </div>
              </td>
              <td class="px-3 py-3.5 text-text-muted">{{ item.totalTrades ?? '—' }}</td>
              <td class="px-3 py-3.5 text-text-primary">{{ percent(item.winRate) }}</td>
              <td
                class="px-3 py-3.5"
                :class="(item.totalReturn ?? 0) > 0 ? 'text-success' : 'text-text-muted'"
              >
                {{ percent(item.totalReturn) }}
              </td>
              <td class="px-3 py-3.5">
                <span
                  v-if="item.qualified"
                  class="rounded-full bg-success/10 px-2.5 py-1 text-xs font-semibold text-success"
                  >Qualified</span
                >
                <span v-else class="text-xs text-text-muted">{{
                  item.reason ?? item.dataStatus
                }}</span>
              </td>
              <td class="px-3 py-3.5 text-xs text-text-muted">
                <div>{{ item.sourceOutcome ?? item.dataStatus }}</div>
                <div v-if="item.firstAvailableDate">
                  {{ item.firstAvailableDate }} → {{ item.lastAvailableDate }}
                </div>
                <div v-if="item.retryAfter">Retry {{ item.retryAfter }}</div>
                <div v-if="item.activated" class="mt-1 font-semibold text-success">
                  Activated — auto-added to wishlist
                </div>
                <div v-if="item.oosTotalTrades != null">
                  OOS: {{ item.oosTotalTrades }} trades, {{ percent(item.oosWinRate) }},
                  {{ percent(item.oosTotalReturn) }}
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div
        v-if="run && totalResults > 0"
        class="flex flex-col gap-3 border-t border-border-subtle px-5 py-3 text-xs text-text-muted sm:flex-row sm:items-center sm:justify-between"
      >
        <span>Showing {{ pageStart + 1 }}–{{ pageEnd }} of {{ totalResults }} results</span>
        <div class="flex items-center gap-3">
          <span v-if="run.status === 'RUNNING'" class="inline-flex items-center gap-2 text-brand"
            ><span class="h-1.5 w-1.5 rounded-full bg-brand pulse-dot" />Scanning in the
            background</span
          >
          <button
            class="rounded-md border border-border-subtle px-2.5 py-1.5 transition-transform duration-150 ease-out hover:border-brand disabled:cursor-not-allowed disabled:opacity-40 active:scale-[0.97]"
            :disabled="pageNumber === 0 || tableLoading"
            @click="changePage(-1)"
          >
            Previous
          </button>
          <span class="tabular-nums">{{ pageNumber + 1 }} / {{ totalPages }}</span>
          <button
            class="rounded-md border border-border-subtle px-2.5 py-1.5 transition-transform duration-150 ease-out hover:border-brand disabled:cursor-not-allowed disabled:opacity-40 active:scale-[0.97]"
            :disabled="pageNumber >= totalPages - 1 || tableLoading"
            @click="changePage(1)"
          >
            Next
          </button>
        </div>
      </div>
    </section>

    <p class="mt-4 text-xs leading-5 text-text-muted">
      Candidate scans never invoke paper trading. Manual scans cover all active stock records;
      qualified symbols are automatically added to the wishlist. After a manual scan, orchestration
      runs over the active watchlist only; scheduled scans hand off qualified symbols only.
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { asAppError } from '../errors/appError'
import {
  cancelCandidateScan,
  getCandidateScan,
  getCandidateScanHistory,
  getCandidateScanResults,
  getCandidateScanSettings,
  openCandidateScanStream,
  pauseCandidateScan,
  resumeCandidateScan,
  startCandidateScan,
  updateCandidateScanSettings,
} from '../api/candidateScan'
import type { CandidateScanLogEvent, CandidateScanResult, CandidateScanRun } from '../api/types'

const run = ref<CandidateScanRun | null>(null)
const results = ref<CandidateScanResult[]>([])
const totalResults = ref(0)
const symbolFilter = ref('')
const signalFilter = ref('')
const pageSize = ref(10)
const pageNumber = ref(0)
const tableLoading = ref(false)
const loading = ref(false)
const error = ref('')
const updatedAt = ref('—')
let timer: ReturnType<typeof setInterval> | undefined
let stream: EventSource | null = null
let filterTimer: ReturnType<typeof setTimeout> | undefined
const logs = ref<CandidateScanLogEvent[]>([])
const streamConnected = ref(false)
const showSettings = ref(false)
const settingsForm = ref({
  minWinRate: 45,
  minTotalReturn: 0,
  maxConcurrent: 3,
  backfillYears: 3,
  minTrades: 15,
  oosDays: 252,
  minStrategyBuys: 2,
})
const settingsSaving = ref(false)
const settingsSaved = ref(false)
const settingsError = ref('')

async function loadSettings() {
  try {
    const settings = await getCandidateScanSettings()
    settingsForm.value = {
      minWinRate: Number(settings['candidate-scan.min-win-rate']),
      minTotalReturn: Number(settings['candidate-scan.min-total-return']),
      maxConcurrent: Number(settings['candidate-scan.max-concurrent']),
      backfillYears: Number(settings['candidate-scan.backfill-years']),
      minTrades: Number(settings['candidate-scan.min-trades'] ?? 15),
      oosDays: Number(settings['candidate-scan.out-of-sample-days'] ?? 252),
      minStrategyBuys: Number(settings['candidate-scan.min-strategy-buys'] ?? 2),
    }
  } catch (e) {
    settingsError.value = asAppError(e).message
  }
}

async function saveSettings() {
  settingsSaving.value = true
  settingsSaved.value = false
  settingsError.value = ''
  try {
    await updateCandidateScanSettings({
      'candidate-scan.min-win-rate': String(settingsForm.value.minWinRate),
      'candidate-scan.min-total-return': String(settingsForm.value.minTotalReturn),
      'candidate-scan.max-concurrent': String(settingsForm.value.maxConcurrent),
      'candidate-scan.backfill-years': String(settingsForm.value.backfillYears),
      'candidate-scan.min-trades': String(settingsForm.value.minTrades),
      'candidate-scan.out-of-sample-days': String(settingsForm.value.oosDays),
      'candidate-scan.min-strategy-buys': String(settingsForm.value.minStrategyBuys),
    })
    settingsSaved.value = true
  } catch (e) {
    settingsError.value = asAppError(e).message
  } finally {
    settingsSaving.value = false
  }
}

const totalPages = computed(() => Math.max(1, Math.ceil(totalResults.value / pageSize.value)))
const pageStart = computed(() => pageNumber.value * pageSize.value)
const pageEnd = computed(() => Math.min(pageStart.value + results.value.length, totalResults.value))
const metrics = computed(() => [
  {
    label: 'Universe',
    value: run.value ? run.value.totalSymbols : '—',
    detail:
      run.value?.scanScope === 'WATCHLIST'
        ? 'active watchlist symbols'
        : run.value?.scanScope === 'ACTIVE_STOCKS'
          ? 'all active stock records'
          : run.value?.scanScope === 'NSE_BROAD'
            ? 'NSE symbol master'
            : 'scope unavailable for legacy run',
    tone: 'text-text-primary',
  },
  {
    label: 'Processed',
    value: run.value ? `${run.value.completedSymbols}/${run.value.totalSymbols}` : '—',
    detail:
      run.value?.status === 'CANCELLED'
        ? 'stopped before the universe finished'
        : run.value?.status === 'COMPLETED'
          ? 'universe scan finished'
          : 'symbols processed so far',
    tone: 'text-text-primary',
  },
  {
    label: 'BUY candidates',
    value: run.value?.qualifiedSymbols ?? '—',
    detail: 'passed gate',
    tone: 'text-brand',
  },
  {
    label: 'Failed',
    value: run.value?.failedSymbols ?? '—',
    detail: 'provider or data errors',
    tone: 'text-error',
  },
  {
    label: 'Status',
    value: run.value?.status ?? 'READY',
    detail:
      run.value?.orchestrationStatus === 'STARTED'
        ? 'watchlist orchestration started'
        : run.value?.orchestrationStatus === 'PENDING'
          ? run.value.scanTrigger === 'MANUAL'
            ? 'watchlist orchestration pending'
            : 'candidate handoff pending'
          : 'screening only; no trades run',
    tone: 'text-text-primary',
  },
])

const percent = (value?: number) => (value == null ? '—' : `${value.toFixed(1)}%`)
const signalClass = (signal?: string) =>
  signal === 'BUY'
    ? 'rounded-full bg-success/10 px-2.5 py-1 text-xs font-semibold text-success'
    : signal === 'SELL'
      ? 'rounded-full bg-error/10 px-2.5 py-1 text-xs font-semibold text-error'
      : 'text-xs text-text-muted'

async function load(runId: string) {
  tableLoading.value = true
  const [nextRun, nextResults] = await Promise.all([
    getCandidateScan(runId),
    getCandidateScanResults(
      runId,
      pageStart.value,
      pageSize.value,
      symbolFilter.value,
      signalFilter.value
    ),
  ])
  run.value = nextRun
  results.value = nextResults.items
  totalResults.value = nextResults.total
  updatedAt.value = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  if (nextRun.status !== 'RUNNING' && nextRun.status !== 'PAUSED') stopPolling()
  tableLoading.value = false
}

function startPolling(runId: string) {
  stopPolling()
  timer = setInterval(() => void load(runId).catch(handleError), 2000)
}
function stopPolling() {
  if (timer) clearInterval(timer)
  timer = undefined
}
function changePage(delta: number) {
  pageNumber.value = Math.max(0, Math.min(totalPages.value - 1, pageNumber.value + delta))
  if (run.value) void load(run.value.runId).catch(handleError)
}
function scheduleFilterLoad() {
  if (filterTimer) clearTimeout(filterTimer)
  filterTimer = setTimeout(() => {
    if (run.value) void load(run.value.runId).catch(handleError)
  }, 180)
}
function closeStream() {
  stream?.close()
  stream = null
  streamConnected.value = false
}
function startStream(runId: string) {
  closeStream()
  stream = openCandidateScanStream(
    runId,
    (event) => {
      streamConnected.value = true
      logs.value = [...logs.value, event].slice(-100)
      if (event.eventType === 'RUN_COMPLETED' || event.eventType === 'RUN_CANCELLED') closeStream()
    },
    () => {
      streamConnected.value = false
    }
  )
}
function handleError(cause: unknown) {
  error.value = asAppError(cause).message
}
function formatLogTime(value: string) {
  return new Date(value).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
function logLevelClass(level: CandidateScanLogEvent['level']) {
  return level === 'ERROR'
    ? 'text-error'
    : level === 'WARN'
      ? 'text-warning'
      : level === 'SUCCESS'
        ? 'text-success'
        : 'text-brand'
}

async function start() {
  loading.value = true
  error.value = ''
  try {
    const started = await startCandidateScan()
    run.value = started
    results.value = []
    totalResults.value = 0
    pageNumber.value = 0
    logs.value = []
    startStream(started.runId)
    startPolling(started.runId)
    await load(started.runId)
  } catch (cause) {
    handleError(cause)
  } finally {
    loading.value = false
  }
}

async function cancel() {
  if (!run.value) return
  try {
    run.value = await cancelCandidateScan(run.value.runId)
    stopPolling()
    closeStream()
  } catch (cause) {
    handleError(cause)
  }
}

async function pause() {
  if (!run.value) return
  try {
    run.value = await pauseCandidateScan(run.value.runId)
  } catch (cause) {
    handleError(cause)
  }
}

async function resume() {
  if (!run.value) return
  try {
    run.value = await resumeCandidateScan(run.value.runId)
  } catch (cause) {
    handleError(cause)
  }
}

onMounted(async () => {
  loadSettings()
  try {
    const history = await getCandidateScanHistory()
    if (history[0]) {
      await load(history[0].runId)
      logs.value = []
      if (run.value?.status === 'RUNNING' || run.value?.status === 'PAUSED') {
        startStream(history[0].runId)
        startPolling(history[0].runId)
      }
    }
  } catch (cause) {
    handleError(cause)
  }
})
watch([symbolFilter, signalFilter, pageSize], () => {
  pageNumber.value = 0
  scheduleFilterLoad()
})
onBeforeUnmount(() => {
  stopPolling()
  closeStream()
})
</script>

<style scoped>
.candidate-explorer {
  --explorer-ease: cubic-bezier(0.23, 1, 0.32, 1);
}
.explorer-kicker {
  color: var(--color-brand);
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}
</style>
