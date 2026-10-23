import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { createElement } from 'react'
import { make as App } from './App.res.js'

const { decodeFile, downloadAll, downloadAllAsZip } = vi.hoisted(() => ({ decodeFile: vi.fn(), downloadAll: vi.fn(), downloadAllAsZip: vi.fn() }))
vi.mock('./interop/imageDecode', () => ({ decodeFile }))
vi.mock('./interop/download', () => ({ downloadAll, downloadAllAsZip, downloadBlob: vi.fn() }))

class AppWorker {
  static instances: AppWorker[] = []
  onmessage: ((event: { data: unknown }) => void) | null = null
  onerror: ((event: unknown) => void) | null = null
  messages: any[] = []
  terminated = false
  constructor() { AppWorker.instances.push(this) }
  postMessage(message: any) {
    this.messages.push(message)
    if (message.type === 'init') queueMicrotask(() => this.onmessage?.({ data: { type: 'ready' } }))
  }
  terminate() { this.terminated = true }
  complete(format = 'webp') {
    const message = [...this.messages].reverse().find(m => m.type === 'compress')
    this.onmessage?.({ data: { type: 'compressed', id: message.id, outputFormat: format, compressedBytes: new Uint8Array([1, 2]) } })
  }
}

const decoded = () => ({ pixels: new Uint8Array(4), originalFileBytes: new Uint8Array(8), width: 1, height: 1, colorType: 6, inputFormat: 'png', previewUrl: 'blob:original' })
const addFiles = (container: HTMLElement, names = ['one.png', 'two.png', 'three.png']) => {
  const files = names.map(name => new File(['original image'], name, { type: 'image/png' }))
  fireEvent.change(container.querySelector('input[type=file]')!, { target: { files } })
}

beforeEach(() => {
  AppWorker.instances = []
  vi.stubGlobal('Worker', AppWorker)
  vi.stubGlobal('URL', class extends URL {
    static createObjectURL() { return 'blob:compressed' }
    static revokeObjectURL = vi.fn()
  })
  vi.spyOn(navigator, 'hardwareConcurrency', 'get').mockReturnValue(8)
  decodeFile.mockReset().mockImplementation(async () => decoded())
  downloadAll.mockReset()
  downloadAllAsZip.mockReset().mockResolvedValue(undefined)
})
afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.unstubAllGlobals() })

