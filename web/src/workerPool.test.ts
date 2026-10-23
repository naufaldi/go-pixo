import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { WorkerPool } from './workerPool'

class ControlledWorker {
  static instances: ControlledWorker[] = []
  onmessage: ((event: { data: unknown }) => void) | null = null
  onerror: ((event: { message: string; preventDefault(): void }) => void) | null = null
  onmessageerror: (() => void) | null = null
  messages: any[] = []
  terminated = false
  constructor() { ControlledWorker.instances.push(this) }
  postMessage(message: unknown) { this.messages.push(message) }
  terminate() { this.terminated = true }
  emit(data: unknown) { this.onmessage?.({ data }) }
  complete(id: string, outputFormat = 'webp') {
    this.emit({ type: 'compressed', id, outputFormat, compressedBytes: new Uint8Array([1, 2, 3]) })
  }
}

const pools: WorkerPool[] = []
const createPool = (size = 2) => {
  const pool = new WorkerPool(size)
  pools.push(pool)
  return pool
}
const flush = async () => { for (let i = 0; i < 5; i++) await Promise.resolve() }

beforeEach(() => {
  ControlledWorker.instances = []
  vi.stubGlobal('Worker', ControlledWorker)
})
afterEach(() => {
  pools.splice(0).forEach(pool => pool.terminate())
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('WorkerPool lifecycle', () => {
  it('fills both workers as each becomes ready, queues excess work, and returns actual output format', async () => {
    const pool = createPool()
    const results = ['a', 'b', 'c'].map(id => pool.submit({ id }, () => {}))
    const [first, second] = ControlledWorker.instances
    first.emit({ type: 'ready' })
    await flush()
    second.emit({ type: 'ready' })
    await flush()
    expect(first.messages.filter(m => m.type === 'compress').map(m => m.id)).toEqual(['a'])
    expect(second.messages.filter(m => m.type === 'compress').map(m => m.id)).toEqual(['b'])
    second.complete('b', 'avif')
    await flush()
    expect(second.messages.filter(m => m.type === 'compress').map(m => m.id)).toEqual(['b', 'c'])
    first.complete('a')
    second.complete('c', 'png')
    expect(await Promise.all(results)).toEqual([
      { compressedBytes: new Uint8Array([1, 2, 3]), outputFormat: 'webp' },
      { compressedBytes: new Uint8Array([1, 2, 3]), outputFormat: 'avif' },
      { compressedBytes: new Uint8Array([1, 2, 3]), outputFormat: 'png' },
    ])
  })

  it('ignores progress and completion for a different request ID', async () => {
    const pool = createPool(1)
    const progress = vi.fn()
    let settled = false
    const result = pool.submit({ id: 'current' }, progress).then(value => { settled = true; return value })
    const worker = ControlledWorker.instances[0]
    worker.emit({ type: 'ready' })
    await flush()
    worker.emit({ type: 'progress', id: 'old', phase: 'encoding', progress: 99 })
    worker.complete('old')
    await flush()
    expect(progress).not.toHaveBeenCalled()
    expect(settled).toBe(false)
    worker.complete('current')
    expect((await result).outputFormat).toBe('webp')
  })

  it('an item error releases its slot so a valid peer still finishes', async () => {
    const pool = createPool(1)
    const failed = pool.submit({ id: 'corrupt' }, () => {}).catch(error => error.message)
    const valid = pool.submit({ id: 'valid' }, () => {})
    const worker = ControlledWorker.instances[0]
    worker.emit({ type: 'ready' })
    await flush()
    worker.emit({ type: 'error', id: 'corrupt', error: 'Invalid image' })
    await flush()
    worker.complete('valid')
    expect(await failed).toBe('Invalid image')
    expect((await valid).compressedBytes).toHaveLength(3)
  })

  it('cancels a running item, rejects it, and continues the waiting peer on a replacement worker', async () => {
    const pool = createPool(1)
    const cancelled = pool.submit({ id: 'remove-me' }, () => {}).catch(error => error.message)
    const valid = pool.submit({ id: 'keep-me' }, () => {})
    const first = ControlledWorker.instances[0]
    first.emit({ type: 'ready' })
    pool.cancel('remove-me')
    expect(await cancelled).toBe('Compression cancelled')
    expect(first.terminated).toBe(true)
    const replacement = ControlledWorker.instances[1]
    replacement.emit({ type: 'ready' })
    replacement.complete('keep-me')
    expect((await valid).outputFormat).toBe('webp')
  })

  it('an initialization timeout settles promises before any worker becomes ready', async () => {
    vi.useFakeTimers()
    const pool = createPool(1)
    const failed = pool.submit({ id: 'waiting' }, () => {}).catch(error => error.message)
    await vi.runAllTimersAsync()
    expect(await failed).toBe('Worker initialization timed out')
  })

  it.each(['init', 'crash', 'terminate', 'timeout'])('%s failure settles active and queued promises', async failure => {
    vi.useFakeTimers()
    const pool = createPool(1)
    const results = ['a', 'b'].map(id => pool.submit({ id }, () => {}).then(() => 'resolved', error => error.message))
    const worker = ControlledWorker.instances[0]
    if (failure !== 'init') worker.emit({ type: 'ready' })
    await flush()
    if (failure === 'init') worker.emit({ type: 'error', error: 'WASM initialization failed' })
    if (failure === 'crash') worker.onerror?.({ message: 'Worker crashed', preventDefault() {} })
    if (failure === 'terminate') pool.terminate()
    if (failure === 'timeout') await vi.runAllTimersAsync()
    await flush()
    const settled = await Promise.all(results)
    expect(settled.every(message => /failed|crashed|terminated|timed out/i.test(message))).toBe(true)
    expect(worker.terminated).toBe(true)
  })
})
