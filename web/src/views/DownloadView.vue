<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import ModeToggle from '../components/ModeToggle.vue'
import JobQueue from '../components/JobQueue.vue'
import { useJobsStore } from '../stores/jobs'
import { useModeStore } from '../stores/mode'
import { isPlaylistURL, isValidYouTubeURL } from '../lib/validate'

const mode = useModeStore()
const jobs = useJobsStore()

const url = ref('')
const error = ref('')
const submitting = ref(false)

let timer: ReturnType<typeof setInterval> | undefined

function tick(): void {
  // The queue is only worth polling while something is in flight.
  if (jobs.activeCount > 0) void jobs.refresh()
}

onMounted(() => {
  void jobs.refresh()
  timer = setInterval(tick, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

async function submit(): Promise<void> {
  const value = url.value.trim()

  if (!value) {
    error.value = 'Paste a YouTube link first.'
    return
  }
  if (!isValidYouTubeURL(value)) {
    error.value = 'That does not look like a YouTube video or playlist link.'
    return
  }

  error.value = ''
  submitting.value = true
  try {
    await jobs.enqueue(value, mode.mode, isPlaylistURL(value))
    url.value = ''
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not start the download.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="view">
    <!-- The download bar is the main thing on this page. -->
    <section class="console frame">
      <span class="tick tick--tl" aria-hidden="true" />
      <span class="tick tick--tr" aria-hidden="true" />
      <span class="tick tick--bl" aria-hidden="true" />
      <span class="tick tick--br" aria-hidden="true" />

      <form class="bar" @submit.prevent="submit">
        <label class="t-label bar-label" for="download-url">Source</label>
        <input
          id="download-url"
          v-model="url"
          class="bar-input"
          type="text"
          inputmode="url"
          spellcheck="false"
          autocomplete="off"
          placeholder="https://www.youtube.com/watch?v=…"
          aria-label="YouTube URL"
          @input="error = ''"
        />
        <button type="submit" class="bar-go" :disabled="submitting">
          <span class="bar-go-text">{{ submitting ? 'Starting' : 'Download' }}</span>
          <span class="bar-go-arrow" aria-hidden="true">→</span>
        </button>
      </form>

      <!-- The settings sit underneath the bar. -->
      <div class="settings">
        <div class="setting">
          <span class="t-label setting-label">Output</span>
          <ModeToggle />
        </div>

        <p class="setting-note t-meta">
          <template v-if="mode.mode === 'music'">
            Best quality MP3, cover art and tags included
          </template>
          <template v-else>
            Original video, best available quality
          </template>
        </p>
      </div>

      <p v-if="error" class="note note--error t-meta">{{ error }}</p>
      <p v-else class="note t-meta">
        Works with youtube.com, youtu.be and playlist?list= links
      </p>
    </section>

    <JobQueue />
  </div>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: var(--gap);
}

/* --- the console --------------------------------------------------------- */

.console {
  display: flex;
  flex-direction: column;
  gap: var(--xl);
  padding: clamp(24px, 4vw, 56px) clamp(20px, 3.4vw, 48px) clamp(20px, 3vw, 40px);
  background: var(--plate);
}

/* --- the bar ------------------------------------------------------------- */

.bar {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--xl);
  padding-bottom: var(--xl);
  border-bottom: 1px solid var(--hairline);
}

.bar-label {
  color: var(--ink-faint);
}

/* The field is the loudest thing on the page. */
.bar-input {
  width: 100%;
  min-width: 0;
  padding: 0;
  background: transparent;
  border: 0;
  color: var(--ink);
  font-family: var(--serif);
  font-size: clamp(24px, 3.4vw, 44px);
  line-height: 1.1;
  letter-spacing: -0.01em;
}

.bar-input::placeholder {
  color: var(--ink-faint);
}

.bar-input:focus-visible {
  outline: 1px solid var(--accent);
  outline-offset: 2px;
}

.bar-go {
  display: inline-flex;
  align-items: center;
  gap: var(--md);
  padding: var(--md) var(--xl);
  border: 1px solid var(--accent);
  color: var(--accent);
  font-family: var(--mono);
  font-size: var(--fs-nav);
  font-weight: 400;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  white-space: nowrap;
  transition:
    background var(--dur-fast) var(--ease-out),
    color var(--dur-fast) var(--ease-out);
}

.bar-go:hover:not(:disabled) {
  background: var(--accent);
  color: var(--bg);
}

.bar-go:disabled {
  border-color: var(--hairline);
  color: var(--ink-faint);
  cursor: progress;
}

.bar-go-arrow {
  transition: transform var(--dur) var(--ease-out);
}

.bar-go:hover:not(:disabled) .bar-go-arrow {
  transform: translateX(8px);
}

/* --- settings ------------------------------------------------------------ */

.settings {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--xxl);
  flex-wrap: wrap;
}

.setting {
  display: flex;
  align-items: center;
  gap: var(--xl);
}

.setting-label {
  color: var(--ink-faint);
}

.setting-note {
  color: var(--ink-faint);
  text-transform: none;
  letter-spacing: 0.04em;
}

/* --- note ---------------------------------------------------------------- */

.note {
  color: var(--ink-faint);
}

.note--error {
  color: var(--accent);
}

@media (max-width: 720px) {
  .bar {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--lg);
  }

  .bar-label {
    order: -1;
  }

  .bar-go {
    justify-content: space-between;
  }
}
</style>
