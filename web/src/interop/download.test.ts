import { Blob as SourceBlob } from 'node:buffer';
import { afterEach, beforeEach, describe, it, expect, vi } from 'vitest';
import { unzipSync, Unzip, UnzipPassThrough, UnzipInflate } from 'fflate';
import { buildCompressedFilename, downloadAll, downloadAllAsZip, downloadBlob } from './download';

const item = (name: string, index: number, extension = '.png') => ({
  file: { name } as File, status: 'Done', compressedUrl: `blob:${index}`, compressedExtension: extension,
});
const readBlob = (blob: Blob) => new Promise<ArrayBuffer>((resolve, reject) => {
  const reader = new FileReader();
  reader.onload = () => resolve(reader.result as ArrayBuffer);
  reader.onerror = reject;
  reader.readAsArrayBuffer(blob);
});
const deferred = <T,>() => {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(r => { resolve = r; });
  return { promise, resolve };
};
let archive: Blob | undefined;
let blobParts: BlobPart[];
let click: ReturnType<typeof vi.spyOn>;
let createUrl: ReturnType<typeof vi.fn<(blob: Blob | MediaSource) => string>>;
let revokeUrl: ReturnType<typeof vi.fn<(url: string) => void>>;
beforeEach(() => {
  archive = undefined;
  blobParts = [];
  const BrowserBlob = Blob;
  vi.stubGlobal('Blob', class extends BrowserBlob {
    constructor(parts: BlobPart[], options: BlobPropertyBag) {
      super(parts, options);
      blobParts = [...parts];
    }
  });
  createUrl = vi.fn((blob: Blob | MediaSource) => { if (blob instanceof Blob) archive = blob; return 'blob:zip'; });
  revokeUrl = vi.fn();
  vi.stubGlobal('URL', class extends URL {
    static createObjectURL = createUrl;
    static revokeObjectURL = revokeUrl;
  });
  click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
});
afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });

