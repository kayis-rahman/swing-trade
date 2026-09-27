<template>
  <div class="view-shell p-4 sm:p-6 animate-fade-in">
    <!-- Page Header -->
    <div class="mb-6 flex items-center justify-between">
      <div>
        <h1 class="font-display text-2xl font-semibold text-text-primary">Watchlist</h1>
        <p class="mt-1 text-sm text-text-muted">
          {{ watchlist.length }} stocks being monitored · {{ buySignalCount }} BUY signal{{
            buySignalCount === 1 ? '' : 's'
          }}
        </p>
      </div>
      <button
        class="flex items-center gap-2 rounded-md bg-brand px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-brand/90"
        @click="showAddForm = !showAddForm"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4v16m8-8H4"
          />
        </svg>
        Add Stock
      </button>
    </div>

    <div class="mb-4 flex flex-wrap items-center gap-2" aria-label="Watchlist filters">
      <button
        v-for="filter in filters"
        :key="filter.value"
        type="button"
        class="rounded-full border px-3 py-1.5 text-xs font-semibold transition-colors"
        :class="
          signalFilter === filter.value
            ? 'border-brand bg-brand text-white'
            : 'border-border-subtle text-text-muted hover:border-brand/50 hover:text-text-primary'
        "
        @click="signalFilter = filter.value"
      >
        {{ filter.label }} <span class="ml-1 opacity-70">{{ filterCount(filter.value) }}</span>
      </button>
    </div>

    <!-- Add Stock Form -->
    <div v-if="showAddForm" class="mb-6 card-panel p-5">
      <h3 class="mb-3 text-sm font-semibold text-text-primary">Add Stock to Watchlist</h3>
      <form class="flex flex-col sm:flex-row gap-3" @submit.prevent="handleSubmit">
        <div class="flex-1">
          <label class="mb-1 block text-xs font-medium text-text-muted">Symbol</label>
          <input
            v-model="newSymbol"
            type="text"
            placeholder="e.g. RELIANCE"
            maxlength="10"
            required
            class="w-full rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted/50 focus:outline-none focus:ring-2 focus:ring-brand/30"
          />
        </div>
        <div class="flex-1">
          <label class="mb-1 block text-xs font-medium text-text-muted"
            >Company Name (optional)</label
          >
          <input
            v-model="newName"
            type="text"
            placeholder="e.g. Reliance Industries"
            class="w-full rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted/50 focus:outline-none focus:ring-2 focus:ring-brand/30"
          />
        </div>
        <div class="flex items-end gap-2">
          <button
            type="submit"
            :disabled="adding"
            class="rounded-md bg-brand px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand/90 disabled:opacity-50"
          >
            {{ adding ? 'Adding...' : 'Add' }}
          </button>
          <button
            type="button"
            class="rounded-md border border-border-subtle px-3 py-2 text-sm text-text-muted transition-colors hover:bg-bg-hover"
            @click="showAddForm = false"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>

    <div v-if="staleWarning" role="status" class="mb-4 text-sm text-warning">
      {{ staleWarning }}
    </div>

    <ErrorMessage
      v-if="mutationError"
      class="mb-4"
      v-bind="mutationError"
      :focus-on-mount="true"
      @action="loadWatchlist"
    />

    <ErrorBoundary :error="Boolean(error)">
      <template #error>
        <div class="flex flex-col items-center justify-center py-20">
          <p class="text-sm text-danger">
            {{ errorMessage }}
          </p>
          <button
            class="mt-2 rounded-md bg-brand px-3 py-1.5 text-xs font-medium text-white"
            @click="loadWatchlist"
          >
            Retry
          </button>
        </div>
      </template>
      <div v-if="loading" class="flex items-center justify-center py-20">
        <LoadingSpinner message="Loading watchlist..." />
      </div>

      <template v-else>
        <div class="card-panel overflow-x-auto">
          <table class="min-w-full">
            <thead>
              <tr class="border-b border-border-subtle bg-bg-primary/50">
                <th
                  class="px-5 py-3 text-left text-xs font-semibold uppercase tracking-wider text-text-muted"
                >
                  Symbol
                </th>
                <th
                  class="px-5 py-3 text-left text-xs font-semibold uppercase tracking-wider text-text-muted"
                >
                  Name
                </th>
                <th
                  class="px-5 py-3 text-left text-xs font-semibold uppercase tracking-wider text-text-muted"
                >
                  Exchange
                </th>
                <th
                  class="px-5 py-3 text-center text-xs font-semibold uppercase tracking-wider text-text-muted"
                >
                  Status
                </th>
                <th
                  class="px-5 py-3 text-center text-xs font-semibold uppercase tracking-wider text-text-muted"
                >
                  Latest signal
                </th>
                <th
                  class="px-5 py-3 text-right text-xs font-semibold uppercase tracking-wider text-text-muted"
                >
                  Candles
                </th>
                <th
                  class="px-5 py-3 text-right text-xs font-semibold uppercase tracking-wider text-text-muted"
                >
                  Actions
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border-subtle/50">
              <tr
                v-for="entry in filteredWatchlist"
                :key="entry.symbol"
                class="transition-colors hover:bg-bg-hover"
              >
                <td class="px-5 py-4 text-sm font-semibold text-text-primary">
                  {{ entry.symbol }}
                </td>
                <td class="px-5 py-4 text-sm text-text-secondary">
                  {{ entry.name || '—' }}
                </td>
                <td class="px-5 py-4 text-sm text-text-muted">
                  {{ entry.exchange }}
                </td>
                <td class="px-5 py-4 text-center">
                  <button
                    class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors"
                    :class="
                      entry.isActive
                        ? 'bg-success-bg text-success hover:bg-success-bg/80'
                        : 'bg-bg-hover text-text-muted hover:bg-border-subtle'
                    "
                    @click="toggleEntry(entry.symbol)"
                  >
                    {{ entry.isActive ? 'Active' : 'Inactive' }}
                  </button>
                </td>
                <td class="px-5 py-4 text-center">
                  <span
                    v-if="latestSignal(entry.symbol)"
                    class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold"
                    :class="signalClass(latestSignal(entry.symbol)!.direction)"
                  >
                    {{ latestSignal(entry.symbol)!.direction }}
                  </span>
                  <span v-else class="text-xs text-text-muted">No signal</span>
                  <span
                    v-if="consensusFor(entry.symbol)"
                    class="ml-1.5 rounded-full bg-brand/10 px-2 py-0.5 text-[11px] font-semibold text-brand"
                    data-testid="consensus-badge"
                    title="Variants signalling BUY / variants with a signal"
                    >{{ consensusFor(entry.symbol) }}</span
                  >
                </td>
                <td class="px-5 py-4 text-right text-sm text-text-secondary">
                  {{ entry.candleCount ?? 0 }}
                </td>
                <td class="px-5 py-4 text-right">
                  <button
                    class="rounded-md p-1 text-text-muted transition-colors hover:bg-danger-bg hover:text-danger"
                    :title="'Remove ' + entry.symbol"
                    @click="removeEntry(entry.symbol)"
                  >
                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M19 7l-.867 12.14A2 2 0 0116.237 21H7.763a2 2 0 01-1.896-1.86L5 7m1 0h4m-4 0V3h4v4m-4 0H5m14-4H15m4 0v4m-4-4h1m-5 0h.01"
                      />
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div
          v-if="watchlist.length > 0 && filteredWatchlist.length === 0"
          class="mt-6 text-center text-sm text-text-muted"
        >
          No {{ signalFilter === 'BUY' ? 'BUY' : '' }} signals in the watchlist.
        </div>
        <div v-else-if="watchlist.length === 0" class="mt-6 text-center text-sm text-text-muted">
          No stocks in watchlist. Click "Add Stock" to get started.
        </div>
      </template>
    </ErrorBoundary>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  getWatchlist,
  addToWatchlist,
  removeFromWatchlist,
  toggleWatchlistActive,
} from '../api/watchlist'
import { getSignals } from '../api/signals'
import { getStrategies } from '../api/strategies'
import { groupSignalsBySymbol } from '../utils/signalGrouping'
import type { Signal, WatchlistEntry } from '../api/types'
import LoadingSpinner from '../components/LoadingSpinner.vue'
import ErrorBoundary from '../components/ErrorBoundary.vue'
import ErrorMessage from '../components/ErrorMessage.vue'
import { useAsyncData } from '../composables/useAsyncData'
import { formatAppError, type FormattedErrorDetail } from '../errors/appError'

