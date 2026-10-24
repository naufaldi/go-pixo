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
  it('holds a large image workspace through compression, then lets the waiting image rasterize', async () => {
    const actual = await vi.importActual<typeof import('./interop/imageDecode')>('./interop/imageDecode');
    const bytes = new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,15,160,0,0,11,184,8,6,0,0,0,0,0,0,0]); // 4000 x 3000
    class Reader {
      result = bytes.buffer; readyState = 2;
      onload?: () => void;
      readAsArrayBuffer() { this.onload?.() }
    }
    class Image {
      static instances: Image[] = [];
      naturalWidth = 4000; naturalHeight = 3000; src = '';
      onload?: () => void; onerror?: () => void;
      constructor() { Image.instances.push(this) }
    }
    vi.stubGlobal('FileReader', Reader); vi.stubGlobal('Image', Image);
    decodeFile.mockImplementation(actual.decodeFile);
    const {container} = render(createElement(App));
    addFiles(container, ['large-one.png','large-two.png']);
    await waitFor(() => expect(decodeFile).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(Image.instances).toHaveLength(1));
    // Keep the synthetic raster tiny. The worker double isolates admission from codec allocation.
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({drawImage() {}, getImageData: () => ({data: new Uint8ClampedArray(4)})} as any);
    await act(async () => Image.instances[0].onload?.());
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(1));
    expect(Image.instances).toHaveLength(1); // Decode completion alone cannot release the budget.
    await act(async () => AppWorker.instances[0].complete('png'));
    await waitFor(() => expect(Image.instances).toHaveLength(2));
  });

  it.each(['read', 'native', 'bundled'].flatMap(stage => ['remove', 'clear', 'unmount'].map(action => ({stage, action}))))('$action during actual $stage decoding releases resources and promptly refills admission', async ({stage, action}) => {
    const actual = await vi.importActual<typeof import('./interop/imageDecode')>('./interop/imageDecode')
    const bytes = stage === 'bundled'
      ? new Uint8Array([73,73,42,0,8,0,0,0,2,0,0,1,3,0,1,0,0,0,1,0,0,0,1,1,3,0,1,0,0,0,1,0,0,0,0,0,0,0])
      : new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,0,1,0,0,0,1,8,6,0,0,0,0,0,0,0])
    class Reader {
      static instances: Reader[] = []
      result: ArrayBuffer | null = null
      readyState = 0
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      onabort: (() => void) | null = null
      aborted = false
      constructor() { Reader.instances.push(this) }
      readAsArrayBuffer() {
        this.readyState = 1
        if (stage !== 'read') { this.result = bytes.buffer; this.readyState = 2; this.onload?.() }
      }
      abort() { this.aborted = true; this.readyState = 2; this.onabort?.() }
    }
    class Image {
      static instances: Image[] = []
      src = ''
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      constructor() { Image.instances.push(this) }
    }
    vi.stubGlobal('FileReader', Reader)
    vi.stubGlobal('Image', Image)
    decodeFile.mockImplementation(actual.decodeFile)
    const {container, unmount} = render(createElement(App))
    addFiles(container)
    await waitFor(() => expect(stage === 'read' ? Reader.instances : stage === 'native' ? Image.instances : AppWorker.instances).toHaveLength(2))
    if (action === 'remove') fireEvent.click(screen.getAllByRole('button', {name: 'Remove image'})[0])
    if (action === 'clear') {
      fireEvent.click(screen.getByRole('button', {name: 'Clear all'}))
      addFiles(container, ['fresh.png'])
    }
    if (action === 'unmount') unmount()
    const count = action === 'remove' ? 1 : 2
    if (stage === 'read') expect(Reader.instances.slice(0, count).every(reader => reader.aborted)).toBe(true)
    if (stage === 'native') {
      expect(Image.instances.slice(0, count).every(image => image.src === '' && image.onload === null)).toBe(true)
      expect(URL.revokeObjectURL).toHaveBeenCalledTimes(count)
    }
    if (stage === 'bundled') expect(AppWorker.instances.slice(0, count).every(worker => worker.terminated)).toBe(true)
    if (action !== 'unmount') await waitFor(() => expect(Reader.instances).toHaveLength(3))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    unmount()
  })

  it.each(['remove', 'clear', 'settings', 'unmount'])('%s aborts decoding but holds capacity until cancelled work settles', async action => {
    const pending: {signal: AbortSignal; settle: () => void}[] = []
    decodeFile.mockImplementation((_file: File, signal: AbortSignal) => new Promise((_resolve, reject) => {
      pending.push({signal, settle: () => reject(new DOMException('Cancelled', 'AbortError'))})
    }))
    const { container, unmount } = render(createElement(App))
    addFiles(container)
    await waitFor(() => expect(pending).toHaveLength(2))
    if (action === 'remove') fireEvent.click(screen.getAllByRole('button', { name: 'Remove image' })[0])
    if (action === 'clear') {
      fireEvent.click(screen.getByRole('button', { name: 'Clear all' }))
      addFiles(container, ['fresh.png'])
    }
    if (action === 'settings') fireEvent.click(screen.getByRole('checkbox'))
    if (action === 'unmount') unmount()
    expect(pending[0].signal?.aborted).toBe(true)
    expect(pending[1].signal?.aborted).toBe(action !== 'remove')
    expect(pending).toHaveLength(2)
    await act(async () => { pending[0].settle(); if (action !== 'remove') pending[1].settle() })
    if (action !== 'unmount') await waitFor(() => expect(pending).toHaveLength(action === 'settings' ? 4 : 3))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    unmount()
    await act(async () => pending.forEach(operation => operation.settle()))
  })

  it('allows RGBA fallback for default preset PNG requests without requiring a palette', async () => {
    decodeFile.mockImplementation(async (file: File) => ({
      ...decoded(),
      pixels: new Uint8Array([12, 34, 56, file.name === 'partial.png' ? 128 : 0]),
    }))
    const { container } = render(createElement(App))
    addFiles(container, ['partial.png', 'transparent.png'])
    await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    const messages = AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')
    for (const message of messages) {
      expect(message).toMatchObject({
        format: 'png', outputFormat: 'same', preset: 1, lossy: true, maxColors: 256,
        dithering: true, ditherStrength: 0.4, qualityTarget: 85, zopfliIterations: 5,
        pngAllowRGBAFallback: true,
      })
    }
    expect(messages.map(message => message.pixels[3]).sort()).toEqual([0, 128])
  })

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
    await act(async () => fireEvent.click(zip))
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

