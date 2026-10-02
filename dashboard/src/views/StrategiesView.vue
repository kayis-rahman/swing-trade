<template>
  <div class="view-shell p-4 sm:p-6 animate-fade-in">
    <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p class="text-xs font-medium uppercase tracking-[0.16em] text-text-muted">
          Strategy desk / configuration
        </p>
        <h1
          class="mt-2 font-display text-2xl font-semibold tracking-tight text-text-primary sm:text-3xl"
        >
          Strategies
        </h1>
        <p class="mt-1 max-w-xl text-sm leading-6 text-text-muted">
          Review the current version and operating mode for every configured strategy variant.
        </p>
      </div>
      <button
        type="button"
        class="rounded-lg border border-border-subtle bg-bg-surface px-3 py-2 text-sm font-medium text-text-muted transition-colors hover:border-border-default hover:text-text-primary disabled:opacity-50"
        :disabled="loading"
        @click="store.load"
      >
        {{ loading ? 'Refreshing...' : 'Refresh' }}
      </button>
    </div>

    <div v-if="loading && !strategies" class="flex items-center justify-center py-20" role="status">
      <span class="text-sm text-text-muted">Loading strategies...</span>
    </div>

    <section v-else-if="error && !strategies" class="card-panel p-6 text-center" role="alert">
      <h2 class="text-base font-semibold text-text-primary">Couldn’t load strategies</h2>
      <p class="mt-1 text-sm text-text-muted">
        {{ errorMessage || 'Strategy configuration is unavailable.' }}
      </p>
      <button
        type="button"
        class="mt-4 rounded-lg bg-brand px-4 py-2 text-sm font-semibold text-brand-text hover:bg-brand-hover"
        @click="store.retry"
      >
        Retry
      </button>
    </section>

    <section v-else-if="strategies?.length === 0" class="card-panel p-10 text-center">
      <h2 class="text-base font-semibold text-text-primary">No strategies configured</h2>
      <p class="mt-1 text-sm text-text-muted">
        Add a strategy variant through the configuration API to see it here.
      </p>
    </section>

    <template v-else>
      <p
        v-if="error"
        class="mb-4 rounded-lg border border-warning/30 bg-warning-bg px-4 py-3 text-sm text-warning"
        role="status"
      >
        Showing the last successful configuration. Refresh failed: {{ errorMessage }}
      </p>
      <p
        v-if="championWarning"
        class="mb-4 rounded-lg border border-warning/30 bg-warning-bg px-4 py-3 text-sm text-warning"
        role="alert"
        data-testid="champion-warning"
      >
        {{ championWarning }}
      </p>
      <div class="grid gap-4 lg:grid-cols-2">
        <article
          v-for="strategy in strategies"
          :key="`${strategy.variantId}-${strategy.version}`"
          class="card-panel p-5 transition-colors hover:border-border-default"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <p class="truncate text-xs font-medium uppercase tracking-[0.14em] text-text-muted">
                {{ strategy.strategyType }} · v{{ strategy.version }}
              </p>
              <h2 class="mt-2 truncate text-lg font-semibold text-text-primary">
                {{ strategy.variantId }}
              </h2>
            </div>
            <span
              class="shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold"
              :class="modeClass(strategy.mode)"
              :aria-label="`Mode: ${strategy.mode}`"
            >
              {{ modeLabel(strategy.mode) }}
            </span>
          </div>
          <dl class="mt-5 grid grid-cols-2 gap-4 border-t border-border-subtle pt-4">
            <div>
              <dt class="text-xs text-text-muted">Paper capital</dt>
              <dd class="mt-1 text-sm font-semibold text-text-primary">
                {{ formatCapital(strategy.paperCapital) }}
              </dd>
            </div>
            <div>
              <dt class="text-xs text-text-muted">Status</dt>
              <dd class="mt-1 text-sm font-semibold text-text-primary">
                {{ strategy.current ? 'Current' : 'Historical' }}
              </dd>
            </div>
          </dl>
          <p v-if="strategy.notes" class="mt-4 text-sm leading-6 text-text-muted">
            {{ strategy.notes }}
          </p>

          <div v-if="strategy.current" class="mt-4 border-t border-border-subtle pt-4">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-xs text-text-muted">Mode</span>
              <template v-for="mode in TOGGLE_MODES" :key="mode">
                <button
                  type="button"
                  :data-testid="`mode-${strategy.variantId}-${mode}`"
                  :aria-pressed="strategy.mode === mode"
                  :disabled="strategy.mode === mode || pendingVariant === strategy.variantId"
                  class="rounded-md border px-2.5 py-1 text-xs font-medium transition-colors disabled:opacity-60"
                  :class="
                    strategy.mode === mode
                      ? 'border-brand bg-brand-subtle text-brand'
                      : 'border-border-subtle text-text-muted hover:text-text-primary'
                  "
                  @click="requestMode(strategy, mode)"
                >
                  {{ modeLabel(mode) }}
                </button>
              </template>
              <button
                type="button"
                :data-testid="`edit-params-${strategy.variantId}`"
                class="ml-auto rounded-md border border-border-subtle px-2.5 py-1 text-xs font-medium text-text-muted hover:text-text-primary"
                @click="openEditor(strategy)"
              >
                Edit params
              </button>
            </div>
            <div
              v-if="confirmVariant === strategy.variantId"
              class="mt-2 flex flex-wrap items-center gap-2 text-xs text-warning"
            >
              <span>Promoting to CHAMPION makes this variant drive live decisions.</span>
              <button
                type="button"
                class="rounded-md bg-brand px-2.5 py-1 font-semibold text-brand-text"
                @click="applyMode(strategy.variantId, 'CHAMPION')"
              >
                Confirm promote
              </button>
              <button type="button" class="text-text-muted" @click="confirmVariant = null">
                Cancel
              </button>
            </div>
            <p
              v-if="modeErrors[strategy.variantId]"
              class="mt-2 text-xs text-danger"
              role="alert"
              :data-testid="`mode-error-${strategy.variantId}`"
            >
              {{ modeErrors[strategy.variantId] }}
            </p>
            <StrategyParamsEditor
              v-if="editingVariant === strategy.variantId"
              :key="`${strategy.variantId}-${strategy.version}`"
              :strategy="strategy"
              :type-info="typeInfoFor(strategy.strategyType)"
              :saving="saving"
              :error="editError"
              @save="saveParams(strategy, $event)"
              @cancel="closeEditor"
            />
          </div>
        </article>
      </div>

      <section v-if="shadowVariantIds.length > 0" class="mt-8">
        <h2 class="font-display text-lg font-semibold text-text-primary">Promotion eligibility</h2>
        <p class="mt-1 max-w-2xl text-sm leading-6 text-text-muted">
          Compares each shadow variant against the current champion. This is informational only —
          promotion still requires a separate manual action.
        </p>
        <div class="mt-4 grid gap-4 lg:grid-cols-2">
          <article
            v-for="variantId in shadowVariantIds"
            :key="variantId"
            class="card-panel p-5"
            data-testid="promotion-eligibility-card"
          >
            <div class="flex items-start justify-between gap-4">
              <h3 class="text-sm font-semibold text-text-primary">{{ variantId }}</h3>
              <span
                v-if="promotionState(variantId)?.result"
                class="shrink-0 rounded-full border border-border-default bg-bg-hover px-2.5 py-1 text-xs font-semibold text-text-primary"
              >
                {{ statusLabel(promotionState(variantId)!.result!.status) }}
              </span>
            </div>

            <p
              v-if="promotionState(variantId)?.loading"
              class="mt-3 text-sm text-text-muted"
              role="status"
            >
              Checking promotion eligibility...
            </p>

            <p
              v-else-if="promotionState(variantId)?.error"
              class="mt-3 text-sm text-danger"
              role="alert"
            >
              {{ promotionState(variantId)?.error }}
            </p>

            <template v-else-if="promotionState(variantId)?.result">
              <p
                v-if="promotionState(variantId)!.result!.status === 'INSUFFICIENT_SAMPLE'"
                class="mt-3 rounded-lg border border-border-subtle bg-bg-hover px-3 py-2 text-sm text-text-muted"
              >
                Not enough data yet to judge — this is not a rejection, just an early result.
              </p>

              <ul class="mt-4 space-y-2 border-t border-border-subtle pt-4">
                <li
                  v-for="condition in promotionState(variantId)!.result!.conditions"
                  :key="condition.name"
                  class="text-sm"
                >
                  <div class="flex items-center gap-2">
                    <span
                      class="shrink-0 rounded-full px-2 py-0.5 text-xs font-semibold"
                      :class="
                        condition.met
                          ? 'bg-bg-hover text-text-primary'
                          : 'bg-bg-hover text-text-muted'
                      "
                    >
                      {{ condition.met ? 'Met' : 'Not met' }}
                    </span>
                    <span class="font-medium text-text-primary">{{ condition.name }}</span>
                  </div>
                  <p class="mt-1 text-text-muted">
                    {{ condition.actualValue }} (threshold: {{ condition.threshold }})
                  </p>
                  <p v-if="condition.note" class="mt-1 text-text-muted">{{ condition.note }}</p>
                </li>
              </ul>

              <div
                v-if="promotionState(variantId)!.result!.dataLimitations.length > 0"
                class="mt-4 border-t border-border-subtle pt-4"
              >
                <p class="text-xs font-medium uppercase tracking-[0.1em] text-text-muted">
                  Data limitations
                </p>
                <ul class="mt-2 space-y-1 text-sm text-text-muted">
                  <li
                    v-for="(limitation, index) in promotionState(variantId)!.result!
                      .dataLimitations"
                    :key="index"
                  >
                    {{ limitation }}
                  </li>
                </ul>
              </div>
            </template>
          </article>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import StrategyParamsEditor from '../components/StrategyParamsEditor.vue'
