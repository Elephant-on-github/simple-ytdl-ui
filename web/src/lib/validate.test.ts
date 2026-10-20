import { describe, expect, it } from 'vitest'
import { isPlaylistURL, isValidYouTubeURL, isVideoURL } from './validate'

describe('isValidYouTubeURL', () => {
  it('accepts a watch url', () => {
    expect(isValidYouTubeURL('https://www.youtube.com/watch?v=abc')).toBe(true)
  })

  it('accepts a youtu.be url', () => {
    expect(isValidYouTubeURL('https://youtu.be/abc')).toBe(true)
  })

  it('accepts a playlist url', () => {
    expect(isValidYouTubeURL('https://www.youtube.com/playlist?list=PL1')).toBe(true)
  })

  it('rejects a non-youtube url', () => {
    expect(isValidYouTubeURL('https://example.com/x.mp3')).toBe(false)
  })

  it('rejects an empty string', () => {
    expect(isValidYouTubeURL('')).toBe(false)
  })

  it('rejects a youtube host without an id', () => {
    expect(isValidYouTubeURL('https://www.youtube.com/')).toBe(false)
    expect(isValidYouTubeURL('https://www.youtube.com/watch?v=')).toBe(false)
  })
})

describe('video versus playlist', () => {
  it('treats a watch url that also carries a list parameter as a video', () => {
    const url = 'https://www.youtube.com/watch?v=abc&list=PL1'
    expect(isVideoURL(url)).toBe(true)
    expect(isPlaylistURL(url)).toBe(false)
  })

  it('treats a playlist url as a playlist', () => {
    const url = 'https://www.youtube.com/playlist?list=PL1'
    expect(isVideoURL(url)).toBe(false)
    expect(isPlaylistURL(url)).toBe(true)
  })
})
