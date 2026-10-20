import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from './session'

describe('session store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('asks for an authorization url that returns to the given page', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ url: 'https://accounts.google.com/o/oauth2/auth?state=abc' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const url = await useSessionStore().authUrl('/playlists')

    expect(url).toContain('accounts.google.com')
    expect(fetchMock.mock.calls[0][0]).toBe('/api/auth/google?redirect=%2Fplaylists')
  })

  it('asks for an authorization url without a page when none is given', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ url: 'https://accounts.google.com/o/oauth2/auth?state=abc' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    await useSessionStore().authUrl()

    expect(fetchMock.mock.calls[0][0]).toBe('/api/auth/google')
  })
})
