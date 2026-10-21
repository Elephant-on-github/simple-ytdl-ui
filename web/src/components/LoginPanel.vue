<script setup lang="ts">
import { ref } from 'vue'
import { useSessionStore } from '../stores/session'

const session = useSessionStore()

const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit(): Promise<void> {
  if (!password.value) {
    error.value = 'Enter the password.'
    return
  }
  busy.value = true
  error.value = ''
  try {
    await session.login(password.value)
    password.value = ''
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not sign in.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="login frame">
    <span class="tick tick--tl" aria-hidden="true" />
    <span class="tick tick--tr" aria-hidden="true" />
    <span class="tick tick--bl" aria-hidden="true" />
    <span class="tick tick--br" aria-hidden="true" />

    <p class="eyebrow">Private</p>

    <h1 class="t-italic login-title">This server is locked</h1>

    <p class="t-serif-body login-copy">
      It asks for a password before it lets you in. Everything you do here stays on
      this machine.
    </p>

    <form class="form" @submit.prevent="submit">
      <label class="t-label form-label" for="login-password">Password</label>
      <input
        id="login-password"
        v-model="password"
        class="field form-input"
        type="password"
        autocomplete="current-password"
        autofocus
        aria-label="Password"
        @input="error = ''"
      />
      <button type="submit" class="form-go" :disabled="busy">
        <span>{{ busy ? 'Signing in' : 'Sign in' }}</span>
        <span aria-hidden="true">→</span>
      </button>
    </form>

    <p v-if="error" class="form-error t-meta">{{ error }}</p>
  </section>
</template>

<style scoped>
.login {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--lg);
  max-width: 520px;
  margin: clamp(40px, 8vw, 120px) auto;
  padding: clamp(24px, 4vw, 48px);
  background: var(--plate);
}

.login-title {
  color: var(--ink);
}

.login-copy {
  color: var(--ink-dim);
  max-width: 42ch;
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--lg);
  width: 100%;
  margin-top: var(--md);
}

.form-label {
  color: var(--ink-faint);
}

.form-input {
  width: 100%;
}

.form-go {
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--md);
  align-self: stretch;
  padding: var(--md) var(--xl);
  border: 1px solid var(--accent);
  color: var(--accent);
  font-family: var(--mono);
  font-size: var(--fs-nav);
  font-weight: 400;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  transition:
    background var(--dur-fast) var(--ease-out),
    color var(--dur-fast) var(--ease-out);
}

.form-go:hover:not(:disabled) {
  background: var(--accent);
  color: var(--bg);
}

.form-go:disabled {
  border-color: var(--hairline);
  color: var(--ink-faint);
  cursor: progress;
}

.form-error {
  color: var(--accent);
  text-transform: none;
  letter-spacing: 0.04em;
}
</style>
