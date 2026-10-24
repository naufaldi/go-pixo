import {afterEach, expect, it, vi} from 'vitest';
const codecs = vi.hoisted(() => ({encodeWebp: vi.fn(), encodeAvif: vi.fn()}));
vi.mock('./interop/codecs', () => codecs);
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks() });
const png = new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,0,1,0,0,0,1,8,6,0,0,0,0,0,0,0]);
const jpeg = new Uint8Array([255,216,255,192,0,7,8,0,1,0,1]);
const webp = new Uint8Array([82,73,70,70,18,0,0,0,87,69,66,80,86,80,56,76,5,0,0,0,47,0,0,0,0,0]);
async function worker() {
  vi.resetModules();
  const messages: any[] = [];
  const scope: any = {
    postMessage: (message: unknown, transfer: Transferable[] = []) => messages.push(structuredClone(message, {transfer})),
    Go: vi.fn(function(this: any) { this.importObject = {}; this.run = () => scope.goWasmInit() }),
    recompressPngLossless: vi.fn((bytes: Uint8Array) => new Uint8Array(bytes)),
  };
  const fetch = vi.fn(async () => ({ok: true, arrayBuffer: async () => new ArrayBuffer(0)}));
  vi.stubGlobal('self', scope); vi.stubGlobal('fetch', fetch);
  vi.spyOn(WebAssembly, 'instantiate').mockResolvedValue({instance: {}} as any);
  await import('./worker');
  await scope.onmessage({data: {type: 'init'}});
  const send = (format: string, bytes?: Uint8Array, extra = {}) => scope.onmessage({data: {
    type: 'compress', id: format, width: 1, height: 1, colorType: 6,
    format, outputFormat: 'same', preset: 2, lossy: false,
    originalFileBytes: bytes && new Uint8Array(bytes), ...extra,
  }});
  return {scope, fetch, messages, send};
}
it('announces transport readiness without initializing Go or any codec', async () => {
  const {scope, fetch, messages} = await worker();
  expect(messages).toEqual([{type: 'ready'}]);
  expect(fetch).not.toHaveBeenCalled();
  expect(scope.Go).not.toHaveBeenCalled();
});
it.each([['jpeg', jpeg], ['webp', webp]] as const)('returns exact original %s bytes without pixels, canvas, or Go', async (format, bytes) => {
  const canvas = vi.fn(() => { throw new Error('Unexpected rasterization') });
  vi.stubGlobal('OffscreenCanvas', canvas);
  const {scope, fetch, messages, send} = await worker();
  await send(format, bytes);
  const output = messages.find(message => message.type === 'compressed');
  expect(output?.outputFormat).toBe(format);
  expect(Array.from(output.compressedBytes)).toEqual(Array.from(bytes));
  expect(canvas).not.toHaveBeenCalled();
  expect(fetch).not.toHaveBeenCalled();
  expect(scope.Go).not.toHaveBeenCalled();
});
it('recompresses PNG original bytes without frontend RGBA and reuses one lazy Go initialization', async () => {
  const canvas = vi.fn(() => { throw new Error('Unexpected rasterization') });
  vi.stubGlobal('OffscreenCanvas', canvas);
  const {scope, fetch, messages, send} = await worker();
  await send('png', png); await send('png', png);
  const output = messages.filter(message => message.type === 'compressed');
  expect(output).toHaveLength(2);
  expect(Array.from(output[0].compressedBytes)).toEqual(Array.from(png));
  expect(scope.recompressPngLossless).toHaveBeenCalledTimes(2);
  expect(scope.Go).toHaveBeenCalledTimes(1);
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(canvas).not.toHaveBeenCalled();
});
it('initializes only the native encoder for a lossy WebP request', async () => {
  codecs.encodeWebp.mockResolvedValue(new Uint8Array(webp));
  const {scope, fetch, messages, send} = await worker();
  await send('png', undefined, {lossy: true, outputFormat: 'webp', pixels: new Uint8Array([1,2,3,4])});
  expect(messages.find(message => message.type === 'compressed')?.outputFormat).toBe('webp');
  expect(scope.Go).not.toHaveBeenCalled();
  expect(fetch).not.toHaveBeenCalled();
});
it('rejects missing pixels for a conversion instead of silently returning the original', async () => {
  const {messages, send} = await worker();
  await send('jpeg', jpeg, {outputFormat: 'png'});
  expect(messages.find(message => message.type === 'error')?.error).toMatch(/pixel/i);
});

it('reports failed lazy Go startup as an isolated initialization failure', async () => {
  const {fetch, messages, send} = await worker();
  fetch.mockRejectedValueOnce(new Error('Go unavailable'));
  vi.spyOn(console, 'error').mockImplementation(() => {});
  await send('png', png);
  const failure = messages.find(message => message.type === 'error');
  expect(failure).toMatchObject({error: 'Go unavailable'});
  expect(failure.id).toBeUndefined();
});
it.each(['success', 'failure'])('resize preserves pixel offsets and releases both canvases after %s', async outcome => {
  const source = new Uint8Array([9,1,2,3,4,5,6,7,8,9]).subarray(1,9);
  const resized = new Uint8ClampedArray([9,8,11,12,13,14,7,6]).subarray(2,6);
  let sourceView: Uint8ClampedArray | undefined;
  const canvases: any[] = [];
  vi.stubGlobal('ImageData', class { constructor(data: Uint8ClampedArray) { sourceView = data } });
  vi.stubGlobal('OffscreenCanvas', class {
    width: number; height: number;
    constructor(width: number, height: number) { this.width = width; this.height = height; canvases.push(this) }
    getContext() { return {
      putImageData() {},
      drawImage() { if (outcome === 'failure') throw new Error('Resize failed') },
      getImageData: () => ({data: resized}),
    } }
  });
  const {messages, send} = await worker();
  codecs.encodeWebp.mockImplementationOnce(async pixels => {
    expect(pixels.buffer).toBe(resized.buffer);
    expect(pixels.byteOffset).toBe(2);
    expect(Array.from(pixels)).toEqual([11,12,13,14]);
    return new Uint8Array(webp);
  });
  await send('png', undefined, {lossy: true, outputFormat: 'webp', pixels: source, width: 2, targetWidth: 1});
  expect(sourceView!.buffer).toBe(source.buffer);
  expect(sourceView!.byteOffset).toBe(1);
  expect(Array.from(sourceView!)).toEqual([1,2,3,4,5,6,7,8]);
  expect(canvases.map(canvas => [canvas.width,canvas.height])).toEqual([[0,0],[0,0]]);
  expect(messages.some(message => message.type === (outcome === 'success' ? 'compressed' : 'error'))).toBe(true);
});

it('keeps the 30 second Go initialization deadline after transport readiness', async () => {
  vi.useFakeTimers();
  try {
    const {fetch, messages, send} = await worker();
    fetch.mockImplementationOnce(() => new Promise(() => {}));
    const pending = send('png', png);
    await vi.advanceTimersByTimeAsync(30_000);
    await pending;
    expect(messages.find(message => message.type === 'error')).toMatchObject({error: expect.stringMatching(/timed out/)});
    expect(vi.getTimerCount()).toBe(0);
  } finally { vi.useRealTimers() }
});
