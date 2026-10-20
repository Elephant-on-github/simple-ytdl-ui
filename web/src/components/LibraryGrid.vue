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
</script>

<template>
  <ul class="grid">
    <li v-for="item in items" :key="item.id" class="card">
      <div class="cover" :class="{ 'cover--broken': broken.has(item.id) }">
        <img
          v-if="!broken.has(item.id)"
          class="cover-img"
          :src="api.coverUrl(item.id)"
          :alt="item.title"
          loading="lazy"
          @error="markBroken(item.id)"
        />
        <span v-else class="cover-fallback" aria-hidden="true">
          <svg v-if="item.mode === 'music'" width="22" height="22" viewBox="0 0 18 18" fill="none">
            <path
              d="M6.4 12.6V4.2l7-1.2v8.4"
              stroke="currentColor"
              stroke-width="1.4"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
            <circle cx="4.6" cy="12.8" r="1.9" stroke="currentColor" stroke-width="1.4" />
            <circle cx="11.6" cy="11.6" r="1.9" stroke="currentColor" stroke-width="1.4" />
          </svg>
          <svg v-else width="22" height="22" viewBox="0 0 18 18" fill="none">
            <rect
              x="2.4"
              y="4.2"
              width="13.2"
              height="9.6"
              rx="2"
              stroke="currentColor"
              stroke-width="1.4"
            />
            <path d="M8 7.2 11.6 9 8 10.8V7.2Z" fill="currentColor" />
          </svg>
        </span>

        <span class="cover-time tnum">{{ formatDuration(item.duration) }}</span>

        <button
          type="button"
          class="cover-play"
          :aria-label="`Play ${item.title}`"
          @click="emit('play', item)"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <path d="M5 3.4 12.4 8 5 12.6V3.4Z" fill="currentColor" />
          </svg>
        </button>

        <button
          type="button"
          class="cover-remove"
          :class="{ 'cover-remove--confirm': confirming === item.id }"
          :aria-label="`Delete ${item.title}`"
          @click.stop="askRemove(item)"
        >
          <template v-if="confirming === item.id">
            <span class="cover-remove-label">Sure?</span>
          </template>
          <svg v-else width="14" height="14" viewBox="0 0 18 18" fill="none" aria-hidden="true">
            <path
              d="M4.6 5.6h8.8M7.4 5.6V4.4h3.2v1.2M6 5.6l.6 8.2h4.8l.6-8.2M7.6 8v3.6M10.4 8v3.6"
              stroke="currentColor"
              stroke-width="1.4"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>

      <div class="meta">
        <p class="title clamp-2">{{ item.title }}</p>
        <p class="artist truncate">
          <span v-if="item.artist">{{ item.artist }}</span>
          <span v-else class="faint">Unknown artist</span>
        </p>
      </div>
    </li>
  </ul>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(184px, 1fr));
  gap: var(--sp-6) var(--sp-5);
}

.card {
  min-width: 0;
}

.cover {
  position: relative;
  aspect-ratio: 1;
  border-radius: var(--r-lg);
  overflow: hidden;
  background: var(--panel);
  border: 1px solid var(--hairline);
  transition:
    border-color var(--dur) var(--ease),
    transform var(--dur) var(--ease);
}

.card:hover .cover {
  border-color: var(--hairline-strong);
}

.cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cover--broken {
  display: grid;
  place-items: center;
  background:
    radial-gradient(120% 90% at 50% 0%, var(--accent-wash), transparent 70%),
    var(--panel-raised);
  color: var(--accent);
}

.cover-fallback {
  opacity: 0.8;
}

.cover-time {
  position: absolute;
  right: 8px;
  bottom: 8px;
  padding: 2px 6px;
  border-radius: var(--r-xs);
  background: rgba(0, 0, 0, 0.66);
  color: rgba(255, 255, 255, 0.88);
  font-size: 11px;
  line-height: 1.5;
  letter-spacing: 0.01em;
}

.cover-play {
  position: absolute;
  left: 50%;
  top: 50%;
  translate: -50% -50%;
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--accent);
  color: var(--accent-ink);
  opacity: 0;
  scale: 0.9;
  transition:
    opacity var(--dur) var(--ease),
    scale var(--dur) var(--ease),
    filter var(--dur) var(--ease);
}

.cover-play svg {
  margin-left: 2px;
}

.card:hover .cover-play,
.cover-play:focus-visible {
  opacity: 1;
  scale: 1;
}

.cover-play:hover {
  filter: brightness(1.08);
}

.cover-remove {
  position: absolute;
  top: 8px;
  right: 8px;
  display: grid;
  place-items: center;
  min-width: 26px;
  height: 26px;
  padding: 0 6px;
  border-radius: var(--r-sm);
  background: rgba(0, 0, 0, 0.62);
  color: rgba(255, 255, 255, 0.72);
  opacity: 0;
  transition:
    opacity var(--dur) var(--ease),
    color var(--dur-fast) var(--ease),
    background var(--dur-fast) var(--ease);
}

.card:hover .cover-remove,
.cover-remove:focus-visible,
.cover-remove--confirm {
  opacity: 1;
}

.cover-remove:hover {
  color: var(--danger);
  background: rgba(0, 0, 0, 0.78);
}

.cover-remove--confirm {
  background: var(--danger);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.01em;
}

.cover-remove-label {
  white-space: nowrap;
}

.meta {
  padding: 10px 2px 0;
}

.title {
  font-size: var(--fs-13);
  font-weight: 500;
  line-height: 1.4;
  letter-spacing: -0.004em;
  color: var(--text);
}

.artist {
  margin-top: 3px;
  font-size: var(--fs-12);
  color: var(--muted);
}

@media (hover: none) {
  .cover-play,
  .cover-remove {
    opacity: 1;
    scale: 1;
  }
}

@media (max-width: 560px) {
  .grid {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: var(--sp-5) var(--sp-4);
  }
}
</style>