interface MutationErrorPresentation {
  title: string
  message: string
  details?: FormattedErrorDetail[]
  actionLabel?: string
}

const { loading, error, errorMessage, execute } = useAsyncData<void>()
const watchlist = ref<WatchlistEntry[]>([])
const signals = ref<Signal[]>([])
const signalFilter = ref<'ALL' | 'BUY'>('ALL')
const filters = [
  { value: 'ALL' as const, label: 'All stocks' },
  { value: 'BUY' as const, label: 'BUY signals' },
]
const mutationError = ref<MutationErrorPresentation | null>(null)
const staleWarning = ref('')
const showAddForm = ref(false)
const newSymbol = ref('')
const newName = ref('')
const adding = ref(false)

const latestSignals = computed(() => {
  const bySymbol = new Map<string, Signal>()
  for (const signal of signals.value) {
    const current = bySymbol.get(signal.symbol)
    if (!current || signal.timestamp > current.timestamp) bySymbol.set(signal.symbol, signal)
  }
  return bySymbol
})

const latestSignal = (symbol: string) => latestSignals.value.get(symbol)

// Only shown when at least one variant (strategy-tagged) signal exists for the symbol.
const variantIds = ref<ReadonlySet<string>>(new Set())
const consensusBySymbol = computed(() => {
  const labels = new Map<string, string>()
  if (variantIds.value.size === 0) return labels
  for (const group of groupSignalsBySymbol(signals.value, [], variantIds.value)) {
    labels.set(group.symbol, group.consensusLabel)
  }
  return labels
})
const consensusFor = (symbol: string) => consensusBySymbol.value.get(symbol) ?? ''
const buySignalCount = computed(
  () => watchlist.value.filter((entry) => latestSignal(entry.symbol)?.direction === 'BUY').length
)
const filteredWatchlist = computed(() =>
  signalFilter.value === 'BUY'
    ? watchlist.value.filter((entry) => latestSignal(entry.symbol)?.direction === 'BUY')
    : watchlist.value
)
const filterCount = (filter: 'ALL' | 'BUY') =>
  filter === 'BUY' ? buySignalCount.value : watchlist.value.length
