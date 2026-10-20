<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import PlaylistList from '../components/PlaylistList.vue'
import { useJobsStore } from '../stores/jobs'
import { useModeStore } from '../stores/mode'
import { useSessionStore } from '../stores/session'
import { api, type Playlist } from '../lib/api'
import { describeOAuthError } from '../lib/oauth'

const session = useSessionStore()
const mode = useModeStore()
const jobs = useJobsStore()
const route = useRoute()
const router = useRouter()

const playlists = ref<Playlist[]>([])
const loading = ref(false)
const error = ref('')
const errorDetail = ref('')
const notice = ref('')
const signingIn = ref(false)
const syncing = ref<string | null>(null)

onMounted(async () => {
  // The sign-in comes back from Google with the result in the query string.
  const code = typeof route.query.error === 'string' ? route.query.error : ''
  const justSignedIn = route.query.signedIn === '1'
  if (code || justSignedIn) {
    void router.replace({ path: route.path, query: {} })
  }

  await session.refresh()

  if (code) {
    error.value = describeOAuthError(code)
    errorDetail.value = ''
    return
  }
  if (!session.googleEnabled || !session.signedIn) return

  if (justSignedIn) {
    notice.value = session.email
      ? `Signed in as ${session.email}.`
      : 'Signed in with Google.'
  }
  await loadPlaylists()
})

async function loadPlaylists(): Promise<void> {
  loading.value = true
  error.value = ''
  errorDetail.value = ''
  try {
    const payload = await api.playlists()
    playlists.value = payload.items ?? []
  } catch (err) {
    error.value = 'Could not load your playlists.'
    errorDetail.value = err instanceof Error ? err.message : String(err)
    playlists.value = []
  } finally {
    loading.value = false
  }
}

async function signIn(): Promise<void> {
  signingIn.value = true
  error.value = ''
  try {
    // Come back to this page when Google is done.
    await session.signIn(route.path)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not start the sign-in.'
    signingIn.value = false
  }
}

async function sync(playlist: Playlist): Promise<void> {
  syncing.value = playlist.id
  error.value = ''
  try {
    await jobs.enqueue(`https://www.youtube.com/playlist?list=${playlist.id}`, mode.mode, true)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not start the sync.'
  } finally {
    syncing.value = null
  }
}
</script>