describe('bounded batch UI', () => {
  it('bounds decoding and encoding together, refills a free slot, and records actual worker format', async () => {
    const { container } = render(createElement(App))
    addFiles(container)
    await waitFor(() => expect(decodeFile).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    await act(async () => { AppWorker.instances[0].complete('avif') })
    await waitFor(() => expect(decodeFile).toHaveBeenCalledTimes(3))
    expect(screen.getByTestId('bottom-bar')).toHaveTextContent('Output: AVIF')
    await act(async () => {
      AppWorker.instances[0].complete('webp')
      AppWorker.instances[1].complete('png')
    })
    await waitFor(() => expect(screen.getByRole('button', { name: 'Compress All' })).toBeEnabled())
    expect(screen.getByTestId('batch-summary')).toHaveTextContent('3 completed')
  })

  it('waits for the entire batch before enabling ZIP and Download All', async () => {
    const { container } = render(createElement(App))
    addFiles(container)
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    await act(async () => { AppWorker.instances[0].complete('png') })
    await waitFor(() => expect(decodeFile).toHaveBeenCalledTimes(3))
    await act(async () => { AppWorker.instances[1].complete('png') })
    const zip = screen.getByRole('button', { name: 'Download ZIP' })
    const all = screen.getByRole('button', { name: 'Download All' })
    expect(zip).toBeDisabled()
    expect(all).toBeDisabled()
    fireEvent.click(zip)
    fireEvent.click(all)
    expect(downloadAllAsZip).not.toHaveBeenCalled()
    expect(downloadAll).not.toHaveBeenCalled()
    await act(async () => { AppWorker.instances[0].complete('png') })
    await waitFor(() => expect(zip).toBeEnabled())
    expect(all).toBeEnabled()
    fireEvent.click(zip)
    expect(downloadAllAsZip).toHaveBeenCalledTimes(1)
    expect(downloadAllAsZip.mock.calls[0][0].map((item: { status: string }) => item.status)).toEqual(['Done', 'Done', 'Done'])
  })

  it('labels successful-only batch downloads after failed files settle', async () => {
    decodeFile.mockImplementation(async (file: File) => {
      if (file.name === 'broken.png') throw new Error('Invalid image')
      return decoded()
    })
    const { container } = render(createElement(App))
    addFiles(container, ['one.png', 'two.png', 'broken.png'])
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    await act(async () => { AppWorker.instances[0].complete('png'); AppWorker.instances[1].complete('png') })
    await waitFor(() => expect(screen.getByTestId('batch-summary')).toHaveTextContent('2 completed, 1 failed, 0 remaining'))
    expect(screen.getByRole('button', { name: 'Download ZIP (2 successful)' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Download successful files (2)' })).toBeEnabled()
    expect(screen.getByTestId('bottom-bar')).toHaveTextContent('1 failed file excluded from downloads.')
  })

  it('restores an accurate completed-settings summary and removes it while new settings run', async () => {
    const { container } = render(createElement(App))
    addFiles(container, ['one.png', 'two.png'])
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    expect(screen.queryByText(/What we did:/)).not.toBeInTheDocument()
    await act(async () => { AppWorker.instances[0].complete('png'); AppWorker.instances[1].complete('png') })
    const bar = screen.getByTestId('bottom-bar')
    expect(bar).toHaveTextContent('What we did: Balanced preset, Lossy compression allowed')
    fireEvent.click(screen.getByRole('checkbox'))
    expect(screen.queryByText(/What we did:/)).not.toBeInTheDocument()
    await waitFor(() => expect(AppWorker.instances.filter(worker => !worker.terminated).flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    await act(async () => { AppWorker.instances.filter(worker => !worker.terminated).forEach(worker => worker.complete('png')) })
    expect(bar).toHaveTextContent('What we did: Balanced preset, Lossless mode')
    expect(bar).not.toHaveTextContent('Lossy compression allowed')
  })

  it('uses the decoded input signature instead of trusting a misleading file name', async () => {
    decodeFile.mockImplementation(async () => ({ ...decoded(), inputFormat: 'jpeg' }))
    const { container } = render(createElement(App))
    addFiles(container, ['misleading.png'])
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(1))
    expect(AppWorker.instances.flatMap(worker => worker.messages).find(m => m.type === 'compress').format).toBe('jpeg')
    expect(screen.getByText('Input: JPEG')).toBeInTheDocument()
  })

  it('passes Auto through to the engine and keeps same-as-input as the default', async () => {
    const { container } = render(createElement(App))
    expect(screen.getByRole('combobox', { name: 'Output format' })).toHaveValue('same')
    fireEvent.change(screen.getByRole('combobox', { name: 'Output format' }), { target: { value: 'auto' } })
    addFiles(container, ['one.png'])
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(1))
    expect(AppWorker.instances.flatMap(worker => worker.messages).find(m => m.type === 'compress').outputFormat).toBe('auto')
  })

  it('applies one resize axis per file so mixed aspect ratios stay intact, with a selected-image preview', async () => {
    decodeFile.mockImplementation(async (file: File) => ({ ...decoded(), width: file.name === 'wide.png' ? 400 : 100, height: 200 }))
    const { container } = render(createElement(App))
    addFiles(container, ['wide.png', 'tall.png'])
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    await act(async () => { AppWorker.instances[0].complete(); AppWorker.instances[1].complete() })
    fireEvent.change(screen.getByPlaceholderText('W'), { target: { value: '100' } })
    expect(screen.getByPlaceholderText('H')).toHaveValue(50)
    await waitFor(() => expect(AppWorker.instances.filter(worker => !worker.terminated).flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    const messages = AppWorker.instances.filter(worker => !worker.terminated).flatMap(worker => worker.messages).filter(m => m.type === 'compress')
    expect(messages.map(message => [message.width, message.height, message.targetWidth, message.targetHeight])).toEqual([
      [400, 200, 100, undefined], [100, 200, 100, undefined],
    ])
  })

  it('a corrupt item settles as failed while valid peers complete', async () => {
    decodeFile.mockImplementation(async (file: File) => {
      if (file.name === 'broken.png') throw new Error('Invalid image')
      return decoded()
    })
    const { container } = render(createElement(App))
    addFiles(container, ['broken.png', 'valid.png'])
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(1))
    await act(async () => AppWorker.instances.find(worker => worker.messages.some(m => m.type === 'compress'))!.complete())
    await waitFor(() => expect(screen.getByTestId('batch-summary')).toHaveTextContent('1 completed, 1 failed, 0 remaining'))
    expect(screen.getByRole('button', { name: 'Compress All' })).toBeEnabled()
    expect(screen.getByRole('alert')).toHaveTextContent('broken.png: Invalid image')
  })

  it('settings during decode ignore the old generation and wait for capacity', async () => {
    const resolves: ((value: ReturnType<typeof decoded>) => void)[] = []
    decodeFile.mockImplementation(() => new Promise(resolve => resolves.push(resolve)))
    const { container } = render(createElement(App))
    addFiles(container, ['one.png', 'two.png'])
    await waitFor(() => expect(resolves).toHaveLength(2))
    fireEvent.click(screen.getByRole('checkbox'))
    expect(decodeFile).toHaveBeenCalledTimes(2)
    await act(async () => { resolves[0](decoded()); resolves[1](decoded()) })
    await waitFor(() => expect(resolves).toHaveLength(4))
    expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(0)
    await act(async () => { resolves[2](decoded()); resolves[3](decoded()) })
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress').every(message => message.lossy === false)).toBe(true)
  })

  it('removing an encoding item prevents its late completion and lets the other item settle', async () => {
    const { container } = render(createElement(App))
    addFiles(container, ['remove.png', 'keep.png'])
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    const removed = AppWorker.instances[0]
    fireEvent.click(screen.getAllByRole('button', { name: 'Remove image' })[0])
    await act(async () => {
      removed.complete('avif')
      AppWorker.instances[1].complete('png')
    })
    await waitFor(() => expect(screen.getByTestId('batch-summary')).toHaveTextContent('1 completed, 0 failed, 0 remaining'))
    expect(screen.queryByText('remove.png')).not.toBeInTheDocument()
    expect(screen.getByTestId('bottom-bar')).toHaveTextContent('Output: PNG')
  })

  it('clearing during decode cannot resurrect files or create stale output URLs', async () => {
    let resolveDecode!: (value: ReturnType<typeof decoded>) => void
    decodeFile.mockImplementation(() => new Promise(resolve => { resolveDecode = resolve }))
    const { container } = render(createElement(App))
    addFiles(container, ['one.png'])
    await waitFor(() => expect(decodeFile).toHaveBeenCalledTimes(1))
    fireEvent.click(screen.getByRole('button', { name: 'Clear all' }))
    await act(async () => resolveDecode(decoded()))
    expect(screen.queryByText('one.png')).not.toBeInTheDocument()
    expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(0)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:original')
  })
})
