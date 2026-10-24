import { expect, test } from '@playwright/test'

const javaBaseUrl = process.env.PARITY_JAVA_BASE_URL
const goBaseUrl = process.env.PARITY_GO_BASE_URL
const requests = [
  '/api/health',
  '/api/holidays',
  '/api/holidays/today',
  '/api/watchlist',
  '/api/watchlist/all',
]

test('Go read endpoints match Java responses on the shared local dataset', async ({ request }) => {
  test.skip(
    !javaBaseUrl || !goBaseUrl,
    'Set PARITY_JAVA_BASE_URL and PARITY_GO_BASE_URL to run API parity'
  )

  for (const path of requests) {
    const [javaResponse, goResponse] = await Promise.all([
      request.get(`${javaBaseUrl}${path}`, { headers: { Accept: 'application/json' } }),
      request.get(`${goBaseUrl}${path}`, { headers: { Accept: 'application/json' } }),
    ])
    expect(goResponse.status(), `${path} status`).toBe(javaResponse.status())
    expect(goResponse.headers()['content-type'], `${path} content type`).toBe(
      javaResponse.headers()['content-type']
    )

    const javaBody: unknown = await javaResponse.json()
    const goBody: unknown = await goResponse.json()
    expect(normalize(goBody, path), `${path} body`).toEqual(normalize(javaBody, path))
  }
})

function normalize(value: unknown, path: string): unknown {
  if (Array.isArray(value)) return value.map((entry) => normalize(entry, path))
  if (typeof value !== 'object' || value === null) return value

  const record = value as Record<string, unknown>
  const normalized = Object.fromEntries(
    Object.keys(record)
      .sort()
      .map((key) => [key, normalize(record[key], path)])
  )
  if (path === '/api/watchlist/all' && Array.isArray(normalized.data)) {
    normalized.data = [...normalized.data].sort((left, right) => {
      const leftId = Number((left as Record<string, unknown>).id)
      const rightId = Number((right as Record<string, unknown>).id)
      return leftId - rightId
    })
  }
  return normalized
}
