<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../lib/api'

const props = defineProps<{ serverUrl: string }>()

const cookies = ref('')
const status = ref('')
const error = ref('')
const busy = ref(false)
const configured = ref(false)

/** The command that exports the session and sends it to this server. */
const command = computed(() => {
  const url = `${props.serverUrl}/api/session/cookies`
  return [
    '# Linux or macOS',
    'yt-dlp --cookies-from-browser chrome --cookies /tmp/yt.txt "https://www.youtube.com/robots.txt" \\',
    `  && curl -X POST --data-binary @/tmp/yt.txt ${url}`,
    '',
    '# Windows PowerShell',
    'yt-dlp --cookies-from-browser chrome --cookies yt.txt "https://www.youtube.com/robots.txt"',
    `Get-Content yt.txt -Raw | curl.exe -X POST --data-binary @- ${url}`,
  ].join('\n')
})

onMounted(async () => {
  try {
    const state = await api.cookies()
    configured.value = state.configured
  } catch {
    /* the server is not reachable; the panel still works */
  }
})

async function save(): Promise<void> {
  if (!cookies.value.trim()) {
    error.value = 'Paste the contents of the cookies.txt first.'
    return
  }
  busy.value = true
  error.value = ''
  status.value = ''
  try {
    const result = await api.saveCookies(cookies.value)
    cookies.value = ''
    configured.value = true
    if (result.ok) {
      status.value = `Connected. The server found ${result.playlists} playlist${
        result.playlists === 1 ? '' : 's'
      }.`
    } else {
      error.value = result.error || 'The server could not use that session.'
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not save the session.'
  } finally {
    busy.value = false
  }
}

async function clear(): Promise<void> {
  busy.value = true
  try {
    await api.clearCookies()
    configured.value = false
    status.value = 'The session was removed.'
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not remove the session.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="session frame">
    <span class="tick tick--tl" aria-hidden="true" />
    <span class="tick tick--tr" aria-hidden="true" />
    <span class="tick tick--bl" aria-hidden="true" />
    <span class="tick tick--br" aria-hidden="true" />

    <p class="t-label">YouTube session</p>

    <h2 class="t-italic session-title">
      {{ configured ? 'The server has your session' : 'Give the server your session' }}
    </h2>

    <p class="t-serif-body session-copy">
      Your playlists live behind your YouTube login. Export them as a
      <code>cookies.txt</code> and paste the file here — or run the command below on the
      machine where your browser is, and it sends the file over by itself. No Google
      Cloud project, no API key, no volume to mount.
    </p>

    <details class="how">
      <summary class="t-label how-summary">Show the command</summary>
      <pre class="how-code">{{ command }}</pre>
    </details>

    <form class="paste" @submit.prevent="save">
      <label class="t-label paste-label" for="cookies">cookies.txt</label>
      <textarea
        id="cookies"
        v-model="cookies"
        class="paste-area"
        rows="5"
        spellcheck="false"
        placeholder="# Netscape HTTP Cookie File&#10;.youtube.com&#9;TRUE&#9;/&#9;TRUE&#9;0&#9;SID&#9;…"
      />
      <div class="paste-actions">
        <button type="submit" class="step" :disabled="busy">
          {{ busy ? 'Checking' : 'Connect' }}
          <span aria-hidden="true">→</span>
        </button>
        <button v-if="configured" type="button" class="step step--quiet" :disabled="busy" @click="clear">
          Forget it
        </button>
      </div>
    </form>

    <p v-if="status" class="session-note t-meta">{{ status }}</p>
    <p v-if="error" class="session-note session-note--error t-meta">{{ error }}</p>
  </section>
</template>

<style scoped>
.session {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--lg);
  max-width: 720px;
  padding: var(--xxl);
  background: var(--plate);
}

.session .t-label {
  color: var(--accent);
}

.session-title {
  color: var(--ink);
}

.session-copy {
  color: var(--ink-dim);
  max-width: 54ch;
}

.session-copy code {
  font-family: var(--mono);
  font-size: 0.8em;
  letter-spacing: 0.04em;
  color: var(--accent);
}

/* --- the command --------------------------------------------------------- */

.how {
  width: 100%;
}

.how-summary {
  cursor: pointer;
  color: var(--ink-faint);
  list-style: none;
}

.how-summary::-webkit-details-marker {
  display: none;
}

.how-summary::before {
  content: '▸ ';
  color: var(--accent);
}

.how[open] .how-summary::before {
  content: '▾ ';
}

.how-code {
  margin: var(--md) 0 0;
  padding: var(--lg);
  width: 100%;
  overflow-x: auto;
  background: var(--bg);
  border: 1px solid var(--hairline);
  color: var(--ink-dim);
  font-family: var(--mono);
  font-size: 11px;
  line-height: 1.7;
  white-space: pre;
}

/* --- pasting ------------------------------------------------------------- */

.paste {
  display: flex;
  flex-direction: column;
  gap: var(--lg);
  width: 100%;
  margin-top: var(--md);
}

.paste-label {
  color: var(--ink-faint);
}

.paste-area {
  width: 100%;
  padding: var(--lg);
  background: var(--bg);
  border: 1px solid var(--hairline);
  color: var(--ink);
  font-family: var(--mono);
  font-size: 11px;
  line-height: 1.7;
  resize: vertical;
  transition: border-color var(--dur-fast) var(--ease-out);
}

.paste-area:focus {
  outline: none;
  border-color: var(--accent);
}

.paste-area::placeholder {
  color: var(--ink-faint);
}

.paste-actions {
  display: flex;
  align-items: center;
  gap: var(--xxl);
}

.step--quiet {
  color: var(--ink-faint);
}

.session-note {
  color: var(--accent);
  text-transform: none;
  letter-spacing: 0.04em;
}

.session-note--error {
  color: var(--danger);
}
</style>
