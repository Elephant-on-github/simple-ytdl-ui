import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, type Schedule } from '../lib/api'

export const useSchedulesStore = defineStore('schedules', () => {
  const items = ref<Schedule[]>([])
  const error = ref('')
  const loading = ref(false)
  const running = ref('')

  async function refresh(): Promise<void> {
    loading.value = true
    try {
      const payload = await api.schedules()
      items.value = payload.schedules ?? []
      error.value = ''
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not load schedules'
      items.value = []
    } finally {
      loading.value = false
    }
  }

  async function save(schedule: Schedule): Promise<Schedule | undefined> {
    try {
      const payload = await api.saveSchedule(schedule)
      if (payload.schedule) {
        // Re-read so the list (and NextRun) matches the server exactly.
        await refresh()
      }
      return payload.schedule
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not save schedule'
      throw err
    }
  }

  async function remove(id: string): Promise<void> {
    try {
      await api.removeSchedule(id)
      // Re-read instead of trusting the optimistic removal: the server is the
      // truth and a failed call must not leave the list lying.
      await refresh()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not remove schedule'
      throw err
    }
  }

  async function run(id: string): Promise<void> {
    running.value = id
    try {
      await api.runSchedule(id)
      error.value = ''
      // NextRun advanced on the server, so the displayed time moves too.
      await refresh()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not run schedule'
      throw err
    } finally {
      running.value = ''
    }
  }

  return { items, error, loading, running, refresh, save, remove, run }
})
