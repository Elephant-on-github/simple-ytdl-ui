<script setup lang="ts">
import { computed } from 'vue'
import { useJobsStore } from '../stores/jobs'
import { formatBytes, formatEta, formatRelative, formatSpeed } from '../lib/format'
import type { Job } from '../lib/api'

const jobs = useJobsStore()

const statusText: Record<Job['status'], string> = {
  queued: 'Queued',
  running: 'Downloading',
  completed: 'Completed',
  failed: 'Failed',
  canceled: 'Canceled',
}

function isActive(job: Job): boolean {
  return job.status === 'queued' || job.status === 'running'
}

function percent(job: Job): number {
  if (job.status === 'completed') return 100
  return Math.max(0, Math.min(100, Math.round(job.progress || 0)))
}

function modeTag(job: Job): string {
  return job.mode === 'music' ? 'MP3' : 'MP4'
}

/** The small line under the progress bar: status, then the useful numbers. */
function metaSegments(job: Job): string[] {
  switch (job.status) {
    case 'completed':
      return [statusText.completed, formatBytes(job.downloaded)]
    case 'failed':
      return [statusText.failed]
    case 'canceled':
      return [statusText.canceled, formatRelative(job.createdAt)]
    default:
      return [statusText[job.status], formatSpeed(job.speed), `${formatEta(job.eta)} left`]
  }
}

const activeCount = computed(() => jobs.activeCount)
</script>

<template>
  <section class="queue">
    <header class="sec-head">
      <h2 class="t-label">Queue</h2>
      <span v-if="jobs.items.length" class="t-meta">
        {{ jobs.items.length }} {{ jobs.items.length === 1 ? 'entry' : 'entries' }}
        <template v-if="activeCount"> · {{ activeCount }} active</template>
      </span>
    </header>

    <p v-if="jobs.error" class="notice t-meta">{{ jobs.error }}</p>

    <div v-if="!jobs.items.length" class="empty">
      <span class="t-meta">Nothing queued yet. Paste a link above to start.</span>
    </div>

    <ul v-else class="jobs">
      <li v-for="(job, index) in jobs.items" :key="job.id" class="job" :class="`job--${job.status}`">
        <span class="job-index t-label">{{ String(index + 1).padStart(2, '0') }}</span>

        <div class="job-body">
          <div class="job-head">
            <span class="job-title t-italic">{{ job.title || job.url || 'Untitled' }}</span>
            <span class="job-pct t-meta tnum">{{ percent(job) }}%</span>
          </div>

          <div
            class="bar"
            role="progressbar"
            :aria-valuenow="percent(job)"
            aria-valuemin="0"
            aria-valuemax="100"
            :aria-label="job.title"
          >
            <span class="bar-fill" :style="{ width: `${percent(job)}%` }" />
          </div>

          <p v-if="job.status === 'failed' && job.error" class="job-error t-meta">
            {{ job.error }}
          </p>
          <p v-else class="job-meta t-meta">
            <template v-for="(segment, i) in metaSegments(job)" :key="i">
              <span class="dot" v-if="i > 0" aria-hidden="true">·</span>
              <span :class="{ 'job-status': i === 0 }">{{ segment }}</span>
            </template>
          </p>
        </div>

        <span class="job-tag t-meta">{{ modeTag(job) }}</span>

        <button
          v-if="isActive(job)"
          type="button"
          class="job-cancel t-nav"
          title="Cancel download"
          aria-label="Cancel download"
          @click="jobs.cancel(job.id)"
        >
          ×
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.queue {
  display: flex;
  flex-direction: column;
  gap: var(--xl);
}

/* --- section head -------------------------------------------------------- */

.sec-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--xxl);
  padding-bottom: var(--lg);
  border-bottom: 1px solid var(--hairline);
}

.sec-head .t-label {
  color: var(--accent);
}

.sec-head .t-meta {
  color: var(--ink-faint);
}

/* --- empty --------------------------------------------------------------- */

.empty {
  padding: var(--xxl) 0;
  border-bottom: 1px solid var(--hairline);
}

.empty .t-meta {
  color: var(--ink-faint);
}

/* --- jobs ---------------------------------------------------------------- */

.jobs {
  display: flex;
  flex-direction: column;
}

.job {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: start;
  gap: var(--xxl);
  padding: var(--xl) 0;
  border-bottom: 1px solid var(--hairline);
  transition: background var(--dur-fast) var(--ease-out);
}

.job:hover {
  background: var(--plate);
}

.job-index {
  color: var(--ink-faint);
  padding-top: 4px;
}

.job-body {
  min-width: 0;
}

.job-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--xxl);
}

.job-title {
  font-size: clamp(18px, 1.6vw, 22px);
  color: var(--ink);
  min-width: 0;
}

.job-pct {
  color: var(--ink-dim);
  flex: none;
}

/* The progress is a rule that fills, not a bar with a radius. */
.bar {
  position: relative;
  height: 1px;
  margin-top: var(--md);
  background: var(--hairline);
}

.bar-fill {
  position: absolute;
  inset: 0 auto 0 0;
  background: var(--ink);
  transition: width var(--dur-fast) var(--ease-out);
}

.job--running .bar-fill {
  background: var(--accent);
}

.job--failed .bar-fill,
.job--canceled .bar-fill {
  background: var(--ink-faint);
}

.job-meta {
  margin-top: var(--md);
  color: var(--ink-faint);
  display: flex;
  align-items: center;
  gap: var(--sm);
  flex-wrap: wrap;
}

.job-meta .dot {
  color: var(--ink-faint);
}

.job-status {
  color: var(--ink-dim);
}

.job--running .job-status {
  color: var(--accent);
}

.job-error {
  margin-top: var(--md);
  color: var(--accent);
  text-transform: none;
  letter-spacing: 0.04em;
  word-break: break-word;
}

.job-tag {
  color: var(--ink-faint);
  padding-top: 4px;
}

.job-cancel {
  color: var(--ink-faint);
  width: 22px;
  height: 22px;
  display: grid;
  place-items: center;
  transition: color var(--dur-fast) var(--ease-out);
}

.job-cancel:hover {
  color: var(--accent);
}

.notice {
  color: var(--accent);
  text-transform: none;
  letter-spacing: 0.04em;
}

@media (max-width: 620px) {
  .job {
    grid-template-columns: auto minmax(0, 1fr) auto;
    gap: var(--lg);
  }

  .job-tag {
    display: none;
  }
}
</style>
