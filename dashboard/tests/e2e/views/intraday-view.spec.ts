import { test, expect } from '@playwright/test'
import { fulfillJson, mockDashboardApi } from '../fixtures/api'

const DASHBOARD = 'http://localhost:3003'

test('scanner candidate handoff selects the returned cutoff in the replay form', async ({
  page,
}) => {
  await mockDashboardApi(page, async (route, path) => {
    if (path === '/intraday/paper/scans' && route.request().method() === 'POST') {
      await fulfillJson(route, {
        mode: 'PAPER_SCAN_AS_OF',
        sessionDate: '2026-10-09',
        asOfBarTime: '10:00:00',
        universe: 'ACTIVE_NSE_WATCHLIST',
        watchlistSize: 1,
        scannedSymbols: 1,
        candidateCount: 1,
        liveBrokerCalls: 0,
        excludedSymbols: [],
        results: [
          {
            symbol: 'INFY',
            strategy: 'INTRADAY_VWAP_PULLBACK',
            status: 'CANDIDATE',
            expectedBars: 4,
            receivedBars: 4,
            completedTrades: 0,
            netPnl: 0,
            openEntryPrice: 100,
            openEntryTime: '2026-10-09T10:00:00+05:30',
          },
        ],
      })
      return true
    }
    return false
  })

  await page.goto(`${DASHBOARD}/intraday`)
  await page.getByLabel('Paper session date').fill('2026-10-09')
  await page.getByLabel('Replay as of bar time').selectOption('10:00')
  await page.getByRole('button', { name: 'Scan watchlist at selected time' }).click()
  await expect(page.getByText('INFY', { exact: true })).toBeVisible()
  await page.getByLabel('Replay as of bar time').selectOption('10:15')
  await page.getByRole('button', { name: 'Use in replay form' }).click()

  await expect(page.getByLabel('Replay as of bar time')).toHaveValue('10:00')
})
