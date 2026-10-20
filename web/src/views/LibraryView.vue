<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import LibraryGrid from '../components/LibraryGrid.vue'
import { useLibraryStore } from '../stores/library'
import { usePlayerStore } from '../stores/player'
import { formatBytes, formatCount } from '../lib/format'
import type { Item } from '../lib/api'

const library = useLibraryStore()
const player = usePlayerStore()

onMounted(() => {
  void library.refresh()
})

async function remove(item: Item): Promise<void> {
  try {
    await library.remove(item.id)
  } catch (err) {
    library.error = err instanceof Error ? err.message : 'Could not delete the file.'
  }
}
</script>

<template>
  <div class="view">
    <header class="head">
      <div>
        <p class="t-label">Local</p>
        <h1 class="t-display head-title">Library</h1>
      </div>
      <p v-if="library.items.length" class="t-fine muted tnum">
        {{ formatCount(library.items.length, 'file') }} · {{ formatBytes(library.totalSize) }}
      </p>
    </header>

    <p v-if="library.error" class="notice notice--error t-fine">{{ library.error }}</p>

    <div v-if="!library.loading && !library.items.length" class="empty panel">
      <span class="empty-icon" aria-hidden="true">
        <svg width="20" height="20" viewBox="0 0 18 18" fill="none">
          <rect x="3.2" y="3.2" width="4.6" height="4.6" rx="1.2" stroke="currentColor" stroke-width="1.4" />
          <rect x="10.2" y="3.2" width="4.6" height="4.6" rx="1.2" stroke="currentColor" stroke-width="1.4" />
          <rect x="3.2" y="10.2" width="4.6" height="4.6" rx="1.2" stroke="currentColor" stroke-width="1.4" />
          <rect x="10.2" y="10.2" width="4.6" height="4.6" rx="1.2" stroke="currentColor" stroke-width="1.4" />
        </svg>
      </span>
      <div>
        <p class="t-meta">Nothing downloaded yet</p>
        <p class="t-fine muted">Files you download show up here, ready to play.</p>
      </div>
      <RouterLink to="/" class="btn empty-cta">Download something</RouterLink>
    </div>

    <LibraryGrid
      v-else
      :items="library.items"
      @play="player.load($event)"
      @remove="remove"
    />
  </div>
</template>

<style scoped>
.view {
  display: flex;
  flex-direction: column;
  gap: var(--sp-7);
  max-width: var(--content-max);
}

.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--sp-4);
  padding-bottom: var(--sp-5);
  border-bottom: 1px solid var(--hairline);
}

.head-title {
  margin-top: 6px;
}

.empty {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
  padding: var(--sp-6);
}

.empty-icon {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex: none;
  border-radius: var(--r-md);
  color: var(--accent);
  background: var(--accent-soft);
  border: 1px solid var(--accent-line);
}

.empty-cta {
  margin-left: auto;
  flex: none;
}

.notice {
  padding: 10px var(--sp-3);
  border-radius: var(--r-sm);
}

.notice--error {
  color: var(--danger);
  background: var(--danger-soft);
  border: 1px solid var(--danger-line);
}

@media (max-width: 620px) {
  .empty {
    flex-wrap: wrap;
  }

  .empty-cta {
    margin-left: 0;
    width: 100%;
  }
}
</style>
