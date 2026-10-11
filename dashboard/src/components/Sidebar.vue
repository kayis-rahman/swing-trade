<template>
  <div
    v-if="mobileOpen"
    class="fixed inset-0 z-30 bg-black/50 lg:hidden"
    aria-hidden="true"
    @click="$emit('close')"
  />
  <aside
    class="fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-border-subtle bg-bg-surface transition-transform duration-300 lg:static lg:z-auto lg:translate-x-0 lg:transition-[width]"
    :class="[mobileOpen ? 'translate-x-0' : '-translate-x-full', collapsed ? 'lg:w-16' : 'lg:w-56']"
  >
    <!-- Logo -->
    <div class="flex h-14 items-center justify-between border-b border-border-subtle px-4">
      <template v-if="!collapsed">
        <div class="flex items-center gap-2">
          <div class="flex h-7 w-7 items-center justify-center rounded-md bg-brand/10">
            <span class="text-sm font-bold text-brand">S</span>
          </div>
          <span class="font-display text-base font-semibold text-text-primary">Swing Trade</span>
        </div>
      </template>
      <template v-else>
        <div class="mx-auto flex h-7 w-7 items-center justify-center rounded-md bg-brand/10">
          <span class="text-sm font-bold text-brand">S</span>
        </div>
      </template>
      <button
        class="flex h-9 w-9 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary lg:hidden"
        aria-label="Close navigation"
        @click="$emit('close')"
      >
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 space-y-1 overflow-y-auto px-2 py-3">
      <template v-for="item in navItems" :key="item.path">
        <p
          v-if="!collapsed && item.group"
          class="px-3 pt-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-text-muted"
        >
          {{ item.group }}
        </p>
        <router-link
          :to="item.path"
          class="flex items-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium transition-all"
          :class="
            $route.path === item.path
              ? 'bg-brand-subtle text-brand'
              : 'text-text-muted hover:bg-bg-hover hover:text-text-primary'
          "
          :title="collapsed ? item.label : undefined"
          @click="$emit('close')"
        >
          <!-- iconPaths is a static, internal SVG path map; no user input reaches v-html. -->
          <!-- eslint-disable-next-line vue/no-v-html -->
          <svg class="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" v-html="item.icon" />
          <template v-if="!collapsed">
            <span class="flex-1">{{ item.label }}</span>
            <span
              v-if="item.badge"
              class="ml-1 inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-brand/10 px-1.5 text-[10px] font-semibold text-brand"
              >{{ item.badge }}</span
            >
          </template>
        </router-link>
      </template>
    </nav>

    <!-- Footer -->
    <div class="border-t border-border-subtle p-3">
      <div v-if="!collapsed" class="rounded-md border border-border-subtle bg-bg-primary/50 p-3">
        <div class="mb-1 flex items-center gap-2">
          <span class="inline-block h-2 w-2 rounded-full" :class="healthDotClass" />
          <span class="text-xs font-medium text-text-muted">System Status</span>
        </div>
        <div
          role="status"
          aria-live="polite"
          :aria-label="`Backend health: ${healthLabel}`"
          class="text-xs font-medium"
          :class="healthTextClass"
        >
          {{ healthLabel }}
        </div>
        <div class="mt-0.5 text-[10px] text-text-muted">Sync {{ lastSync }}</div>
        <div class="mt-2 border-t border-border-subtle pt-2 text-[10px] text-text-muted">
          Broker: <span class="font-medium text-text-secondary">{{ brokerLabel }}</span>
        </div>
      </div>
      <button
        v-else
        class="hidden h-8 w-8 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary lg:flex"
        @click="$emit('toggle')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M13 5l7 7-7 7M5 5l7 7-7 7"
          />
        </svg>
      </button>
    </div>

    <!-- Collapse Toggle -->
    <button
      v-if="!collapsed"
      class="absolute right-0 top-16 hidden -translate-x-1/2 rounded-l-md border border-border-subtle bg-bg-surface px-1 py-1 text-text-muted transition-colors hover:text-text-primary lg:block"
      @click="$emit('toggle')"
    >
      <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M11 19l-7-7 7-7M15 5l7 7-7 7"
        />
      </svg>
    </button>
  </aside>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { iconPaths } from './Icons'
