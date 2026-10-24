// @vitest-environment node
import { readFile } from 'node:fs/promises';
import { deflateSync } from 'node:zlib';
import { Inflate } from 'pako';
import { afterEach, expect, it, vi } from 'vitest';
import { decodeAvif, decodeTiff, encodeAvif } from './codecs';
import { inspectImage, sniffInputFormat } from './imageFormats';

const tiffBoundary=vi.hoisted(()=>({original:undefined as typeof import('utif')['decodeImage'] | undefined}));
vi.mock('utif',async importOriginal=>{
  const actual=await importOriginal<typeof import('utif')>();
  const exports=(actual as {default?: typeof actual}).default ?? actual;
  tiffBoundary.original=exports.decodeImage;
  return {...actual,...exports,decode:vi.fn(exports.decode),decodeImage:vi.fn(exports.decodeImage)};
});

afterEach(async () => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  const UTIF=await import('utif');
  vi.mocked(UTIF.decode).mockClear();
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
  const pixelOffset=tags.t273?.[0] ?? 1000;
  const bytes=new Uint8Array(Math.max(pixelOffset+pixels.length,14+entries.length*12+entries.reduce((n,[,v])=>n+(v.length>1 ? v.length*4 : 0),0)));
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
  bytes.set(pixels,pixelOffset);
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

it('rejects aliased metadata by aggregate value count before UTIF parses arrays', async () => {
  // Each entry fits the file, but all 64 entries alias the same 1024 LONGs.
  const base=makeTiff(Object.fromEntries(Array.from({length:64},(_,i)=>[`t${40000+i}`,[0]])));
  const bytes=new Uint8Array(6000); bytes.set(base);
  const view=new DataView(bytes.buffer);
  for (let i=0;i<view.getUint16(8,true);i++) {
    const p=10+i*12;
    if (view.getUint16(p,true)>=40000) {
      view.setUint32(p+4,1024,true); view.setUint32(p+8,1900,true);
    }
  }
  const UTIF=await import('utif');
  await expect(decodeTiff(bytes)).rejects.toThrow(/metadata.*limit|metadata.*budget/i);
  expect(UTIF.decode).not.toHaveBeenCalled();
});

it('rejects oversized sample tag counts before UTIF parses arrays', async () => {
  const UTIF=await import('utif');
  await expect(decodeTiff(makeTiff({t258:Array(128).fill(8)}))).rejects.toThrow(/sample|channel|bit/i);
  expect(UTIF.decode).not.toHaveBeenCalled();
});

it('drops non-decoding tags from the private TIFF copy', async () => {
  const bytes=makeTiff({t40000:[1,2,3],t34665:[0]});
  const original=bytes.slice();
  const image=await decodeTiff(bytes);
  const UTIF=await import('utif');
  const tags=UTIF.decode(vi.mocked(UTIF.decode).mock.calls[0][0])[0];
  expect(tags.t40000).toBeUndefined();
  expect(tags.t34665).toBeUndefined();
  expect(bytes).toEqual(original);
  expect(Array.from(image.pixels)).toEqual([255,0,0,255,0,255,0,255]);
});

it('rejects a DEFLATE strip exceeding the exact image budget before UTIF inflation', async () => {
  const UTIF=await import('utif');
  const rasterize=vi.mocked(UTIF.decodeImage).mockImplementation(()=>{ throw new Error('Unsafe rasterization reached'); });
  const bytes=makeTiff({t259:[8]},Array.from(deflateSync(new Uint8Array(4096))));
  await expect(decodeTiff(bytes)).rejects.toThrow(/strip.*(budget|limit|length)/i);
  expect(rasterize).not.toHaveBeenCalled();
});

it('stops streaming inflation at the first over-budget byte without retaining full output', async () => {
  const push=vi.spyOn(Inflate.prototype,'push');
  const bytes=makeTiff({t259:[8]},Array.from(deflateSync(new Uint8Array(4096))));
  await expect(decodeTiff(bytes)).rejects.toThrow(/strip.*budget/i);
  const stream=push.mock.contexts[0] as Inflate & {strm:{total_out:number}};
  // The 2x1 RGB image allows six bytes; only seven are expanded, not 4096.
  expect(stream.strm.total_out).toBe(7);
});

it.each(['truncated','checksum','short','trailing'])('rejects a %s DEFLATE strip', async kind => {
  let data=new Uint8Array(deflateSync(new Uint8Array(kind==='short' ? 3 : 6)));
  if (kind==='truncated') data=data.slice(0,-1);
  if (kind==='checksum') data[data.length-1]^=1;
  if (kind==='trailing') data=new Uint8Array([...data,0]);
  await expect(decodeTiff(makeTiff({t259:[8]},Array.from(data)))).rejects.toThrow(/strip/i);
});

it('rejects TIFF metadata aliasing the directory that is sanitized before UTIF parsing', async () => {
  const bytes=makeTiff({t40000:[8,8,8]});
  const view=new DataView(bytes.buffer);
  // BitsPerSample now aliases the inline fields of several directory entries.
  for (let i=0;i<view.getUint16(8,true);i++) if(view.getUint16(10+i*12,true)===258) view.setUint32(18+i*12,10,true);
  const UTIF=await import('utif');
  await expect(decodeTiff(bytes)).rejects.toThrow(/metadata|directory/i);
  expect(UTIF.decode).not.toHaveBeenCalled();
});

it.each([new Uint8Array([255,0,0]), new Uint8Array(7)])('rejects short or overlong stored strips', async pixels => {
  await expect(decodeTiff(makeTiff({},Array.from(pixels)))).rejects.toThrow(/strip/i);
});

it('rejects extra aliased strips beyond the declared row layout', async () => {
  const strip=Array.from(deflateSync(new Uint8Array([255,0,0,0,255,0])));
  await expect(decodeTiff(makeTiff({t259:[8],t273:[1000,1000],t279:[strip.length,strip.length]},strip))).rejects.toThrow(/strip/i);
});

it.each([1,8,32946])('retains RGB and palette strips with compression %i', async compression => {
  const encode=(pixels: number[])=>compression===1 ? pixels : Array.from(deflateSync(new Uint8Array(pixels)));
  const rgb=await decodeTiff(makeTiff({t259:[compression]},encode([255,0,0,0,255,0])));
  expect(Array.from(rgb.pixels)).toEqual([255,0,0,255,0,255,0,255]);
  const map=Array(768).fill(0); map[0]=65535; map[257]=65535;
  // A larger header accommodates a real 8-bit palette, independently of pixels.
  const data=encode([0,1]);
  const bytes=makeTiff({t259:[compression],t258:[8],t262:[3],t277:[1],t320:map,t273:[5000]},data);
  const image=await decodeTiff(bytes);
  expect(Array.from(image.pixels)).toEqual([255,0,0,255,0,255,0,255]);
});

it.each([1,8])('retains multiple strips, a partial final strip and row predictors (%i)', async compression => {
  const row=[255,0,0,1,255,0]; // horizontal RGB differences for red, green
  const first=compression===1 ? [...row,...row] : Array.from(deflateSync(new Uint8Array([...row,...row])));
  const last=compression===1 ? row : Array.from(deflateSync(new Uint8Array(row)));
  const image=await decodeTiff(makeTiff({t257:[3],t259:[compression],t278:[2],t317:[2],
    t273:[1000,1000+first.length],t279:[first.length,last.length]},[...first,...last]));
  expect([image.width,image.height]).toEqual([2,3]);
  expect(Array.from(image.pixels)).toEqual([
    255,0,0,255,0,255,0,255,255,0,0,255,0,255,0,255,255,0,0,255,0,255,0,255,
  ]);
});

it('rejects compression without a bounded decoder with an actionable error', async () => {
  await expect(decodeTiff(makeTiff({t259:[5]}))).rejects.toThrow(/compression.*stored.*DEFLATE/i);
});
