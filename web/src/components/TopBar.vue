<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useModeStore } from '../stores/mode'
import { useSessionStore } from '../stores/session'

const route = useRoute()
const mode = useModeStore()
const session = useSessionStore()

const links = [
  { name: 'download', label: 'Download', to: '/' },
  { name: 'library', label: 'Library', to: '/library' },
  { name: 'playlists', label: 'Playlists', to: '/playlists' },
]

const activeName = computed(() => String(route.name ?? 'download'))
const modeLabel = computed(() => (mode.mode === 'music' ? 'MP3' : 'Video'))
</script>

<template>
  <aside class="rail">
    <RouterLink to="/" class="brand" aria-label="simple-ytdl-ui home">
      <span class="brand-mark" aria-hidden="true">
        <svg width="14" height="14" viewBox="0 0 18 18" fill="none">
          <path
            d="M9 2.5v9m0 0 3.4-3.4M9 11.5 5.6 8.1M3.6 14.6h10.8"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </span>
      <span class="brand-text">
        <span class="brand-name">simple-ytdl</span>
        <span class="brand-suffix">-ui</span>
      </span>
    </RouterLink>

    <nav class="nav" aria-label="Views">
      <RouterLink
        v-for="link in links"
        :key="link.name"
        :to="link.to"
        class="nav-item"
        :class="{ 'nav-item--active': activeName === link.name }"
      >
        <span class="nav-rail" aria-hidden="true" />
        <svg width="16" height="16" viewBox="0 0 18 18" fill="none" aria-hidden="true">
          <template v-if="link.name === 'download'">
            <path
              d="M9 2.6v8.8m0 0 3.3-3.3M9 11.4l-3.3-3.3M3.7 14.7h10.6"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </template>
          <template v-else-if="link.name === 'library'">
            <rect
              x="3.2"
              y="3.2"
              width="4.6"
              height="4.6"
              rx="1.2"
              stroke="currentColor"
              stroke-width="1.5"
            />
            <rect
              x="10.2"
              y="3.2"
              width="4.6"
              height="4.6"
              rx="1.2"
              stroke="currentColor"
              stroke-width="1.5"
            />
            <rect
              x="3.2"
              y="10.2"
              width="4.6"
              height="4.6"
              rx="1.2"
              stroke="currentColor"
              stroke-width="1.5"
            />
            <rect
              x="10.2"
              y="10.2"
              width="4.6"
              height="4.6"
              rx="1.2"
              stroke="currentColor"
              stroke-width="1.5"
            />
          </template>
          <template v-else>
            <path
              d="M4.4 5h10M4.4 9h10M4.4 13h6.4"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
            />
          </template>
        </svg>
        <span class="nav-label">{{ link.label }}</span>
      </RouterLink>
    </nav>

    <div class="rail-foot">
      <span class="foot-label">Downloading as</span>
      <span class="chip chip--accent foot-chip">{{ modeLabel }}</span>
      <span v-if="session.version" class="foot-version tnum">v{{ session.version }}</span>
    </div>
  </aside>
</template>

<style scoped>
.rail {
  position: sticky;
  top: 0;
  align-self: start;
  height: 100vh;
  display: flex;
  flex-direction: column;
  gap: var(--sp-7);
  padding: var(--sp-6) var(--sp-4) var(--sp-5);
  background: var(--rail);
  border-right: 1px solid var(--hairline);
  z-index: 20;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 var(--sp-2);
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border-radius: 8px;
  color: var(--accent);
  background: var(--accent-soft);
  border: 1px solid var(--accent-line);
  transition:
    background var(--dur) var(--ease),
    border-color var(--dur) var(--ease);
}

.brand-text {
  font-size: var(--fs-15);
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text);
}

.brand-suffix {
  color: var(--faint);
  font-weight: 500;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 11px;
  height: 38px;
  padding: 0 var(--sp-3);
  border-radius: var(--r-sm);
  color: var(--muted);
  font-size: var(--fs-13);
  font-weight: 500;
  letter-spacing: -0.004em;
  transition:
    color var(--dur-fast) var(--ease),
    background var(--dur-fast) var(--ease);
}

.nav-item:hover {
  color: var(--text-dim);
  background: rgba(255, 255, 255, 0.035);
}

.nav-item--active {
  color: var(--text);
  background: rgba(255, 255, 255, 0.05);
}

.nav-rail {
  position: absolute;
  left: calc(var(--sp-4) * -1);
  top: 50%;
  width: 2px;
  height: 0;
  border-radius: 0 2px 2px 0;
  background: var(--accent);
  transform: translateY(-50%);
  transition: height var(--dur) var(--ease);
}

.nav-item--active .nav-rail {
  height: 18px;
}

.nav-item--active svg {
  color: var(--accent);
}

.nav-label {
  position: relative;
  top: 0.5px;
}

.rail-foot {
  margin-top: auto;
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: 0 var(--sp-2);
}

.foot-label {
  font-size: var(--fs-12);
  color: var(--faint);
  white-space: nowrap;
}

.foot-chip {
  height: 20px;
  padding: 0 8px;
  font-size: 10px;
}

.foot-version {
  margin-left: auto;
  font-size: var(--fs-11);
  color: var(--faint);
}

/* --- rail collapses into a top bar on narrow screens ---------------------- */

@media (max-width: 900px) {
  .rail {
    position: sticky;
    top: 0;
    width: 100%;
    height: auto;
    flex-direction: row;
    align-items: center;
    gap: var(--sp-4);
    padding: 10px var(--sp-5);
    border-right: none;
    border-bottom: 1px solid var(--hairline);
    background: var(--rail);
  }

  .nav {
    flex-direction: row;
    gap: 4px;
    margin-left: auto;
    overflow-x: auto;
    scrollbar-width: none;
  }

  .nav::-webkit-scrollbar {
    display: none;
  }

  .nav-item {
    height: 34px;
    padding: 0 10px;
  }

  .nav-rail {
    display: none;
  }

  .rail-foot {
    margin-top: 0;
  }

  .foot-label,
  .foot-version {
    display: none;
  }
}

@media (max-width: 560px) {
  .rail {
    padding: 9px var(--sp-4);
    gap: var(--sp-3);
  }

  .brand-text {
    display: none;
  }
}
</style>
