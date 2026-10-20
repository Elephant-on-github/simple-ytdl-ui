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
})
