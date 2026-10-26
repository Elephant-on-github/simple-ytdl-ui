<script setup lang="ts">
import { ref } from 'vue'
import { useSchedulesStore } from '../stores/schedules'
import type { Schedule } from '../lib/api'

const schedules = useSchedulesStore()

const url = ref('')
const mode = ref<'video' | 'music'>('video')
const folder = ref('')
const frequency = ref<'daily' | 'weekly' | 'monthly'>('daily')
const error = ref('')
const loading = ref(false)

async function addSchedule(): Promise<void> {
  error.value = ''
  loading.value = true
  try {
    await schedules.save({
      id: '',
      url: url.value.trim(),
      mode: mode.value,
      folder: folder.value.trim(),
      frequency: frequency.value,
      title: '',
    })
    url.value = ''
    folder.value = ''
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not create schedule'
  } finally {
    loading.value = false
  }
}

async function remove(id: string): Promise<void> {
  try {
    await schedules.remove(id)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not remove schedule'
  }
}

const frequencyLabel = (f: string) => {
  switch (f) {
    case 'daily':
      return 'Daily'
    case 'weekly':
      return 'Weekly'
    case 'monthly':
      return 'Monthly'
    default:
      return f
  }
}

const modeLabel = (m: string) => (m === 'music' ? 'MP3' : 'Video')
</script>

<template>
  <div class="view">
    <header class="head">
      <div>
        <p class="eyebrow">Schedule</h1>
        <h1 class="t-display head-title">Schedule Downloads</h1>
      </div>
    </header>

    <p v-if="error" class="notice notice--error t-meta">{{ error }}</p>

    <section class="card panel">
      <p class="t-label">New schedule</p>

      <form class="form" @submit.prevent="addSchedule">
        <div class="entry">
          <label class="t-label">URL</label>
          <input
            v-model="url"
            class="field entry-input"
            type="text"
            placeholder="https://www.youtube.com/playlist?list=..."
            autocomplete="off"
          />
        </div>

        <div class="entry">
          <label class="t-label">Mode</label>
          <select
            class="field entry-input"
            v-model="mode"
          >
            <option value="video">Video</option>
            <option value="music">MP3</option>
          </select>
        </div>

        <div class="entry">
          <label class="t-label">Folder</label>
          <input
            v-model="folder"
            class="field entry-input"
            type="text"
            placeholder="Optional folder name"
          />
        </div>

        <div class="entry">
          <label class="t-label">Frequency</label>
          <select class="field entry-input" v-model="frequency">
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
          </select>
        </div>

        <button type="submit" class="btn btn--primary entry-go">
          <span class="btn-go-text">Schedule</span>
          <span class="bar-go-arrow" aria-hidden="true">→</span>
        </button>
      </form>
    </section>

    <section class="schedule-list">
      <header class="sec-head">
        <h2 class="t-label">Scheduled downloads</h2>
        <span v-if="schedules.items.length" class="t-meta">
          {{ schedules.items.length }} scheduled
        </span>
      </header>

      <ul v-if="schedules.items.length" class="jobs">
        <li
          v-for="(item, index) in schedules.items"
          :key="item.id"
          class="job"
        >
          <span class="job-index t-label">{{ String(index + 1).padStart(2, '0') }}</span>

          <div class="job-body">
            <div class="job-head">
              <span class="job-title t-italic">{{ item.url }}</span>
              <span class="job-pct t-meta tnum">{{ item.mode === 'music' ? 'MP3' : 'Video' }}</span>
            </div>

            <div class="job-item">
              <span class="pl-item-time t-meta tnum">{{ item.frequency }}</span>
              <span class="pl-item-channel t-meta tnum">{{ item.folder || '(default)' }}</span>
            </div>
          </div>

          <button
            type="button"
            class="job-cancel"
            :aria-label="`Remove ${item.id}`"
            @click="remove(item.id)"
          >
            ×
          </button>
        </li>
      </ul>
    </section>

    <div v-if="!schedules.items.length" class="empty">
      <span class="t-meta">No scheduled downloads yet.</span>
    </section>
  </div>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: var(--gap);
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--lg);
  max-width: 720px;
}

.entry {
  display: flex;
  flex-direction: column;
  gap: var(--sm);
}

.entry label {
  font-size: var(--fs-label);
  color: var(--faint);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.entry-input {
  width: 100%;
}

.entry-go {
  width: 100%;
}

.schedule-list {
  margin-top: var(--gap);
}

.plates {
  display: flex;
  flex-direction: column;
  gap: var(--lg);
}

.plate {
  border-bottom: 1px solid var(--hairline);
  padding: var(--xl) 0;
}

.plate-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-4);
  padding-bottom: var(--sp-5);
  border-bottom: 1px solid var(--hairline);
}

.plate-title {
  font-size: var(--fs-body);
  color: var(--ink);
}

.muted {
  color: var(--faint);
}

.empty {
  padding: var(--gap) 0;
}

.empty .t-meta {
  color: var(--ink-faint);
}

.job {
  border-bottom: 1px solid var(--hairline);
  padding: var(--sp-7) 0;
  display: flex;
  align-items: center;
  gap: var(--xl);
}

.job-index {
  color: var(--ink-faint);
  padding-top: 4px;
}

.job-body {
  flex: 1;
}

.job-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--xl);
}

.job-title {
  font-size: 17px;
  color: var(--ink);
}

.job-pct {
  color: var(--ink-dim);
  flex: none;
}

.job-item {
  display: flex;
  gap: var(--xl);
  color: var(--ink-faint);
  font-size: 14px;
}

.job-action {
  color: var(--ink-faint);
  width: 22px;
  height: 22px;
  display: grid;
  place-items: center;
  font-family: var(--mono);
  font-size: 10px;
  transition: color var(--dur-fast) var(--ease-out);
}

.job-action:hover {
  color: var(--accent);
}

.btn--secondary {
  height: 40px;
  padding: 0 var(--sp-4);
  border-radius: var(--r-md);
  border: 1px solid var(--hairline);
  background: transparent;
  color: var(--text);
  font-size: var(--fs-13);
  font-weight: 500;
  letter-spacing: -0.004em;
  white-space: nowrap;
  transition:
    background var(--dur-fast) var(--ease-out),
    border-color var(--dur-fast) var(--ease-out),
    color var(--dur-fast) var(--ease-out);
}

.btn--secondary:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.04);
  border-color: var(--hairline-strong);
}

.btn--danger:hover {
  color: var(--danger);
}
</style>