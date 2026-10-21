<script setup lang="ts">
import { ref } from 'vue'
import { api, type Mode, type Playlist, type PlaylistItem } from '../lib/api'
import { formatDuration } from '../lib/format'

defineProps<{ playlists: Playlist[]; mode: Mode; busy?: string | null }>()

const emit = defineEmits<{ (e: 'sync', playlist: Playlist): void }>()

const openId = ref<string | null>(null)
const previews = ref<Record<string, PlaylistItem[]>>({})
const nextTokens = ref<Record<string, string>>({})
const loadingId = ref<string | null>(null)
const brokenThumbs = ref<Set<string>>(new Set())

async function toggle(playlist: Playlist): Promise<void> {
  if (openId.value === playlist.id) {
    openId.value = null
    return
  }
  openId.value = playlist.id
  if (previews.value[playlist.id]) return
  await loadPage(playlist.id, '')
}

// loadPage fetches one page of playlist items. The YouTube API returns 50
// items per page, so a long playlist needs more than one request.
async function loadPage(playlistId: string, pageToken: string): Promise<void> {
  loadingId.value = playlistId
  try {
    const payload = await api.playlistItems(playlistId, pageToken || undefined)
    const known = previews.value[playlistId] ?? []
    const seen = new Set(known.map((entry) => entry.id))
    const fresh = (payload.items ?? []).filter((entry) => !seen.has(entry.id))
    previews.value = { ...previews.value, [playlistId]: [...known, ...fresh] }
    nextTokens.value = { ...nextTokens.value, [playlistId]: payload.nextPageToken ?? '' }
  } catch {
    previews.value = { ...previews.value, [playlistId]: previews.value[playlistId] ?? [] }
  } finally {
    loadingId.value = null
  }
}

function markBroken(id: string): void {
  const next = new Set(brokenThumbs.value)
  next.add(id)
  brokenThumbs.value = next
}
</script>

<template>
  <ul class="playlists">
    <li v-for="playlist in playlists" :key="playlist.id" class="pl">
      <div class="pl-row">
        <button
          type="button"
          class="pl-toggle"
          :aria-expanded="openId === playlist.id"
          @click="toggle(playlist)"
        >
          <span class="pl-thumb" :class="{ 'pl-thumb--broken': brokenThumbs.has(playlist.id) }">
            <img
              v-if="!brokenThumbs.has(playlist.id)"
              :src="playlist.thumb"
              :alt="playlist.title"
              loading="lazy"
              @error="markBroken(playlist.id)"
            />
            <span v-else class="t-label">No Cover</span>
          </span>

          <span class="pl-text">
            <span class="pl-title t-italic">{{ playlist.title }}</span>
            <span v-if="playlist.itemCount" class="pl-count t-meta tnum">
              {{ playlist.itemCount }} {{ playlist.itemCount === 1 ? 'item' : 'items' }}
            </span>
          </span>

          <svg
            class="pl-caret"
            :class="{ 'pl-caret--open': openId === playlist.id }"
            width="14"
            height="14"
            viewBox="0 0 14 14"
            fill="none"
            aria-hidden="true"
          >
            <path d="M4 5.5 7 8.5l3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
          </svg>
        </button>

        <div class="pl-actions">
          <span class="t-label">{{ mode === 'music' ? 'MP3' : 'MP4' }}</span>
          <button
            type="button"
            class="step"
            :disabled="busy === playlist.id"
            @click="emit('sync', playlist)"
          >
            {{ busy === playlist.id ? 'Syncing' : 'Sync' }}
            <span aria-hidden="true">→</span>
          </button>
        </div>
      </div>

      <Transition name="expand">
        <div v-if="openId === playlist.id" class="pl-preview">
          <p v-if="loadingId === playlist.id" class="t-meta faint">Loading items…</p>
          <p v-else-if="!previews[playlist.id]?.length" class="t-meta faint">
            No items to preview.
          </p>
          <ul v-else class="pl-items">
            <li v-for="entry in previews[playlist.id]" :key="entry.id" class="pl-item">
              <span class="pl-item-title t-serif-body truncate">{{ entry.title }}</span>
              <span class="pl-item-channel t-meta truncate">{{ entry.channel }}</span>
              <span class="pl-item-time t-meta tnum">{{ formatDuration(entry.duration) }}</span>
            </li>
          </ul>
          <button
            v-if="nextTokens[playlist.id]"
            type="button"
            class="step pl-more"
            :disabled="loadingId === playlist.id"
            @click="loadPage(playlist.id, nextTokens[playlist.id])"
          >
            Show more
            <span aria-hidden="true">→</span>
          </button>
        </div>
      </Transition>
    </li>
  </ul>
</template>

<style scoped>
.playlists {
  display: flex;
  flex-direction: column;
}

.pl {
  border-bottom: 1px solid var(--hairline);
}

.pl-row {
  display: flex;
  align-items: center;
  gap: var(--xxl);
  padding: var(--xl) 0;
}

.pl-toggle {
  display: flex;
  align-items: center;
  gap: var(--xl);
  flex: 1;
  min-width: 0;
  text-align: left;
}

.pl-thumb {
  display: grid;
  place-items: center;
  width: 56px;
  height: 56px;
  flex: none;
  overflow: hidden;
  background: var(--plate);
  border: 1px solid var(--hairline);
  color: var(--ink-faint);
}

.pl-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.pl-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.pl-title {
  font-size: clamp(18px, 1.6vw, 24px);
  color: var(--ink);
  transition: color var(--dur-fast) var(--ease-out);
}

.pl-toggle:hover .pl-title {
  color: var(--accent);
}

.pl-count {
  color: var(--ink-faint);
}

.pl-caret {
  flex: none;
  color: var(--ink-faint);
  transition: transform var(--dur) var(--ease-out);
}

.pl-caret--open {
  transform: rotate(180deg);
}

.pl-actions {
  display: flex;
  align-items: center;
  gap: var(--xxl);
  flex: none;
}

.pl-actions .t-label {
  color: var(--accent);
}

/* --- preview ------------------------------------------------------------- */

.pl-preview {
  padding: 0 0 var(--xl);
}

.pl-items {
  display: flex;
  flex-direction: column;
}

.pl-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 150px 46px;
  gap: var(--lg);
  align-items: center;
  padding: var(--sm) 0;
  border-bottom: 1px solid rgba(74, 72, 67, 0.35);
}

.pl-item-title {
  font-size: 15px;
  color: var(--ink-dim);
}

.pl-item-channel {
  color: var(--ink-faint);
}

.pl-item-time {
  text-align: right;
  color: var(--ink-faint);
}

.pl-more {
  margin-top: var(--lg);
  color: var(--ink-dim);
}

@media (max-width: 620px) {
  .pl-row {
    flex-wrap: wrap;
    gap: var(--lg);
  }

  .pl-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .pl-item {
    grid-template-columns: minmax(0, 1fr) 44px;
  }

  .pl-item-channel {
    display: none;
  }
}
</style>
