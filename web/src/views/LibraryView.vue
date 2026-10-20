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
        <p class="eyebrow">Local</p>
        <h1 class="t-display head-title">Library</h1>
      </div>
      <p v-if="library.items.length" class="t-meta tnum">
        {{ formatCount(library.items.length, 'file') }} · {{ formatBytes(library.totalSize) }}
      </p>
    </header>

    <p v-if="library.error" class="notice t-meta">{{ library.error }}</p>

    <div v-if="!library.loading && !library.items.length" class="empty">
      <p class="t-meta">Nothing downloaded yet</p>
      <p class="t-serif-body empty-copy">
        The files you download show up here, ready to play.
      </p>
      <RouterLink to="/" class="step empty-cta">
        Download something
        <span aria-hidden="true">→</span>
      </RouterLink>
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
  gap: var(--gap);
}

.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--xxl);
  padding-bottom: var(--xl);
  border-bottom: 1px solid var(--hairline);
}

.head-title {
  margin-top: var(--md);
}

.head .t-meta {
  color: var(--ink-faint);
}

/* --- empty --------------------------------------------------------------- */

.empty {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--lg);
  padding: var(--gap) 0;
  border-bottom: 1px solid var(--hairline);
}

.empty .t-meta {
  color: var(--ink-faint);
}

.empty-copy {
  color: var(--ink-dim);
  max-width: 44ch;
}

.empty-cta {
  margin-top: var(--md);
}

.notice {
  color: var(--accent);
  text-transform: none;
  letter-spacing: 0.04em;
}
</style>
