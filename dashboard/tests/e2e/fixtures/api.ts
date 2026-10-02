import type { Page, Route } from '@playwright/test'

export type ApiResponder = (route: Route, path: string) => Promise<boolean> | boolean

export async function fulfillJson(route: Route, body: unknown, status = 200): Promise<void> {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}

/**
 * Prevent E2E cases from depending on a running backend. Individual tests can
 * override a route by returning true from `responder`.
 */
export async function mockDashboardApi(page: Page, responder?: ApiResponder): Promise<void> {
  await page.route(/^https?:\/\/[^/]+\/api(?:\/|$)/, async (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname.replace(/^\/api/, '')

    if ((await responder?.(route, path)) === true) return

    if (path === '/health') {
      await fulfillJson(route, { status: 'UP', components: {} })
      return
    }
    if (path === '/signals/latest' || path === '/paper-portfolios') {
      await fulfillJson(route, [])
      return
    }
    if (path === '/positions' || path.startsWith('/positions/closed')) {
      await fulfillJson(route, {
        content: [],
        totalElements: 0,
        totalPages: 0,
        size: 20,
        number: 0,
      })
      return
    }

    await fulfillJson(route, {})
  })
}
