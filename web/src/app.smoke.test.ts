import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import App from './App.vue'
import DownloadView from './views/DownloadView.vue'
import LibraryView from './views/LibraryView.vue'
import PlaylistsView from './views/GoogleView.vue'
import { useModeStore } from './stores/mode'
import { usePlayerStore } from './stores/player'
import type { Item, Job } from './lib/api'

const jobs: Job[] = [
  {
    id: 'j1', title: 'Blinding Lights', url: 'u', videoId: 'v', mode: 'music', status: 'running',
    progress: 42.5, downloaded: 2202009, total: 5242880, speed: 1048576, eta: 3, error: '',
    createdAt: '2026-10-20T10:02:00Z', updatedAt: '2026-10-20T10:02:10Z',
  },
  {
    id: 'j3', title: 'Private Video', url: 'u', videoId: 'v', mode: 'video', status: 'failed',
    progress: 8, downloaded: 0, total: 100, speed: 0, eta: 0, error: 'ERROR: Private video',
    createdAt: '2026-10-20T09:58:00Z', updatedAt: '2026-10-20T09:58:30Z',
  },
]

const library: Item[] = [
  {
    id: 'yt_1', videoId: '1', title: 'Nightcall', artist: 'Kavinsky', duration: 257, file: 'f',
    cover: 'c', mime: 'audio/mpeg', size: 7340032, mode: 'music', sourceUrl: 'u', playlist: '',
    addedAt: '2026-10-20T09:12:00Z',
  },
  {
    id: 'yt_2', videoId: '2', title: 'City Lights', artist: 'Marrow', duration: 321, file: 'f',
    cover: 'c', mime: 'audio/mpeg', size: 6291456, mode: 'music', sourceUrl: 'u', playlist: '',
    addedAt: '2026-10-19T09:12:00Z',
  },
]

const responses: Record<string, unknown> = {
  '/api/health': { status: 'ok', version: '0.1.0' },
  '/api/config': { googleEnabled: true, defaultMode: 'video', concurrent: 2 },
  '/api/jobs': { items: jobs },
  '/api/library': { items: library },
  '/api/playlists': { items: [{ id: 'PL1', title: 'Late Night Drive', itemCount: 42, thumb: '' }] },
  '/api/playlists/PL1/items': {
    items: [{ id: 'v1', title: 'Midnight City', thumb: '', channel: 'M83', duration: 244 }],
    nextPageToken: '',
  },
}

/** The session the server reports. Tests may change it. */
let sessionResponse: Record<string, unknown> = {
  authenticated: true,
  authRequired: false,
  googleCanAuth: false,
  playlistsAvailable: true,
  playlistSource: 'google',
  signedIn: true,
  email: 'dev@example.com',
}

function stubFetch(): void {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input).split('?')[0]
      const method = init?.method ?? 'GET'

      if (path === '/api/auth/session') {
        return new Response(JSON.stringify(sessionResponse), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (path === '/api/session/cookies' && method === 'POST') {
        return new Response(JSON.stringify({ ok: true, playlists: 3 }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      const body = responses[path]
      if (body === undefined) {
        return new Response('{}', { status: 404, headers: { 'Content-Type': 'application/json' } })
      }
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    }),
  )
}

/** jsdom has no media engine — stub the two calls the player makes. */
function stubMedia(): void {
  Object.defineProperty(HTMLMediaElement.prototype, 'load', { value: () => undefined })
  Object.defineProperty(HTMLMediaElement.prototype, 'play', {
    value: () => Promise.resolve(),
  })
  Object.defineProperty(HTMLMediaElement.prototype, 'pause', { value: () => undefined })
}

async function mount(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: DownloadView },
      { path: '/library', component: LibraryView },
      { path: '/playlists', component: PlaylistsView },
    ],
  })
  router.push(path)
  await router.isReady()

  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  const host = document.createElement('div')
  document.body.appendChild(host)
  app.mount(host)

  await nextTick()
  await new Promise((resolve) => setTimeout(resolve, 50))
  await nextTick()

  return { app, html: host.innerHTML }
}

