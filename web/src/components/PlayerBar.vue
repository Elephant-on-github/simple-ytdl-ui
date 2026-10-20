<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '../lib/api'
import { usePlayerStore } from '../stores/player'
import { formatDuration } from '../lib/format'

const player = usePlayerStore()
const mediaEl = ref<HTMLMediaElement | null>(null)

const src = computed(() => (player.current ? api.streamUrl(player.current.id) : ''))
/** A video download is not audio: it needs a <video> element to play. */
const isVideo = computed(() => (player.current?.mime ?? '').startsWith('video/'))
const coverSrc = computed(() => (player.current ? api.coverUrl(player.current.id) : ''))
const seekMax = computed(() => Math.max(1, player.duration || 0))
const seekValue = computed(() => Math.min(player.currentTime, seekMax.value))
const fill = computed(() => (player.duration ? (player.currentTime / player.duration) * 100 : 0))
const volumeFill = computed(() => (player.muted ? 0 : player.volume) * 100)

/** `play()` returns a promise in modern browsers — but not everywhere. */
function safePlay(el: HTMLMediaElement): void {
  try {
    const result = el.play() as Promise<void> | undefined
    if (result && typeof result.catch === 'function') result.catch(() => undefined)
  } catch {
    /* autoplay was blocked — the user can press play again */
  }
}

watch(
  src,
  (value) => {
    const el = mediaEl.value
    if (!el || !value) return
    el.load()
    if (player.playing) safePlay(el)
  },
  { flush: 'post' },
)

watch(
  () => player.playing,
  (value) => {
    const el = mediaEl.value
    if (!el) return
    if (value) safePlay(el)
    else el.pause()
  },
)

watch(
  [() => player.volume, () => player.muted],
  () => {
    const el = mediaEl.value
    if (!el) return
    el.volume = player.muted ? 0 : player.volume
  },
)

function onTimeUpdate(): void {
  const el = mediaEl.value
  if (el) player.currentTime = el.currentTime
}

function onDurationChange(): void {
  const el = mediaEl.value
  if (el && Number.isFinite(el.duration)) player.duration = el.duration
}

function onEnded(): void {
  player.pause()
  player.currentTime = player.duration
}

function onSeek(event: Event): void {
  const value = Number((event.target as HTMLInputElement).value)
  player.seek(value)
  if (mediaEl.value) mediaEl.value.currentTime = value
}

function onVolume(event: Event): void {
  player.setVolume(Number((event.target as HTMLInputElement).value))
}
</script>

<template>
  <Transition name="player">
    <div v-if="player.isOpen" class="player" role="region" aria-label="Player">
      <component
        :is="isVideo ? 'video' : 'audio'"
        ref="mediaEl"
        class="pb-media"
        :src="src"
        preload="metadata"
        @timeupdate="onTimeUpdate"
        @durationchange="onDurationChange"
        @ended="onEnded"
        @play="player.play()"
        @pause="player.pause()"
      />

      <div class="pb-inner">
        <div class="pb-now">
          <span class="pb-cover">
            <img :src="coverSrc" :alt="player.current?.title ?? ''" />
          </span>
          <span class="pb-text">
            <span class="pb-title t-italic truncate">{{ player.current?.title }}</span>
            <span class="pb-artist t-meta truncate">
              {{ player.current?.artist || 'Unknown artist' }}
            </span>
          </span>
        </div>

        <div class="pb-center">
          <button
            type="button"
            class="pb-play"
            :aria-label="player.playing ? 'Pause' : 'Play'"
            @click="player.toggle()"
          >
            <svg v-if="player.playing" width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <path d="M4.5 3.2h2.6v9.6H4.5zM8.9 3.2h2.6v9.6H8.9z" fill="currentColor" />
            </svg>
            <svg v-else width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <path d="M5 3.4 12.4 8 5 12.6V3.4Z" fill="currentColor" />
            </svg>
          </button>

          <div class="pb-seek">
            <input
              class="range"
              type="range"
              min="0"
              :max="seekMax"
              step="0.1"
              :value="seekValue"
              :style="{ '--fill': `${fill}%` }"
              aria-label="Seek"
              @input="onSeek"
            />
          </div>

          <span class="pb-time t-meta tnum">
            {{ formatDuration(player.currentTime) }} / {{ formatDuration(player.duration) }}
          </span>
        </div>

        <div class="pb-right">
          <button
            type="button"
            class="pb-mute"
            :aria-label="player.muted ? 'Unmute' : 'Mute'"
            @click="player.toggleMute()"
          >
            <svg v-if="player.muted || player.volume === 0" width="15" height="15" viewBox="0 0 18 18" fill="none" aria-hidden="true">
              <path
                d="M4 7.2h2.4L9.6 4.4v9.2L6.4 10.8H4V7.2Z"
                stroke="currentColor"
                stroke-width="1.4"
                stroke-linejoin="round"
              />
              <path
                d="M12.4 7.2l3.4 3.6m0-3.6-3.4 3.6"
                stroke="currentColor"
                stroke-width="1.4"
                stroke-linecap="round"
              />
            </svg>
            <svg v-else width="15" height="15" viewBox="0 0 18 18" fill="none" aria-hidden="true">
              <path
                d="M4 7.2h2.4L9.6 4.4v9.2L6.4 10.8H4V7.2Z"
                stroke="currentColor"
                stroke-width="1.4"
                stroke-linejoin="round"
              />
              <path
                d="M12 7.4a3 3 0 0 1 0 3.2M14 5.4a5.6 5.6 0 0 1 0 7.2"
                stroke="currentColor"
                stroke-width="1.4"
                stroke-linecap="round"
              />
            </svg>
          </button>

          <input
            class="range range--vol"
            type="range"
            min="0"
            max="1"
            step="0.01"
            :value="player.muted ? 0 : player.volume"
            :style="{ '--fill': `${volumeFill}%` }"
            aria-label="Volume"
            @input="onVolume"
          />

          <button type="button" class="pb-close" aria-label="Close player" @click="player.close()">
            <svg width="12" height="12" viewBox="0 0 14 14" fill="none" aria-hidden="true">
              <path d="M3.5 3.5l7 7m0-7-7 7" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.player {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 40;
  height: 76px;
  background: var(--bg);
  border-top: 1px solid var(--hairline);
}

