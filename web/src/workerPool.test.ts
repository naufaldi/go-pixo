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

  it('explicit termination settles active and queued promises and remains terminal', async () => {
    vi.useFakeTimers()
    const pool = createPool(1)
    const results = ['a', 'b'].map(id => pool.submit({ id }, () => {}).then(() => 'resolved', error => error.message))
    const worker = ControlledWorker.instances[0]
    worker.emit({ type: 'ready' })
    await flush()
    pool.terminate()
    await flush()
    const settled = await Promise.all(results)
    expect(settled.every(message => /failed|crashed|terminated|timed out/i.test(message))).toBe(true)
    expect(worker.terminated).toBe(true)
    await expect(pool.submit({ id: 'later' }, () => {})).rejects.toThrow(/terminated/)
    expect(pool.recover()).toBe(false)
  })

  it.each(['crash', 'timeout', 'messageerror'])('replaces only the %s slot, preserving its peer and queued/future work without retrying input', async failure => {
    vi.useFakeTimers()
    const pool = createPool()
    const failed = pool.submit({ id: 'failed' }, () => {}).catch(error => error.message)
    const peer = pool.submit({ id: 'peer' }, () => {})
    const queued = pool.submit({ id: 'queued' }, () => {})
    const [first, second] = ControlledWorker.instances
    second.emit({ type: 'ready' }) // Takes failed; peer starts later for a distinct deadline.
    await vi.advanceTimersByTimeAsync(1000)
    first.emit({ type: 'ready' })
    if (failure === 'crash') second.onerror?.({ message: 'Crashed', preventDefault() {} })
    if (failure === 'messageerror') second.onmessageerror?.()
    if (failure === 'timeout') await vi.advanceTimersByTimeAsync(299_000)
    expect(await failed).toMatch(/crashed|timed out|response/i)
    expect(first.terminated).toBe(false)
    first.complete('peer')
    expect((await peer).outputFormat).toBe('webp')
    const replacement = ControlledWorker.instances[2]
    expect(replacement).toBeDefined()
    replacement.emit({ type: 'ready' })
    const queuedWorker = ControlledWorker.instances.find(worker => !worker.terminated && worker.messages.some(m => m.type === 'compress' && m.id === 'queued'))!
    queuedWorker.complete('queued')
    expect((await queued).outputFormat).toBe('webp')
    const future = pool.submit({ id: 'future' }, () => {})
    const futureWorker = ControlledWorker.instances.find(worker => !worker.terminated && worker.messages.some(m => m.type === 'compress' && m.id === 'future'))!
    futureWorker.complete('future')
    expect((await future).outputFormat).toBe('webp')
    expect(ControlledWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress' && m.id === 'failed')).toHaveLength(1)
    expect(ControlledWorker.instances.filter(worker => !worker.terminated)).toHaveLength(2)
  })

  it('bounds initialization replacements, settles a systemic failure, and requires explicit recovery', async () => {
    const pool = createPool(1)
    const failed = pool.submit({ id: 'waiting' }, () => {}).catch(error => error.message)
    ControlledWorker.instances[0].emit({type: 'error', error: 'WASM initialization failed'})
    expect(ControlledWorker.instances).toHaveLength(2)
    ControlledWorker.instances[1].emit({type: 'error', error: 'WASM initialization failed'})
    expect(await failed).toMatch(/initialization failed/)
    await expect(pool.submit({id: 'blocked'}, () => {})).rejects.toThrow(/initialization failed/)
    expect(ControlledWorker.instances).toHaveLength(2)
    expect(pool.recover()).toBe(true)
    const recovered = pool.submit({id: 'recovered'}, () => {})
    ControlledWorker.instances[2].emit({type: 'ready'})
    ControlledWorker.instances[2].complete('recovered')
    expect((await recovered).outputFormat).toBe('webp')
  })

  it('keeps a healthy slot serving queued work when its peer exhausts initialization', async () => {
    const pool = createPool()
    const results = ['a', 'b', 'c'].map(id => pool.submit({id}, () => {}))
    const [first, second] = ControlledWorker.instances
    first.emit({type: 'ready'})
    second.emit({type: 'error', error: 'Cannot initialize'})
    ControlledWorker.instances[2].emit({type: 'error', error: 'Cannot initialize'})
    expect(first.terminated).toBe(false)
    first.complete('a'); first.complete('b'); first.complete('c')
    expect((await Promise.all(results)).map(result => result.outputFormat)).toEqual(['webp', 'webp', 'webp'])
    expect(ControlledWorker.instances).toHaveLength(3)
    pool.recover()
    const future = pool.submit({id: 'future'}, () => {})
    expect(ControlledWorker.instances).toHaveLength(4)
    first.complete('future')
    expect((await future).outputFormat).toBe('webp')
  })

  it('a synchronous send failure rejects only that input and ignores callbacks from its replaced worker', async () => {
    const pool = createPool()
    const failed = pool.submit({id: 'bad'}, () => {}).catch(error => error.message)
    const peer = pool.submit({id: 'peer'}, () => {})
    const [first, second] = ControlledWorker.instances
    const oldMessage = first.onmessage, oldError = first.onerror
    vi.spyOn(first, 'postMessage').mockImplementationOnce(() => { throw new Error('Cannot send') })
    first.emit({type: 'ready'})
    expect(await failed).toBe('Cannot send')
    second.emit({type: 'ready'}); second.complete('peer')
    expect((await peer).outputFormat).toBe('webp')
    oldMessage?.({data: {type: 'ready'}})
    oldError?.({message: 'Late crash', preventDefault() {}})
    expect(ControlledWorker.instances).toHaveLength(3)
  })

  it('bounds synchronous constructor failures until explicitly recovered', async () => {
    let attempts = 0
    vi.stubGlobal('Worker', class { constructor() { attempts++; throw new Error('Worker unavailable') } })
    const pool = createPool()
    await expect(pool.submit({id: 'waiting'}, () => {})).rejects.toThrow('Worker unavailable')
    expect(attempts).toBe(4)
    await expect(pool.submit({id: 'future'}, () => {})).rejects.toThrow('Worker unavailable')
    expect(attempts).toBe(4)
    vi.stubGlobal('Worker', ControlledWorker)
    pool.recover()
    const recovered = pool.submit({id: 'recovered'}, () => {})
    ControlledWorker.instances[0].emit({type: 'ready'})
    ControlledWorker.instances[0].complete('recovered')
    expect((await recovered).outputFormat).toBe('webp')
  })

  it.each(['typed', 'view', 'buffer'])('receives %s output without copying and preserves byte offsets', async kind => {
    const pool = createPool(1)
    const result = pool.submit({id: 'bytes'}, () => {})
    const worker = ControlledWorker.instances[0]; worker.emit({type: 'ready'})
    const storage = new Uint8Array([9, 1, 2, 3, 8])
    const bytes = kind === 'typed' ? storage.subarray(1, 4) : kind === 'view' ? new DataView(storage.buffer, 1, 3) : storage.buffer
    worker.emit({type: 'compressed', id: 'bytes', outputFormat: 'png', compressedBytes: bytes})
    const received = (await result).compressedBytes
    expect(received.buffer).toBe(storage.buffer)
    expect(Array.from(received)).toEqual(kind === 'buffer' ? [9,1,2,3,8] : [1,2,3])
    expect(received.byteOffset).toBe(kind === 'buffer' ? 0 : 1)
    if (kind === 'typed') expect(received).toBe(bytes)
  })
})

it('bounds failures from lazy Go startup even after transport readiness', async () => {
  const pool = createPool(1)
  const first = pool.submit({id: 'first-go'}, () => {}).catch(error => error.message)
  ControlledWorker.instances[0].emit({type: 'ready'})
  ControlledWorker.instances[0].emit({type: 'error', error: 'Go startup failed'})
  expect(await first).toBe('Go startup failed')
  const second = pool.submit({id: 'second-go'}, () => {}).catch(error => error.message)
  ControlledWorker.instances[1].emit({type: 'ready'})
  ControlledWorker.instances[1].emit({type: 'error', error: 'Go startup failed'})
  expect(await second).toBe('Go startup failed')
  expect(ControlledWorker.instances).toHaveLength(2)
  await expect(pool.submit({id: 'later'}, () => {})).rejects.toThrow('Go startup failed')
  expect(pool.recover()).toBe(true)
})
