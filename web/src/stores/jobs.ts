import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, type Job } from '../lib/api'

/** Newest first — the order the queue is rendered in. */
function byNewest(a: Job, b: Job): number {
  const diff = Date.parse(b.createdAt) - Date.parse(a.createdAt)
  if (!Number.isNaN(diff) && diff !== 0) return diff
  return a.id < b.id ? 1 : a.id > b.id ? -1 : 0
}

export const useJobsStore = defineStore('jobs', () => {
  const items = ref<Job[]>([])
  const error = ref('')
  const loading = ref(false)

  const activeCount = computed(
    () => items.value.filter((job) => job.status === 'queued' || job.status === 'running').length,
  )

  /**
   * Merge a polled payload into the queue: known jobs are updated in place,
   * unknown ones are inserted, and the result stays sorted newest-first.
   */
  function merge(incoming: Job[]): void {
    const byId = new Map(items.value.map((job) => [job.id, job]))
    for (const job of incoming) {
      const known = byId.get(job.id)
      byId.set(job.id, known ? { ...known, ...job } : { ...job })
    }
    items.value = [...byId.values()].sort(byNewest)
  }

  async function refresh(): Promise<void> {
    loading.value = true
    try {
      const payload = await api.jobs()
      merge(payload.items ?? [])
      error.value = ''
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not load the queue'
    } finally {
      loading.value = false
    }
  }

  async function enqueue(url: string, mode: 'video' | 'music', playlist: boolean): Promise<Job[]> {
    const payload = await api.createDownload({ url, mode, playlist })
    merge(payload.jobs ?? [])
    return payload.jobs ?? []
  }

  async function cancel(id: string): Promise<void> {
    try {
      await api.cancelJob(id)
    } finally {
      await refresh()
    }
  }

  async function pause(id: string): Promise<void> {
    try {
      await api.pauseJob(id)
    } finally {
      await refresh()
    }
  }

  async function resume(id: string): Promise<void> {
    try {
      await api.resumeJob(id)
    } finally {
      await refresh()
    }
  }

  async function remove(id: string): Promise<void> {
    try {
      await api.removeJob(id)
    } finally {
      await refresh()
    }
  }

  async function pauseAll(): Promise<void> {
    try {
      await api.pauseAll()
    } finally {
      await refresh()
    }
  }

  async function cancelAll(): Promise<void> {
    try {
      await api.cancelAll()
    } finally {
      await refresh()
    }
  }

  async function clearFinished(): Promise<void> {
    try {
      await api.clearFinished()
    } finally {
      await refresh()
    }
  }

  function clear(): void {
    items.value = []
  }

  return { items, error, loading, activeCount, merge, refresh, enqueue, cancel, pause, resume, remove, pauseAll, cancelAll, clearFinished, clear }
})
