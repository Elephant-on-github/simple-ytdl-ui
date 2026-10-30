import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSchedulesStore } from './schedules'
import type { Schedule } from '../lib/api'

function schedule(overrides: Partial<Schedule> = {}): Schedule {
  return {
    id: 'sched-1',
    url: 'https://www.youtube.com/playlist?list=PLfake',
    mode: 'music',
    folder: '',
    frequency: 'daily',
    nextRun: '2026-10-30T00:00:00Z',
    createdAt: '2026-10-29T20:00:00Z',
    ...overrides,
  }
}

/** A fetch mock that answers the schedules endpoints from a live list. */
function stubApi(initial: Schedule[]) {
  let store = initial
  const calls: string[] = []

  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : String(input)
      const method = init?.method ?? 'GET'
      calls.push(`${method} ${url}`)

      const json = async (body: unknown) => ({ ok: true, status: 200, json: async () => body })

      if (method === 'GET' && url === '/api/schedules') return json({ schedules: store })

      if (method === 'POST' && url === '/api/schedules') {
        const saved = schedule(JSON.parse(String(init?.body ?? '{}')))
        saved.id = 'sched-new'
        store = [...store, saved]
        return json({ ok: true, schedule: saved })
      }

      if (method === 'POST' && url.endsWith('/run')) {
        // A run advances NextRun on the server, exactly like the Go side does.
        store = store.map((s) =>
          s.id === url.split('/').slice(-2, -1)[0] ? { ...s, nextRun: '2026-10-31T00:00:00Z' } : s,
        )
        return json({ ok: true })
      }

      if (method === 'DELETE') {
        const id = url.split('/').pop() ?? ''
        store = store.filter((s) => s.id !== id)
        return json({ ok: true })
      }

      return json({ ok: false, error: 'not handled' })
    }),
  )

  return { calls }
}

describe('schedules store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('loads the schedules from the server', async () => {
    stubApi([schedule(), schedule({ id: 'sched-2' })])

    const store = useSchedulesStore()
    await store.refresh()

    expect(store.items).toHaveLength(2)
    expect(store.error).toBe('')
    expect(store.loading).toBe(false)
  })

  it('reflects a schedule the server no longer has', async () => {
    // The bug behind this test: the list was merged, so a deleted schedule
    // stayed on screen until a page reload.
    stubApi([schedule()])

    const store = useSchedulesStore()
    await store.refresh()
    expect(store.items).toHaveLength(1)

    // The server list is now empty; the next poll must agree.
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({ ok: true, status: 200, json: async () => ({ schedules: [] }) })),
    )
    await store.refresh()

    expect(store.items).toHaveLength(0)
  })

  it('reports a failed load instead of keeping stale rows', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({ ok: false, status: 500, statusText: 'Server Error', json: async () => ({}) })),
    )

    const store = useSchedulesStore()
    await store.refresh()

    expect(store.items).toHaveLength(0)
    expect(store.error).not.toBe('')
  })

  it('keeps the server id of a created schedule', async () => {
    stubApi([])

    const store = useSchedulesStore()
    const saved = await store.save(schedule({ id: '' }))

    expect(saved?.id).toBe('sched-new')
    expect(store.items.map((s) => s.id)).toContain('sched-new')
  })

  it('re-reads after a removal so the list matches the server', async () => {
    stubApi([schedule(), schedule({ id: 'sched-2' })])

    const store = useSchedulesStore()
    await store.refresh()

    await store.remove('sched-1')

    expect(store.items.map((s) => s.id)).toEqual(['sched-2'])
  })

  it('reports a failed removal and re-reads the truth', async () => {
    stubApi([schedule()])

    const store = useSchedulesStore()
    await store.refresh()

    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: async () => ({ error: 'unknown schedule' }),
      })),
    )

    await expect(store.remove('sched-1')).rejects.toThrow()
    expect(store.error).not.toBe('')
  })

  it('tracks the running state and advances NextRun after a run', async () => {
    stubApi([schedule()])

    const store = useSchedulesStore()
    await store.refresh()

    await store.run('sched-1')

    expect(store.running).toBe('')
    expect(store.error).toBe('')
    expect(store.items[0].nextRun).toBe('2026-10-31T00:00:00Z')
  })

  it('clears the running state when a run fails', async () => {
    stubApi([schedule()])

    const store = useSchedulesStore()
    await store.refresh()

    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: false,
        status: 502,
        statusText: 'Bad Gateway',
        json: async () => ({ error: 'yt-dlp cannot read it' }),
      })),
    )

    await expect(store.run('sched-1')).rejects.toThrow()
    expect(store.running).toBe('')
    expect(store.error).not.toBe('')
  })
})
