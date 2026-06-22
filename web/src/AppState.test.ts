import { afterEach, describe, expect, it, vi } from 'vitest'
import { initialState, reducer } from './AppState.res.js'

const originalRevoke = URL.revokeObjectURL

afterEach(() => {
  URL.revokeObjectURL = originalRevoke
})

function queueItem(overrides: Record<string, unknown> = {}) {
  return {
    id: 'item-1',
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
