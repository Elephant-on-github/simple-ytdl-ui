<script setup lang="ts">
import { ref } from 'vue'
import { api, type Item } from '../lib/api'
import { formatDuration } from '../lib/format'

defineProps<{ items: Item[] }>()

const emit = defineEmits<{
  (e: 'play', item: Item): void
  (e: 'remove', item: Item): void
}>()

const broken = ref<Set<string>>(new Set())
const confirming = ref<string | null>(null)
let confirmTimer: ReturnType<typeof setTimeout> | undefined

/** The plate number, in roman. */
function roman(index: number): string {
  const numerals = ['I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII', 'IX', 'X']
  if (index < numerals.length) return numerals[index]
  return String(index + 1)
}

function markBroken(id: string): void {
  const next = new Set(broken.value)
  next.add(id)
  broken.value = next
}

function askRemove(item: Item): void {
  if (confirming.value === item.id) {
    clearTimeout(confirmTimer)
    confirming.value = null
    emit('remove', item)
    return
  }
  confirming.value = item.id
  clearTimeout(confirmTimer)
  confirmTimer = setTimeout(() => {
    confirming.value = null
  }, 3000)
}

/** A video download is watched, not played in the audio bar. */
function isVideo(item: Item): boolean {
  return (item.mime ?? '').startsWith('video/')
}

function open(item: Item): void {
  if (isVideo(item)) {
    window.open(api.streamUrl(item.id), '_blank', 'noopener')
    return
  }
  emit('play', item)
}
</script>

<template>
  <ul class="plates">
    <li v-for="(item, index) in items" :key="item.id" class="plate">
      <div class="art frame" :class="{ 'art--broken': broken.has(item.id) }">
        <span class="tick tick--tl" aria-hidden="true" />
        <span class="tick tick--tr" aria-hidden="true" />
        <span class="tick tick--bl" aria-hidden="true" />
        <span class="tick tick--br" aria-hidden="true" />

        <img
          v-if="!broken.has(item.id)"
          class="art-img"
          :src="api.coverUrl(item.id)"
          :alt="item.title"
          loading="lazy"
          @error="markBroken(item.id)"
        />
        <span v-else class="art-fallback t-meta">No Cover</span>

        <span class="art-numeral" aria-hidden="true">{{ roman(index) }}</span>
        <span class="art-time t-meta tnum">{{ formatDuration(item.duration) }}</span>
      </div>

      <div class="meta">
        <div class="meta-top">
          <span class="t-label">{{ item.mode === 'music' ? 'MP3' : 'MP4' }}</span>
          <button
            type="button"
            class="remove t-nav"
            :class="{ 'remove--confirm': confirming === item.id }"
            :aria-label="`Delete ${item.title}`"
            @click="askRemove(item)"
          >
            {{ confirming === item.id ? 'Sure?' : 'Delete' }}
          </button>
        </div>

        <h3 class="title t-plate-title">{{ item.title }}</h3>
        <p class="artist t-meta truncate">
          <template v-if="item.artist">{{ item.artist }}</template>
          <template v-else>Unknown artist</template>
        </p>

        <button type="button" class="step plate-action" @click="open(item)">
          {{ isVideo(item) ? 'Watch' : 'Play' }}
          <span aria-hidden="true">→</span>
        </button>
      </div>
    </li>
  </ul>
</template>

<style scoped>
.plates {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--gap) var(--xxl);
}

/* --- the plate ----------------------------------------------------------- */

.plate {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--lg);
}

/* The art frame: a plate of black with the cover in it. */
.art {
  position: relative;
  aspect-ratio: 1;
  background: var(--plate);
  overflow: hidden;
}

.art-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 1.2s var(--ease-out);
}

.plate:hover .art-img {
  transform: scale(1.04);
}

/* The artwork fades into the page at the top and the bottom. */
.art::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(
    to bottom,
    rgba(0, 0, 0, 0.55) 0%,
    rgba(0, 0, 0, 0) 26%,
    rgba(0, 0, 0, 0) 62%,
    rgba(0, 0, 0, 0.78) 100%
  );
}

.art-fallback {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: var(--ink-faint);
  letter-spacing: 0.32em;
}

/* The roman numeral sits inside the plate. */
.art-numeral {
  position: absolute;
  left: var(--lg);
  bottom: var(--lg);
  z-index: 1;
  font-family: var(--serif);
  font-size: clamp(28px, 3.4vw, 46px);
  line-height: 0.95;
  color: var(--roman-overlay);
  text-shadow: 0 0 16px rgba(0, 0, 0, 0.45);
}

.art-time {
  position: absolute;
  right: var(--lg);
  bottom: var(--lg);
  z-index: 1;
  color: var(--roman-overlay);
}

/* --- meta ---------------------------------------------------------------- */

.meta {
  display: flex;
  flex-direction: column;
  gap: var(--sm);
}

.meta-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--lg);
}

.meta-top .t-label {
  color: var(--accent);
}

.remove {
  color: var(--ink-faint);
  transition: color var(--dur-fast) var(--ease-out);
}

.remove:hover {
  color: var(--accent);
}

.remove--confirm {
  color: var(--accent);
}

.title {
  color: var(--ink);
  transition: color var(--dur-fast) var(--ease-out);
}

.plate:hover .title {
  color: var(--accent);
}

.artist {
  color: var(--ink-faint);
}

.plate-action {
  margin-top: var(--md);
  align-self: flex-start;
  color: var(--ink-dim);
}

.plate:hover .plate-action {
  color: var(--ink);
}
</style>
