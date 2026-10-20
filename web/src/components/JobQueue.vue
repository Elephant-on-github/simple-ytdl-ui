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
      <h2 class="t-title sec-title">Queue</h2>
      <span v-if="jobs.items.length" class="chip tnum">{{ jobs.items.length }}</span>
      <span v-if="activeCount" class="chip chip--accent">{{ activeCount }} active</span>
    </header>

    <p v-if="jobs.error" class="notice notice--error">{{ jobs.error }}</p>

    <div v-if="!jobs.items.length" class="empty panel">
      <svg width="20" height="20" viewBox="0 0 18 18" fill="none" aria-hidden="true">
        <path
          d="M9 2.6v8.8m0 0 3.3-3.3M9 11.4l-3.3-3.3M3.7 14.7h10.6"
          stroke="currentColor"
          stroke-width="1.4"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
      <p class="t-meta muted">Nothing queued yet. Paste a link above to start.</p>
    </div>

    <ul v-else class="jobs">
      <li v-for="job in jobs.items" :key="job.id" class="job" :class="`job--${job.status}`">
        <span class="job-tag" :title="job.mode === 'music' ? 'MP3' : 'Video'">{{ modeTag(job) }}</span>

        <div class="job-body">
          <div class="job-head">
            <span class="job-title clamp-2">{{ job.title || job.url || 'Untitled' }}</span>
            <span class="job-pct tnum">{{ percent(job) }}<i>%</i></span>
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

          <p v-if="job.status === 'failed' && job.error" class="job-error">{{ job.error }}</p>
          <p v-else class="job-meta">
            <template v-for="(segment, index) in metaSegments(job)" :key="index">
              <span v-if="index > 0" class="dot" aria-hidden="true" />
              <span :class="index === 0 ? 'job-status' : 'tnum'">{{ segment }}</span>
            </template>
          </p>
        </div>

        <button
          v-if="isActive(job)"
          type="button"
          class="job-cancel"
          title="Cancel download"
          aria-label="Cancel download"
          @click="jobs.cancel(job.id)"
        >
          <svg width="13" height="13" viewBox="0 0 14 14" fill="none" aria-hidden="true">
            <path
              d="M3.5 3.5l7 7m0-7-7 7"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
            />
          </svg>
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.sec-head {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-bottom: var(--sp-4);
}

.sec-title {
  margin-right: var(--sp-1);
}

.empty {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-5);
  color: var(--faint);
}

.jobs {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}

.job {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr) auto;
  align-items: start;
  gap: var(--sp-4);
  padding: var(--sp-4) var(--sp-4) var(--sp-4) var(--sp-4);
  background: var(--panel);
  border: 1px solid var(--hairline);
  border-radius: var(--r-lg);
  transition:
    border-color var(--dur) var(--ease),
    opacity var(--dur) var(--ease);
}

.job:hover {
  border-color: var(--hairline-strong);
}

.job--completed {
  opacity: 0.72;
}

.job--canceled {
  opacity: 0.5;
}

.job--failed {
  border-color: var(--danger-line);
}

.job-tag {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: var(--r-sm);
  background: var(--accent-soft);
  border: 1px solid var(--accent-line);
  color: var(--accent);
  font-size: var(--fs-11);
  font-weight: 600;
  letter-spacing: 0.04em;
}

.job--failed .job-tag {
  background: var(--danger-soft);
  border-color: var(--danger-line);
  color: var(--danger);
}

.job-body {
  min-width: 0;
}

.job-head {
  display: flex;
  align-items: baseline;
  gap: var(--sp-4);
  margin-bottom: 10px;
}

.job-title {
  flex: 1;
  min-width: 0;
  font-size: var(--fs-13);
  font-weight: 500;
  line-height: 1.4;
  color: var(--text);
}

.job-pct {
  font-size: var(--fs-20);
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.03em;
  color: var(--text);
}

.job-pct i {
  font-style: normal;
  font-size: var(--fs-12);
  font-weight: 500;
  color: var(--faint);
  margin-left: 1px;
}

.job--failed .job-pct {
  color: var(--danger);
}

.bar {
  height: 3px;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.07);
  overflow: hidden;
}

.bar-fill {
  display: block;
  height: 100%;
  border-radius: 2px;
  background: var(--accent);
  transition: width var(--dur-slow) var(--ease);
}

.job--completed .bar-fill {
  background: rgba(255, 255, 255, 0.28);
}

.job--canceled .bar-fill {
  background: rgba(255, 255, 255, 0.18);
}

.job--failed .bar-fill {
  background: var(--danger);
}

.job-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--sp-2);
  margin-top: 10px;
  font-size: var(--fs-12);
  color: var(--muted);
}

.job-status {
  color: var(--text-dim);
  font-weight: 500;
}

.job--failed .job-status {
  color: var(--danger);
}

.dot {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--faint);
}

.job-error {
  margin-top: 10px;
  font-size: var(--fs-12);
  line-height: 1.5;
  color: var(--danger);
  word-break: break-word;
}

.job-cancel {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: var(--r-sm);
  border: 1px solid var(--hairline);
  color: var(--faint);
  transition:
    color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease),
    background var(--dur-fast) var(--ease);
}

.job-cancel:hover {
  color: var(--danger);
  border-color: var(--danger-line);
  background: var(--danger-soft);
}

.notice {
  margin-bottom: var(--sp-4);
  padding: 10px var(--sp-3);
  border-radius: var(--r-sm);
  font-size: var(--fs-12);
}

.notice--error {
  color: var(--danger);
  background: var(--danger-soft);
  border: 1px solid var(--danger-line);
}

@media (max-width: 560px) {
  .job {
    grid-template-columns: 32px minmax(0, 1fr) auto;
    gap: var(--sp-3);
    padding: var(--sp-3);
  }

  .job-tag {
    width: 32px;
    height: 32px;
    font-size: 10px;
  }

  .job-head {
    gap: var(--sp-2);
  }

  .job-pct {
    font-size: var(--fs-15);
  }
}
</style>
