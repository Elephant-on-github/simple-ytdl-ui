<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PlaylistList from '../components/PlaylistList.vue'
import SessionPanel from '../components/SessionPanel.vue'
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

/** The address of this server, for the command in the session panel. */
const serverURL = typeof window === 'undefined' ? '' : window.location.origin

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

  // The playlists come from the YouTube session in the cookies, or from the
  // YouTube Data API once the Google sign-in is done.
  if (session.playlistsAvailable) {
    if (justSignedIn && session.playlistSource === 'google') {
      notice.value = session.email
        ? `Signed in as ${session.email}.`
        : 'Signed in with Google.'
    }
    await loadPlaylists()
  }
})

async function loadPlaylists(): Promise<void> {
  loading.value = true
  error.value = ''
  errorDetail.value = ''
  try {
    const payload = await api.playlists()
    playlists.value = payload.items ?? []
  } catch (err) {
    // The server explains what went wrong (no session, expired cookies, …).
    error.value = err instanceof Error ? err.message : 'Could not load your playlists.'
    errorDetail.value = ''
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
        <p class="eyebrow">Your YouTube</p>
        <h1 class="t-display head-title">Playlists</h1>
      </div>
      <div v-if="session.signedIn" class="account">
        <span class="t-meta truncate">{{ session.email }}</span>
        <button type="button" class="step" @click="session.signOut()">
          Sign out
          <span aria-hidden="true">→</span>
        </button>
      </div>
    </header>

    <p v-if="notice" class="notice notice--ok t-meta">{{ notice }}</p>
    <p v-if="error" class="notice notice--error t-meta" :title="errorDetail || undefined">
      {{ error }}
    </p>

    <!-- No YouTube session yet: offer the two ways to get one. -->
    <template v-if="session.loaded && !session.playlistsAvailable">
      <SessionPanel :server-url="serverURL" />
      <p class="t-meta faint">
        Would rather sign in with a button? A Google OAuth client works too — see the
        README. It needs a project in the Google Cloud console, the session above does
        not.
      </p>
    </template>

    <!-- Google is configured but the visitor has not signed in. -->
    <section
      v-else-if="session.loaded && session.googleEnabled && !session.signedIn"
      class="card frame"
    >
      <span class="tick tick--tl" aria-hidden="true" />
      <span class="tick tick--tr" aria-hidden="true" />
      <span class="tick tick--bl" aria-hidden="true" />
      <span class="tick tick--br" aria-hidden="true" />

      <p class="t-label">Step one</p>
      <h2 class="t-italic card-title">Sign in to see your playlists</h2>
      <p class="t-serif-body card-copy">
        Google only ever grants read access to your playlists. Downloads still happen on
        your own server.
      </p>
      <button type="button" class="step card-cta" :disabled="signingIn" @click="signIn">
        {{ signingIn ? 'Redirecting' : 'Continue with Google' }}
        <span aria-hidden="true">→</span>
      </button>
    </section>

    <!-- The playlists are there — from the session in the cookies or from Google. -->
    <template v-else-if="session.playlistsAvailable">
      <div class="toolbar">
        <p class="t-meta">
          Syncing queues every track as
          <strong>{{ mode.mode === 'music' ? 'MP3' : 'VIDEO' }}</strong>.
        </p>
        <button type="button" class="step" :disabled="loading" @click="loadPlaylists">
          {{ loading ? 'Refreshing' : 'Refresh' }}
          <span aria-hidden="true">→</span>
        </button>
      </div>

      <p v-if="loading && !playlists.length" class="t-meta faint">Loading your playlists…</p>
      <p v-else-if="!playlists.length" class="t-meta faint">You have no playlists yet.</p>
      <PlaylistList
        v-else
        :playlists="playlists"
        :mode="mode.mode"
        :busy="syncing"
        @sync="sync"
      />
    </template>

    <p v-else class="t-meta faint">Checking the server…</p>
  </div>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: var(--gap);
}

.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--xxl);
  padding-bottom: var(--xl);
  border-bottom: 1px solid var(--hairline);
}

.head-title {
  margin-top: var(--md);
}

.account {
  display: flex;
  align-items: center;
  gap: var(--xxl);
  max-width: 45%;
}

/* --- card ---------------------------------------------------------------- */

.card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--lg);
  max-width: 640px;
  padding: var(--xxl);
  border: 1px solid var(--hairline);
}

.card .t-label {
  color: var(--accent);
}

.card-title {
  color: var(--ink);
}

.card-copy {
  color: var(--ink-dim);
  max-width: 46ch;
}

.card-copy code {
  font-family: var(--mono);
  font-size: 0.8em;
  letter-spacing: 0.04em;
  color: var(--accent);
}

.card-note {
  color: var(--ink-faint);
  text-transform: none;
  letter-spacing: 0.04em;
  max-width: 46ch;
}

.card-cta {
  margin-top: var(--md);
  color: var(--ink);
}

/* --- toolbar ------------------------------------------------------------- */

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--xxl);
  padding-bottom: var(--lg);
  border-bottom: 1px solid var(--hairline);
}

.toolbar .t-meta {
  color: var(--ink-faint);
}

.toolbar strong {
  color: var(--accent);
  font-weight: 400;
}

/* --- notices ------------------------------------------------------------- */

.notice {
  padding: var(--md) 0;
  border-bottom: 1px solid var(--hairline);
  text-transform: none;
  letter-spacing: 0.04em;
}

.notice--ok {
  color: var(--accent);
}

.notice--error {
  color: var(--accent);
}

@media (max-width: 620px) {
  .account {
    max-width: 55%;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--sm);
  }
}
</style>
