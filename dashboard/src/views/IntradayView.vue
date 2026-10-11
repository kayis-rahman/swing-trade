<template>
  <div class="view-shell space-y-5 p-4 sm:p-6 animate-fade-in">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="mb-1 text-[10px] font-semibold uppercase tracking-[0.18em] text-brand">
          Paper operations / NSE
        </p>
        <h1 class="font-display text-2xl font-semibold text-text-primary">Intraday cockpit</h1>
        <p class="mt-1 max-w-2xl text-sm text-text-muted">
          Replay a complete 15-minute session and inspect its risk-gated paper ledger. No live
          broker orders are sent.
        </p>
      </div>
      <div
        class="flex items-center gap-2 rounded-md border border-brand/20 bg-brand-subtle px-3 py-2"
      >
        <span class="h-2 w-2 rounded-full bg-brand" aria-hidden="true" />
        <span class="text-xs font-semibold tracking-wide text-brand">PAPER ONLY</span>
      </div>
    </header>

    <div
      v-if="isCandidateHandoff"
      class="rounded-md border border-info/30 bg-info/10 px-4 py-3 text-xs text-text-secondary"
      role="status"
    >
      Candidate {{ form.symbol }} is prefilled from Candidate Explorer and already auto-added to the
      wishlist. Choose the intraday session date, strategy, and cutoff below; daily swing strategy
      outcomes do not select an intraday strategy. The paper replay starts only when you explicitly
      submit this form.
    </div>

    <section class="card-panel p-4 sm:p-5" aria-labelledby="session-form-heading">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 id="session-form-heading" class="text-sm font-semibold text-text-primary">
            Session replay
          </h2>
          <p class="mt-1 text-xs text-text-muted">
            Full-session replay needs complete NSE coverage; as-of replay needs bars through the
            selected time.
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            :disabled="loading || scanning"
            class="rounded-md border border-brand/30 bg-brand-subtle px-3 py-2 text-xs font-semibold text-brand hover:bg-brand/15 disabled:opacity-50"
            @click="loadSeededSession"
          >
            Load seeded session
          </button>
          <button
            type="button"
            :disabled="loading || scanning || !canScanWatchlist"
            class="rounded-md border border-info/30 bg-info/10 px-3 py-2 text-xs font-semibold text-info hover:bg-info/15 disabled:cursor-not-allowed disabled:opacity-50"
            @click="scanWatchlist"
          >
            {{ scanning ? 'Scanning watchlist…' : 'Scan watchlist at selected time' }}
          </button>
        </div>
      </div>

      <form
        class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-9"
        @submit.prevent="runSession"
      >
        <label class="text-xs font-medium text-text-muted">
          Symbol
          <input
            v-model.trim="form.symbol"
            aria-label="Paper session symbol"
            required
            maxlength="10"
            class="mt-1 w-full rounded-md border bg-bg-primary px-3 py-2 text-sm text-text-primary"
          />
        </label>
        <label class="text-xs font-medium text-text-muted">
          Session date
          <input
            v-model="form.sessionDate"
            aria-label="Paper session date"
            type="date"
            required
            class="mt-1 w-full rounded-md border bg-bg-primary px-3 py-2 text-sm text-text-primary"
          />
        </label>
        <label class="text-xs font-medium text-text-muted">
          Replay as of (IST)
          <select
            v-model="form.asOfBarTime"
            aria-label="Replay as of bar time"
            class="mt-1 w-full rounded-md border bg-bg-primary px-3 py-2 text-sm text-text-primary"
          >
            <option v-if="isCandidateHandoff" :value="AS_OF_SELECTION_REQUIRED" disabled>
              Select cutoff or end of session
            </option>
            <option value="">End of session</option>
            <option v-for="time in sessionBarTimes" :key="time" :value="time">{{ time }}</option>
          </select>
        </label>
        <label class="text-xs font-medium text-text-muted xl:col-span-2">
          Strategy
          <select
            v-model="form.strategyType"
            aria-label="Paper session strategy"
            class="mt-1 w-full rounded-md border bg-bg-primary px-3 py-2 text-sm text-text-primary"
          >
            <option value="" disabled>Select an intraday strategy</option>
            <option value="INTRADAY_ORB">Opening range breakout</option>
            <option value="INTRADAY_VWAP_PULLBACK">VWAP pullback</option>
            <option value="INTRADAY_RSI2">RSI(2) reversal</option>
          </select>
        </label>
        <label class="text-xs font-medium text-text-muted">
          Quantity
          <input
            v-model.number="form.quantity"
            aria-label="Paper order quantity"
            type="number"
            min="1"
            required
            class="mt-1 w-full rounded-md border bg-bg-primary px-3 py-2 text-sm text-text-primary"
          />
        </label>
        <label class="text-xs font-medium text-text-muted">
          Max trades/session
          <input
            v-model.number="form.maxTradesPerDay"
            aria-label="Maximum trades per session"
            type="number"
            min="1"
            required
            class="mt-1 w-full rounded-md border bg-bg-primary px-3 py-2 text-sm text-text-primary"
          />
        </label>
        <label class="text-xs font-medium text-text-muted">
          Max daily loss (₹)
          <input
            v-model.number="form.maxDailyLoss"
            aria-label="Maximum daily loss"
            type="number"
            min="1"
            required
            class="mt-1 w-full rounded-md border bg-bg-primary px-3 py-2 text-sm text-text-primary"
          />
        </label>
        <div class="flex items-end">
          <button
            type="submit"
            :disabled="loading"
            class="w-full rounded-md bg-brand px-4 py-2 text-sm font-semibold text-text-inverse hover:bg-brand/90 disabled:cursor-wait disabled:opacity-50"
          >
            {{ loading ? 'Replaying…' : 'Run paper replay' }}
          </button>
        </div>
      </form>
      <div class="mt-3 flex flex-wrap items-center gap-x-5 gap-y-1 text-[11px] text-text-muted">
        <span>Capital: {{ formatCurrency(form.initialCapital, { prefix: '₹' }) }}</span>
        <span>Max trades/session: {{ form.maxTradesPerDay }}</span>
        <span>Cost model: ₹20/order + NSE charges + slippage</span>
        <span class="text-info"
          >Scanner needs an as-of time; blank means full-session replay only.</span
        >
      </div>

      <section
        v-if="scanError || scanResult"
        class="mt-5 border-t border-border-subtle pt-4"
        aria-live="polite"
      >
        <p v-if="scanError" role="alert" class="mb-3 text-sm text-danger">{{ scanError }}</p>
        <template v-if="scanResult">
          <div class="mb-3 flex flex-wrap items-end justify-between gap-3">
            <div>
              <h3 class="text-sm font-semibold text-text-primary">
                Watchlist scan · {{ scanResult.sessionDate }} · {{ scanResult.asOfBarTime }} IST
              </h3>
              <p class="mt-1 text-xs text-text-muted">
                {{ scanResult.candidateCount }} open setups from {{ scanResult.scannedSymbols }} of
                {{ scanResult.watchlistSize }} active watchlist symbols ·
                {{ scanResult.results.length }} strategy checks.
              </p>
              <p class="mt-1 text-[11px] text-text-muted">
                Historical as-of replay over stored bars; no provider calls, no durable scan
                history, no live broker calls.
              </p>
            </div>
            <span
              class="rounded border border-brand/25 bg-brand-subtle px-2 py-1 font-mono text-[10px] text-brand"
              >{{ scanResult.liveBrokerCalls }} live broker calls</span
            >
          </div>
          <p v-if="scanResult.excludedSymbols.length" class="mb-3 text-xs text-warning">
            Excluded non-NSE watchlist entries: {{ scanResult.excludedSymbols.join(', ') }}
          </p>
          <p
            v-if="scanResult.results.length === 0"
            class="rounded-md border border-border-subtle p-3 text-xs text-text-muted"
          >
            No active NSE watchlist symbols are available for this scan.
          </p>
          <p
            v-else-if="scanResult.candidateCount === 0"
            class="mb-3 rounded-md border border-border-subtle p-3 text-xs text-text-muted"
          >
            No open setup was found at this as-of time. Incomplete bar coverage and kill-switch
            blocks remain visible in the table.
          </p>
          <div
            v-if="scanResult.results.length"
            class="overflow-x-auto rounded-md border border-border-subtle"
          >
            <table class="w-full min-w-[680px] text-left text-xs">
              <thead class="bg-bg-elevated text-[10px] uppercase tracking-wide text-text-muted">
                <tr>
                  <th class="px-3 py-2">Symbol</th>
                  <th class="px-3 py-2">Strategy</th>
                  <th class="px-3 py-2">Outcome</th>
                  <th class="px-3 py-2">Bars</th>
                  <th class="px-3 py-2 text-right">Trades / net</th>
                  <th class="px-3 py-2">Action</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border-subtle/60">
                <tr v-for="row in scanResult.results" :key="`${row.symbol}-${row.strategy}`">
                  <td class="px-3 py-2 font-semibold text-text-primary">{{ row.symbol }}</td>
                  <td class="px-3 py-2 text-text-secondary">{{ strategyLabel(row.strategy) }}</td>
                  <td class="px-3 py-2">
                    <span
                      :class="
                        row.status === 'CANDIDATE'
                          ? 'text-brand'
                          : row.status === 'INCOMPLETE_DATA'
                            ? 'text-warning'
                            : row.status === 'BLOCKED_KILL_SWITCH'
                              ? 'text-danger'
                              : 'text-text-muted'
                      "
                      >{{ row.status.replace(/_/g, ' ') }}</span
                    >
                  </td>
                  <td class="px-3 py-2 font-mono text-text-muted">
                    {{ row.receivedBars }}/{{ row.expectedBars }}
                  </td>
                  <td class="px-3 py-2 text-right font-mono text-text-secondary">
                    {{ row.completedTrades }} /
                    {{ formatSignedCurrency(row.netPnl, { prefix: '₹' }) }}
                  </td>
                  <td class="px-3 py-2">
                    <button
                      v-if="row.status === 'CANDIDATE'"
                      type="button"
                      class="rounded border border-brand/30 px-2 py-1 text-brand hover:bg-brand-subtle"
                      @click="useCandidate(row)"
                    >
                      Use in replay form</button
                    ><span v-else class="text-text-muted">—</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </section>
    </section>

    <ErrorMessage
      v-if="errorMessage"
      title="Couldn’t load the paper session"
      :message="errorMessage"
      action-label="Retry"
      :busy="loading"
      @action="runSession"
    />

    <div v-if="loading" class="card-panel flex min-h-36 items-center justify-center">
      <LoadingSpinner message="Replaying the complete NSE session…" />
    </div>

    <section
      v-else-if="!result && !errorMessage"
      class="card-panel flex min-h-40 flex-col items-center justify-center px-6 text-center"
      aria-live="polite"
    >
      <span
        class="mb-3 grid h-10 w-10 place-items-center rounded-full bg-bg-elevated text-brand"
        aria-hidden="true"
      >
        ↗
      </span>
      <h2 class="text-sm font-semibold text-text-primary">No session loaded</h2>
      <p class="mt-1 max-w-md text-xs leading-5 text-text-muted">
        Run a replay for stored 15-minute candles, or load the seeded TCS session to inspect the
        completed paper ledger.
      </p>
    </section>

    <template v-else-if="result">
      <section
        class="card-panel overflow-hidden"
        aria-labelledby="session-state-heading"
        aria-live="polite"
      >
        <div
          class="flex flex-wrap items-center justify-between gap-4 border-b border-border-subtle px-4 py-4 sm:px-5"
        >
          <div>
            <div class="mb-1 flex flex-wrap items-center gap-2">
              <h2 id="session-state-heading" class="text-sm font-semibold text-text-primary">
                {{ result.symbol }} · {{ result.sessionDate }}
              </h2>
              <StatusBadge :status="result.status" :label="result.status.replace(/_/g, ' ')" />
            </div>
            <p class="text-xs text-text-muted">
              {{ strategyLabel(result.strategy) }} · NSE · 15-minute bars
            </p>
          </div>
          <div class="text-right">
            <p class="text-[10px] font-semibold uppercase tracking-[0.16em] text-text-muted">
              Session completeness
            </p>
            <p class="mt-1 font-mono text-sm font-semibold text-text-primary">
              {{ result.receivedBars }} / {{ result.expectedBars }} bars
            </p>
          </div>
        </div>

        <div v-if="result.status === 'INCOMPLETE_DATA'" class="px-4 py-4 sm:px-5">
          <p class="text-sm font-semibold text-warning">
            Incomplete session — no paper ledger created
          </p>
          <p class="mt-1 text-xs text-text-muted">
            Every expected NSE bar is required. No simulated order, position, or trade was emitted.
          </p>
        </div>
        <div v-else-if="result.status === 'BLOCKED_KILL_SWITCH'" class="px-4 py-4 sm:px-5">
          <p class="text-sm font-semibold text-danger">Blocked by kill switch</p>
          <p class="mt-1 text-xs text-text-muted">No paper orders or positions were created.</p>
        </div>
        <div v-else class="grid grid-cols-2 divide-x divide-border-subtle sm:grid-cols-4">
          <div class="px-4 py-4 sm:px-5">
            <p class="text-[10px] font-semibold uppercase tracking-[0.14em] text-text-muted">
              Market state
            </p>
            <p class="mt-2 text-sm font-semibold text-text-primary">As of {{ resultAsOfLabel }}</p>
          </div>
          <div class="px-4 py-4 sm:px-5">
            <p class="text-[10px] font-semibold uppercase tracking-[0.14em] text-text-muted">
              Net P&amp;L
            </p>
            <p class="mt-2 font-mono text-lg font-semibold" :class="pnlClass(result.netPnl)">
              {{ formatSignedCurrency(result.netPnl, { prefix: '₹' }) }}
            </p>
          </div>
          <div class="px-4 py-4 sm:px-5">
            <p class="text-[10px] font-semibold uppercase tracking-[0.14em] text-text-muted">
              Costs
            </p>
            <p class="mt-2 font-mono text-sm font-semibold text-text-primary">
              {{ formatCurrency(totalCosts, { prefix: '₹' }) }}
            </p>
          </div>
          <div class="px-4 py-4 sm:px-5">
            <p class="text-[10px] font-semibold uppercase tracking-[0.14em] text-text-muted">
              Square-off
            </p>
            <p class="mt-2 text-sm font-semibold text-text-primary">{{ squareOffLabel }}</p>
          </div>
        </div>
      </section>

      <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
        <section class="card-panel p-4 sm:p-5" aria-labelledby="open-positions-heading">
          <div class="mb-3 flex items-center justify-between gap-3">
            <div>
              <h2 id="open-positions-heading" class="text-sm font-semibold text-text-primary">
                Open paper positions
              </h2>
              <p class="mt-1 text-xs text-text-muted">
                {{ openPositions.length }} currently open · limit 1
              </p>
            </div>
            <span
              class="rounded-md border border-border-subtle px-2 py-1 font-mono text-xs text-text-muted"
            >
              {{ result.openPositionsAtEnd }} open at selected time
            </span>
          </div>
          <div
            v-if="openPositions.length === 0"
            class="rounded-md border border-dashed border-border-default bg-bg-primary/40 px-4 py-6 text-center"
          >
            <p class="text-xs font-medium text-text-secondary">No open paper positions</p>
            <p class="mt-1 text-[11px] text-text-muted">
              {{
                result.status === 'COMPLETED'
                  ? 'The completed replay is flat; closed positions remain in the audit below.'
                  : 'No position record was created for this run.'
              }}
            </p>
          </div>
          <div v-else class="space-y-2">
            <article
              v-for="position in openPositions"
              :key="position.positionId"
              class="rounded-md border border-border-subtle bg-bg-primary/40 p-3"
            >
              <div class="flex items-center justify-between text-sm">
                <span class="font-semibold text-text-primary">{{ position.symbol }}</span>
                <span class="font-mono text-text-secondary">{{ position.quantity }} shares</span>
              </div>
              <p class="mt-2 text-xs text-text-muted">
                Entry {{ formatCurrency(position.entryPrice, { prefix: '₹' }) }} ·
                {{ timeLabel(position.entryTime) }}
              </p>
              <p class="mt-1 text-xs text-text-muted">
                Mark {{ formatCurrency(position.currentPrice, { prefix: '₹' }) }} · unrealized
                {{ formatSignedCurrency(position.unrealizedPnl, { prefix: '₹' }) }}
              </p>
            </article>
          </div>
        </section>

        <section class="card-panel p-4 sm:p-5" aria-labelledby="strategy-outcomes-heading">
          <div class="mb-3 flex items-center justify-between gap-3">
            <div>
              <h2 id="strategy-outcomes-heading" class="text-sm font-semibold text-text-primary">
                Strategy outcomes
              </h2>
              <p class="mt-1 text-xs text-text-muted">
                {{ result.strategy }} · {{ result.trades.length }} completed trades
              </p>
            </div>
            <span
              class="rounded-md border border-border-subtle px-2 py-1 font-mono text-[11px] text-text-muted"
            >
              max {{ result.maxConcurrentPositions }} concurrent
            </span>
          </div>
          <div
            v-if="result.trades.length === 0"
            class="rounded-md border border-dashed border-border-default bg-bg-primary/40 px-4 py-6 text-center text-xs text-text-muted"
          >
            {{
              result.status === 'COMPLETED'
                ? 'No strategy signals produced a paper trade in this session.'
                : 'No trades were recorded for this session.'
            }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="min-w-full text-left">
              <thead>
                <tr
                  class="border-b border-border-subtle text-[10px] font-semibold uppercase tracking-wider text-text-muted"
                >
                  <th class="py-2 pr-3">Exit</th>
                  <th class="py-2 pr-3">Time</th>
                  <th class="py-2 pr-3 text-right">Net P&amp;L</th>
                  <th class="py-2">Audit links</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border-subtle/60">
                <tr v-for="trade in result.trades" :key="trade.positionId" class="text-xs">
                  <td class="py-3 pr-3 font-medium text-text-primary">
                    {{ trade.exitReason.replace(/_/g, ' ') }}
                  </td>
                  <td class="py-3 pr-3 font-mono text-text-muted">
                    {{ timeLabel(trade.exitTime) }}
                  </td>
                  <td
                    class="py-3 pr-3 text-right font-mono font-semibold"
                    :class="pnlClass(trade.pnl)"
                  >
                    {{ formatSignedCurrency(trade.pnl, { prefix: '₹' }) }}
                  </td>
                  <td class="py-3 font-mono text-[10px] text-text-muted">
                    <span :title="trade.positionId">P {{ shortId(trade.positionId) }}</span>
                    <span :title="trade.entryOrderId" class="ml-1"
                      >· E {{ shortId(trade.entryOrderId) }}</span
                    >
                    <span :title="trade.exitOrderId" class="ml-1"
                      >· X {{ shortId(trade.exitOrderId) }}</span
                    >
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
        <section class="card-panel p-4 sm:p-5" aria-labelledby="position-ledger-heading">
          <h2 id="position-ledger-heading" class="mb-3 text-sm font-semibold text-text-primary">
            Position ledger
          </h2>
          <div
            v-if="result.positions.length === 0"
            class="py-6 text-center text-xs text-text-muted"
          >
            No position records.
          </div>
          <div v-else class="overflow-x-auto">
            <table class="min-w-full text-left">
              <thead>
                <tr
                  class="border-b border-border-subtle text-[10px] font-semibold uppercase tracking-wider text-text-muted"
                >
                  <th class="py-2 pr-3">Position</th>
                  <th class="py-2 pr-3">State</th>
                  <th class="py-2 pr-3">Costs</th>
                  <th class="py-2 text-right">Realized</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border-subtle/60">
                <tr v-for="position in result.positions" :key="position.positionId" class="text-xs">
                  <td class="py-3 pr-3 font-mono text-text-secondary" :title="position.positionId">
                    {{ shortId(position.positionId) }}
                  </td>
                  <td class="py-3 pr-3 text-text-muted">
                    {{ position.status }} · {{ position.exitReason.replace(/_/g, ' ') }}
                  </td>
                  <td class="py-3 pr-3 font-mono text-text-secondary">
                    {{ formatCurrency(position.costs, { prefix: '₹' }) }}
                  </td>
                  <td
                    class="py-3 text-right font-mono font-semibold"
                    :class="pnlClass(position.realizedPnl)"
                  >
                    {{ formatSignedCurrency(position.realizedPnl, { prefix: '₹' }) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="card-panel p-4 sm:p-5" aria-labelledby="order-ledger-heading">
          <div class="mb-3 flex items-center justify-between gap-3">
            <h2 id="order-ledger-heading" class="text-sm font-semibold text-text-primary">
              Simulated order ledger
            </h2>
            <span class="text-[10px] font-semibold uppercase tracking-wider text-brand"
              >No broker route</span
            >
          </div>
          <div v-if="result.orders.length === 0" class="py-6 text-center text-xs text-text-muted">
            No simulated orders.
          </div>
          <div v-else class="overflow-x-auto">
            <table class="min-w-full text-left">
              <thead>
                <tr
                  class="border-b border-border-subtle text-[10px] font-semibold uppercase tracking-wider text-text-muted"
                >
                  <th class="py-2 pr-3">Side</th>
                  <th class="py-2 pr-3">Fill time</th>
                  <th class="py-2 pr-3 text-right">Price</th>
                  <th class="py-2 text-right">Qty</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border-subtle/60">
                <tr v-for="order in result.orders" :key="order.orderId" class="text-xs">
                  <td
                    class="py-3 pr-3 font-semibold"
                    :class="order.side === 'BUY' ? 'text-brand' : 'text-warning'"
                  >
                    {{ order.side }}
                  </td>
                  <td class="py-3 pr-3 font-mono text-text-muted">
                    {{ timeLabel(order.filledAt) }} IST
                  </td>
                  <td class="py-3 pr-3 text-right font-mono text-text-secondary">
                    {{ formatCurrency(order.fillPrice, { prefix: '₹' }) }}
                  </td>
                  <td class="py-3 text-right font-mono text-text-secondary">
                    {{ order.quantity }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import ErrorMessage from '../components/ErrorMessage.vue'
import LoadingSpinner from '../components/LoadingSpinner.vue'
import StatusBadge from '../components/StatusBadge.vue'
import {
  runIntradayPaperSession,
  scanIntradayPaperWatchlist,
  type IntradayPaperScanResult,
  type IntradayPaperScanRow,
  type IntradayPaperSessionResult,
} from '../api/intraday'
import { formatAppError } from '../errors/appError'
import { formatCurrency, formatSignedCurrency } from '../utils/format'

const route = useRoute()
const AS_OF_SELECTION_REQUIRED = '__SELECT_AS_OF__'
const form = reactive({
  symbol: 'TCS',
  sessionDate: '2026-10-09',
  asOfBarTime: '',
  strategyType: 'INTRADAY_ORB',
  initialCapital: 10_000,
  quantity: 10,
  maxTradesPerDay: 1,
  maxDailyLoss: 1_000,
})

const isCandidateHandoff = computed(
  () => route.query.source === 'candidate-explorer' && typeof route.query.symbol === 'string'
)
const canScanWatchlist = computed(() =>
  Boolean(form.sessionDate && form.asOfBarTime && form.asOfBarTime !== AS_OF_SELECTION_REQUIRED)
)
const loading = ref(false)
const scanning = ref(false)
const scanResult = ref<IntradayPaperScanResult | null>(null)
const scanError = ref<string | null>(null)
const result = ref<IntradayPaperSessionResult | null>(null)
const resultAsOfLabel = ref<string | null>(null)
const errorMessage = ref<string | null>(null)
const sessionBarTimes = Array.from({ length: 25 }, (_, index) => {
  const totalMinutes = 9 * 60 + 15 + index * 15
  return `${String(Math.floor(totalMinutes / 60)).padStart(2, '0')}:${String(totalMinutes % 60).padStart(2, '0')}`
})
watch(
  () => [route.query.source, route.query.symbol] as const,
  ([source, candidateSymbol]) => {
    if (
      source !== 'candidate-explorer' ||
      typeof candidateSymbol !== 'string' ||
      !/^[A-Z0-9.-]{1,10}$/i.test(candidateSymbol)
    ) {
      return
    }
    form.symbol = candidateSymbol.toUpperCase()
    form.sessionDate = ''
    form.asOfBarTime = AS_OF_SELECTION_REQUIRED
    form.strategyType = ''
    scanResult.value = null
    result.value = null
    resultAsOfLabel.value = null
    errorMessage.value = null
  },
  { immediate: true }
)

const openPositions = computed(
  () => result.value?.positions.filter((position) => position.status === 'OPEN') ?? []
)
const totalCosts = computed(
  () => result.value?.positions.reduce((sum, position) => sum + position.costs, 0) ?? 0
)

const squareOffLabel = computed(() => {
  if (!result.value) return 'Not run'
  if (result.value.status === 'INCOMPLETE_DATA') return 'Not run · incomplete data'
  if (result.value.status === 'BLOCKED_KILL_SWITCH') return 'Blocked · kill switch'
  if (openPositions.value.length > 0) return 'Open · awaiting exit'
  if (result.value.trades.some((trade) => trade.exitReason === 'FORCED_SQUARE_OFF')) {
    return 'Forced · 15:15 bar proxy'
  }
  return result.value.trades.length > 0 ? 'Flat · all exits recorded' : 'Flat · no position opened'
})

function pnlClass(value: number): string {
  return Number.isFinite(value) && value >= 0 ? 'text-success' : 'text-danger'
}

function strategyLabel(strategy: string): string {
  const labels: Record<string, string> = {
    INTRADAY_ORB: 'Opening range breakout',
    INTRADAY_VWAP_PULLBACK: 'VWAP pullback',
    INTRADAY_RSI2: 'RSI(2) reversal',
  }
  return labels[strategy] ?? strategy
}

function timeLabel(value: string): string {
  const match = value.match(/T(\d{2}:\d{2})/)
  return match?.[1] ?? value
}

function shortId(value: string): string {
  return value.length > 18 ? `${value.slice(0, 8)}…${value.slice(-5)}` : value
}

async function scanWatchlist() {
  if (!canScanWatchlist.value) return
  scanning.value = true
  scanError.value = null
  scanResult.value = null
  try {
    scanResult.value = await scanIntradayPaperWatchlist({
      sessionDate: form.sessionDate,
      asOfBarTime: form.asOfBarTime,
      riskConfig: {
        initialCapital: form.initialCapital,
        quantity: form.quantity,
        maxTradesPerDay: form.maxTradesPerDay,
        maxDailyLoss: form.maxDailyLoss,
      },
    })
  } catch (error) {
    scanError.value = formatAppError(error, {
      title: 'Couldn’t scan the active watchlist',
      operation: 'mutation',
      retryLabel: 'Retry',
    }).message
  } finally {
    scanning.value = false
  }
}

function useCandidate(candidate: IntradayPaperScanRow) {
  form.symbol = candidate.symbol
  form.strategyType = candidate.strategy
  form.sessionDate = scanResult.value?.sessionDate ?? form.sessionDate
  const scanAsOfBarTime = scanResult.value?.asOfBarTime
  form.asOfBarTime = scanAsOfBarTime ? scanAsOfBarTime.slice(0, 5) : form.asOfBarTime
}

async function runSession() {
  if (
    isCandidateHandoff.value &&
    (!form.sessionDate || !form.strategyType || form.asOfBarTime === AS_OF_SELECTION_REQUIRED)
  ) {
    errorMessage.value =
      'Choose an intraday session date, strategy, and cutoff (or End of session) before replay.'
    return
  }
  const submittedAsOfLabel = form.asOfBarTime ? `${form.asOfBarTime} IST` : 'end of session'
  loading.value = true
  result.value = null
  resultAsOfLabel.value = null
  errorMessage.value = null
  try {
    result.value = await runIntradayPaperSession({
      symbol: form.symbol.trim().toUpperCase(),
      sessionDate: form.sessionDate,
      ...(form.asOfBarTime ? { asOfBarTime: form.asOfBarTime } : {}),
      strategyType: form.strategyType,
      riskConfig: {
        initialCapital: form.initialCapital,
        quantity: form.quantity,
        maxTradesPerDay: form.maxTradesPerDay,
        maxDailyLoss: form.maxDailyLoss,
      },
    })
    resultAsOfLabel.value = submittedAsOfLabel
  } catch (error) {
    errorMessage.value = formatAppError(error, {
      title: 'Couldn’t load the paper session',
      operation: 'mutation',
      retryLabel: 'Retry',
    }).message
  } finally {
    loading.value = false
  }
}

function loadSeededSession() {
  form.symbol = 'TCS'
  form.sessionDate = '2026-10-09'
  form.asOfBarTime = ''
  form.strategyType = 'INTRADAY_ORB'
  form.initialCapital = 10_000
  form.quantity = 10
  form.maxTradesPerDay = 1
  form.maxDailyLoss = 1_000
  void runSession()
}
</script>
