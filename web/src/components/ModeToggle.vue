<script setup lang="ts">
import { useModeStore } from '../stores/mode'

const mode = useModeStore()
</script>

<template>
  <div class="seg" role="group" aria-label="Download quality">
    <button
      type="button"
      class="seg-btn"
      :class="{ 'seg-btn--on': mode.mode === 'video' }"
      :aria-pressed="mode.mode === 'video'"
      @click="mode.set('video')"
    >
      <svg width="15" height="15" viewBox="0 0 18 18" fill="none" aria-hidden="true">
        <rect
          x="2.4"
          y="4.2"
          width="13.2"
          height="9.6"
          rx="2"
          stroke="currentColor"
          stroke-width="1.5"
        />
        <path d="M8 7.2 11.6 9 8 10.8V7.2Z" fill="currentColor" />
      </svg>
      <span>Video</span>
    </button>
    <button
      type="button"
      class="seg-btn"
      :class="{ 'seg-btn--on': mode.mode === 'music' }"
      :aria-pressed="mode.mode === 'music'"
      @click="mode.set('music')"
    >
      <svg width="15" height="15" viewBox="0 0 18 18" fill="none" aria-hidden="true">
        <path
          d="M6.4 12.6V4.2l7-1.2v8.4"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <circle cx="4.6" cy="12.8" r="1.9" stroke="currentColor" stroke-width="1.5" />
        <circle cx="11.6" cy="11.6" r="1.9" stroke="currentColor" stroke-width="1.5" />
      </svg>
      <span>MP3</span>
    </button>
  </div>
</template>

<style scoped>
.seg {
  display: inline-flex;
  gap: 2px;
  padding: 3px;
  border-radius: var(--r-md);
  background: rgba(255, 255, 255, 0.035);
  border: 1px solid var(--hairline);
}

.seg-btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 38px;
  padding: 0 16px;
  border-radius: 7px;
  color: var(--muted);
  font-size: var(--fs-13);
  font-weight: 500;
  letter-spacing: -0.004em;
  transition:
    background var(--dur) var(--ease),
    color var(--dur) var(--ease),
    box-shadow var(--dur) var(--ease);
}

.seg-btn:hover {
  color: var(--text-dim);
}

.seg-btn--on {
  background: var(--accent);
  color: var(--accent-ink);
  font-weight: 600;
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.12) inset;
}

.seg-btn--on:hover {
  color: var(--accent-ink);
}

@media (max-width: 420px) {
  .seg-btn {
    padding: 0 10px;
  }
}
</style>