describe('app smoke', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    stubFetch()
    stubMedia()
  })

  it('renders the download view with the live queue', async () => {
    const { app, html } = await mount('/')

    // The download bar is the main thing, the settings sit underneath it.
    expect(html).toContain('https://www.youtube.com/watch?v=…')
    expect(html).toContain('Download')
    expect(html).toContain('Output')

    expect(html).toContain('Blinding Lights')
    expect(html).toContain('aria-valuenow="43"')
    expect(html).toContain('style="width: 43%;"')
    expect(html).toContain('ERROR: Private video')
    expect(html).toContain('1.0 MB/s')

    app.unmount()
  })

  it('renders the library grid and loads an item into the player', async () => {
    const { app, html } = await mount('/library')

    expect(html).toContain('Nightcall')
    expect(html).toContain('Kavinsky')
    expect(html).toContain('4:17')

    const player = usePlayerStore()
    player.load(library[0])
    await nextTick()

    expect(player.isOpen).toBe(true)
    expect(player.current?.title).toBe('Nightcall')
    expect(player.playing).toBe(true)

    app.unmount()
  })

  it('renders the playlists view with a sync action', async () => {
    const { app, html } = await mount('/playlists')

    expect(html).toContain('Late Night Drive')
    expect(html).toContain('42 items')
    expect(html).toContain('Sync')

    app.unmount()
  })

  it('shows why a google sign-in failed when it comes back from google', async () => {
    const { app, html } = await mount('/playlists?error=exchange_failed')

    expect(html).toContain('did not hand out an access token')

    app.unmount()
  })

  it('confirms a sign-in that came back from google', async () => {
    const { app, html } = await mount('/playlists?signedIn=1')

    expect(html).toContain('Signed in as dev@example.com')

    app.unmount()
  })

  it('shows the login screen with the google option when the server asks for one', async () => {
    const before = { ...sessionResponse }
    sessionResponse = {
      authenticated: false,
      authRequired: true,
      googleCanAuth: true,
      playlistsAvailable: false,
      playlistSource: '',
      signedIn: false,
      email: '',
    }

    const { app, html } = await mount('/')

    expect(html).toContain('This server is locked')
    expect(html).toContain('Continue with Google')
    // The app itself is not rendered while we are outside.
    expect(html).not.toContain('youtube.com/watch?v=')

    sessionResponse = before
    app.unmount()
  })

  it('loads the playlists once a session is connected', async () => {
    // No session yet: the panel shows.
    const before = { ...sessionResponse }
    sessionResponse = {
      authenticated: true,
      authRequired: false,
      googleCanAuth: false,
      playlistsAvailable: false,
      playlistSource: '',
      signedIn: false,
      email: '',
    }

    const { app } = await mount('/playlists')

    expect(document.body.innerHTML).toContain('Give the server your session')

    // The server has a session from now on.
    sessionResponse = { ...before, playlistsAvailable: true, playlistSource: 'cookies' }

    const area = document.querySelector('.paste-area') as HTMLTextAreaElement
    const setter = Object.getOwnPropertyDescriptor(
      window.HTMLTextAreaElement.prototype,
      'value',
    )?.set
    setter?.call(area, '# Netscape HTTP Cookie File')
    area.dispatchEvent(new Event('input', { bubbles: true }))
    ;(document.querySelector('.paste button[type=submit]') as HTMLButtonElement).click()

    await new Promise((resolve) => setTimeout(resolve, 300))
    await nextTick()

    // The playlists are on the page now, without a reload.
    expect(document.body.innerHTML).toContain('Late Night Drive')

    sessionResponse = before
    app.unmount()
  })

  it('swaps the accent identity when the mode changes', async () => {
    const { app } = await mount('/')
    const mode = useModeStore()

    mode.toggle()
    await nextTick()

    expect(mode.mode).toBe('music')
    expect(document.documentElement.dataset.mode).toBe('music')

    app.unmount()
  })
})
