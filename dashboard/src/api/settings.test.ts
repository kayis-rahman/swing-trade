import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiRequestMock = vi.hoisted(() => vi.fn())

vi.mock('./shared', () => ({ apiRequest: apiRequestMock }))

import { startMlxServer } from './settings'

describe('startMlxServer', () => {
  beforeEach(() => apiRequestMock.mockReset())

  it('allows enough time for the MLX model startup window', async () => {
    await startMlxServer()

    expect(apiRequestMock).toHaveBeenCalledWith(
      '/settings/mlx/start',
      expect.objectContaining({ method: 'POST', timeoutMs: 130_000 })
    )
  })
})
