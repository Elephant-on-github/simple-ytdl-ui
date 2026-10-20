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
const coverSrc = computed(() =>
  player.current ? api.coverUrl(player.current.id) : '',
)
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
        <div class="pb-rail" aria-hidden="true" />

        <div class="pb-content">
          <div class="pb-now">
            <span class="pb-cover">
              <img :src="coverSrc" :alt="player.current?.title ?? ''" />
            </span>
            <span class="pb-text">
              <span class="pb-title truncate">{{ player.current?.title }}</span>
              <span class="pb-artist truncate">{{ player.current?.artist || 'Unknown artist' }}</span>
            </span>
          </div>

          <div class="pb-center">
            <button
              type="button"
              class="pb-play"
              :aria-label="player.playing ? 'Pause' : 'Play'"
              @click="player.toggle()"
            >
              <svg v-if="player.playing" width="15" height="15" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                <path d="M4.5 3.2h2.6v9.6H4.5zM8.9 3.2h2.6v9.6H8.9z" fill="currentColor" />
              </svg>
              <svg v-else width="15" height="15" viewBox="0 0 16 16" fill="none" aria-hidden="true">
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

            <span class="pb-time tnum">
              {{ formatDuration(player.currentTime) }}
              <i>/</i>
              {{ formatDuration(player.duration) }}
            </span>
          </div>

          <div class="pb-right">
            <button
              type="button"
              class="pb-mute"
              :aria-label="player.muted ? 'Unmute' : 'Mute'"
              @click="player.toggleMute()"
            >
              <svg v-if="player.muted || player.volume === 0" width="16" height="16" viewBox="0 0 18 18" fill="none" aria-hidden="true">
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
              <svg v-else width="16" height="16" viewBox="0 0 18 18" fill="none" aria-hidden="true">
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
              <svg width="13" height="13" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                <path d="M3.5 3.5l7 7m0-7-7 7" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
              </svg>
            </button>
          </div>
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
  height: var(--player-h);
  background: var(--panel);
  border-top: 1px solid var(--hairline);
}

/* The media element is driven by the bar; it is never shown itself. */
.pb-media {
  display: none;
}

/* The bar spans the window; its content lines up with the main column. */
.pb-inner {
  display: grid;
  grid-template-columns: var(--rail-w) minmax(0, 1fr);
  height: 100%;
}

.pb-content {
  display: grid;
  grid-template-columns: minmax(180px, 260px) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--sp-5);
  height: 100%;
  max-width: calc(var(--content-max) + var(--sp-9) * 2);
  padding: 0 var(--sp-9);
}

.pb-now {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  min-width: 0;
}

.pb-cover {
  width: 44px;
  height: 44px;
  flex: none;
  border-radius: var(--r-sm);
  overflow: hidden;
  background: var(--panel-raised);
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
  font-size: var(--fs-13);
  font-weight: 500;
  color: var(--text);
}

.pb-artist {
  font-size: var(--fs-12);
  color: var(--muted);
}

.pb-center {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
  min-width: 0;
}

.pb-play {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  flex: none;
  border-radius: 50%;
  background: var(--accent);
  color: var(--accent-ink);
  transition:
    filter var(--dur-fast) var(--ease),
    transform var(--dur-fast) var(--ease);
}

.pb-play:hover {
  filter: brightness(1.08);
}

.pb-play:active {
  transform: scale(0.96);
}

.pb-seek {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
}

.pb-time {
  flex: none;
  font-size: var(--fs-12);
  color: var(--muted);
  letter-spacing: 0;
}

.pb-time i {
  font-style: normal;
  color: var(--faint);
  padding: 0 1px;
}

.pb-right {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
}

.pb-mute {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: var(--r-sm);
  color: var(--muted);
  transition:
    color var(--dur-fast) var(--ease),
    background var(--dur-fast) var(--ease);
}

.pb-mute:hover {
  color: var(--text);
  background: rgba(255, 255, 255, 0.05);
}

.pb-close {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: var(--r-sm);
  border: 1px solid transparent;
  color: var(--faint);
  transition:
    color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease);
}

.pb-close:hover {
  color: var(--text);
  border-color: var(--hairline);
}

/* --- range inputs -------------------------------------------------------- */

.range {
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  height: 16px;
  background: transparent;
  cursor: pointer;
}

.range::-webkit-slider-runnable-track {
  height: 3px;
  border-radius: 2px;
  background: linear-gradient(
    to right,
    var(--accent) 0%,
    var(--accent) var(--fill, 0%),
    rgba(255, 255, 255, 0.12) var(--fill, 0%),
    rgba(255, 255, 255, 0.12) 100%
  );
}

.range::-moz-range-track {
  height: 3px;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.12);
}

.range::-moz-range-progress {
  height: 3px;
  border-radius: 2px;
  background: var(--accent);
}

.range::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 11px;
  height: 11px;
  margin-top: -4px;
  border-radius: 50%;
  background: var(--text);
  border: none;
  transition: scale var(--dur-fast) var(--ease);
}

.range::-moz-range-thumb {
  width: 11px;
  height: 11px;
  border: none;
  border-radius: 50%;
  background: var(--text);
}

.range:hover::-webkit-slider-thumb {
  scale: 1.15;
}

.range--vol {
  width: 88px;
}

/* --- transitions --------------------------------------------------------- */

.player-enter-active,
.player-leave-active {
  transition:
    transform var(--dur-slow) var(--ease),
    opacity var(--dur-slow) var(--ease);
}

.player-enter-from,
.player-leave-to {
  transform: translateY(100%);
  opacity: 0;
}

/* --- narrow screens ------------------------------------------------------ */

@media (max-width: 900px) {
  .pb-inner {
    grid-template-columns: minmax(0, 1fr);
  }

  .pb-content {
    max-width: none;
    padding: 0 var(--sp-5);
  }
}

@media (max-width: 860px) {
  .pb-content {
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-areas:
      'now right'
      'center center';
    gap: var(--sp-2) var(--sp-4);
  }

  .pb-now {
    grid-area: now;
  }

  .pb-center {
    grid-area: center;
  }

  .pb-right {
    grid-area: right;
  }

  .range--vol {
    display: none;
  }
}

@media (max-width: 560px) {
  .player {
    height: auto;
  }

  .pb-content {
    padding: var(--sp-2) var(--sp-4) var(--sp-3);
    gap: var(--sp-2);
  }

  .pb-cover {
    width: 38px;
    height: 38px;
  }

  .pb-center {
    gap: var(--sp-3);
  }

  .pb-time {
    font-size: var(--fs-11);
  }
}
</style>
