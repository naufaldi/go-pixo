import {afterEach, beforeEach, expect, it, vi} from 'vitest';
import {jpegWithExif} from './test/jpeg';
const codecs = vi.hoisted(() => ({encodeWebp: vi.fn(), encodeAvif: vi.fn()}));
vi.mock('./interop/codecs', () => codecs);
const validation = vi.hoisted(() => vi.fn());
const samePixels = vi.hoisted(() => vi.fn());
vi.mock('./interop/codecsNative', () => ({decodeNative: validation, assertSameDecodedPixels: samePixels}));
beforeEach(() => {
  validation.mockReset().mockResolvedValue(undefined);
  samePixels.mockReset().mockResolvedValue(undefined);
  vi.stubGlobal('createImageBitmap',vi.fn());
});
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks() });
const png = new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,0,1,0,0,0,1,8,6,0,0,0,0,0,0,0]);
const jpeg = new Uint8Array([255,216,255,192,0,7,8,0,1,0,1]);
const webp = new Uint8Array([82,73,70,70,18,0,0,0,87,69,66,80,86,80,56,76,5,0,0,0,47,0,0,0,0,0]);
async function worker() {
  vi.resetModules();
  const messages: any[] = [];
  const scope: any = {
    postMessage: (message: unknown, options: StructuredSerializeOptions = {}) => messages.push(structuredClone(message, options)),
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
it.each([['jpeg', jpeg], ['png', new Uint8Array([...png.slice(0,24),16,...png.slice(25)])]] as const)('does not report metadata-only %s as a successful original candidate', async (format, bytes) => {
  validation.mockRejectedValueOnce(new Error('Invalid or corrupt image data'));
  const {scope, fetch, messages, send} = await worker();
  await send(format,bytes);
  expect(messages.some(message => message.type === 'compressed')).toBe(false);
  expect(messages.find(message => message.type === 'error')?.error).toMatch(/corrupt/);
  expect(scope.Go).not.toHaveBeenCalled();
  expect(scope.recompressPngLossless).not.toHaveBeenCalled();
  expect(fetch).not.toHaveBeenCalled();
});
it('returns exact original webp bytes without pixels, canvas, or Go', async () => {
  const canvas = vi.fn(() => { throw new Error('Unexpected rasterization') });
  vi.stubGlobal('OffscreenCanvas', canvas);
  const {scope, fetch, messages, send} = await worker();
  await send('webp', webp);
  const output = messages.find(message => message.type === 'compressed');
  expect(output?.outputFormat).toBe('webp');
  expect(Array.from(output.compressedBytes)).toEqual(Array.from(webp));
  expect(canvas).not.toHaveBeenCalled();
  expect(fetch).not.toHaveBeenCalled();
  expect(scope.Go).not.toHaveBeenCalled();
});
it('optimizes JPEG original bytes losslessly in Go and checks decoded pixels', async () => {
  const canvas = vi.fn(() => { throw new Error('Unexpected rasterization') });
  vi.stubGlobal('OffscreenCanvas', canvas);
  const withComment = new Uint8Array([255,216,255,254,0,10,...new Uint8Array(8),...jpeg.slice(2)]);
  const {scope, messages, send} = await worker();
  scope.optimizeJpegLossless = vi.fn(() => new Uint8Array(jpeg));
  // The selected output is transferred (detached) afterwards, so record sizes at call time.
  const compared: number[][] = [];
  samePixels.mockImplementation(async (expected: Uint8Array, actual: Uint8Array) => { compared.push([expected.length, actual.length]); });
  await send('jpeg', withComment);
  const output = messages.find(message => message.type === 'compressed');
  expect(output?.outputFormat).toBe('jpeg');
  expect(Array.from(output.compressedBytes)).toEqual(Array.from(jpeg));
  expect(Array.from(scope.optimizeJpegLossless.mock.calls[0][0])).toEqual(Array.from(withComment));
  expect(compared).toEqual([[withComment.length, jpeg.length]]);
  expect(messages.some(message => message.type === 'progress' && message.phase === 'optimizing-jpeg')).toBe(true);
  expect(canvas).not.toHaveBeenCalled();
});
it.each([[2, false], [1, true]])('lossy preset %i tries lossless JPEG recoding: %s', async (preset, tried) => {
  const comment = (n: number) => new Uint8Array([255,216,255,254,0,n+2,...new Uint8Array(n),...jpeg.slice(2)]);
  const {scope, messages, send} = await worker();
  // The lossy result is barely smaller than the original, so only the preset decides.
  scope.encodeJpegAdvanced = vi.fn(() => comment(7));
  scope.optimizeJpegLossless = vi.fn(() => new Uint8Array(jpeg));
  await send('jpeg', comment(8), {lossy: true, preset, pixels: new Uint8Array([1,2,3,255])});
  expect(messages.find(message => message.type === 'compressed')).toBeDefined();
  expect(scope.optimizeJpegLossless).toHaveBeenCalledTimes(tried ? 1 : 0);
});
it('keeps exact original JPEG bytes when the Go optimizer is unavailable', async () => {
  const {messages, send} = await worker();
  await send('jpeg', jpeg);
  const output = messages.find(message => message.type === 'compressed');
  expect(Array.from(output.compressedBytes)).toEqual(Array.from(jpeg));
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

it('validates JPEG byte-only requests against display dimensions without canvas', async () => {
  const bytes=jpegWithExif(3000,4000,6);
  const canvas = vi.fn(() => { throw new Error('Unexpected rasterization') });
  vi.stubGlobal('OffscreenCanvas', canvas);
  const {messages,send}=await worker();
  await send('jpeg',bytes,{width:4000,height:3000,targetWidth:4000});
  const output=messages.find(message => message.type==='compressed');
  expect(Array.from(output.compressedBytes)).toEqual(Array.from(bytes));
  expect(canvas).not.toHaveBeenCalled();
});
it('rejects storage dimensions masquerading as display dimensions on a JPEG byte request', async () => {
  const {messages,send}=await worker();
  await send('jpeg',jpegWithExif(3000,4000,6),{width:3000,height:4000,targetWidth:3000});
  expect(messages.find(message => message.type==='error')?.error).toMatch(/pixel/i);
});

it('forwards actual PNG phase units and leaves opaque stages indeterminate until the result', async () => {
  const {scope, messages, send} = await worker();
  scope.recompressPngLossless.mockImplementation((bytes: Uint8Array, _preset: number, _iterations: number, report: (phase: string, value: number) => void) => {
    report('histogram', 0); report('histogram', 25); report('histogram', 100);
    report('palette-sort', 0); report('quantize', 0); report('quantize', 50); report('quantize', 100);
    report('deflate', 0); vi.spyOn(performance, 'now').mockReturnValueOnce(performance.now() + 100); report('deflate', 50); report('deflate', 100);
    report('finalize', 100);
    return new Uint8Array(bytes);
  });
  await send('png', png);
  const updates = messages.filter(message => message.type === 'progress');
  expect(updates.find(message => message.phase === 'histogram' && message.phasePercent === 100)).toBeDefined();
  expect(updates.find(message => message.phase === 'quantize' && message.phasePercent === 100)).toBeDefined();
  expect(updates.find(message => message.phase === 'deflate' && message.phasePercent === 50)).toBeDefined();
  for (const update of updates.filter(message => ['initializing','palette-sort','finalize','finalizing'].includes(message.phase))) {
    expect(update.phasePercent).toBeUndefined();
    expect(update.progress).toBeLessThan(100);
    expect(update.predictable).not.toBe(true);
  }
  expect(messages.at(-1).type).toBe('compressed');
});
