// @vitest-environment node
import { readFile } from 'node:fs/promises';
import { beforeAll, expect, it } from 'vitest';
import { compositeRgbaToRgbWhite } from './pixels';
import { encodeJpegRgba, type JpegEncodeOptions } from './jpegParallel';
import { unwrapWasmResult } from './wasmResult';

type GoExports = {
  Go: new () => { importObject: WebAssembly.Imports; run(instance: WebAssembly.Instance): Promise<void> };
  goWasmInit?: () => void;
  encodeJpegAdvanced(...args: unknown[]): Uint8Array | string;
  encodeJpegBand(...args: unknown[]): Uint8Array | string;
  encodeJpegBands(...args: unknown[]): Uint8Array | string;
};
const go = globalThis as unknown as GoExports;

// The real main.wasm, as the compression and band workers load it.
beforeAll(async () => {
  // eslint-disable-next-line no-eval
  (0, eval)(await readFile(new URL('../../public/wasm_exec.js', import.meta.url), 'utf8'));
  const runtime = new go.Go();
  const ready = new Promise<void>(resolve => { go.goWasmInit = resolve; });
  const { instance } = await WebAssembly.instantiate(await readFile(new URL('../../public/main.wasm', import.meta.url)), runtime.importObject);
  void runtime.run(instance);
  await ready;
});

// Partial edge MCUs, translucent pixels and an uneven band split.
function photo(width: number, height: number): Uint8Array {
  const rgba = new Uint8Array(width * height * 4);
  for (let y = 0; y < height; y++) for (let x = 0; x < width; x++) {
    const i = (y * width + x) * 4;
    rgba[i] = (x * 7 + y * 3 + ((x * y) % 13)) & 255;
    rgba[i + 1] = (128 + 100 * Math.sin(x / 9) * Math.cos(y / 7)) & 255;
    rgba[i + 2] = (x ^ y) & 255;
    rgba[i + 3] = x < 20 && y < 20 ? x * 12 : 255;
  }
  return rgba;
}

it.each([
  ['4:2:0 trellis progressive', { quality: 85, subsampling: 0, progressive: true, trellis: true, optimizeHuffman: true, preset: 1 }],
  ['4:4:4 baseline', { quality: 88, subsampling: 1, progressive: false, trellis: false, optimizeHuffman: false, preset: 2 }],
] as [string, JpegEncodeOptions][])('band workers produce the serial encoder bytes (%s)', async (_name, options) => {
  const width = 301, height = 203;
  const rgba = photo(width, height);
  const args = [width, height, 3, options.quality, options.subsampling, options.progressive, options.trellis,
    options.optimizeHuffman, options.preset];
  const serial = unwrapWasmResult(go.encodeJpegAdvanced(compositeRgbaToRgbWhite(rgba, width, height), ...args));
  const bands: number[] = [];
  const parallel = await encodeJpegRgba(rgba, width, height, options.subsampling, {
    bands: 4,
    serial: () => { throw new Error('unexpected serial fallback'); },
    band: (rgb, band) => { bands.push(band.rows); return unwrapWasmResult(go.encodeJpegBand(rgb, ...args, band.firstRow, band.rows)); },
    // As jpegBand.worker.ts does with the RGBA rows it receives.
    remote: async (_index, slice, band) => {
      bands.push(band.rows);
      return unwrapWasmResult(go.encodeJpegBand(compositeRgbaToRgbWhite(slice, width, band.height), ...args, band.firstRow, band.rows));
    },
    assemble: coefficients => unwrapWasmResult(go.encodeJpegBands(coefficients, ...args)),
    release: () => {},
  });
  expect(bands).toHaveLength(4);
  expect(bands.reduce((a, b) => a + b)).toBe(Math.ceil(height / (options.subsampling === 0 ? 16 : 8)));
  expect(Buffer.from(parallel).equals(Buffer.from(serial))).toBe(true);
});
