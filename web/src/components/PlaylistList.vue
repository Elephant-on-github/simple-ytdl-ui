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
        <button type="button" class="pl-toggle" :aria-expanded="openId === playlist.id" @click="toggle(playlist)">
          <span class="pl-thumb" :class="{ 'pl-thumb--broken': brokenThumbs.has(playlist.id) }">
            <img
              v-if="!brokenThumbs.has(playlist.id)"
              :src="playlist.thumb"
              :alt="playlist.title"
              loading="lazy"
              @error="markBroken(playlist.id)"
            />
            <svg v-else width="16" height="16" viewBox="0 0 18 18" fill="none" aria-hidden="true">
              <path
                d="M4.4 5h10M4.4 9h10M4.4 13h6.4"
                stroke="currentColor"
                stroke-width="1.4"
                stroke-linecap="round"
              />
            </svg>
          </span>

          <span class="pl-text">
            <span class="pl-title truncate">{{ playlist.title }}</span>
            <span class="pl-count tnum">
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
          <span class="chip">{{ mode === 'music' ? 'MP3' : 'Video' }}</span>
          <button
            type="button"
            class="btn btn--sync"
            :disabled="busy === playlist.id"
            @click="emit('sync', playlist)"
          >
            <svg width="14" height="14" viewBox="0 0 18 18" fill="none" aria-hidden="true">
              <path
                d="M9 2.8v8.4m0 0 3.2-3.2M9 11.2l-3.2-3.2M3.8 14.4h10.4"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            {{ busy === playlist.id ? 'Syncing…' : 'Sync' }}
          </button>
        </div>
      </div>

      <Transition name="expand">
        <div v-if="openId === playlist.id" class="pl-preview">
          <p v-if="loadingId === playlist.id" class="t-fine faint">Loading items…</p>
          <p v-else-if="!previews[playlist.id]?.length" class="t-fine faint">No items to preview.</p>
          <ul v-else class="pl-items">
            <li v-for="entry in previews[playlist.id]" :key="entry.id" class="pl-item">
              <span class="pl-item-title truncate">{{ entry.title }}</span>
              <span class="pl-item-channel truncate">{{ entry.channel }}</span>
              <span class="pl-item-time tnum">{{ formatDuration(entry.duration) }}</span>
            </li>
          </ul>
          <button
            v-if="nextTokens[playlist.id]"
            type="button"
            class="btn btn--ghost pl-more"
            :disabled="loadingId === playlist.id"
            @click="loadPage(playlist.id, nextTokens[playlist.id])"
          >
            {{ loadingId === playlist.id ? 'Loading…' : 'Show more' }}
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
  gap: var(--sp-2);
}

.pl {
  border: 1px solid var(--hairline);
  border-radius: var(--r-lg);
  background: var(--panel);
  transition: border-color var(--dur) var(--ease);
}

.pl:hover {
  border-color: var(--hairline-strong);
}

.pl-row {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
  padding: var(--sp-3) var(--sp-4);
}

.pl-toggle {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  flex: 1;
  min-width: 0;
  text-align: left;
}

.pl-thumb {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  flex: none;
  border-radius: var(--r-sm);
  overflow: hidden;
  background: var(--panel-raised);
  border: 1px solid var(--hairline);
  color: var(--faint);
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
  font-size: var(--fs-13);
  font-weight: 500;
  color: var(--text);
}

.pl-count {
  font-size: var(--fs-12);
  color: var(--muted);
}

.pl-caret {
  flex: none;
  color: var(--faint);
  transition: transform var(--dur) var(--ease);
}

.pl-caret--open {
  transform: rotate(180deg);
}

.pl-actions {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  flex: none;
}

.btn--sync {
  height: 32px;
  padding: 0 12px;
  color: var(--text-dim);
}

.btn--sync:hover {
  color: var(--accent);
  border-color: var(--accent-line);
  background: var(--accent-soft);
}

.pl-preview {
  border-top: 1px solid var(--hairline);
  padding: var(--sp-3) var(--sp-4) var(--sp-4);
}

.pl-items {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.pl-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 150px 46px;
  gap: var(--sp-3);
  align-items: center;
  padding: 6px 0;
  font-size: var(--fs-12);
  color: var(--text-dim);
}

.pl-item + .pl-item {
  border-top: 1px solid rgba(255, 255, 255, 0.045);
}

.pl-item-channel {
  color: var(--faint);
}

.pl-item-time {
  text-align: right;
  color: var(--muted);
}

.pl-more {
  height: 30px;
  margin-top: var(--sp-3);
  font-size: var(--fs-12);
}

.expand-enter-active,
.expand-leave-active {
  transition:
    opacity var(--dur) var(--ease),
    transform var(--dur) var(--ease);
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

@media (max-width: 620px) {
  .pl-row {
    flex-wrap: wrap;
    gap: var(--sp-3);
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
