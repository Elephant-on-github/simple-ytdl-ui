<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterView } from 'vue-router'
import TopBar from './components/TopBar.vue'
import PlayerBar from './components/PlayerBar.vue'
import { usePlayerStore } from './stores/player'
import { useSessionStore } from './stores/session'

const player = usePlayerStore()
const session = useSessionStore()
const hasPlayer = computed(() => player.isOpen)

onMounted(() => {
  // The rail shows the server version, so the session is needed everywhere.
  void session.refresh()
})
</script>

<template>
  <div class="shell" :class="{ 'shell--player': hasPlayer }">
    <TopBar class="shell-rail" />
    <main class="shell-main">
      <RouterView v-slot="{ Component }">
        <Transition name="view" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
    <PlayerBar />
  </div>
</template>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: var(--rail-w) minmax(0, 1fr);
  min-height: 100vh;
  background: var(--canvas);
}

.shell-main {
  min-width: 0;
  padding: var(--sp-9) var(--sp-9) var(--sp-9);
}

/* Reserve room for the docked player only while it is on screen. */
.shell--player .shell-main {
  padding-bottom: calc(var(--player-h) + var(--sp-7));
}

@media (max-width: 900px) {
  .shell {
    grid-template-columns: minmax(0, 1fr);
  }

  .shell-main {
    padding: var(--sp-7) var(--sp-5) var(--sp-7);
  }

  .shell--player .shell-main {
    padding-bottom: calc(var(--player-h) + var(--sp-6));
  }
}

@media (max-width: 560px) {
  .shell-main {
    padding: var(--sp-6) var(--sp-4) var(--sp-6);
  }

  .shell--player .shell-main {
    padding-bottom: calc(var(--player-h) + var(--sp-5));
  }
}

.view-enter-active,
.view-leave-active {
  transition:
    opacity var(--dur) var(--ease),
    transform var(--dur) var(--ease);
}

.view-enter-from {
  opacity: 0;
  transform: translateY(6px);
}

.view-leave-to {
  opacity: 0;
}
</style>