const signalClass = (direction: Signal['direction']) =>
  direction === 'BUY'
    ? 'bg-success-bg text-success'
    : direction === 'SELL'
      ? 'bg-danger-bg text-danger'
      : 'bg-bg-hover text-text-muted'

const loadWatchlist = async (): Promise<boolean> => {
  await execute(async () => {
    const [entries, latest] = await Promise.all([getWatchlist(), getSignals()])
    watchlist.value = entries
    signals.value = latest
    // Supplementary: consensus badges simply stay hidden if variants can't be listed.
    getStrategies()
      .then((variants) => {
        variantIds.value = new Set(variants.map((v) => v.variantId))
      })
      .catch(() => {
        variantIds.value = new Set()
      })
  })
  return error.value === null
}

async function refreshAfterConfirmedMutation() {
  staleWarning.value = ''
  if (!(await loadWatchlist())) {
    staleWarning.value =
      'Watchlist update confirmed, but refresh failed. Displayed rows may be stale.'
    error.value = null
    errorMessage.value = ''
  }
}

function showMutationError(errorLike: unknown, title: string) {
  const formatted = formatAppError(errorLike, {
    title,
    operation: 'mutation',
    refreshLabel: 'Refresh watchlist',
  })
  mutationError.value = {
    title: formatted.title,
    message: formatted.message,
    details: formatted.details,
    actionLabel: formatted.action?.label,
  }
}

const handleSubmit = async () => {
  if (!newSymbol.value.trim()) return
  adding.value = true
  mutationError.value = null
  const symbol = newSymbol.value.trim().toUpperCase()
  try {
    const added = await addToWatchlist(newSymbol.value, newName.value || undefined)
    watchlist.value = [...watchlist.value.filter((item) => item.symbol !== added.symbol), added]
    newSymbol.value = ''
    newName.value = ''
    showAddForm.value = false
    await refreshAfterConfirmedMutation()
  } catch (err: unknown) {
    showMutationError(err, `Couldn’t add ${symbol || 'stock'}`)
  } finally {
    adding.value = false
  }
}

const removeEntry = async (symbol: string) => {
  if (!confirm(`Remove ${symbol} from watchlist?`)) return
  mutationError.value = null
  try {
    await removeFromWatchlist(symbol)
    watchlist.value = watchlist.value.filter((item) => item.symbol !== symbol)
    await refreshAfterConfirmedMutation()
  } catch (err: unknown) {
    showMutationError(err, `Couldn’t remove ${symbol}`)
  }
}

const toggleEntry = async (symbol: string) => {
  const entry = watchlist.value.find((item) => item.symbol === symbol)
  if (!entry) return
  mutationError.value = null
  try {
    const updated = await toggleWatchlistActive(symbol, !entry.isActive)
    watchlist.value = watchlist.value.map((item) => (item.symbol === symbol ? updated : item))
    await refreshAfterConfirmedMutation()
  } catch (err: unknown) {
    showMutationError(err, `Couldn’t update ${symbol}`)
  }
}

onMounted(() => {
  loadWatchlist()
})
</script>
