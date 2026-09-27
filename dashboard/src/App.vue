<template>
  <div class="flex h-screen w-full overflow-hidden bg-bg-primary">
    <!-- Sidebar -->
    <Sidebar
      :collapsed="sidebarCollapsed"
      :mobile-open="mobileNavOpen"
      :health-status="appState.healthStatus"
      @toggle="sidebarCollapsed = !sidebarCollapsed"
      @close="mobileNavOpen = false"
    />

    <!-- Main Content -->
    <div class="flex flex-1 flex-col overflow-hidden">
      <!-- Backend Down Banner -->
      <BackendDownBanner />
      <NotificationHost />

      <!-- Header -->
      <Header :sidebar-collapsed="sidebarCollapsed" @toggle-sidebar="toggleNav" />

      <!-- Page Content -->
      <main class="flex-1 overflow-auto bg-bg-primary">
        <router-view v-slot="{ Component, route }">
          <RuntimeErrorBoundary :reset-key="route.fullPath" :route="route.fullPath">
            <Transition name="route" mode="out-in" @after-leave="mobileNavOpen = false">
              <component :is="Component" />
            </Transition>
          </RuntimeErrorBoundary>
        </router-view>
      </main>

      <!-- Bottom Status Bar -->
      <div
        class="flex h-8 items-center justify-between border-t border-border-subtle bg-bg-surface px-4 text-xs text-text-muted"
      >
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1.5">
            <span class="inline-block h-1.5 w-1.5 rounded-full" :class="healthDotClass" />
            <span :class="healthTextClass">{{ healthLabel }}</span>
          </span>
          <span class="text-border-subtle">│</span>
          <span>NSE/BSE</span>
          <span class="text-border-subtle">│</span>
          <span>{{
            settings.tradingConfig.mode === 'live' ? 'Live Trading' : 'Paper Trading'
          }}</span>
        </div>
        <div class="flex items-center gap-3">
          <span>{{ currentTime }}</span>
          <span class="text-border-subtle">│</span>
          <span>v1.0.0</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import Sidebar from './components/Sidebar.vue'
import Header from './components/Header.vue'
import BackendDownBanner from './components/BackendDownBanner.vue'
import NotificationHost from './components/NotificationHost.vue'
import RuntimeErrorBoundary from './components/RuntimeErrorBoundary.vue'
import { useAppStateStore } from './stores/appState'
import { getSettings } from './stores/settings'

const appState = useAppStateStore()
const settings = getSettings()
const sidebarCollapsed = ref(false)
const mobileNavOpen = ref(false)
const currentTime = ref('')

const DESKTOP_BREAKPOINT = 1024

const toggleNav = () => {
  if (window.innerWidth < DESKTOP_BREAKPOINT) {
    mobileNavOpen.value = !mobileNavOpen.value
  } else {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }
}

const healthLabel = computed(() => {
  const labels = {
    checking: 'Checking',
    healthy: 'Healthy',
    degraded: 'Degraded',
    unavailable: 'Unavailable',
  } as const
  return labels[appState.healthStatus]
})

const healthDotClass = computed(() => {
  if (appState.healthStatus === 'unavailable') return 'bg-error'
  if (appState.healthStatus === 'degraded') return 'bg-warning'
  if (appState.healthStatus === 'checking') return 'bg-text-muted pulse-dot'
  return 'bg-success pulse-dot'
})

const healthTextClass = computed(() => {
  if (appState.healthStatus === 'unavailable') return 'text-error'
  if (appState.healthStatus === 'degraded') return 'text-warning'
  return appState.healthStatus === 'healthy' ? 'text-success' : ''
})

let timer: number
const updateTime = () => {
  currentTime.value = new Date().toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

onMounted(() => {
  updateTime()
  timer = window.setInterval(updateTime, 1000)
  appState.startHealthPolling()
})

onUnmounted(() => {
  clearInterval(timer)
  appState.stopHealthPolling()
})
</script>
