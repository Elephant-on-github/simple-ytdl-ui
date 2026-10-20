import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, type Item } from '../lib/api'

export const useLibraryStore = defineStore('library', () => {
  const items = ref<Item[]>([])
  const error = ref('')
  const loading = ref(false)

  const totalSize = computed(() => items.value.reduce((sum, item) => sum + (item.size || 0), 0))

  async function refresh(): Promise<void> {
    loading.value = true
    try {
      const payload = await api.library()
      items.value = payload.items ?? []
      error.value = ''
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not load the library'
    } finally {
      loading.value = false
    }
  }

  async function remove(id: string): Promise<void> {
    await api.deleteItem(id)
    items.value = items.value.filter((item) => item.id !== id)
  }

  function find(id: string): Item | undefined {
    return items.value.find((item) => item.id === id)
  }

  return { items, error, loading, totalSize, refresh, remove, find }
})
