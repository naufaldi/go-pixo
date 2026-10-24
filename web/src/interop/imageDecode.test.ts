import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { decodeFile } from './imageDecode';
import {jpegWithExif} from '../test/jpeg';

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

  it('waits for header-based admission before either native or bundled rasterization', async () => {
    for (const bytes of [png, tiff]) {
      let admit!: () => void;
      const admitted = new Promise<void>(resolve => { admit = resolve });
      const reserve = vi.fn(() => admitted);
      const controller = new AbortController();
      const outcome = decodeFile(new File(['image'], 'input'), controller.signal, {lossy: true, admit: reserve}).catch(error => error.name);
      PendingReader.instances.at(-1)!.complete(bytes);
      await flush();
      expect(reserve).toHaveBeenCalledWith(32);
      expect(PendingImage.instances).toHaveLength(0);
      expect(PendingDecoder.instances).toHaveLength(0);
      admit(); await flush();
      expect(bytes === png ? PendingImage.instances : PendingDecoder.instances).toHaveLength(1);
      controller.abort();
      expect(await outcome).toBe('AbortError');
      PendingImage.instances = []; PendingDecoder.instances = [];
    }
  });

  it.each(['png','jpeg','webp'])('prepares unchanged lossless %s bytes and an owned original preview without rasterizing', async format => {
    const bytes = format === 'png' ? png : format === 'jpeg'
      ? new Uint8Array([255,216,255,192,0,7,8,0,1,0,1])
      : new Uint8Array([82,73,70,70,18,0,0,0,87,69,66,80,86,80,56,76,5,0,0,0,47,0,0,0,0,0]);
    const canvas = vi.spyOn(document, 'createElement');
    const reserve = vi.fn(async () => {});
    const outcome = decodeFile(new File(['image'], 'input'), undefined, {lossy: false, outputFormat: 'same', admit: reserve});
    PendingReader.instances[0].complete(bytes); await flush();
    expect(PendingImage.instances).toHaveLength(0);
    expect(PendingDecoder.instances).toHaveLength(0);
    const result = await outcome;
    expect(result.originalFileBytes.buffer).toBe(bytes.buffer);
    expect(result.pixels).toBeUndefined();
    expect([result.width,result.height,result.previewUrl]).toEqual([1,1,'blob:decode']);
    expect(reserve).toHaveBeenCalledWith(format === 'png' ? 32 : 0);
    expect(canvas).not.toHaveBeenCalled();
    expect(URL.revokeObjectURL).not.toHaveBeenCalled();
    canvas.mockRestore();
  });

  it('returns JPEG display dimensions to the UI and worker without rasterizing unchanged bytes', async () => {
    const bytes=jpegWithExif(3000,4000,6);
    const outcome=decodeFile(new File(['image'],'rotated.jpg'),undefined,{lossy:false,targetWidth:4000});
    PendingReader.instances[0].complete(bytes); await flush();
    // Avoid a pending decoder obscuring the wrong-path failure.
    expect(PendingImage.instances).toHaveLength(0);
    const result=await outcome;
    expect(result).toMatchObject({width:4000,height:3000,previewUrl:'blob:decode'});
    expect(result.pixels).toBeUndefined();
  });

  it('reserves stored JPEG pixels before a display-axis downsize starts native decoding', async () => {
    const controller=new AbortController();
    const reserve=vi.fn(async () => {});
    const outcome=decodeFile(new File(['image'],'rotated.jpg'),controller.signal,{lossy:false,targetWidth:3000,admit:reserve})
      .catch(error => error.name);
    PendingReader.instances[0].complete(jpegWithExif(3000,4000,6)); await flush();
    expect(reserve).toHaveBeenCalledWith(384_000_000);
    expect(PendingImage.instances).toHaveLength(1);
    controller.abort(); expect(await outcome).toBe('AbortError');
  });

  it('rejects a native JPEG display mismatch before allocating its canvas', async () => {
    const canvas=vi.spyOn(document,'createElement');
    try {
      const outcome=decodeFile(new File(['image'],'rotated.jpg'),undefined,{lossy:true}).catch(error => error.message);
      PendingReader.instances[0].complete(jpegWithExif(3,4,6)); await flush();
      const image=PendingImage.instances[0]; image.naturalWidth=3; image.naturalHeight=4;
      image.onload?.();
      expect(await outcome).toMatch(/EXIF preflight/);
      expect(canvas).not.toHaveBeenCalled();
      expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:decode');
    } finally { canvas.mockRestore(); }
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
    const data = new Uint8ClampedArray([9,8,1,2,3,4,7,6]).subarray(2,6);
    vi.spyOn(canvas, 'getContext').mockReturnValue({drawImage() {}, getImageData: () => ({data})} as any);
    const outcome = decodeFile(new File(['image'], 'input'));
    PendingReader.instances[0].complete(png); await flush();
    PendingImage.instances[0].onload?.();
    const result = await outcome;
    expect(result.previewUrl).toBe('blob:decode');
    expect(result.pixels!.buffer).toBe(data.buffer);
    expect(result.pixels!.byteOffset).toBe(2);
    expect(Array.from(result.pixels!)).toEqual([1,2,3,4]);
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
