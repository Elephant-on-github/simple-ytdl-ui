import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Item } from '../lib/api'

export const usePlayerStore = defineStore('player', () => {
  const current = ref<Item | null>(null)
  const queue = ref<Item[]>([])
  const queueIndex = ref(0)
  const playing = ref(false)
  const currentTime = ref(0)
  const duration = ref(0)
  const volume = ref(0.9)
  const muted = ref(false)

  const progress = computed(() =>
    duration.value > 0 ? Math.min(1, Math.max(0, currentTime.value / duration.value)) : 0,
  )

  const isOpen = computed(() => current.value !== null)

  /** Load a single item and start playing it. */
  function load(item: Item): void {
    if (current.value?.id === item.id) {
      playing.value = true
      return
    }
    queue.value = [item]
    queueIndex.value = 0
    current.value = item
    currentTime.value = 0
    duration.value = item.duration || 0
    playing.value = true
  }

  /** Load a queue of items and start playing the first one. */
  function loadQueue(items: Item[]): void {
    if (items.length === 0) return
    queue.value = items
    queueIndex.value = 0
    const item = items[0]
    current.value = item
    currentTime.value = 0
    duration.value = item.duration || 0
    playing.value = true
  }

  /** Play the next track in the queue. */
  function next(): void {
    if (queueIndex.value >= queue.value.length - 1) return
    queueIndex.value++
    const item = queue.value[queueIndex.value]
    current.value = item
    currentTime.value = 0
    duration.value = item.duration || 0
    playing.value = true
  }

  /** Play the previous track in the queue. */
  function prev(): void {
    if (queueIndex.value <= 0) return
    queueIndex.value--
    const item = queue.value[queueIndex.value]
    current.value = item
    currentTime.value = 0
    duration.value = item.duration || 0
    playing.value = true
  }

  const hasNext = computed(() => queueIndex.value < queue.value.length - 1)
  const hasPrev = computed(() => queueIndex.value > 0)
  const queueLength = computed(() => queue.value.length)

  function play(): void {
    playing.value = true
  }

  function pause(): void {
    playing.value = false
  }

  function toggle(): void {
    playing.value = !playing.value
  }

  function seek(seconds: number): void {
    currentTime.value = Math.max(0, Math.min(duration.value || 0, seconds))
  }

  function setVolume(next: number): void {
    volume.value = Math.min(1, Math.max(0, next))
    if (volume.value > 0) muted.value = false
  }

  function toggleMute(): void {
    muted.value = !muted.value
  }

  function close(): void {
    current.value = null
    queue.value = []
    queueIndex.value = 0
    playing.value = false
    currentTime.value = 0
    duration.value = 0
  }

  return {
    current,
    queue,
    queueIndex,
    playing,
    currentTime,
    duration,
    volume,
    muted,
    progress,
    isOpen,
    hasNext,
    hasPrev,
    queueLength,
    load,
    loadQueue,
    next,
    prev,
    play,
    pause,
    toggle,
    seek,
    setVolume,
    toggleMute,
    close,
  }
})
