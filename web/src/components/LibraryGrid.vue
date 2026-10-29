<script setup lang="ts">
import { ref, computed } from 'vue'
import { api, type Item } from '../lib/api'
import { formatDuration } from '../lib/format'

const props = defineProps<{ items: Item[] }>()

const emit = defineEmits<{
  (e: 'play', item: Item): void
  (e: 'remove', item: Item): void
  (e: 'playPlaylist', items: Item[]): void
}>()

const broken = ref<Set<string>>(new Set())
const confirming = ref<string | null>(null)
const countdown = ref(3)
let confirmTimer: ReturnType<typeof setTimeout> | undefined
let countdownTimer: ReturnType<typeof setInterval> | undefined

/** Grouped view: playlists as compound cards, singles as individual plates. */
interface Group {
  kind: 'playlist' | 'single'
  key: string
  title: string
  items: Item[]
  coverIds: string[]
  duration: number
  mode: string
}

const groups = computed<Group[]>(() => {
  const map = new Map<string, Item[]>()
  const singles: Item[] = []
  for (const it of props.items) {
    if (it.playlist) {
      const list = map.get(it.playlist) ?? []
      list.push(it)
      map.set(it.playlist, list)
    } else {
      singles.push(it)
    }
  }
  const out: Group[] = []
  for (const [title, list] of map) {
    out.push({
      kind: 'playlist',
      key: `pl:${title}`,
      title,
      items: list,
      coverIds: list.slice(0, 4).map((i) => i.id),
      duration: list.reduce((s, i) => s + (i.duration || 0), 0),
      mode: list[0]?.mode ?? 'music',
    })
  }
  for (const it of singles) {
    out.push({
      kind: 'single',
      key: `single:${it.id}`,
      title: it.title,
      items: [it],
      coverIds: [it.id],
      duration: it.duration,
      mode: it.mode,
    })
  }
  return out
})

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
    clearInterval(countdownTimer)
    confirming.value = null
    countdown.value = 3
    emit('remove', item)
    return
  }
  confirming.value = item.id
  countdown.value = 3
  clearTimeout(confirmTimer)
  clearInterval(countdownTimer)
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      clearInterval(countdownTimer)
    }
  }, 1000)
  confirmTimer = setTimeout(() => {
    confirming.value = null
    countdown.value = 3
    clearInterval(countdownTimer)
  }, 3000)
}

/** A video download is watched, not played in the audio bar. */
function isVideo(item: Item): boolean {
  return (item.mime ?? '').startsWith('video/')
}

function openGroup(group: Group): void {
  if (group.kind === 'playlist') {
    emit('playPlaylist', group.items)
    return
  }
  const item = group.items[0]
  if (isVideo(item)) {
    window.open(api.streamUrl(item.id), '_blank', 'noopener')
    return
  }
  emit('play', item)
}


</script>

<template>
  <ul class="plates">
    <li v-for="(group, index) in groups" :key="group.key" class="plate">
      <div
        class="art frame"
        :class="{ 'art--broken': group.coverIds.every((id) => broken.has(id)) }"
        role="button"
        tabindex="0"
        :aria-label="`Play ${group.title}`"
        @click="openGroup(group)"
        @keydown.enter="openGroup(group)"
        @keydown.space.prevent="openGroup(group)"
      >
        <span class="tick tick--tl" aria-hidden="true" />
        <span class="tick tick--tr" aria-hidden="true" />
        <span class="tick tick--bl" aria-hidden="true" />
        <span class="tick tick--br" aria-hidden="true" />

        <!-- Compound art for playlists: 2x2 grid of covers -->
        <div v-if="group.kind === 'playlist'" class="compound">
          <img
            v-for="id in group.coverIds"
            :key="id"
            class="compound-cell"
            :src="api.coverUrl(id)"
            :alt="group.title"
            loading="lazy"
            @error="markBroken(id)"
          />
          <div v-if="group.coverIds.length < 4" class="compound-fill" />
        </div>

        <!-- Single cover -->
        <img
          v-else-if="!broken.has(group.coverIds[0])"
          class="art-img"
          :src="api.coverUrl(group.coverIds[0])"
          :alt="group.title"
          loading="lazy"
          @error="markBroken(group.coverIds[0])"
        />
        <span v-else class="art-fallback t-meta">No Cover</span>

        <span class="art-numeral" aria-hidden="true">{{ roman(index) }}</span>
        <span class="art-time t-meta tnum">{{ formatDuration(group.duration) }}</span>
        <span class="art-play" aria-hidden="true">▶</span>
        <span v-if="group.kind === 'playlist'" class="art-count t-meta tnum">{{ group.items.length }}</span>
      </div>

      <div class="meta">
        <div class="meta-top">
          <span class="t-label">{{ group.kind === 'playlist' ? 'PLAYLIST' : (group.mode === 'music' ? 'MP3' : 'MP4') }}</span>
          <button
            v-if="group.kind === 'single'"
            type="button"
            class="remove t-nav"
            :class="{ 'remove--confirm': confirming === group.items[0].id }"
            :aria-label="`Delete ${group.title}`"
            @click="askRemove(group.items[0])"
          >
            {{ confirming === group.items[0].id ? `Sure? (${countdown})` : 'Delete' }}
          </button>
          <button
            v-else
            type="button"
            class="remove t-nav"
            :aria-label="`Delete playlist ${group.title}`"
            @click.stop="emit('remove', group.items[0])"
          >
            Delete all
          </button>
        </div>

        <h3 class="title t-body">{{ group.title }}</h3>
        <p class="artist t-meta truncate">
          <template v-if="group.kind === 'playlist'">{{ group.items.length }} tracks</template>
          <template v-else>
            <template v-if="group.items[0].artist">{{ group.items[0].artist }}</template>
            <template v-else>Unknown artist</template>
          </template>
        </p>
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
  cursor: pointer;
}

.art:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
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
  z-index: 1;
}

.art-fallback {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: var(--ink-faint);
  letter-spacing: 0.32em;
}

/* Compound art for playlists: 2x2 grid */
.compound {
  position: absolute;
  inset: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 2px;
  background: var(--bg);
}

.compound-cell {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 1.2s var(--ease-out);
}

.plate:hover .compound-cell {
  transform: scale(1.04);
}

.compound-fill {
  background: var(--plate);
}

/* The roman numeral sits inside the plate. */
.art-numeral {
  position: absolute;
  left: var(--lg);
  bottom: var(--lg);
  z-index: 3;
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
  z-index: 3;
  color: var(--roman-overlay);
}

.art-count {
  position: absolute;
  top: var(--lg);
  right: var(--lg);
  z-index: 3;
  color: var(--roman-overlay);
  background: rgba(0, 0, 0, 0.5);
  padding: 2px 6px;
  border: 1px solid var(--hairline);
}

.art-play {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: grid;
  place-items: center;
  font-size: clamp(24px, 4vw, 40px);
  color: var(--roman-overlay);
  opacity: 0;
  transition: opacity var(--dur-fast) var(--ease-out);
  pointer-events: none;
}

.art:hover .art-play,
.art:focus-visible .art-play {
  opacity: 1;
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
  color: var(--accent);
  transition: color var(--dur-fast) var(--ease-out);
}

.remove:hover {
  color: var(--accent);
}

.remove--confirm {
  color: var(--error);
  font-weight: 600;
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
</style>