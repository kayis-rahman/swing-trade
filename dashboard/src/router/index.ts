import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import { reportRuntimeError } from '../stores/runtimeErrors'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/news',
      name: 'News',
      component: () => import('../views/NewsView.vue'),
    },
    {
      path: '/',
      name: 'Dashboard',
      component: DashboardView,
    },
    {
      path: '/positions',
      name: 'Positions',
      component: () => import('../views/PositionsView.vue'),
    },
    {
      path: '/intraday',
      name: 'Intraday paper',
      component: () => import('../views/IntradayView.vue'),
    },
    {
      path: '/symbols/:symbol',
      name: 'SymbolDetail',
      component: () => import('../views/SymbolDetailView.vue'),
    },
    {
      path: '/signals',
      name: 'Signals',
      component: () => import('../views/SignalsView.vue'),
    },
    {
      path: '/gate-effectiveness',
      name: 'Gate Effectiveness',
      component: () => import('../views/GateEffectivenessView.vue'),
    },
    {
      path: '/portfolio',
      name: 'Portfolio',
      component: () => import('../views/PortfolioView.vue'),
    },
    {
      path: '/watchlist',
      name: 'Watchlist',
      component: () => import('../views/WatchlistView.vue'),
    },
    {
      path: '/candidate-explorer',
      name: 'Candidate Explorer',
      component: () => import('../views/CandidateExplorerView.vue'),
    },
    {
      path: '/backtest',
      name: 'Backtest',
      component: () => import('../views/BacktestView.vue'),
    },
    {
      path: '/strategies',
      name: 'Strategies',
      component: () => import('../views/StrategiesView.vue'),
    },
    {
      path: '/strategy-report',
      name: 'StrategyReport',
      component: () => import('../views/StrategyReportView.vue'),
    },
    {
      path: '/sentiment',
      name: 'Sentiment',
      component: () => import('../views/SentimentView.vue'),
    },
    {
      path: '/monitoring',
      name: 'Monitoring',
      component: () => import('../views/MonitoringView.vue'),
    },
    {
      path: '/data',
      name: 'Data Ingestion',
      component: () => import('../views/DataIngestionView.vue'),
    },
    {
      path: '/settings',
      name: 'Settings',
      component: () => import('../views/SettingsView.vue'),
    },
    {
      path: '/orchestrator',
      name: 'Orchestrator',
      component: () => import('../views/OrchestratorView.vue'),
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'NotFound',
      component: () => import('../views/NotFoundView.vue'),
    },
  ],
})

router.onError((error, to) => {
  reportRuntimeError(error, {
    source: 'router',
    ...(to?.fullPath ? { route: to.fullPath } : {}),
  })
})

export default router
