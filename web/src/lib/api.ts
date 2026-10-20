/**
 * Typed client for the frozen simple-ytdl-ui API contract.
 * Field names mirror the Go structs exactly (camelCase on the wire).
 */

export type Mode = 'video' | 'music'

export type JobStatus = 'queued' | 'running' | 'completed' | 'failed' | 'canceled'

export interface Health {
  status: string
  version: string
}

export interface AppConfig {
  googleEnabled: boolean
  defaultMode: Mode
  concurrent: number
}

export interface Session {
  signedIn: boolean
  email: string
}

export interface Job {
  id: string
  title: string
  url: string
  videoId: string
  mode: Mode
  status: JobStatus
  progress: number
  downloaded: number
  total: number
  speed: number
  eta: number
  error: string
  createdAt: string
  updatedAt: string
}

export interface Item {
  id: string
  videoId: string
  title: string
  artist: string
  duration: number
  file: string
  cover: string
  mime: string
  size: number
  mode: Mode
  sourceUrl: string
  playlist: string
  addedAt: string
}

export interface Playlist {
  id: string
  title: string
  itemCount: number
  thumb: string
}

export interface PlaylistItem {
  id: string
  title: string
  thumb: string
  channel: string
  duration: number
}

export interface DownloadRequest {
  url: string
  mode: Mode
  playlist: boolean
}

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response
  try {
    res = await fetch(path, {
      ...init,
      headers: {
        Accept: 'application/json',
        ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      },
    })
  } catch (err) {
    throw new ApiError(0, err instanceof Error ? err.message : 'Network request failed')
  }

  if (!res.ok) {
    let message = `${res.status} ${res.statusText || 'Request failed'}`
    try {
      const body = await res.json()
      if (body && typeof body.error === 'string' && body.error) message = body.error
      else if (body && typeof body.message === 'string' && body.message) message = body.message
    } catch {
      /* body was not json — keep the status line */
    }
    throw new ApiError(res.status, message)
  }

  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

const enc = encodeURIComponent

export const api = {
  health: () => request<Health>('/api/health'),

  config: () => request<AppConfig>('/api/config'),

  googleAuthUrl: () => request<{ url: string }>('/api/auth/google'),

  session: () => request<Session>('/api/auth/session'),

  logout: () => request<{ ok: boolean }>('/api/auth/logout', { method: 'POST' }),

  playlists: () => request<{ items: Playlist[] }>('/api/playlists'),

  playlistItems: (playlistId: string) =>
    request<{ items: PlaylistItem[]; nextPageToken: string }>(
      `/api/playlists/${enc(playlistId)}/items`,
    ),

  createDownload: (body: DownloadRequest) =>
    request<{ jobs: Job[] }>('/api/downloads', { method: 'POST', body: JSON.stringify(body) }),

  jobs: () => request<{ items: Job[] }>('/api/jobs'),

  cancelJob: (id: string) => request<{ ok: boolean }>(`/api/jobs/${enc(id)}/cancel`, { method: 'POST' }),

  library: () => request<{ items: Item[] }>('/api/library'),

  deleteItem: (id: string) => request<{ ok: boolean }>(`/api/library/${enc(id)}`, { method: 'DELETE' }),

  streamUrl: (id: string) => `/api/library/${enc(id)}/stream`,

  coverUrl: (id: string) => `/api/library/${enc(id)}/cover`,
}