describe('download helpers', () => {
  it('builds compressed filenames with selected extension', () => {
    expect(buildCompressedFilename('photo.png', '.jpg')).toBe('compressed_photo.jpg');
    expect(buildCompressedFilename('image.jpeg', '.webp')).toBe('compressed_image.webp');
  });

  it('normalizes path separators before deduplicating actual archive entries', async () => {
    const bytes = [1, 2, 3, 4, 5, 6, 7].map(value => new Uint8Array([value]));
    vi.stubGlobal('fetch', vi.fn(async (url: string) => ({ ok: true, blob: async () => new SourceBlob([bytes[Number(url.slice(5))]]) })));
    await downloadAllAsZip([
      item('x\\..\\..\\escape.png', 0), item('../../escape.png', 1),
      item('escape.png', 2), item('escape (2).png', 3),
      item('C:\\folder\\photo.png', 4), item('..', 5), item('a\u0000:b?.png', 6),
    ]);
    const files = unzipSync(new Uint8Array(await readBlob(archive!)));
    expect(Object.keys(files)).toEqual([
      'compressed_escape.png', 'compressed_escape (2).png', 'compressed_escape (3).png',
      'compressed_escape (2) (2).png', 'compressed_photo.png', 'compressed_image.png', 'compressed_a__b_.png',
    ]);
    expect(Object.values(files)).toEqual(bytes);
    expect(() => buildCompressedFilename('photo.png', '.png/../../escape')).toThrow('Invalid output extension');
  });

  it('downloads completed PNG/JPEG files sequentially and settles after the final click', async () => {
    vi.useFakeTimers();
    const fetcher = vi.fn(async () => ({ ok: true, blob: async () => new Blob(['pixels']) }));
    vi.stubGlobal('fetch', fetcher);
    const names: string[] = [];
    click.mockImplementation(function (this: HTMLAnchorElement) { names.push(this.download); });
    const outcome = downloadAll([item('a.png', 0), { ...item('pending.png', 2), status: 'Pending' }, item('b.jpeg', 1, '.jpg')]);
    await vi.advanceTimersByTimeAsync(0);
    expect(fetcher).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(100);
    await outcome;
    expect(names).toEqual(['compressed_a.png', 'compressed_b.jpg']);
    expect(revokeUrl).toHaveBeenCalledTimes(2);
    expect(vi.getTimerCount()).toBe(0);
  });

  it('aborts a delayed batch before URLs are revoked, and a retry has its own lifetime', async () => {
    vi.useFakeTimers();
    const fetcher = vi.fn(async () => ({ ok: true, blob: async () => new Blob(['pixels']) }));
    vi.stubGlobal('fetch', fetcher);
    const controller = new AbortController();
    const outcome = downloadAll([item('a.png', 0), item('b.png', 1)], controller.signal).catch(error => error);
    await vi.advanceTimersByTimeAsync(0);
    expect(click).toHaveBeenCalledTimes(1);
    controller.abort();
    expect(await outcome).toMatchObject({ name: 'AbortError' });
    expect(vi.getTimerCount()).toBe(0);
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetcher).toHaveBeenCalledTimes(1);
    await downloadAll([item('retry.jpg', 2, '.jpg')], new AbortController().signal);
    expect(click).toHaveBeenCalledTimes(2);
  });

  it('aborts in-flight single downloads and observes late fetch rejections', async () => {
    let rejectFetch!: (error: Error) => void;
    vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>((_resolve, reject) => { rejectFetch = reject; })));
    const controller = new AbortController();
    const outcome = downloadBlob('blob:0', 'image.png', controller.signal).catch(error => error);
    controller.abort();
    expect(await outcome).toMatchObject({ name: 'AbortError' });
    rejectFetch(new Error('URL revoked'));
    await Promise.resolve();
    expect(createUrl).not.toHaveBeenCalled();
    expect(click).not.toHaveBeenCalled();
  });

  it('does not download late blob bytes after cancellation', async () => {
    const body = deferred<Blob>();
    const started = deferred<void>();
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: true, blob: () => { started.resolve(); return body.promise; } })));
    const controller = new AbortController();
    const outcome = downloadAll([item('a.png', 0), item('b.png', 1)], controller.signal).catch(error => error);
    await started.promise;
    controller.abort();
    expect(await outcome).toMatchObject({ name: 'AbortError' });
    body.resolve(new Blob(['late']));
    await Promise.resolve();
    expect(createUrl).not.toHaveBeenCalled();
    expect(click).not.toHaveBeenCalled();
  });

  it('rejects failed batch reads once and does not schedule peers', async () => {
    vi.useFakeTimers();
    const fetcher = vi.fn().mockRejectedValue(new Error('URL revoked'));
    vi.stubGlobal('fetch', fetcher);
    await expect(downloadAll([item('a.png', 0), item('b.png', 1)])).rejects.toThrow('URL revoked');
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(vi.getTimerCount()).toBe(0);
    expect(click).not.toHaveBeenCalled();
  });

  it('rejects nonlocal URLs and unsuccessful responses before reading bytes', async () => {
    const blob = vi.fn();
    const fetcher = vi.fn().mockResolvedValue({ ok: false, blob });
    vi.stubGlobal('fetch', fetcher);
    await expect(downloadBlob('https://example.com/image.png', 'image.png')).rejects.toThrow('local blob URL');
    await expect(downloadAllAsZip([{ ...item('a.png', 0), compressedUrl: 'https://example.com/a.png' }])).rejects.toThrow('local blob URL');
    expect(fetcher).not.toHaveBeenCalled();
    await expect(downloadAll([item('a.png', 0)])).rejects.toThrow('Could not read compressed image');
    await expect(downloadAllAsZip([item('a.png', 0)])).rejects.toThrow('Could not read compressed image');
    expect(blob).not.toHaveBeenCalled();
  });

  it('removes the single-download anchor and revokes its URL when dispatch fails', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: true, blob: async () => new Blob(['pixels']) })));
    click.mockImplementation(() => { throw new Error('Download blocked'); });
    await expect(downloadBlob('blob:0', 'image.png')).rejects.toThrow('Download blocked');
    expect(revokeUrl).toHaveBeenCalledWith('blob:zip');
    expect(document.querySelector('a[download]')).toBeNull();
  });

  it('round-trips stored entries with duplicate, pre-suffixed and mixed-format names and exact bytes', async () => {
    const bytes = [
      new Uint8Array([137, 80, 78, 71, 1]), new Uint8Array([137, 80, 78, 71, 2]),
      new Uint8Array([137, 80, 78, 71, 3]), new Uint8Array([255, 216, 255, 4]),
      new Uint8Array([82, 73, 70, 70, 5]), new Uint8Array([0, 0, 0, 32, 6]), new Uint8Array(),
    ];
    vi.stubGlobal('fetch', vi.fn(async (url: string) => ({ ok: true, blob: async () => new SourceBlob([bytes[Number(url.slice(5))]]) })));
    await downloadAllAsZip([
      item('photo.png', 0), item('photo.png', 1), item('photo (2).png', 2),
      item('photo.png', 3, '.jpg'), item('photo.png', 4, '.webp'), item('photo.png', 5, '.avif'), item('empty.png', 6),
      { ...item('pending.png', 0), status: 'Pending' },
    ]);
    const zipped = new Uint8Array(await readBlob(archive!));
    const files = unzipSync(zipped);
    expect(Object.keys(files)).toEqual([
      'compressed_photo.png', 'compressed_photo (2).png', 'compressed_photo (2) (2).png',
      'compressed_photo.jpg', 'compressed_photo.webp', 'compressed_photo.avif', 'compressed_empty.png',
    ]);
    expect(Object.values(files)).toEqual(bytes);
    const methods: number[] = [];
    const unzip = new Unzip(file => { methods.push(file.compression); file.ondata = () => {}; file.start(); });
    unzip.register(UnzipPassThrough);
    unzip.register(UnzipInflate);
    unzip.push(zipped, true);
    expect(methods).toEqual([0, 0, 0, 0, 0, 0, 0]);
    expect(blobParts.length).toBeGreaterThan(bytes.length);
    expect(archive!.type).toBe('application/zip');
    expect(click).toHaveBeenCalledTimes(1);
    expect(revokeUrl).toHaveBeenCalledWith('blob:zip');
  });

  it('reads one bounded slice at a time and finishes a file before fetching the next', async () => {
    const gate = deferred<void>();
    const firstRead = deferred<void>();
    const sources = [new SourceBlob([new Uint8Array(200_000).fill(7)]), new SourceBlob([new Uint8Array(80_000).fill(9)])];
    const ranges: number[] = [];
    let active = 0;
    let maxActive = 0;
    let reads = 0;
    let releases = 0;
    for (const source of sources) {
      vi.spyOn(source, 'arrayBuffer').mockRejectedValue(new Error('Full-file reads are forbidden'));
      const originalSlice = source.slice.bind(source);
      vi.spyOn(source, 'slice').mockImplementation((start = 0, end = source.size) => {
        ranges.push(end - start);
        const part = originalSlice(start, end);
        const stream = part.stream.bind(part);
        vi.spyOn(part, 'stream').mockImplementation(() => {
          const reader = stream().getReader();
          active++;
          maxActive = Math.max(maxActive, active);
          return { getReader: () => ({
            read: async () => {
              if (++reads === 1) { firstRead.resolve(); await gate.promise; }
              return reader.read();
            },
            cancel: () => reader.cancel(),
            releaseLock: () => { active--; releases++; reader.releaseLock(); },
          }) } as ReturnType<typeof stream>;
        });
        return part;
      });
    }
    const fetcher = vi.fn(async (url: string) => ({ ok: true, blob: async () => sources[Number(url.slice(5))] }));
    vi.stubGlobal('fetch', fetcher);
    const outcome = downloadAllAsZip([item('a.png', 0), item('b.png', 1)]).then(() => undefined, error => error);
    await Promise.race([firstRead.promise, outcome]);
    expect(reads).toBe(1);
    expect(fetcher).toHaveBeenCalledTimes(1);
    gate.resolve();
    expect(await outcome).toBeUndefined();
    expect(maxActive).toBe(1);
    expect(active).toBe(0);
    expect(Math.max(...ranges)).toBeLessThanOrEqual(65_536);
    expect(releases).toBe(ranges.length);
    const files = unzipSync(new Uint8Array(await readBlob(archive!)));
    expect(files['compressed_a.png']).toEqual(new Uint8Array(200_000).fill(7));
    expect(files['compressed_b.png']).toEqual(new Uint8Array(80_000).fill(9));
  });

  it('cancels and releases a failed reader without fetching peers or downloading a partial archive', async () => {
    const reader = { read: vi.fn().mockResolvedValueOnce({ done: false, value: new Uint8Array([1, 2]) }).mockRejectedValue(new Error('Read failed')), cancel: vi.fn().mockResolvedValue(undefined), releaseLock: vi.fn() };
    const fetcher = vi.fn().mockResolvedValue({ ok: true, blob: async () => ({ size: 10, slice: () => ({ stream: () => ({ getReader: () => reader }) }) }) });
    vi.stubGlobal('fetch', fetcher);
    await expect(downloadAllAsZip([item('a.png', 0), item('b.png', 1)])).rejects.toThrow('Read failed');
    expect(reader.cancel).toHaveBeenCalledTimes(1);
    expect(reader.releaseLock).toHaveBeenCalledTimes(1);
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(createUrl).not.toHaveBeenCalled();
    expect(click).not.toHaveBeenCalled();
  });

  it('aborts a pending read promptly even when cancellation never settles, and ignores late bytes', async () => {
    const pendingRead = deferred<ReadableStreamReadResult<Uint8Array>>();
    const started = deferred<void>();
    const reader = {
      read: vi.fn(() => { started.resolve(); return pendingRead.promise; }),
      cancel: vi.fn(() => new Promise(() => {})), releaseLock: vi.fn(),
    };
    const fetcher = vi.fn().mockResolvedValue({ ok: true, blob: async () => ({ size: 10, slice: () => ({ stream: () => ({ getReader: () => reader }) }) }) });
    vi.stubGlobal('fetch', fetcher);
    const controller = new AbortController();
    const outcome = downloadAllAsZip([item('a.png', 0), item('b.png', 1)], controller.signal).catch(error => error);
    await Promise.race([started.promise, outcome]);
    expect(reader.read).toHaveBeenCalledTimes(1);
    controller.abort();
    expect(await Promise.race([outcome, new Promise(resolve => setTimeout(() => resolve('still pending'), 0))])).toMatchObject({ name: 'AbortError' });
    expect(reader.cancel).toHaveBeenCalledTimes(1);
    expect(reader.releaseLock).toHaveBeenCalledTimes(1);
    pendingRead.resolve({ done: false, value: new Uint8Array([1, 2]) });
    await Promise.resolve();
    expect(reader.read).toHaveBeenCalledTimes(1);
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(createUrl).not.toHaveBeenCalled();
    expect(click).not.toHaveBeenCalled();
  });

  it('aborts during fetch and prevents a late response from starting reads', async () => {
    const response = deferred<Response>();
    const blob = vi.fn();
    const fetcher = vi.fn(() => response.promise);
    vi.stubGlobal('fetch', fetcher);
    const controller = new AbortController();
    const outcome = downloadAllAsZip([item('a.png', 0)], controller.signal).catch(error => error);
    controller.abort();
    expect(await Promise.race([outcome, new Promise(resolve => setTimeout(() => resolve('still pending'), 0))])).toMatchObject({ name: 'AbortError' });
    expect(fetcher).toHaveBeenCalledWith('blob:0', { signal: controller.signal });
    response.resolve({ ok: true, blob } as unknown as Response);
    await Promise.resolve();
    await Promise.resolve();
    expect(blob).not.toHaveBeenCalled();
    expect(click).not.toHaveBeenCalled();
  });

  it('does no work for an already aborted signal', async () => {
    const fetcher = vi.fn();
    vi.stubGlobal('fetch', fetcher);
    const controller = new AbortController();
    controller.abort();
    await expect(downloadAllAsZip([item('a.png', 0)], controller.signal)).rejects.toMatchObject({ name: 'AbortError' });
    expect(fetcher).not.toHaveBeenCalled();
    expect(createUrl).not.toHaveBeenCalled();
  });

  it('rejects batches beyond classic ZIP entry or size limits without truncating or downloading', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, blob: async () => ({ size: 0x1_0000_0000 }) });
    vi.stubGlobal('fetch', fetcher);
    await expect(downloadAllAsZip(Array.from({ length: 65_536 }, (_, index) => item('a.png', index)))).rejects.toThrow(/ZIP.*limit/i);
    expect(fetcher).not.toHaveBeenCalled();
    await expect(downloadAllAsZip([item('a.png', 0)])).rejects.toThrow(/ZIP.*limit/i);
    expect(createUrl).not.toHaveBeenCalled();
    expect(click).not.toHaveBeenCalled();
  });

  it('revokes the archive URL and removes the anchor when download dispatch fails', async () => {
    vi.stubGlobal('fetch', async () => ({ ok: true, blob: async () => new SourceBlob([new Uint8Array([1])]) }));
    click.mockImplementation(() => { throw new Error('Download blocked'); });
    await expect(downloadAllAsZip([item('a.png', 0)])).rejects.toThrow('Download blocked');
    expect(revokeUrl).toHaveBeenCalledWith('blob:zip');
    expect(document.querySelector('a[download]')).toBeNull();
  });
});