/* The media element is driven by the bar; it is never shown itself. */
.pb-media {
  display: none;
}

.pb-inner {
  display: grid;
  grid-template-columns: minmax(220px, 300px) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--xxl);
  height: 100%;
  padding: 0 var(--pad-x);
}

/* --- now playing --------------------------------------------------------- */

.pb-now {
  display: flex;
  align-items: center;
  gap: var(--lg);
  min-width: 0;
}

.pb-cover {
  width: 44px;
  height: 44px;
  flex: none;
  overflow: hidden;
  background: var(--plate);
  border: 1px solid var(--hairline);
}

.pb-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.pb-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.pb-title {
  font-size: 17px;
  color: var(--ink);
}

.pb-artist {
  color: var(--ink-faint);
}

/* --- transport ----------------------------------------------------------- */

.pb-center {
  display: flex;
  align-items: center;
  gap: var(--xl);
  min-width: 0;
}

.pb-play {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  flex: none;
  border: 1px solid var(--hairline);
  color: var(--ink);
  transition:
    color var(--dur-fast) var(--ease-out),
    border-color var(--dur-fast) var(--ease-out);
}

.pb-play:hover {
  color: var(--accent);
  border-color: var(--accent-line);
}

.pb-seek {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
}

.pb-time {
  color: var(--ink-dim);
  flex: none;
}

/* The range is a hairline rule that fills. */
.range {
  --fill: 0%;
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  height: 14px;
  background: transparent;
  cursor: pointer;
}

.range::-webkit-slider-runnable-track {
  height: 1px;
  background: linear-gradient(
    to right,
    var(--accent) 0 var(--fill),
    var(--hairline) var(--fill) 100%
  );
}

.range::-moz-range-track {
  height: 1px;
  background: var(--hairline);
}

.range::-moz-range-progress {
  height: 1px;
  background: var(--accent);
}

.range::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 9px;
  height: 9px;
  margin-top: -4px;
  border-radius: 50%;
  background: var(--ink);
}

.range::-moz-range-thumb {
  width: 9px;
  height: 9px;
  border: 0;
  border-radius: 50%;
  background: var(--ink);
}

.range--vol {
  width: 76px;
  flex: none;
}

/* --- right --------------------------------------------------------------- */

.pb-right {
  display: flex;
  align-items: center;
  gap: var(--lg);
  flex: none;
}

.pb-mute,
.pb-close {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  color: var(--ink-faint);
  transition: color var(--dur-fast) var(--ease-out);
}

.pb-mute:hover,
.pb-close:hover {
  color: var(--accent);
}

.player-enter-active,
.player-leave-active {
  transition:
    opacity var(--dur-fast) var(--ease-out),
    transform var(--dur-fast) var(--ease-out);
}

.player-enter-from,
.player-leave-to {
  opacity: 0;
  transform: translateY(100%);
}

@media (max-width: 860px) {
  .pb-inner {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: var(--lg);
  }

  .pb-center {
    grid-column: 1 / -1;
    grid-row: 2;
  }

  .pb-right {
    grid-column: 2;
    grid-row: 1;
  }
}
</style>
