import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import type { Mode } from '../lib/api'

export const MODE_STORAGE_KEY = 'simple-ytdl-ui:mode'

/** The mode the user picked, or null when they never picked one. */
function readStored(): Mode | null {
  try {
    const stored = window.localStorage.getItem(MODE_STORAGE_KEY)
    if (stored === 'video' || stored === 'music') return stored
  } catch {
    /* storage unavailable (private mode, disabled) — fall through to default */
  }
  return null
}

export const useModeStore = defineStore('mode', () => {
  const stored = readStored()
  const mode = ref<Mode>(stored ?? 'video')

  // Until the user picks a mode, the server decides which one it is.
  let followsServer = stored === null
  let fromServer = false

  // One place owns the side effects: persist the choice and swap the accent
  // identity on <html>. Components only ever read `mode`.
  watch(
    mode,
    (value) => {
      if (fromServer) {
        // The server default is not a choice of the user: do not store it.
        fromServer = false
      } else {
        followsServer = false
        try {
          window.localStorage.setItem(MODE_STORAGE_KEY, value)
        } catch {
          /* ignore */
        }
      }
      if (typeof document !== 'undefined') {
        document.documentElement.dataset.mode = value
      }
    },
    { flush: 'sync' },
  )

  /** Adopt the default the server announces, as long as the user has no say. */
  function applyServerDefault(next: Mode): void {
    if (!followsServer || next === mode.value) return
    fromServer = true
    mode.value = next
  }

  function set(next: Mode) {
    mode.value = next
  }

  function toggle() {
    mode.value = mode.value === 'video' ? 'music' : 'video'
  }

  return { mode, set, toggle, applyServerDefault }
})
