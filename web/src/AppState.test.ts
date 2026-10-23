import { afterEach, describe, expect, it, vi } from 'vitest'
import { createQueueItem, initialState, reducer } from './AppState.res.js'

const originalRevoke = URL.revokeObjectURL

afterEach(() => {
  URL.revokeObjectURL = originalRevoke
})

function queueItem(overrides: Record<string, unknown> = {}) {
  return {
    id: 'item-1',
    generation: 0,
    file: { name: 'image.png', size: 1234, type: 'image/png' },
    kind: 'Png',
    status: 'Done',
    originalUrl: 'blob:original',
    compressedUrl: 'blob:compressed',
    originalBytes: 1234,
    compressedBytes: 1000,
    width: 10,
    height: 10,
    compressionTime: 15,
    compressedMime: 'image/png',
    compressedExtension: '.png',
    ...overrides,
  }
}

describe('AppState reducer', () => {
  it('settings invalidate decoding, compressing and completed generations immediately', () => {
    URL.revokeObjectURL = vi.fn()
    const items = ['Decoding', 'Compressing', 'Done'].map((status, index) => queueItem({ id: `item-${index}`, status }))
    const next = reducer({ ...initialState, items, processingAll: true }, 'RequeueProcessedItemsForSettings')
    expect(next.items.map(item => [item.status, item.generation, item.compressedUrl])).toEqual([
      ['Pending', 1, undefined], ['Pending', 1, undefined], ['Pending', 1, undefined],
    ])
    expect(next.activeCompressions).toEqual([])
  })

  it('stale completion cannot replace a newer generation or recreate a removed item', () => {
    const updater = vi.fn(item => ({ ...item, status: 'Done' }))
    const state = { ...initialState, items: [queueItem({ generation: 2, status: 'Pending' })] }
    const action = { TAG: 'UpdateTask', _0: 'item-1', _1: 1, _2: updater }
    expect(reducer(state, action).items[0].status).toBe('Pending')
    expect(reducer({ ...state, items: [] }, action).items).toEqual([])
    expect(updater).not.toHaveBeenCalled()
  })

  it('recognizes broad formats case-insensitively', () => {
    const kinds = ['PHOTO.AVIF', 'photo.GIF', 'photo.BMP', 'photo.SVG', 'photo.HEIC', 'photo.TIFF', 'unknown.txt'].map(name =>
      createQueueItem({ name, size: 10, type: '' }).kind,
    )
    expect(kinds).toEqual(['Avif', 'Gif', 'Bmp', 'Svg', 'Heic', 'Tiff', 'Unknown'])
  })
  it('UpdateItem calls updater once and revokes cleared object URLs', () => {
    const revoke = vi.fn()
    URL.revokeObjectURL = revoke
    const updater = vi.fn((item) => ({
      ...item,
      compressedUrl: undefined,
    }))

    const next = reducer(
      { ...initialState, items: [queueItem()] },
      { TAG: 'UpdateItem', _0: 'item-1', _1: updater },
    )

    expect(updater).toHaveBeenCalledTimes(1)
    expect(revoke).toHaveBeenCalledWith('blob:compressed')
    expect(next.items[0].compressedUrl).toBeUndefined()
  })
})
