import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import type { Mode } from '../lib/api'

export const MODE_STORAGE_KEY = 'simple-ytdl-ui:mode'

function readStored(): Mode {
  try {
    const stored = window.localStorage.getItem(MODE_STORAGE_KEY)
    if (stored === 'video' || stored === 'music') return stored
  } catch {
    /* storage unavailable (private mode, disabled) — fall through to default */
  }
  return 'video'
}

export const useModeStore = defineStore('mode', () => {
  const mode = ref<Mode>(readStored())

  // One place owns the side effects: persist the choice and swap the accent
  // identity on <html>. Components only ever read `mode`.
  watch(
    mode,
    (value) => {
      try {
        window.localStorage.setItem(MODE_STORAGE_KEY, value)
      } catch {
        /* ignore */
      }
      if (typeof document !== 'undefined') {
        document.documentElement.dataset.mode = value
      }
    },
    { flush: 'sync' },
  )

  function set(next: Mode) {
    mode.value = next
  }

  function toggle() {
    mode.value = mode.value === 'video' ? 'music' : 'video'
  }

  return { mode, set, toggle }
})