<template>
  <div class="view">
    <header class="head">
      <div>
        <p class="t-label">Your YouTube</p>
        <h1 class="t-display head-title">Playlists</h1>
      </div>
      <div v-if="session.signedIn" class="account">
        <span class="t-fine muted truncate">{{ session.email }}</span>
        <button type="button" class="btn btn--ghost" @click="session.signOut()">Sign out</button>
      </div>
    </header>

    <p v-if="notice" class="notice notice--ok t-fine">{{ notice }}</p>
    <p v-if="error" class="notice notice--error t-fine" :title="errorDetail || undefined">
      {{ error }}
    </p>

    <!-- Google is not configured — the app stays fully usable without it. -->
    <section v-if="session.loaded && !session.googleEnabled" class="card panel">
      <span class="card-icon" aria-hidden="true">
        <svg width="20" height="20" viewBox="0 0 18 18" fill="none">
          <path
            d="M3.2 6.4h11.6v7.2a1.4 1.4 0 0 1-1.4 1.4H4.6a1.4 1.4 0 0 1-1.4-1.4V6.4Z"
            stroke="currentColor"
            stroke-width="1.4"
          />
          <path d="M6.2 6.4V5a2.8 2.8 0 0 1 5.6 0v1.4" stroke="currentColor" stroke-width="1.4" />
        </svg>
      </span>
      <div class="card-body">
        <h2 class="t-title">Google sign-in is not configured</h2>
        <p class="t-meta muted">
          Browsing your own YouTube playlists needs a Google OAuth client. Set
          <code>GOOGLE_CLIENT_ID</code> and <code>GOOGLE_CLIENT_SECRET</code> on the server to enable it.
        </p>
        <p class="t-fine faint">
          Everything else works without it — paste a playlist link on the Download page and it syncs
          straight into your library.
        </p>
        <RouterLink to="/" class="btn card-cta">Paste a playlist link</RouterLink>
      </div>
    </section>

    <!-- Google is configured but the visitor has not signed in. -->
    <section v-else-if="session.loaded && !session.signedIn" class="card panel">
      <span class="card-icon" aria-hidden="true">
        <svg width="20" height="20" viewBox="0 0 18 18" fill="none">
          <path
            d="M9 2.4a6.6 6.6 0 1 1 0 13.2 6.6 6.6 0 0 1 0-13.2Z"
            stroke="currentColor"
            stroke-width="1.4"
          />
          <path d="M1.8 9h14.4M9 2.4c1.9 2 2.8 4.3 2.8 6.6S10.9 13.6 9 15.6c-1.9-2-2.8-4.3-2.8-6.6S7.1 4.4 9 2.4Z" stroke="currentColor" stroke-width="1.4" />
        </svg>
      </span>
      <div class="card-body">
        <h2 class="t-title">Sign in to see your playlists</h2>
        <p class="t-meta muted">
          Google only ever grants read access to your playlists. Downloads still happen on your own
          server.
        </p>
        <button type="button" class="btn btn--primary card-cta" :disabled="signingIn" @click="signIn">
          {{ signingIn ? 'Redirecting…' : 'Continue with Google' }}
        </button>
      </div>
    </section>

    <!-- Signed in. -->
    <template v-else-if="session.signedIn">
      <div class="toolbar">
        <p class="t-fine muted">
          Syncing queues every track as
          <strong>{{ mode.mode === 'music' ? 'MP3' : 'video' }}</strong>.
        </p>
        <button type="button" class="btn btn--ghost" :disabled="loading" @click="loadPlaylists">
          {{ loading ? 'Refreshing…' : 'Refresh' }}
        </button>
      </div>

      <p v-if="loading && !playlists.length" class="t-fine faint">Loading your playlists…</p>
      <p v-else-if="!playlists.length" class="t-fine faint">You have no playlists yet.</p>
      <PlaylistList
        v-else
        :playlists="playlists"
        :mode="mode.mode"
        :busy="syncing"
        @sync="sync"
      />
    </template>

    <p v-else class="t-fine faint">Checking the server…</p>
  </div>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
  max-width: var(--content-max);
}

.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--sp-4);
  padding-bottom: var(--sp-5);
  border-bottom: 1px solid var(--hairline);
}

.head-title {
  margin-top: 6px;
}

.account {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  max-width: 45%;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-4);
}

.toolbar strong {
  color: var(--text-dim);
  font-weight: 500;
}

.card {
  display: flex;
  gap: var(--sp-5);
  padding: var(--sp-6);
  max-width: 640px;
}

.card-icon {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex: none;
  border-radius: var(--r-md);
  color: var(--accent);
  background: var(--accent-soft);
  border: 1px solid var(--accent-line);
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  min-width: 0;
}

.card-body code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  padding: 1px 5px;
  border-radius: var(--r-xs);
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-dim);
}

.card-cta {
  align-self: flex-start;
  margin-top: var(--sp-1);
}

.notice {
  padding: 10px var(--sp-3);
  border-radius: var(--r-sm);
}

.notice--error {
  color: var(--danger);
  background: var(--danger-soft);
  border: 1px solid var(--danger-line);
}

.notice--ok {
  color: var(--accent);
  background: var(--accent-soft);
  border: 1px solid var(--accent-line);
}

@media (max-width: 620px) {
  .card {
    flex-direction: column;
    gap: var(--sp-4);
    padding: var(--sp-5);
  }

  .account {
    max-width: 55%;
  }
}
</style>
