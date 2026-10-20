import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './api'

function stubFetch(): ReturnType<typeof vi.fn> {
  const fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({ items: [], nextPageToken: '' }),
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('api.playlistItems', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('asks for the first page without a token', async () => {
    const fetchMock = stubFetch()

    await api.playlistItems('PL1')

    expect(fetchMock.mock.calls[0][0]).toBe('/api/playlists/PL1/items')
  })

  it('passes the page token of the next page', async () => {
    const fetchMock = stubFetch()

    await api.playlistItems('PL1', 'tok 123')

    expect(fetchMock.mock.calls[0][0]).toBe('/api/playlists/PL1/items?pageToken=tok%20123')
  })

  it('escapes the playlist id', async () => {
    const fetchMock = stubFetch()

    await api.playlistItems('PL/1?x')

    expect(fetchMock.mock.calls[0][0]).toBe('/api/playlists/PL%2F1%3Fx/items')
  })
})
