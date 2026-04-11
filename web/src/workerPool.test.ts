import { describe, it, expect, vi, beforeEach } from 'vitest'

// Mock the Worker global before importing WorkerPool
const mockWorkerInstances: any[] = []

class MockWorker {
  url: any
  options: any
  onmessage: ((e: MessageEvent) => void) | null = null
  private _terminated = false

  constructor(url: any, options?: any) {
    this.url = url
    this.options = options
    mockWorkerInstances.push(this)
  }

  postMessage(msg: any) {
    if (this._terminated) return
    // Simulate async ready/result based on message type
    if (msg?.type === 'init') {
      setTimeout(() => {
        this.onmessage?.({ data: { type: 'ready' } } as MessageEvent)
      }, 0)
    } else if (msg?.type === 'compress') {
      setTimeout(() => {
        this.onmessage?.({
          data: {
            type: 'compressed',
            id: msg.id,
            compressedBytes: new Uint8Array([1, 2, 3]).buffer,
          },
        } as unknown as MessageEvent)
      }, 5)
    }
  }

  terminate() {
    this._terminated = true
  }
}

vi.stubGlobal('Worker', MockWorker)

// Use vi.mock to handle the dynamic Worker URL inside WorkerPool
vi.mock('./workerPool', async (importOriginal) => {
  return await importOriginal()
})

import { WorkerPool } from './workerPool'

describe('WorkerPool', () => {
  beforeEach(() => {
    mockWorkerInstances.length = 0
  })

  it('submits a task and resolves with Uint8Array', async () => {
    const pool = new WorkerPool(1)
    const onProgress = vi.fn()

    const result = await pool.submit(
      {
        type: 'compress',
        id: 'test-1',
        pixels: new Uint8Array(4),
        width: 1,
        height: 1,
        colorType: 6,
        format: 'png',
        preset: 1,
        lossy: false,
        maxColors: 0,
        dithering: false,
      },
      onProgress
    )

    expect(result).toBeInstanceOf(Uint8Array)
    pool.terminate()
  })

  it('terminate cleans up all workers', () => {
    const pool = new WorkerPool(2)
    pool.terminate()
    expect(() => pool.terminate()).not.toThrow()
  })
})