import { getSettings, brokerLabels } from '../stores/settings'
import type { BackendHealthStatus } from '../stores/appState'
import { getSignals } from '../api/signals'

const settings = getSettings()
const props = withDefaults(
  defineProps<{
    collapsed: boolean
    mobileOpen?: boolean
    healthStatus?: BackendHealthStatus
  }>(),
  { healthStatus: 'checking', mobileOpen: false }
)
defineEmits<{ toggle: []; close: [] }>()

const lastSync = computed(() => {
  const now = new Date()
  return now.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
})

const healthLabel = computed(() => {
  const labels: Record<BackendHealthStatus, string> = {
    checking: 'Checking',
    healthy: 'Healthy',
    degraded: 'Degraded',
    unavailable: 'Unavailable',
  }
  return labels[props.healthStatus]
})

const healthDotClass = computed(() => {
  if (props.healthStatus === 'unavailable') return 'bg-error'
  if (props.healthStatus === 'degraded') return 'bg-warning'
  if (props.healthStatus === 'checking') return 'bg-text-muted pulse-dot'
  return 'bg-success pulse-dot'
})

const healthTextClass = computed(() => {
  if (props.healthStatus === 'unavailable') return 'text-error'
  if (props.healthStatus === 'degraded') return 'text-warning'
  if (props.healthStatus === 'checking') return 'text-text-muted'
  return 'text-success'
})

const brokerLabel = computed(() => brokerLabels[settings.selectedBroker] || 'Unknown')

const buySignalCount = ref<number | null>(null)
let signalCountTimer: ReturnType<typeof setInterval> | undefined

const refreshBuySignalCount = async () => {
  try {
    const signals = await getSignals()
    buySignalCount.value = signals.filter((signal) => signal.direction === 'BUY').length
  } catch {
    // A stale/unavailable count is worse than no badge.
    buySignalCount.value = null
  }
}

const navItems = computed(() => [
  {
    path: '/',
    label: 'Dashboard',
    icon: iconPaths.dashboard,
    badge: undefined,
    group: 'Workspace',
  },
  { path: '/data', label: 'Data', icon: iconPaths.data, badge: undefined },
  {
    path: '/strategies',
    label: 'Strategies',
    icon: iconPaths.settings,
    badge: undefined,
    group: 'Research flow',
  },
  {
    path: '/candidate-explorer',
    label: 'Candidate Explorer',
    icon: iconPaths.search,
    badge: undefined,
  },
  { path: '/backtest', label: 'Backtest', icon: iconPaths.backtest, badge: undefined },
  { path: '/orchestrator', label: 'Orchestrator', icon: iconPaths.dashboard, badge: undefined },
  {
    path: '/signals',
    label: 'Signals',
    icon: iconPaths.signals,
    badge: buySignalCount.value ? String(buySignalCount.value) : undefined,
  },
  {
    path: '/gate-effectiveness',
    label: 'Gate Effectiveness',
    icon: iconPaths.intelligence,
    badge: undefined,
  },
  {
    path: '/strategy-report',
    label: 'Strategy report',
    icon: iconPaths.strategies,
    badge: undefined,
  },
  {
    path: '/watchlist',
    label: 'Watchlist',
    icon: iconPaths.watchlist,
    badge: undefined,
    group: 'Portfolio',
  },
  { path: '/portfolio', label: 'Portfolio', icon: iconPaths.portfolio, badge: undefined },
  { path: '/positions', label: 'Positions', icon: iconPaths.positions, badge: undefined },
  {
    path: '/intraday',
    label: 'Intraday paper',
    icon: iconPaths.backtest,
    badge: undefined,
    group: 'Paper trading',
  },
  {
    path: '/news',
    label: 'News',
    icon: iconPaths.intelligence,
    badge: undefined,
    group: 'Intelligence & operations',
  },
  { path: '/sentiment', label: 'Sentiment', icon: iconPaths.intelligence, badge: undefined },
  { path: '/monitoring', label: 'Monitoring', icon: iconPaths.intelligence, badge: undefined },
  { path: '/settings', label: 'Settings', icon: iconPaths.settings, badge: undefined },
])

onMounted(() => {
  void refreshBuySignalCount()
  signalCountTimer = setInterval(() => void refreshBuySignalCount(), 30_000)
})

onBeforeUnmount(() => {
  if (signalCountTimer) clearInterval(signalCountTimer)
})
</script>
