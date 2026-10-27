import { expect, test } from '@playwright/test';
import { deflateSync } from 'node:zlib';

function tiff(deflate: boolean, oversized = false): number[] {
  const pixels=oversized ? new Uint8Array(4096) : new Uint8Array([255,0,0,0,255,0]);
  const strip=deflate ? deflateSync(pixels) : pixels;
  const tags=[[256,2],[257,1],[258,8],[259,deflate ? 8 : 1],[262,2],
    [273,256],[277,3],[278,1],[279,strip.length]];
  const bytes=new Uint8Array(256+strip.length);
  bytes.set([73,73,42,0,8,0,0,0]);
  const view=new DataView(bytes.buffer);
  view.setUint16(8,tags.length,true);
  tags.forEach(([tag,value],i)=>{
    const p=10+i*12;
    view.setUint16(p,tag,true); view.setUint16(p+2,3,true);
    view.setUint32(p+4,tag===258 ? 3 : 1,true);
    view.setUint32(p+8,tag===258 ? 200 : value,true);
  });
  view.setUint16(200,8,true); view.setUint16(202,8,true); view.setUint16(204,8,true);
  bytes.set(strip,256);
  return Array.from(bytes);
}

test('real browser imports bounded TIFF strips and static SVG, and rejects excess strip output and unmatched URLs', async ({page}) => {
  await page.goto('/');
  const result=await page.evaluate(async fixtures=>{
    const modulePath='/src/interop/imageDecode.ts';
    const {decodeFile}=await import(modulePath);
    const images=[];
    for (const bytes of fixtures.valid) {
      const image=await decodeFile(new File([new Uint8Array(bytes)],'image.tiff',{type:'image/tiff'}));
      images.push({width:image.width,height:image.height,pixels:Array.from(image.pixels)});
      URL.revokeObjectURL(image.previewUrl);
    }
    const svg='<svg xmlns="http://www.w3.org/2000/svg" width="2" height="1"><defs><path id="red" d="M0 0h2v1H0z"/></defs><use href="#red" fill="red"/></svg>';
    const staticImage=await decodeFile(new File([svg],'static.svg',{type:'image/svg+xml'}));
    const staticPixels=Array.from(staticImage.pixels);
    URL.revokeObjectURL(staticImage.previewUrl);
    const errors=[];
    for (const file of [new File([new Uint8Array(fixtures.bad)],'excess.tiff'),
      new File([`<svg width="2" height="1"><!--${'url('.repeat(2048)}--></svg>`],'unmatched.svg')]) {
      try { await decodeFile(file); errors.push('unexpected success'); }
      catch(error) { errors.push(error instanceof Error ? error.message : String(error)); }
    }
    return {images,staticPixels,errors};
  },{valid:[tiff(false),tiff(true)],bad:tiff(true,true)});
  expect(result.images).toEqual([
    {width:2,height:1,pixels:[255,0,0,255,0,255,0,255]},
    {width:2,height:1,pixels:[255,0,0,255,0,255,0,255]},
  ]);
  expect(result.staticPixels).toEqual([255,0,0,255,255,0,0,255]);
  expect(result.errors[0]).toMatch(/strip.*budget/i);
  expect(result.errors[1]).toMatch(/unterminated URL/i);
});
