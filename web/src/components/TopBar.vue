<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

const route = useRoute()

const links = [
  { name: 'download', label: 'Download', to: '/' },
  { name: 'library', label: 'Library', to: '/library' },
  { name: 'google', label: 'Google', to: '/google' },
  { name: 'schedule', label: 'Schedule', to: '/schedule' },
]

const activeName = computed(() => String(route.name ?? 'download'))
</script>

<template>
  <header class="topbar">
    <RouterLink to="/" class="wordmark" aria-label="simple-ytdl-ui">
      <span class="wordmark-text">Simple-Ytdl</span>
      <span class="wordmark-coord t-nav">51°30′N 0°07′W</span>
    </RouterLink>

    <span class="edition t-nav">Edition 01 — Self-Hosted</span>

    <nav class="nav" aria-label="Views">
      <RouterLink
        v-for="link in links"
        :key="link.name"
        :to="link.to"
        class="nav-link t-nav"
        :class="{ 'nav-link--active': activeName === link.name }"
      >
        {{ link.label }}
      </RouterLink>
    </nav>
  </header>
</template>

<style scoped>
.topbar {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: var(--xxl);
  padding: var(--pad-y) var(--pad-x);
  border-bottom: 1px solid var(--hairline);
}

/* --- wordmark ------------------------------------------------------------ */

.wordmark {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  justify-self: start;
}

.wordmark-text {
  font-family: var(--serif);
  font-size: var(--fs-wordmark);
  font-weight: 400;
  line-height: 1.5;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: var(--ink);
}

/* A hidden coordinate line that shows up when the wordmark is held. */
.wordmark-coord {
  color: var(--accent);
  opacity: 0;
  transform: translateY(-4px);
  transition:
    opacity var(--dur) var(--ease-out),
    transform var(--dur) var(--ease-out);
}

.wordmark:hover .wordmark-coord {
  opacity: 1;
  transform: translateY(0);
}

/* --- edition ------------------------------------------------------------- */

.edition {
  justify-self: center;
  color: var(--ink-faint);
}

/* --- nav ----------------------------------------------------------------- */

.nav {
  display: flex;
  align-items: center;
  gap: var(--xxl);
  justify-self: end;
}

.nav-link {
  position: relative;
  color: var(--ink-dim);
  padding-bottom: 2px;
  transition: color var(--dur-fast) var(--ease-out);
}

/* The underline draws from the right on hover. */
.nav-link::after {
  content: '';
  position: absolute;
  left: 0;
  bottom: 0;
  width: 100%;
  height: 1px;
  background: currentColor;
  transform: scaleX(0);
  transform-origin: right;
  transition: transform var(--dur) var(--ease-out);
}

.nav-link:hover {
  color: var(--ink);
}

.nav-link:hover::after {
  transform: scaleX(1);
  transform-origin: left;
}

.nav-link--active {
  color: var(--accent);
}

.nav-link--active::after {
  background: var(--accent);
  transform: scaleX(1);
}

@media (max-width: 720px) {
  .topbar {
    grid-template-columns: 1fr auto;
    row-gap: var(--md);
  }

  .edition {
    display: none;
  }

  .nav {
    gap: var(--lg);
  }
}
</style>
