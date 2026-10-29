<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import TopBar from './components/TopBar.vue'
import PlayerBar from './components/PlayerBar.vue'
import LoginPanel from './components/LoginPanel.vue'
import { setUnauthorizedHandler } from './lib/api'
import { usePlayerStore } from './stores/player'
import { useSessionStore } from './stores/session'
import { useModeStore } from './stores/mode'

const player = usePlayerStore()
const session = useSessionStore()
const mode = useModeStore()
const route = useRoute()
const router = useRouter()
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
  // A refused request means the session is gone: show the login again.
  setUnauthorizedHandler(() => {
    session.authenticated = false
  })
  // The result of a sign-in arrives in the query string; it is read by the
  // view that cares about it and then tidied away.
  if (route.query.signedIn || route.query.error) {
    void router.replace({ path: route.path, query: {} })
  }
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  setUnauthorizedHandler(null)
})
</script>

<template>
  <div class="shell" :class="{ 'shell--player': hasPlayer }">
    <span class="grain" aria-hidden="true" />
    <span class="vignette" aria-hidden="true" />

    <a href="#main-content" class="skip-link">Skip to main content</a>

    <TopBar class="shell-top" />

    <main id="main-content" class="shell-stage" tabindex="-1">
      <LoginPanel v-if="session.needsLogin" />
      <RouterView v-else v-slot="{ Component }">
        <Transition name="view" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>

    <footer class="shell-foot">
      <span class="t-meta">{{ clock }}</span>
      <span class="t-meta">{{ mode.mode === 'music' ? 'MP3' : 'VIDEO' }} MODE</span>
      <span class="t-meta">{{ session.version ? `BUILD ${session.version}` : 'SELF-HOSTED' }}</span>
      <button
        v-if="session.authRequired && session.authenticated"
        type="button"
        class="foot-out t-nav"
        @click="session.logout()"
      >
        Sign out
      </button>
    </footer>

    <PlayerBar />
  </div>
</template>

<style scoped>
.skip-link {
  position: absolute;
  top: -40px;
  left: 0;
  background: var(--accent);
  color: var(--bg);
  padding: var(--sm) var(--md);
  z-index: 100;
  transition: top var(--dur-fast) var(--ease-out);
  font-family: var(--mono);
  font-size: 11px;
}

.skip-link:focus {
  top: 0;
}

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

/* The way out of a locked server. */
.foot-out {
  color: var(--ink-faint);
  padding-bottom: 2px;
  border-bottom: 1px solid transparent;
  transition:
    color var(--dur-fast) var(--ease-out),
    border-color var(--dur-fast) var(--ease-out);
}

.foot-out:hover {
  color: var(--accent);
  border-bottom-color: var(--accent);
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
