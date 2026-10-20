<script setup lang="ts">
import { useModeStore } from '../stores/mode'

const mode = useModeStore()
</script>

<template>
  <div class="seg" role="group" aria-label="Download quality">
    <button
      type="button"
      class="seg-btn t-nav"
      :class="{ 'seg-btn--on': mode.mode === 'video' }"
      :aria-pressed="mode.mode === 'video'"
      @click="mode.set('video')"
    >
      Video
    </button>
    <button
      type="button"
      class="seg-btn t-nav"
      :class="{ 'seg-btn--on': mode.mode === 'music' }"
      :aria-pressed="mode.mode === 'music'"
      @click="mode.set('music')"
    >
      MP3
    </button>
  </div>
</template>

<style scoped>
.seg {
  display: inline-flex;
  border: 1px solid var(--hairline);
}

.seg-btn {
  position: relative;
  height: 34px;
  padding: 0 var(--xl);
  color: var(--ink-dim);
  transition: color var(--dur-fast) var(--ease-out);
}

/* A hairline separates the two, never a fill. */
.seg-btn + .seg-btn {
  border-left: 1px solid var(--hairline);
}

.seg-btn:hover {
  color: var(--ink);
}

/* The active one is marked with the accent and a rule under it — the amber
   is a signal, never a fill. */
.seg-btn--on {
  color: var(--accent);
}

.seg-btn--on::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 1px;
  background: var(--accent);
}

.seg-btn--on:hover {
  color: var(--accent);
}

@media (max-width: 420px) {
  .seg-btn {
    padding: 0 var(--md);
  }
}
</style>