import { useStrategiesStore } from '../stores/strategies'
import type {
  PromotionEligibilityStatus,
  StrategyConfig,
  StrategyMode,
  StrategyTypeInfo,
} from '../api/types'

const store = useStrategiesStore()
const strategies = computed(() => store.strategies)
const loading = computed(() => store.loading)
const error = computed(() => store.error)
const errorMessage = computed(() => store.errorMessage)

const championVariantId = computed<string | null>(() => {
  const champion = (strategies.value ?? []).find(
    (strategy: StrategyConfig) => strategy.current && strategy.mode === 'CHAMPION'
  )
  return champion?.variantId ?? null
})

const shadowVariantIds = computed<string[]>(() => {
  const champion = championVariantId.value
  if (!champion) return []
  return (strategies.value ?? [])
    .filter((strategy: StrategyConfig) => strategy.current && strategy.mode === 'SHADOW')
    .map((strategy: StrategyConfig) => strategy.variantId)
    .filter((variantId: string) => variantId !== champion)
})

const TOGGLE_MODES: StrategyMode[] = ['SHADOW', 'CHAMPION']

const championWarning = computed<string>(() => {
  const list = strategies.value
  if (!list || list.length === 0) return ''
  const count = list.filter((s) => s.current && s.mode === 'CHAMPION').length
  if (count === 0) {
    return 'No CHAMPION strategy is set. Live decisions have no champion until one is promoted.'
  }
  if (count > 1) {
    return `${count} champions are set. Only one current variant may be CHAMPION; the API rejects additional promotions.`
  }
  return ''
})

