import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, type Mode } from '../lib/api'
import { useModeStore } from './mode'

export const useSessionStore = defineStore('session', () => {
  const googleEnabled = ref(false)
  const signedIn = ref(false)
  const email = ref('')
  const defaultMode = ref<Mode>('video')
  const concurrent = ref(0)
  const version = ref('')
  const loaded = ref(false)
  const error = ref('')

  const status = computed(() => {
    if (!googleEnabled.value) return 'unconfigured'
    return signedIn.value ? 'signed-in' : 'signed-out'
  })

  let inflight: Promise<void> | null = null

  async function load(): Promise<void> {
    try {
      const [config, health] = await Promise.allSettled([api.config(), api.health()])
      if (config.status === 'fulfilled') {
        googleEnabled.value = !!config.value.googleEnabled
        defaultMode.value = config.value.defaultMode === 'music' ? 'music' : 'video'
        concurrent.value = config.value.concurrent ?? 0
        // The server decides which mode the toggle starts in, until the user
        // picks one.
        useModeStore().applyServerDefault(defaultMode.value)
      }
      if (health.status === 'fulfilled') version.value = health.value.version ?? ''

      const session = await api.session()
      signedIn.value = !!session.signedIn
      email.value = session.email ?? ''
      error.value = ''
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Could not reach the server'
    } finally {
      loaded.value = true
    }
  }

  /** Fetch the server config and the current session — de-duplicated. */
  function refresh(): Promise<void> {
    if (!inflight) {
      inflight = load().finally(() => {
        inflight = null
      })
    }
    return inflight
  }

  /** Redirect the browser to Google's consent screen. */
  async function signIn(): Promise<void> {
    const { url } = await api.googleAuthUrl()
    window.location.href = url
  }

  async function signOut(): Promise<void> {
    try {
      await api.logout()
    } finally {
      signedIn.value = false
      email.value = ''
    }
  }

  return {
    googleEnabled,
    signedIn,
    email,
    defaultMode,
    concurrent,
    version,
    loaded,
    error,
    status,
    refresh,
    signIn,
    signOut,
  }
})
