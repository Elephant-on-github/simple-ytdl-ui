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

  /** The app itself: is a password set, and is this browser signed in? */
  const authRequired = ref(false)
  const authenticated = ref(true)
  /** Can a Google account sign in to the app itself? */
  const googleCanAuth = ref(false)

  const status = computed(() => {
    if (!googleEnabled.value) return 'unconfigured'
    return signedIn.value ? 'signed-in' : 'signed-out'
  })

  /** True when the shell has to show the login screen. */
  const needsLogin = computed(() => authRequired.value && !authenticated.value)

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
      authRequired.value = !!session.authRequired
      authenticated.value = session.authenticated !== false
      googleCanAuth.value = !!session.googleCanAuth
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

  /**
   * The URL of Google's consent screen. The redirect says where the browser
   * should end up afterwards, so the sign-in stays inside the app.
   */
  async function authUrl(redirect?: string): Promise<string> {
    const { url } = await api.googleAuthUrl(redirect)
    return url
  }

  /** Send the browser to Google's consent screen. */
  async function signIn(redirect?: string): Promise<void> {
    const url = await authUrl(redirect)
    window.location.href = url
  }

  /** Sign out of Google, keeping the app session. */
  async function signOut(): Promise<void> {
    try {
      await api.googleLogout()
    } finally {
      signedIn.value = false
      email.value = ''
    }
  }

  /** Sign in to the app itself. */
  async function login(password: string): Promise<void> {
    try {
      await api.login(password)
      authenticated.value = true
      error.value = ''
    } catch (err) {
      // A refused password means we are still outside.
      authenticated.value = false
      throw err
    }
  }

  /** Sign out of the app itself. */
  async function logout(): Promise<void> {
    try {
      await api.logout()
    } finally {
      authenticated.value = false
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
    authRequired,
    authenticated,
    googleCanAuth,
    needsLogin,
    status,
    refresh,
    authUrl,
    signIn,
    signOut,
    login,
    logout,
  }
})
