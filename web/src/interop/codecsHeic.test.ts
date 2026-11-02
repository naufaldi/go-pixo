// @vitest-environment node
import { readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { afterEach, expect, it, vi } from 'vitest';
import { decodeHeic, isPatchedLibheif } from './codecs';

const libheifUrl = vi.hoisted(() => new URL('../../node_modules/heic-to/src/lib/libheif-without-unsafe-eval.js', import.meta.url).href);
// Node imports the module from disk; Vite serves the same file as a URL asset.
vi.mock('../../node_modules/heic-to/src/lib/libheif-without-unsafe-eval.js?url', () => ({ default: libheifUrl }));

// The Emscripten build takes its Node branch under Vitest and expects CommonJS
// globals; browsers and workers never reach that branch.
Object.assign(globalThis, { require: createRequire(import.meta.url), __dirname: process.cwd() });
afterEach(() => vi.unstubAllGlobals());

const fixture = (name: string) => readFile(new URL(`./codecs-fixtures/${name}`, import.meta.url));

it.each([
  ['1.23.7', true], ['1.23.8', true], ['1.24.0', true], ['2.0.0', true],
  ['1.23.6', false], ['1.23.5', false], ['1.22.9', false], ['0.99.99', false],
  ['1.23', false], ['', false], ['1.23.x', false],
])('treats libheif %s as patched=%s', (version, patched) => {
  expect(isPatchedLibheif(version)).toBe(patched);
});

it('bundles a patched libheif build', async () => {
  const module = await import(/* @vite-ignore */ libheifUrl);
  const lib = await module.default();
  expect(isPatchedLibheif(lib.heif_get_version())).toBe(true);
}, 30000);

it('decodes real HEIC locally when the browser has no native HEIC decoder', async () => {
  // Official libheif test image: tests/data/rainbow-451x461.heic (7080 bytes).
  const image = await decodeHeic(await fixture('rainbow-451x461.heic'));
  expect([image.width, image.height, image.pixels.length]).toEqual([451, 461, 831644]);
  // A decoded rainbow must contain a range of channel values, not a blank raster.
  expect(new Set(image.pixels).size).toBeGreaterThan(100);
  expect(image.pixels[3]).toBe(255);
}, 30000);

it('falls back to libheif when the native decoder rejects HEIC', async () => {
  const createImageBitmap = vi.fn(async () => { throw new DOMException('unsupported', 'InvalidStateError'); });
  vi.stubGlobal('createImageBitmap', createImageBitmap);
  vi.stubGlobal('OffscreenCanvas', class {});
  const image = await decodeHeic(await fixture('rainbow-451x461.heic'));
  expect(createImageBitmap).toHaveBeenCalledOnce();
  expect([image.width, image.height]).toEqual([451, 461]);
}, 30000);

it('uses the native HEIC decoder when available, including irot-transposed output', async () => {
  const pixels = new Uint8ClampedArray(461 * 451 * 4).fill(7);
  vi.stubGlobal('createImageBitmap', vi.fn(async () => ({ width: 461, height: 451, close: vi.fn() })));
  vi.stubGlobal('OffscreenCanvas', class {
    width: number; height: number;
    constructor(width: number, height: number) { this.width = width; this.height = height; }
    getContext() { return { drawImage: vi.fn(), getImageData: () => ({ data: pixels }) }; }
  });
  const image = await decodeHeic(await fixture('rainbow-451x461.heic'));
  expect([image.width, image.height]).toEqual([461, 451]);
  expect(image.pixels.every(value => value === 7)).toBe(true);
});

it('rejects a multi-image header without pixel data instead of crashing', async () => {
  // ftyp + meta (949 bytes) from official examples/example.heic, mdat omitted.
  await expect(decodeHeic(await fixture('multi-image-header.heic'))).rejects.toThrow();
}, 30000);

it('rejects high-bit-depth HEIC before the 8-bit rendering boundary', async () => {
  const bytes = new Uint8Array(await fixture('rainbow-451x461.heic'));
  const tag = Buffer.from(bytes).indexOf('hvcC');
  bytes[tag + 4 + 17] = 250; // HEVCDecoderConfigurationRecord: bitDepthLumaMinus8 = 2
  await expect(decodeHeic(bytes)).rejects.toThrow(/8.bit/i);
});

it('rejects malformed HEIC before loading the decoder', async () => {
  await expect(decodeHeic(new Uint8Array())).rejects.toThrow(/empty|invalid|corrupt/i);
});
