import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { MODE_STORAGE_KEY, useModeStore } from './mode'

describe('mode store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('defaults to video', () => {
    expect(useModeStore().mode).toBe('video')
  })

  it('toggles between video and music', () => {
    const store = useModeStore()
    store.toggle()
    expect(store.mode).toBe('music')
    store.toggle()
    expect(store.mode).toBe('video')
  })

  it('persists the mode under simple-ytdl-ui:mode', () => {
    expect(MODE_STORAGE_KEY).toBe('simple-ytdl-ui:mode')

    const store = useModeStore()
    store.toggle()

    expect(localStorage.getItem('simple-ytdl-ui:mode')).toBe('music')
  })

  it('restores a stored mode on creation', () => {
    localStorage.setItem('simple-ytdl-ui:mode', 'music')
    expect(useModeStore().mode).toBe('music')
  })

  it('swaps the accent identity on the document element', () => {
    const store = useModeStore()
    store.toggle()
    expect(document.documentElement.dataset.mode).toBe('music')
  })

  it('takes the default the server announces while the user has no preference', () => {
    const store = useModeStore()

    store.applyServerDefault('music')

    expect(store.mode).toBe('music')
    // The server default is not a choice of the user, so it is not stored.
    expect(localStorage.getItem('simple-ytdl-ui:mode')).toBeNull()
  })

  it('keeps the choice of the user over the server default', () => {
    localStorage.setItem('simple-ytdl-ui:mode', 'video')
    const store = useModeStore()

    store.applyServerDefault('music')

    expect(store.mode).toBe('video')
  })

  it('lets the user overrule the server default', () => {
    const store = useModeStore()
    store.applyServerDefault('music')

    store.toggle()

    expect(store.mode).toBe('video')
    expect(localStorage.getItem('simple-ytdl-ui:mode')).toBe('video')
  })
})
