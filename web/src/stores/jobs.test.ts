import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useJobsStore } from './jobs'
import type { Job } from '../lib/api'

function job(overrides: Partial<Job> = {}): Job {
  return {
    id: 'job-1',
    title: 'A Song',
    url: 'https://youtu.be/abc',
    videoId: 'abc',
    mode: 'music',
    status: 'running',
    progress: 10,
    downloaded: 100,
    total: 1000,
    speed: 1024,
    eta: 12,
    error: '',
    createdAt: '2026-10-20T10:00:00Z',
    updatedAt: '2026-10-20T10:00:00Z',
    ...overrides,
  }
}

describe('jobs store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('merges a polled payload', () => {
    const store = useJobsStore()

    store.merge([job({ id: 'a', progress: 5 })])
    store.merge([job({ id: 'a', progress: 42 }), job({ id: 'b', progress: 0 })])

    expect(store.items).toHaveLength(2)
    expect(store.items.find((item) => item.id === 'a')?.progress).toBe(42)
    expect(store.items.find((item) => item.id === 'b')?.progress).toBe(0)
  })

  it('keeps jobs newest-first', () => {
    const store = useJobsStore()

    store.merge([
      job({ id: 'older', createdAt: '2026-10-19T10:00:00Z' }),
      job({ id: 'newer', createdAt: '2026-10-20T10:00:00Z' }),
    ])

    expect(store.items.map((item) => item.id)).toEqual(['newer', 'older'])
  })

  it('exposes activeCount', () => {
    const store = useJobsStore()

    store.merge([
      job({ id: 'a', status: 'queued' }),
      job({ id: 'b', status: 'running' }),
      job({ id: 'c', status: 'completed' }),
      job({ id: 'd', status: 'failed' }),
      job({ id: 'e', status: 'canceled' }),
    ])

    expect(store.activeCount).toBe(2)
  })

  it('drops activeCount back to zero once every job settled', () => {
    const store = useJobsStore()

    store.merge([job({ id: 'a', status: 'running' })])
    expect(store.activeCount).toBe(1)

    store.merge([job({ id: 'a', status: 'completed' })])
    expect(store.activeCount).toBe(0)
  })

  it('replaces the whole list on refresh, so server deletions show up', async () => {
    // The bug behind this test: refresh merged, so a job removed on the
    // server stayed visible until a page reload — and a removed queued job
    // pinned activeCount at 1, polling forever.
    const store = useJobsStore()

    store.merge([job({ id: 'a', status: 'queued' }), job({ id: 'b', status: 'running' })])
    expect(store.activeCount).toBe(2)

    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: true,
        status: 200,
        json: async () => ({ items: [job({ id: 'b', status: 'completed' })] }),
      })),
    )

    await store.refresh()

    expect(store.items.map((item) => item.id)).toEqual(['b'])
    expect(store.activeCount).toBe(0)
  })
})
