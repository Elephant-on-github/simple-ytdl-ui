import { defineStore } from 'pinia'
import { api, type Schedule } from '../lib/api'

export const useSchedulesStore = defineStore('schedules', () => {
  const items = ref<Schedule[]>([])
  const error = ref('')
  const loading = ref(false)

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

  async function save(schedule: Schedule): Promise<void> {
    try {
      const payload = await api.saveSchedule(schedule)
      items.value = payload.jobs ? payload.schedules : [...items.value, schedule]
      error.value = ''
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not save schedule'
      throw err
    }
  }

  async function remove(id: string): Promise<void> {
    try {
      await api.removeSchedule(id)
      items.value = items.value.filter((s) => s.id !== id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not remove schedule'
      throw err
    }
  }

  return { items, error, loading, refresh, save, remove }
})