const readyForZip = async () => {
  const view = render(createElement(App))
  addFiles(view.container, ['one.png', 'two.png'])
  await waitFor(() => expect(AppWorker.instances.flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
  await act(async () => { AppWorker.instances[0].complete('png'); AppWorker.instances[1].complete('png') })
  return view
}

describe('ZIP export lifecycle', () => {
  it('shows busy state, suppresses duplicate clicks and allows another export after completion', async () => {
    let finish!: () => void
    downloadAllAsZip.mockImplementation(() => new Promise<void>(resolve => { finish = resolve }))
    await readyForZip()
    const button = screen.getByRole('button', { name: 'Download ZIP' })
    act(() => { fireEvent.click(button); fireEvent.click(button) })
    expect(downloadAllAsZip).toHaveBeenCalledTimes(1)
    expect(button).toBeDisabled()
    expect(button).toHaveTextContent('Exporting ZIP')
    expect(button).toHaveAttribute('aria-busy', 'true')
    expect(downloadAllAsZip.mock.calls[0][1]).toBeInstanceOf(AbortSignal)
    await act(async () => finish())
    expect(button).toBeEnabled()
    fireEvent.click(button)
    expect(downloadAllAsZip).toHaveBeenCalledTimes(2)
    await act(async () => finish())
  })

  it('reports export errors and restores the ZIP button for retry', async () => {
    downloadAllAsZip.mockRejectedValue(new Error('Read failed'))
    await readyForZip()
    fireEvent.click(screen.getByRole('button', { name: 'Download ZIP' }))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('ZIP download failed: Read failed'))
    expect(screen.getByRole('button', { name: 'Download ZIP' })).toBeEnabled()
  })

  it.each(['resolve', 'reject'] as const)('adding files aborts the previous ZIP and ignores its late %s', async settlement => {
    let signal!: AbortSignal
    let finish!: () => void
    let fail!: (error: Error) => void
    downloadAllAsZip.mockImplementation((_items, exportSignal) => {
      signal = exportSignal
      return new Promise<void>((resolve, reject) => { finish = resolve; fail = reject })
    })
    const { container } = await readyForZip()
    fireEvent.click(screen.getByRole('button', { name: 'Download ZIP' }))
    await act(async () => addFiles(container, ['new.png']))
    expect(signal.aborted).toBe(true)
    await act(async () => {
      if (settlement === 'resolve') finish()
      else fail(new Error('Stale export failure'))
    })
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(downloadAllAsZip).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('button', { name: 'Download ZIP' })).toBeDisabled()
  })

  it.each(['resolve', 'reject'] as const)('clear aborts the export and an old %s cannot change a newer export', async settlement => {
    const pending: { signal: AbortSignal; reject: (error: Error) => void; resolve: () => void }[] = []
    downloadAllAsZip.mockImplementation((_items, signal) => new Promise<void>((resolve, reject) => pending.push({ signal, resolve, reject })))
    const { container } = await readyForZip()
    fireEvent.click(screen.getByRole('button', { name: 'Download ZIP' }))
    fireEvent.click(screen.getByRole('button', { name: 'Clear all' }))
    expect(pending[0].signal?.aborted).toBe(true)
    addFiles(container, ['new-one.png', 'new-two.png'])
    await waitFor(() => expect(AppWorker.instances.filter(worker => !worker.terminated).flatMap(worker => worker.messages).filter(m => m.type === 'compress')).toHaveLength(2))
    await act(async () => AppWorker.instances.filter(worker => !worker.terminated).forEach(worker => worker.complete('png')))
    fireEvent.click(screen.getByRole('button', { name: 'Download ZIP' }))
    await act(async () => {
      if (settlement === 'resolve') pending[0].resolve()
      else pending[0].reject(new Error('Late old failure'))
    })
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    const button = screen.getByRole('button', { name: /Exporting ZIP/ })
    expect(button).toBeDisabled()
    expect(pending[1].signal.aborted).toBe(false)
    await act(async () => pending[1].resolve())
    expect(button).toBeEnabled()
  })

  it.each(['settings', 'remove'] as const)('aborts before %s revokes an export source', async action => {
    let signal!: AbortSignal
    downloadAllAsZip.mockImplementation((_items, exportSignal) => { signal = exportSignal; return new Promise(() => {}) })
    await readyForZip()
    fireEvent.click(screen.getByRole('button', { name: 'Download ZIP' }))
    const abortedAtRevoke: boolean[] = []
    vi.mocked(URL.revokeObjectURL).mockImplementation(() => abortedAtRevoke.push(signal.aborted))
    await act(async () => {
      if (action === 'settings') fireEvent.click(screen.getByRole('checkbox'))
      else fireEvent.click(screen.getAllByRole('button', { name: 'Remove image' })[0])
    })
    expect(signal.aborted).toBe(true)
    expect(abortedAtRevoke.length).toBeGreaterThan(0)
    expect(abortedAtRevoke.every(Boolean)).toBe(true)
  })

  it('unmount aborts the active export before URLs are released', async () => {
    let signal!: AbortSignal
    downloadAllAsZip.mockImplementation((_items, exportSignal) => { signal = exportSignal; return new Promise(() => {}) })
    const { unmount } = await readyForZip()
    fireEvent.click(screen.getByRole('button', { name: 'Download ZIP' }))
    const abortAtRevoke: boolean[] = []
    vi.mocked(URL.revokeObjectURL).mockImplementation(() => { abortAtRevoke.push(signal?.aborted ?? false) })
    unmount()
    expect(signal?.aborted).toBe(true)
    expect(abortAtRevoke.length).toBeGreaterThan(0)
    expect(abortAtRevoke.every(Boolean)).toBe(true)
  })
})

