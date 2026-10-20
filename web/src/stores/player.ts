import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Item } from '../lib/api'

export const usePlayerStore = defineStore('player', () => {
  const current = ref<Item | null>(null)
  const playing = ref(false)
  const currentTime = ref(0)
  const duration = ref(0)
  const volume = ref(0.9)
  const muted = ref(false)

  const progress = computed(() =>
    duration.value > 0 ? Math.min(1, Math.max(0, currentTime.value / duration.value)) : 0,
  )

  const isOpen = computed(() => current.value !== null)

  /** Load an item and start playing it. */
  function load(item: Item): void {
    if (current.value?.id === item.id) {
      playing.value = true
      return
    }
    current.value = item
    currentTime.value = 0
    duration.value = item.duration || 0
    playing.value = true
  }

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
    playing.value = false
    currentTime.value = 0
    duration.value = 0
  }

  return {
    current,
    playing,
    currentTime,
    duration,
    volume,
    muted,
    progress,
    isOpen,
    load,
    play,
    pause,
    toggle,
    seek,
    setVolume,
    toggleMute,
    close,
  }
})
