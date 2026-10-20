/**
 * YouTube URL recognition.
 *
 * The backend classifies URLs on its own (a `watch?v=…&list=…` link is always a
 * *video*, never a playlist), but the UI needs the same answer to label the
 * request and to validate the field before it ever reaches the network.
 */

const VIDEO_ID = /^[A-Za-z0-9_-]{3,}$/

function parse(raw: string): URL | null {
  const trimmed = (raw ?? '').trim()
  if (!trimmed) return null
  const candidate = /^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(trimmed) ? trimmed : `https://${trimmed}`
  try {
    return new URL(candidate)
  } catch {
    return null
  }
}

function isYouTubeHost(host: string): boolean {
  const h = host.toLowerCase()
  return h === 'youtube.com' || h.endsWith('.youtube.com') || h === 'youtu.be' || h.endsWith('.youtu.be')
}

/** The video id of a watch / short / embed / youtu.be link, if there is one. */
export function extractVideoID(raw: string): string | null {
  const url = parse(raw)
  if (!url || !isYouTubeHost(url.hostname)) return null

  const host = url.hostname.toLowerCase()
  const path = url.pathname.replace(/^\/+/, '')

  if (host === 'youtu.be' || host.endsWith('.youtu.be')) {
    const id = path.split('/')[0] ?? ''
    return VIDEO_ID.test(id) ? id : null
  }

  const queryVideo = url.searchParams.get('v')
  if (queryVideo && VIDEO_ID.test(queryVideo)) return queryVideo

  const segments = path.split('/').filter(Boolean)
  if (segments[0] === 'shorts' || segments[0] === 'embed' || segments[0] === 'live') {
    const id = segments[1] ?? ''
    return VIDEO_ID.test(id) ? id : null
  }

  return null
}

/** The playlist id of a `playlist?list=…` link. */
export function extractPlaylistID(raw: string): string | null {
  const url = parse(raw)
  if (!url || !isYouTubeHost(url.hostname)) return null
  const list = url.searchParams.get('list')
  if (!list || !VIDEO_ID.test(list)) return null
  return list
}

/**
 * True when the link points at a single video — even if it also carries a
 * `list=` parameter, exactly like the backend.
 */
export function isVideoURL(raw: string): boolean {
  return extractVideoID(raw) !== null
}

/**
 * True when the link points at a playlist *page* (not a watch link that
 * happens to carry a `list=` parameter).
 */
export function isPlaylistURL(raw: string): boolean {
  if (extractVideoID(raw) !== null) return false
  return extractPlaylistID(raw) !== null
}

/** True for any link the backend can act on. */
export function isValidYouTubeURL(raw: string): boolean {
  return isVideoURL(raw) || isPlaylistURL(raw)
}
