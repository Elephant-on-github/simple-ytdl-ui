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

let timer: number | undefined

function tick(): void {
  // The queue is only worth polling while something is in flight.
  if (jobs.activeCount > 0) void jobs.refresh()
}

onMounted(() => {
  void jobs.refresh()
  timer = window.setInterval(tick, 1000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
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
    <section class="hero">
      <span class="hero-wash" aria-hidden="true" />
      <span class="hero-wash hero-wash--right" aria-hidden="true" />

      <p class="eyebrow">
        <span class="eyebrow-dot" aria-hidden="true" />
        Paste a link
      </p>

      <h1 class="t-display hero-title">Take it with you,<br />in the quality you want.</h1>

      <p class="lede t-body">
        <template v-if="mode.mode === 'music'">
          Any YouTube video or playlist, pulled as a <strong>best-quality MP3</strong> with cover art
          and tags baked in.
        </template>
        <template v-else>
          Any YouTube video or playlist. Keep the <strong>original video</strong>, or pull a
          best-quality MP3 with cover art and tags baked in.
        </template>
      </p>

      <form class="entry" @submit.prevent="submit">
        <div class="entry-field">
          <svg
            class="entry-icon"
            width="16"
            height="16"
            viewBox="0 0 18 18"
            fill="none"
            aria-hidden="true"
          >
            <path
              d="M6.6 11.4V3.8l7-1.2v7.6"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
            <circle cx="4.8" cy="11.6" r="1.9" stroke="currentColor" stroke-width="1.5" />
            <circle cx="11.8" cy="10.4" r="1.9" stroke="currentColor" stroke-width="1.5" />
          </svg>
          <input
            v-model="url"
            class="field entry-input"
            type="text"
            inputmode="url"
            spellcheck="false"
            autocomplete="off"
            placeholder="https://www.youtube.com/watch?v=…"
            aria-label="YouTube URL"
            @input="error = ''"
          />
        </div>

        <ModeToggle />

        <button type="submit" class="btn btn--primary entry-go" :disabled="submitting">
          {{ submitting ? 'Starting…' : 'Download' }}
        </button>
      </form>

      <p v-if="error" class="entry-error t-fine">{{ error }}</p>
      <p v-else class="entry-hint t-fine">
        Works with youtube.com, youtu.be and <code>playlist?list=</code> links.
      </p>
    </section>

    <JobQueue />
  </div>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: var(--sp-8);
  max-width: var(--content-max);
}

/* --- hero ---------------------------------------------------------------- */

.hero {
  position: relative;
  overflow: hidden;
  padding: var(--sp-8) var(--sp-8) var(--sp-7);
  background: var(--panel);
  border: 1px solid var(--hairline);
  border-radius: var(--r-xl);
}

.hero-wash {
  position: absolute;
  top: -220px;
  left: -140px;
  width: 620px;
  height: 480px;
  pointer-events: none;
  background: radial-gradient(closest-side, var(--accent-wash), transparent 72%);
  opacity: 0.85;
}

.hero-wash--right {
  top: -280px;
  left: auto;
  right: -200px;
  width: 520px;
  height: 420px;
  opacity: 0.4;
}

.eyebrow {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: var(--fs-12);
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--muted);
}

.eyebrow-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.hero-title {
  position: relative;
  margin-top: var(--sp-4);
  max-width: 17ch;
}

.lede {
  position: relative;
  margin-top: var(--sp-4);
  max-width: 54ch;
  color: var(--muted);
}

.lede strong {
  color: var(--text-dim);
  font-weight: 500;
}

/* --- entry --------------------------------------------------------------- */

.entry {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: var(--sp-3);
  margin-top: var(--sp-7);
}

.entry-field {
  position: relative;
  display: flex;
  align-items: center;
}

.entry-icon {
  position: absolute;
  left: 14px;
  color: var(--faint);
  pointer-events: none;
  transition: color var(--dur-fast) var(--ease);
}

.entry-field:focus-within .entry-icon {
  color: var(--accent);
}

.entry-input {
  padding-left: 40px;
  font-size: var(--fs-15);
}

.entry-go {
  min-width: 118px;
  height: 46px;
}

.entry-hint {
  position: relative;
  margin-top: var(--sp-3);
  color: var(--faint);
}

.entry-hint code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  padding: 1px 5px;
  border-radius: var(--r-xs);
  background: rgba(255, 255, 255, 0.05);
  color: var(--muted);
}

.entry-error {
  position: relative;
  margin-top: var(--sp-3);
  color: var(--danger);
}

@media (max-width: 720px) {
  .hero {
    padding: var(--sp-6) var(--sp-5) var(--sp-6);
  }

  .hero-title {
    max-width: none;
  }

  .entry {
    grid-template-columns: minmax(0, 1fr);
    justify-items: stretch;
  }

  .entry :deep(.seg) {
    justify-self: start;
  }

  .entry-go {
    width: 100%;
  }
}
</style>
