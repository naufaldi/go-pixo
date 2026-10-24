import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { decodeFile } from './imageDecode';

afterEach(() => vi.unstubAllGlobals());

class PendingReader {
  static instances: PendingReader[] = [];
  result: ArrayBuffer | null = null;
  readyState = 1;
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onabort: (() => void) | null = null;
  aborted = false;
  constructor() { PendingReader.instances.push(this); }
  readAsArrayBuffer() {}
  abort() { this.aborted = true; this.readyState = 2; this.onabort?.(); }
  complete(bytes: Uint8Array) { this.result = bytes.buffer as ArrayBuffer; this.readyState = 2; this.onload?.(); }
}
class PendingImage {
  static instances: PendingImage[] = [];
  src = '';
  naturalWidth = 1;
  naturalHeight = 1;
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  constructor() { PendingImage.instances.push(this); }
}
class PendingDecoder {
  static instances: PendingDecoder[] = [];
  onmessage: ((event: {data: unknown}) => void) | null = null;
  onerror: (() => void) | null = null;
  onmessageerror: (() => void) | null = null;
  terminated = false;
  constructor() { PendingDecoder.instances.push(this); }
  postMessage() {}
  terminate() { this.terminated = true; }
}
const png = new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,0,1,0,0,0,1,8,6,0,0,0,0,0,0,0]);
const tiff = new Uint8Array([73,73,42,0,8,0,0,0,2,0,0,1,3,0,1,0,0,0,1,0,0,0,1,1,3,0,1,0,0,0,1,0,0,0,0,0,0,0]);
const flush = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };

describe('decode cancellation resources', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    PendingReader.instances = []; PendingImage.instances = []; PendingDecoder.instances = [];
    vi.stubGlobal('FileReader', PendingReader);
    vi.stubGlobal('Image', PendingImage);
    vi.stubGlobal('Worker', PendingDecoder);
    vi.stubGlobal('URL', class extends URL {
      static createObjectURL = vi.fn(() => 'blob:decode');
      static revokeObjectURL = vi.fn();
    });
  });
  afterEach(() => vi.useRealTimers());

  it.each(['read', 'native', 'bundled'])('aborts and settles promptly during %s, detaching resources and late callbacks', async stage => {
    const controller = new AbortController();
    let settled = false;
    const outcome = decodeFile(new File(['image'], 'input'), controller.signal)
      .then(() => 'resolved', error => { settled = true; return error.name; });
    const reader = PendingReader.instances[0];
    if (stage !== 'read') reader.complete(stage === 'native' ? png : tiff);
    await flush();
    const img = PendingImage.instances[0], worker = PendingDecoder.instances[0];
    const lateLoad = img?.onload, lateMessage = worker?.onmessage;
    controller.abort();
    await flush();
    expect(settled).toBe(true);
    expect(await outcome).toBe('AbortError');
    expect(reader.onload).toBeNull();
    expect(reader.onerror).toBeNull();
    expect(reader.onabort).toBeNull();
    if (stage === 'read') expect(reader.aborted).toBe(true);
    if (stage === 'native') {
      expect(img.src).toBe('');
      expect(img.onload).toBeNull();
      expect(img.onerror).toBeNull();
      expect(URL.revokeObjectURL).toHaveBeenCalledTimes(1);
      expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:decode');
    }
    if (stage === 'bundled') {
      expect(worker.terminated).toBe(true);
      expect(worker.onmessage).toBeNull();
      expect(worker.onerror).toBeNull();
      expect(worker.onmessageerror).toBeNull();
    }
    lateLoad?.();
    lateMessage?.({data: {pixels: new Uint8Array(4), width: 1, height: 1, previewBytes: png}});
    await flush();
    expect(URL.createObjectURL).toHaveBeenCalledTimes(stage === 'native' ? 1 : 0);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('does not start reading an already aborted request', async () => {
    const controller = new AbortController(); controller.abort();
    await expect(decodeFile(new File(['image'], 'input'), controller.signal)).rejects.toMatchObject({name: 'AbortError'});
    expect(PendingReader.instances).toHaveLength(0);
  });

  it('a read deadline aborts the reader and settles instead of holding admission forever', async () => {
    const outcome = decodeFile(new File(['image'], 'input')).catch(error => error.message);
    await vi.advanceTimersByTimeAsync(30_000);
    expect(PendingReader.instances[0].aborted).toBe(true);
    expect(await outcome).toMatch(/timed out/);
  });

  it('releases native image and canvas resources after success while retaining the owned preview', async () => {
    const canvas = document.createElement('canvas');
    vi.spyOn(document, 'createElement').mockReturnValueOnce(canvas);
    vi.spyOn(canvas, 'getContext').mockReturnValue({drawImage() {}, getImageData: () => ({data: new Uint8ClampedArray([1,2,3,4])})} as any);
    const outcome = decodeFile(new File(['image'], 'input'));
    PendingReader.instances[0].complete(png); await flush();
    PendingImage.instances[0].onload?.();
    expect((await outcome).previewUrl).toBe('blob:decode');
    expect(PendingImage.instances[0].src).toBe('');
    expect([canvas.width, canvas.height]).toEqual([0, 0]);
    expect(URL.revokeObjectURL).not.toHaveBeenCalled();
    vi.restoreAllMocks();
  });
});

it('rejects animated files before a browser can flatten them', async () => {
  // Browser decode must never run for a known animation, even when MIME lies.
  vi.stubGlobal('Image', class {
    onerror?: () => void;
    set src(_: string) { queueMicrotask(() => this.onerror?.()); }
  });
  vi.stubGlobal('URL', {createObjectURL: () => 'blob:test', revokeObjectURL: () => {}});
  const frame = [44,0,0,0,0,1,0,1,0,0,2,2,68,1,0];
  const bytes = new Uint8Array([71,73,70,56,57,97,1,0,1,0,0,0,0,...frame,...frame,59]);
  await expect(decodeFile(new File([bytes], 'fake.png', {type:'IMAGE/PNG'}))).rejects.toThrow(/animat/i);
});
