// @vitest-environment node
import { readFile } from 'node:fs/promises';
import { afterEach, expect, it, vi } from 'vitest';
import { decodeAvif, decodeTiff, encodeAvif } from './codecs';
import { inspectImage, sniffInputFormat } from './imageFormats';

const tiffBoundary=vi.hoisted(()=>({original:undefined as typeof import('utif')['decodeImage'] | undefined}));
vi.mock('utif',async importOriginal=>{
  const actual=await importOriginal<typeof import('utif')>();
  const exports=(actual as {default?: typeof actual}).default ?? actual;
  tiffBoundary.original=exports.decodeImage;
  return {...actual,...exports,decodeImage:vi.fn(exports.decodeImage)};
});

afterEach(async () => {
  vi.unstubAllGlobals();
  const UTIF=await import('utif');
  vi.mocked(UTIF.decodeImage).mockReset().mockImplementation(tiffBoundary.original!);
});

it('encodes real AVIF bytes and losslessly decodes dimensions and alpha locally', async () => {
  // Vite turns ?url into /node_modules/ URLs. Feed those exact bundled assets to WASM.
  vi.stubGlobal('fetch', async (url: string) => new Response(await readFile(new URL(`../..${url}`, import.meta.url))));
  const pixels=new Uint8Array([255,0,0,255,0,255,0,128,0,0,255,255,255,255,255,255]);
  const encoded=await encodeAvif(pixels,2,2,100,true,8);
  expect(sniffInputFormat(encoded)).toBe('avif');
  expect(inspectImage(encoded)).toMatchObject({format:'avif',width:2,height:2});
  const decoded=await decodeAvif(encoded);
  expect([decoded.width,decoded.height]).toEqual([2,2]);
  expect(Array.from(decoded.pixels)).toEqual(Array.from(pixels));
},15000);

it('decodes a real single-page TIFF and rejects multi-page TIFF before rasterization', async () => {
  const UTIF=await import('utif');
  const pixels=new Uint8Array([255,0,0,255,0,255,0,255]);
  const bytes=new Uint8Array(UTIF.encodeImage(pixels.buffer,2,1));
  const decoded=await decodeTiff(bytes);
  expect([decoded.width,decoded.height]).toEqual([2,1]);
  expect(Array.from(decoded.pixels)).toEqual(Array.from(pixels));
  const multi=new Uint8Array(UTIF.encode([{t256:[1],t257:[1]},{t256:[1],t257:[1]}]));
  await expect(decodeTiff(multi)).rejects.toThrow(/page|multiple/i);
});

function makeTiff(tags: Record<string, number[]>, pixels = [255,0,0,0,255,0]): Uint8Array {
  const entries=Object.entries({
    t256:[2],t257:[1],t258:[8,8,8],t259:[1],t262:[2],t273:[1000],
    t277:[3],t278:[1],t279:[pixels.length],t284:[1],...tags,
  }).sort(([a],[b])=>Number(a.slice(1))-Number(b.slice(1)));
  const bytes=new Uint8Array(1000+pixels.length);
  bytes.set([73,73,42,0,8,0,0,0]);
  const view=new DataView(bytes.buffer);
  view.setUint16(8,entries.length,true);
  let offset=10+entries.length*12+4;
  entries.forEach(([tag,values],index)=>{
    const p=10+index*12;
    view.setUint16(p,Number(tag.slice(1)),true); view.setUint16(p+2,4,true); view.setUint32(p+4,values.length,true);
    if (values.length<=1) view.setUint32(p+8,values[0] ?? 0,true);
    else {
      view.setUint32(p+8,offset,true);
      values.forEach(value=>{ view.setUint32(offset,value,true); offset+=4; });
    }
  });
  bytes.set(pixels,1000);
  return bytes;
}

it('rejects separate planar TIFF channels before UTIF rasterization', async () => {
  const bytes=makeTiff({t284:[2],t273:[1000,1002,1004],t279:[2,2,2]},[255,0,0,255,0,0]);
  const UTIF=await import('utif');
  const rasterize=vi.mocked(UTIF.decodeImage).mockImplementation(()=>{ throw new Error('Unsafe rasterization reached'); });
  await expect(decodeTiff(bytes)).rejects.toThrow(/planar|interleaved/i);
  expect(rasterize).not.toHaveBeenCalled();
});

const tiledDeclarations: Record<string, number[]>[]=[
  {t322:[16384],t323:[16384],t324:[1000],t325:[1]},
  {t323:[16384]},
  {t324:[1000]},
  {t325:[1]},
];
it.each(tiledDeclarations)('rejects any TIFF tile declaration before allocating a tile buffer (%j)', async tags => {
  // Tiny encoded file, bounded 16x16 image, but a declared 805 MB tile.
  const bytes=makeTiff({t256:[16],t257:[16],...tags},[0]);
  const UTIF=await import('utif');
  const rasterize=vi.mocked(UTIF.decodeImage).mockImplementation(()=>{ throw new Error('Unsafe rasterization reached'); });
  await expect(decodeTiff(bytes)).rejects.toThrow(/tile/i);
  expect(rasterize).not.toHaveBeenCalled();
});

const invalidSampleDeclarations: Record<string, number[]>[]=[
  {t277:[0]},
  {t277:[65535]},
  {t277:[3,3]},
  {t258:[]},
  {t258:[0,0,0]},
  {t258:[8,4,8]},
  {t258:Array(128).fill(8)},
  {t339:[1,1]},
];
it.each(invalidSampleDeclarations)('rejects invalid TIFF sample allocation declarations before rasterization (%j)', async tags => {
  const bytes=makeTiff(tags);
  const UTIF=await import('utif');
  const rasterize=vi.mocked(UTIF.decodeImage).mockImplementation(()=>{ throw new Error('Unsafe rasterization reached'); });
  await expect(decodeTiff(bytes)).rejects.toThrow(/sample|channel|bit/i);
  expect(rasterize).not.toHaveBeenCalled();
});

it('retains ordinary interleaved RGB TIFF pixels', async () => {
  const image=await decodeTiff(makeTiff({}));
  expect([image.width,image.height]).toEqual([2,1]);
  expect(Array.from(image.pixels)).toEqual([255,0,0,255,0,255,0,255]);
});
