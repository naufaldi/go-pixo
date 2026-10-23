// @vitest-environment node
import { readFile } from 'node:fs/promises';
import { expect, it, vi } from 'vitest';
import { decodeHeic } from './codecs';

vi.mock('libheif-js/libheif-wasm/libheif-bundle.mjs?url',()=>({
  default:new URL('../../node_modules/libheif-js/libheif-wasm/libheif-bundle.mjs',import.meta.url).href,
}));

it('decodes real HEIC locally without a browser-native decoder', async () => {
  // Official libheif test image: tests/data/rainbow-451x461.heic (7080 bytes).
  const bytes=await readFile(new URL('./codecs-fixtures/rainbow-451x461.heic',import.meta.url));
  const image=await decodeHeic(bytes);
  expect([image.width,image.height,image.pixels.length]).toEqual([451,461,831644]);
  // A decoded rainbow must contain a range of channel values, not a blank raster.
  expect(new Set(image.pixels).size).toBeGreaterThan(100);
  expect(image.pixels[3]).toBe(255);
},15000);

it('rejects a multi-image HEIC before decoding any frame', async () => {
  // ftyp + meta (949 bytes) from official examples/example.heic, mdat omitted.
  // Handles are readable; pixel decoding would fail, so this proves rejection happens first.
  const bytes=await readFile(new URL('./codecs-fixtures/multi-image-header.heic',import.meta.url));
  await expect(decodeHeic(bytes)).rejects.toThrow(/multiple|single/i);
});

it('rejects high-bit-depth HEIC before the 8-bit rendering boundary', async () => {
  const bytes=new Uint8Array(await readFile(new URL('./codecs-fixtures/rainbow-451x461.heic',import.meta.url)));
  const tag=Buffer.from(bytes).indexOf('hvcC');
  bytes[tag+4+17]=250; // HEVCDecoderConfigurationRecord: bitDepthLumaMinus8 = 2
  await expect(decodeHeic(bytes)).rejects.toThrow(/8.bit/i);
});