const pendingVariant = ref<string | null>(null)
const confirmVariant = ref<string | null>(null)
const modeErrors = ref<Record<string, string>>({})
const editingVariant = ref<string | null>(null)
const editError = ref('')
const saving = ref(false)

const typeInfoFor = (type: string): StrategyTypeInfo | null =>
  store.strategyTypes?.find((t) => t.type === type) ?? null

function requestMode(strategy: StrategyConfig, mode: StrategyMode): void {
  modeErrors.value = { ...modeErrors.value, [strategy.variantId]: '' }
  if (mode === 'CHAMPION') {
    confirmVariant.value = strategy.variantId
    return
  }
  void applyMode(strategy.variantId, mode)
}

async function applyMode(variantId: string, mode: StrategyMode): Promise<void> {
  confirmVariant.value = null
  pendingVariant.value = variantId
  const result = await store.changeMode(variantId, mode)
  pendingVariant.value = null
  if (!result.ok) {
    const hint = result.outcomeUnknown
      ? ' The change could not be confirmed; refresh to see the current mode.'
      : ''
    modeErrors.value = { ...modeErrors.value, [variantId]: result.message + hint }
  }
}

function openEditor(strategy: StrategyConfig): void {
  editError.value = ''
  editingVariant.value = strategy.variantId
  if (!store.strategyTypes) void store.loadStrategyTypes()
}

function closeEditor(): void {
  editingVariant.value = null
  editError.value = ''
}

async function saveParams(strategy: StrategyConfig, params: Record<string, unknown>) {
  saving.value = true
  editError.value = ''
  const result = await store.saveConfig({
    variantId: strategy.variantId,
    strategyType: strategy.strategyType,
    params,
    overlays: strategy.overlays,
    mode: strategy.mode,
    paperCapital: strategy.paperCapital,
    notes: strategy.notes,
  })
  saving.value = false
  if (result.ok) closeEditor()
  else {
    editError.value =
      result.message +
      (result.outcomeUnknown ? ' The save could not be confirmed; refresh before retrying.' : '')
  }
}

const promotionState = (variantId: string) => store.promotionEligibility[variantId]

const modeLabel = (mode: StrategyMode): string => mode.replace('_', ' ')

const statusLabel = (status: PromotionEligibilityStatus): string =>
  ({
    ELIGIBLE: 'Eligible',
    NOT_ELIGIBLE: 'Not eligible',
    INSUFFICIENT_SAMPLE: 'Insufficient sample',
  })[status]

const modeClass = (mode: StrategyMode): string =>
  ({
    CHAMPION: 'bg-success-bg text-success',
    SHADOW: 'bg-brand-subtle text-brand',
    BACKTEST_ONLY: 'bg-warning-bg text-warning',
    OFF: 'bg-bg-hover text-text-muted',
  })[mode]

const formatCapital = (value: number): string =>
  new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    maximumFractionDigits: 0,
  }).format(value)

function loadPromotionEligibilityForShadows(): void {
  for (const variantId of shadowVariantIds.value) {
    if (!store.promotionEligibility[variantId]) {
      void store.loadPromotionEligibility(variantId)
    }
  }
}

watch(shadowVariantIds, loadPromotionEligibilityForShadows)

onMounted(() => {
  void store.loadStrategyTypes()
  if (!store.strategies) void store.load()
  else loadPromotionEligibilityForShadows()
})
</script>
