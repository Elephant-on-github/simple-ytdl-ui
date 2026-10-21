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

  it('knows when the server wants a password', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ authenticated: false, authRequired: true, signedIn: false, email: '' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const store = useSessionStore()
    await store.refresh()

    expect(store.authRequired).toBe(true)
    expect(store.authenticated).toBe(false)
    expect(store.needsLogin).toBe(true)
  })

  it('signs in with the password', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true, required: true }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const store = useSessionStore()
    store.authRequired = true
    store.authenticated = false
    await store.login('hunter2')

    expect(store.authenticated).toBe(true)
    expect(fetchMock.mock.calls[0][0]).toBe('/api/auth/login')
    expect(fetchMock.mock.calls[0][1]?.body).toContain('hunter2')
  })

  it('reports a refused password', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: async () => ({ error: 'that password is not right' }),
      }),
    )

    const store = useSessionStore()
    await expect(store.login('wrong')).rejects.toThrow()
    expect(store.authenticated).toBe(false)
  })
})
