<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { useSessionStore } from '../stores/session'

const session = useSessionStore()
const route = useRoute()

const password = ref('')
const error = ref('')
const busy = ref(false)
const googleBusy = ref(false)

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

async function signInWithGoogle(): Promise<void> {
  googleBusy.value = true
  error.value = ''
  try {
    // Come back to this page when Google is done.
    await session.signIn(route.path)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not start the sign-in.'
    googleBusy.value = false
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

    <div v-if="session.googleCanAuth" class="or">
      <span class="or-rule" aria-hidden="true" />
      <span class="t-label or-label">Or</span>
      <span class="or-rule" aria-hidden="true" />
    </div>

    <button
      v-if="session.googleCanAuth"
      type="button"
      class="google"
      :disabled="googleBusy"
      @click="signInWithGoogle"
    >
      <svg width="14" height="14" viewBox="0 0 18 18" fill="none" aria-hidden="true">
        <path
          d="M17.6 9.2c0-.6-.1-1.3-.2-1.9H9v3.6h4.8a4.1 4.1 0 0 1-1.8 2.7v2.2h2.9c1.7-1.6 2.7-3.9 2.7-6.6Z"
          fill="currentColor"
        />
        <path
          d="M9 18c2.4 0 4.5-.8 6-2.2l-2.9-2.2c-.8.5-1.8.9-3.1.9-2.4 0-4.4-1.6-5.1-3.8H.9v2.3A9 9 0 0 0 9 18Z"
          fill="currentColor"
          opacity=".72"
        />
        <path
          d="M3.9 10.7A5.4 5.4 0 0 1 3.6 9c0-.6.1-1.2.3-1.7V5H.9A9 9 0 0 0 0 9c0 1.5.4 2.8 1 4l2.9-2.3Z"
          fill="currentColor"
          opacity=".48"
        />
        <path
          d="M9 3.6c1.3 0 2.5.5 3.4 1.3l2.5-2.5C13.5.8 11.4 0 9 0A9 9 0 0 0 .9 5l3 2.3C4.6 5.2 6.6 3.6 9 3.6Z"
          fill="currentColor"
          opacity=".9"
        />
      </svg>
      {{ googleBusy ? 'Redirecting' : 'Continue with Google' }}
    </button>
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

/* --- the other way in ---------------------------------------------------- */

.or {
  display: flex;
  align-items: center;
  gap: var(--lg);
  width: 100%;
  margin-top: var(--md);
}

.or-rule {
  flex: 1;
  height: 1px;
  background: var(--hairline);
}

.or-label {
  color: var(--ink-faint);
}

.google {
  display: inline-flex;
  align-items: center;
  gap: var(--md);
  align-self: stretch;
  justify-content: center;
  padding: var(--md) var(--xl);
  border: 1px solid var(--hairline);
  color: var(--ink);
  font-family: var(--mono);
  font-size: var(--fs-nav);
  font-weight: 400;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  transition:
    border-color var(--dur-fast) var(--ease-out),
    color var(--dur-fast) var(--ease-out);
}

.google:hover:not(:disabled) {
  border-color: var(--accent);
  color: var(--accent);
}

.google:disabled {
  color: var(--ink-faint);
  cursor: progress;
}
</style>
