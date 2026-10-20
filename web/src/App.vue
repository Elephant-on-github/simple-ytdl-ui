<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterView } from 'vue-router'
import TopBar from './components/TopBar.vue'
import PlayerBar from './components/PlayerBar.vue'
import { usePlayerStore } from './stores/player'
import { useSessionStore } from './stores/session'
import { useModeStore } from './stores/mode'

const player = usePlayerStore()
const session = useSessionStore()
const mode = useModeStore()
const hasPlayer = computed(() => player.isOpen)

/* The footbar carries a live UTC clock, the way a studio page would. */
const clock = ref('')
let timer: ReturnType<typeof setInterval> | undefined

function tick(): void {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  clock.value = `${pad(now.getUTCHours())}:${pad(now.getUTCMinutes())}:${pad(
    now.getUTCSeconds(),
  )} UTC`
}

onMounted(() => {
  // The footer shows the server version, so the session is needed everywhere.
  void session.refresh()
  tick()
  timer = setInterval(tick, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="shell" :class="{ 'shell--player': hasPlayer }">
    <span class="grain" aria-hidden="true" />
    <span class="vignette" aria-hidden="true" />

    <TopBar class="shell-top" />

    <main class="shell-stage">
      <RouterView v-slot="{ Component }">
        <Transition name="view" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>

    <footer class="shell-foot">
      <span class="t-meta">{{ clock }}</span>
      <span class="t-meta">{{ mode.mode === 'music' ? 'MP3' : 'VIDEO' }} MODE</span>
      <span class="t-meta">{{ session.version ? `BUILD ${session.version}` : 'SELF-HOSTED' }}</span>
    </footer>

    <PlayerBar />
  </div>
</template>

<style scoped>
.shell {
  display: grid;
  grid-template-rows: auto 1fr auto;
  min-height: 100vh;
  background: var(--bg);
}

/* The stage holds the page and keeps the editorial rhythm. */
.shell-stage {
  min-width: 0;
  padding: var(--gap) var(--pad-x);
}

/* The footbar: two rules with a clock between them. */
.shell-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--xxl);
  padding: var(--lg) var(--pad-x);
  border-top: 1px solid var(--hairline);
}

.shell-foot .t-meta {
  color: var(--ink-faint);
}

.shell-foot .t-meta:nth-child(2) {
  color: var(--accent);
}

/* Leave room for the docked player only while it is on screen. */
.shell--player .shell-stage {
  padding-bottom: calc(76px + var(--gap));
}

.view-enter-active,
.view-leave-active {
  transition:
    opacity var(--dur-fast) var(--ease-out),
    transform var(--dur-fast) var(--ease-out);
}

.view-enter-from {
  opacity: 0;
  transform: translateY(6px);
}

.view-leave-to {
  opacity: 0;
}

@media (max-width: 620px) {
  .shell-foot {
    gap: var(--lg);
  }

  .shell-foot .t-meta:nth-child(3) {
    display: none;
  }
}
</style>