describe('live progress lifecycle', () => {
  it('keeps silent decode/worker clocks live, forwards phase units and cleans up a new settings generation', async () => {
    vi.useFakeTimers({toFake: ['setInterval', 'clearInterval', 'performance']})
    try {
      let finishDecode!: (value: ReturnType<typeof decoded>) => void
      decodeFile.mockImplementationOnce(() => new Promise(resolve => { finishDecode = resolve }))
      const {container, unmount} = render(createElement(App))
      await act(async () => addFiles(container, ['silent.png']))
      expect(screen.getByTestId('compression-phase')).toHaveTextContent(/decoding/i)
      expect(screen.getAllByRole('progressbar').every(bar => !bar.hasAttribute('aria-valuenow'))).toBe(true)
      await act(async () => vi.advanceTimersByTime(12_500))
      expect(screen.getByTestId('compression-elapsed')).toHaveTextContent('12.5s')
      expect(screen.getByTestId('file-compression-elapsed')).toHaveTextContent('12.5s')
      await act(async () => finishDecode(decoded()))
      expect(screen.queryAllByText(/has not reported progress/i)).toHaveLength(0)
      const worker = AppWorker.instances[0]
      const request = worker.messages.find(message => message.type === 'compress')
      await act(async () => worker.onmessage?.({data: {type: 'progress', id: request.id, phase: 'quantize', progress: 12, phasePercent: 25}}))
      expect(screen.getByTestId('compression-percent')).toHaveTextContent('25% of stage')
      await act(async () => vi.advanceTimersByTime(2500))
      expect(screen.getByTestId('compression-elapsed')).toHaveTextContent('15.0s')
      expect(screen.getByTestId('file-compression-percent')).toHaveTextContent('25% of stage')
      await act(async () => fireEvent.click(screen.getByRole('checkbox')))
      expect(screen.getByTestId('compression-elapsed')).toHaveTextContent('0.0s')
      expect(screen.getByTestId('compression-percent')).not.toHaveTextContent('25%')
      await act(async () => AppWorker.instances.filter(worker => !worker.terminated && worker.messages.some(message => message.type === 'compress')).forEach(worker => worker.complete('png')))
      expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
      expect(vi.getTimerCount()).toBe(0)
      await act(async () => fireEvent.click(screen.getByRole('checkbox')))
      expect(vi.getTimerCount()).toBe(1)
      unmount()
      expect(vi.getTimerCount()).toBe(0)
    } finally { vi.useRealTimers() }
  })
})
