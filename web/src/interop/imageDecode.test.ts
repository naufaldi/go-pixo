import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { decodeFile, MAX_FALLBACK_PIXELS } from './imageDecode';
import {jpegWithExif} from '../test/jpeg';
import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';

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
  postMessage = vi.fn((_message: {bytes: ArrayBuffer; validateOnly: boolean}, _transfers: ArrayBuffer[]) => {});
  constructor() { PendingDecoder.instances.push(this); }
  terminate() { this.terminated = true; }
}
const png = new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,0,1,0,0,0,1,8,6,0,0,0,0,0,0,0]);
const tiff = new Uint8Array([73,73,42,0,8,0,0,0,2,0,0,1,3,0,1,0,0,0,1,0,0,0,1,1,3,0,1,0,0,0,1,0,0,0,0,0,0,0]);
const flush = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };

describe('local image decode boundary', () => {
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
  afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); vi.restoreAllMocks(); });

  it.each(['read', 'native', 'bundled', 'fallback'])('cancels %s and ignores late completion', async stage => {
    const controller = new AbortController();
    const outcome = decodeFile(new File(['image'], 'input'), controller.signal).catch(error => error.name);
    const reader = PendingReader.instances[0];
    if (stage !== 'read') reader.complete(stage === 'bundled' ? tiff : png);
    await flush();
    const worker = PendingDecoder.instances[0];
    if (stage === 'fallback') { worker.onmessage?.({data: {unsupportedNative: true}}); await flush(); }
    const image = PendingImage.instances[0];
    const lateMessage = worker?.onmessage, lateLoad = image?.onload;
    controller.abort();
    expect(await outcome).toBe('AbortError');
    expect(reader.onload).toBeNull();
    if (stage === 'read') expect(reader.aborted).toBe(true);
    if (worker) {
      expect(worker.terminated).toBe(true);
      expect(worker.onmessage).toBeNull();
      expect(worker.onerror).toBeNull();
      expect(worker.onmessageerror).toBeNull();
    }
    if (image) {
      expect(image.src).toBe('');
      expect(image.onload).toBeNull();
      expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:decode');
    }
    lateMessage?.({data: {pixels: new Uint8Array(4), width: 1, height: 1}});
    lateLoad?.(); await flush();
    expect(URL.createObjectURL).toHaveBeenCalledTimes(stage === 'fallback' ? 1 : 0);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('admits full decode workspace before creating the worker, including unchanged JPEG', async () => {
    let admit!: () => void;
    const reserve = vi.fn(() => new Promise<void>(resolve => { admit = resolve; }));
    const controller = new AbortController();
    const outcome = decodeFile(new File(['image'], 'input'), controller.signal, {lossy:false,admit:reserve}).catch(error => error.name);
    PendingReader.instances[0].complete(jpegWithExif(3000,4000,6)); await flush();
    expect(reserve).toHaveBeenCalledWith(384_000_000);
    expect(PendingDecoder.instances).toHaveLength(0);
    admit(); await flush();
    expect(PendingDecoder.instances).toHaveLength(1);
    controller.abort(); expect(await outcome).toBe('AbortError');
  });

  it.each(['png', 'jpeg'])('requires successful decode validation before retaining unchanged %s', async format => {
    const bytes = format === 'png' ? png.slice() : jpegWithExif(1,1,1);
    if (format === 'png') bytes[24] = 16;
    const canvas = vi.spyOn(document, 'createElement');
    let resolved = false;
    const outcome = decodeFile(new File(['image'], 'input'), undefined, {lossy:false}).then(result => { resolved = true; return result; });
    PendingReader.instances[0].complete(bytes); await flush();
    expect(resolved).toBe(false);
    const worker = PendingDecoder.instances[0];
    const [message, transfers] = worker.postMessage.mock.calls[0];
    expect(message.validateOnly).toBe(true);
    expect(transfers).toEqual([message.bytes]);
    expect(message.bytes).not.toBe(bytes.buffer);
    worker.onmessage?.({data: {width:1,height:1}});
    const result = await outcome;
    expect(result.originalFileBytes.buffer).toBe(bytes.buffer);
    expect(result.pixels).toBeUndefined();
    expect(canvas).not.toHaveBeenCalled();
    expect(worker.terminated).toBe(true);
  });

  it('returns oriented dimensions and worker-owned pixels without main-thread canvas work', async () => {
    const canvas = vi.spyOn(document, 'createElement');
    const outcome = decodeFile(new File(['image'], 'input'));
    PendingReader.instances[0].complete(jpegWithExif(3,4,6)); await flush();
    const pixels = new Uint8Array(48);
    PendingDecoder.instances[0].onmessage?.({data: {width:4,height:3,pixels}});
    expect(await outcome).toMatchObject({width:4,height:3,pixels,previewUrl:'blob:decode'});
    expect(PendingImage.instances).toHaveLength(0);
    expect(canvas).not.toHaveBeenCalled();
  });

  it.each(['error','messageerror','timeout','corrupt'])('settles worker %s without retrying the decoder on the main thread', async failure => {
    const outcome = decodeFile(new File(['image'], 'input'), undefined, {lossy:false}).catch(error => error.message);
    PendingReader.instances[0].complete(png); await flush();
    const worker = PendingDecoder.instances[0];
    if (failure === 'timeout') await vi.advanceTimersByTimeAsync(30_000);
    else if (failure === 'error') worker.onerror?.();
    else if (failure === 'messageerror') worker.onmessageerror?.();
    else worker.onmessage?.({data: {error:'Invalid or corrupt image data'}});
    expect(await outcome).toMatch(/failed|response|timed out|corrupt/);
    expect(worker.terminated).toBe(true);
    expect(PendingImage.instances).toHaveLength(0);
    expect(URL.createObjectURL).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
  });

  it('falls back only on missing worker capabilities and keeps byte validation canvas-free', async () => {
    const canvas = vi.spyOn(document, 'createElement');
    const outcome = decodeFile(new File(['image'], 'input'), undefined, {lossy:false});
    PendingReader.instances[0].complete(png); await flush();
    PendingDecoder.instances[0].onmessage?.({data: {unsupportedNative:true}}); await flush();
    PendingImage.instances[0].onload?.();
    expect((await outcome).pixels).toBeUndefined();
    expect(canvas).not.toHaveBeenCalled();
    expect(URL.revokeObjectURL).not.toHaveBeenCalled();
  });

  it.each(['success','failure'])('releases a small fallback canvas after %s and retains only successful previews', async outcome => {
    const pixels = new Uint8ClampedArray([9,8,1,2,3,4,7,6]).subarray(2,6);
    const canvas = {
      width:0,height:0,
      getContext:() => ({
        drawImage() {},
        getImageData() { if (outcome === 'failure') throw new Error('Readback failed'); return {data:pixels}; },
      }),
    };
    vi.stubGlobal('document',{createElement:vi.fn(() => canvas)});
    const pending = decodeFile(new File(['image'],'input'));
    const result = pending.then(image => image,error => error.message);
    PendingReader.instances[0].complete(png); await flush();
    PendingDecoder.instances[0].onmessage?.({data:{unsupportedNative:true}}); await flush();
    PendingImage.instances[0].onload?.();
    if (outcome === 'success') {
      const image = await pending;
      expect(image.pixels?.buffer).toBe(pixels.buffer);
      expect(image.pixels?.byteOffset).toBe(2);
      expect(URL.revokeObjectURL).not.toHaveBeenCalled();
    } else {
      expect(await result).toBe('Readback failed');
      expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:decode');
    }
    expect(canvas).toMatchObject({width:0,height:0});
    expect(PendingImage.instances[0].src).toBe('');
    expect(vi.getTimerCount()).toBe(0);
  });

  it('rejects a corrupt fallback and allows a fresh retry', async () => {
    const first = decodeFile(new File(['image'],'input'),undefined,{lossy:false}).catch(error => error.message);
    PendingReader.instances[0].complete(png); await flush();
    PendingDecoder.instances[0].onmessage?.({data:{unsupportedNative:true}}); await flush();
    PendingImage.instances[0].onerror?.();
    expect(await first).toMatch(/Invalid or unsupported/);
    expect(URL.revokeObjectURL).toHaveBeenCalledOnce();
    const retry = decodeFile(new File(['image'],'input'),undefined,{lossy:false});
    PendingReader.instances[1].complete(png); await flush();
    PendingDecoder.instances[1].onmessage?.({data:{width:1,height:1}});
    expect((await retry).originalFileBytes).toEqual(png);
    expect(PendingDecoder.instances.every(worker => worker.terminated)).toBe(true);
  });

  it('reports HEIC as temporarily unavailable before admission or worker creation, including retry', async () => {
    const bytes = new Uint8Array(await readFile(resolve('src/interop/codecs-fixtures/rainbow-451x461.heic')));
    const admit = vi.fn();
    for (let attempt=0;attempt<2;attempt++) {
      const pending = decodeFile(new File(['image'],'input.heic'),undefined,{lossy:true,admit}).catch(error => error.message);
      PendingReader.instances.at(-1)!.complete(bytes);
      expect(await pending).toMatch(/temporarily unavailable.*security.*retry/i);
    }
    expect(admit).not.toHaveBeenCalled();
    expect(PendingDecoder.instances).toHaveLength(0);
    expect(URL.createObjectURL).not.toHaveBeenCalled();
  });

  it('rejects large main-thread raster fallback before loading an Image', async () => {
    const outcome = decodeFile(new File(['image'], 'input')).catch(error => error.message);
    PendingReader.instances[0].complete(jpegWithExif(1001,1000,1)); await flush();
    expect(1001*1000).toBeGreaterThan(MAX_FALLBACK_PIXELS);
    PendingDecoder.instances[0].onmessage?.({data: {unsupportedNative:true}});
    expect(await outcome).toMatch(/worker.*support/i);
    expect(PendingImage.instances).toHaveLength(0);
  });

  it.each([{width:4,height:3,pixels:new Uint8Array(47)}, {width:3,height:4,pixels:new Uint8Array(48)}, {width:24_000_001,height:1}])('rejects invalid worker dimensions or pixels before creating a preview', async response => {
    const outcome = decodeFile(new File(['image'], 'input')).catch(error => error.message);
    PendingReader.instances[0].complete(jpegWithExif(3,4,6)); await flush();
    PendingDecoder.instances[0].onmessage?.({data: response});
    expect(await outcome).toMatch(/dimensions|pixel|limit/);
    expect(URL.createObjectURL).not.toHaveBeenCalled();
  });

  it('does not start an already aborted request', async () => {
    const controller = new AbortController(); controller.abort();
    await expect(decodeFile(new File(['image'], 'input'), controller.signal)).rejects.toMatchObject({name:'AbortError'});
    expect(PendingReader.instances).toHaveLength(0);
  });

  it('aborts a stalled reader at its deadline', async () => {
    const outcome = decodeFile(new File(['image'], 'input')).catch(error => error.message);
    await vi.advanceTimersByTimeAsync(30_000);
    expect(await outcome).toMatch(/timed out/);
    expect(PendingReader.instances[0].aborted).toBe(true);
  });
});

it('rejects animated files before a native decoder can flatten them', async () => {
  const worker = vi.fn(); vi.stubGlobal('Worker',worker);
  const frame = [44,0,0,0,0,1,0,1,0,0,2,2,68,1,0];
  const bytes = new Uint8Array([71,73,70,56,57,97,1,0,1,0,0,0,0,...frame,...frame,59]);
  await expect(decodeFile(new File([bytes],'fake.png',{type:'IMAGE/PNG'}))).rejects.toThrow(/animat/i);
  expect(worker).not.toHaveBeenCalled();
  vi.unstubAllGlobals();
});
