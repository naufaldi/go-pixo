import { describe, expect, it } from 'vitest';
import {jpegWithExif} from '../test/jpeg';
import { assertSafeDimensions, imageDisplayDimensions, inspectImage, sniffInputFormat } from './imageFormats';
import { imageOperation } from './imageOperation';

const ascii = (value: string) => Uint8Array.from(value, c => c.charCodeAt(0));
const png = (w = 2, h = 1, animated = false) => {
  const b = new Uint8Array(animated ? 65 : 45);
  b.set([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82]);
  const v = new DataView(b.buffer); v.setUint32(16, w); v.setUint32(20, h);
  b.set([8,6], 24);
  if (animated) { v.setUint32(33, 8); b.set(ascii('acTL'), 37); v.setUint32(41, 2); }
  return b;
};
const gifFrame = [44,0,0,0,0,1,0,1,0,0,2,2,68,1,0];
const gif = (frames: number) => new Uint8Array([...ascii('GIF89a'),1,0,1,0,0,0,0,...Array.from({length: frames}, () => gifFrame).flat(),59]);
const bmff = (brand: string) => new Uint8Array([0,0,0,20,...ascii('ftyp'),...ascii(brand),0,0,0,0,...ascii(brand)]);

describe('image byte preflight', () => {
  it('uses bytes instead of misleading, empty or uppercase MIME', () => {
    expect(sniffInputFormat(png(), 'image/jpeg')).toBe('png');
    expect(sniffInputFormat(gif(1), '')).toBe('gif');
    expect(sniffInputFormat(ascii('<svg width="2" height="1" xmlns="http://www.w3.org/2000/svg"/>'), 'IMAGE/SVG+XML')).toBe('svg');
    expect(sniffInputFormat(ascii('not an image'), 'IMAGE/PNG')).toBe('unknown');
    expect(sniffInputFormat(bmff('avif'))).toBe('avif');
    expect(sniffInputFormat(bmff('heic'))).toBe('heic');
  });
  it('rejects animated GIF but accepts a single frame', () => {
    expect(inspectImage(gif(1)).format).toBe('gif');
    expect(() => inspectImage(gif(2))).toThrow(/animat/i);
  });
  it('rejects APNG, animated WebP, and AVIF sequences before decoding', () => {
    expect(() => inspectImage(png(2, 1, true))).toThrow(/animat/i);
    const webp = new Uint8Array([...ascii('RIFF'),14,0,0,0,...ascii('WEBPVP8X'),2,0,0,0,2,0]);
    expect(() => inspectImage(webp)).toThrow(/animat/i);
    expect(() => inspectImage(bmff('avis'))).toThrow(/animat/i);
  });
  it('uses the primary item irot for HEIC display dimensions and resize planning', () => {
    const box = (tag: string, body: number[] = []) => [0,0,0,8+body.length,...ascii(tag),...body];
    const u32 = (v: number) => [v>>>24&255, v>>>16&255, v>>>8&255, v&255];
    // ipco: 1 = ispe 4032x3024, 2 = irot 90 degrees anticlockwise.
    const ipco = box('ipco',[...box('ispe',[0,0,0,0,...u32(4032),...u32(3024)]),...box('irot',[1])]);
    const heic = (rotatedItem: number) => new Uint8Array([
      ...box('ftyp',[...ascii('heic'),0,0,0,0,...ascii('mif1heic')]),
      ...box('meta',[0,0,0,0,
        ...box('pitm',[0,0,0,0,0,1]),
        ...box('iprp',[...ipco,
          // ipma v0: item 1 -> ispe; item `rotatedItem` -> irot.
          ...box('ipma',[0,0,0,0,...u32(2), 0,1,1,1, 0,rotatedItem,1,2])])]),
    ]);
    const portrait = inspectImage(heic(1));
    expect(imageDisplayDimensions(portrait)).toEqual({width:3024,height:4032});
    expect(imageOperation(portrait,{lossy:true,targetHeight:5000})).toMatchObject({width:3750,height:5000});
    expect(imageDisplayDimensions(inspectImage(heic(2)))).toEqual({width:4032,height:3024});

    // Associations split over several ipma boxes (narrow and wide indexes):
    // the primary item's irot sits in the first box, another item in the second.
    const split = (wide: number) => new Uint8Array([
      ...box('ftyp',[...ascii('heic'),0,0,0,0,...ascii('mif1heic')]),
      ...box('meta',[0,0,0,0,
        ...box('pitm',[0,0,0,0,0,1]),
        ...box('iprp',[...ipco,
          ...box('ipma',[0,0,0,wide,...u32(1), 0,1,2,...(wide ? [0,1,0,2] : [1,2])]),
          ...box('ipma',[0,0,0,0,...u32(1), 0,2,1,1])])]),
    ]);
    expect(imageDisplayDimensions(inspectImage(split(0)))).toEqual({width:3024,height:4032});
    expect(imageDisplayDimensions(inspectImage(split(1)))).toEqual({width:3024,height:4032});
  });
  it.each([
    ['truncated version 1', [1,0,0,0,0,1]],
    ['unsupported version 2', [2,0,0,0,0,0,0,1]],
  ])('ignores a %s pitm instead of reading past it', (_case, pitm) => {
    const box = (tag: string, body: number[] = []) => [0,0,0,8+body.length,...ascii(tag),...body];
    const u32 = (v: number) => [v>>>24&255, v>>>16&255, v>>>8&255, v&255];
    const ipco = box('ipco',[...box('ispe',[0,0,0,0,...u32(40),...u32(30)]),...box('irot',[1])]);
    const bytes = new Uint8Array([
      ...box('ftyp',[...ascii('heic'),0,0,0,0,...ascii('mif1heic')]),
      ...box('meta',[0,0,0,0,...box('iprp',[...ipco,...box('ipma',[0,0,0,0,...u32(1), 0,1,2,1,2])]),...box('pitm',pitm)]),
    ]);
    expect(imageDisplayDimensions(inspectImage(bytes))).toEqual({width:40,height:30});
  });
  it('treats HEIC as a sequence only when it has a moov track and no still image', () => {
    const box = (tag: string, body: number[] = []) => [0,0,0,8+body.length,...ascii(tag),...body];
    const ispe = box('ispe',[0,0,0,0, 0,0,0,4, 0,0,0,3]);
    const heic = (brands: string, ...boxes: number[][]) =>
      new Uint8Array([...box('ftyp',[...ascii(brands.slice(0,4)),0,0,0,0,...ascii(brands)]),...boxes.flat()]);
    // libheif's examples/example.heic: major 'mif1', compatible 'mif1heichevc'.
    const meta = box('meta',[0,0,0,0,...box('iprp',box('ipco',ispe))]);
    expect(inspectImage(heic('mif1heichevc', meta))).toMatchObject({format:'heic', width:4, height:3});
    expect(inspectImage(heic('msf1hevc', meta, box('moov')))).toMatchObject({format:'heic', width:4, height:3});
    expect(() => inspectImage(heic('msf1hevc', box('moov', box('trak',ispe))))).toThrow(/animat/i);
  });
  it('rejects oversized dimensions, corrupt data and external or animated SVG', () => {
    expect(() => inspectImage(png(100000,100000))).toThrow(/dimension|pixel/i);
    expect(() => inspectImage(new Uint8Array())).toThrow(/empty/i);
    expect(() => inspectImage(new Uint8Array([137,80,78,71]))).toThrow(/unsupported|invalid|corrupt/i);
    expect(() => inspectImage(ascii('<svg width="10" height="10"><image href="https://example.com/a.png"/></svg>'))).toThrow(/external|self.contained/i);
    expect(() => inspectImage(ascii('<svg width="10" height="10"><animate attributeName="x"/></svg>'))).toThrow(/animat/i);
    expect(() => assertSafeDimensions(0, 20)).toThrow();
    expect(() => assertSafeDimensions(10, Infinity)).toThrow();
  });
  it('accepts local quoted SVG gradients but rejects CSS animation properties', () => {
    expect(inspectImage(ascii('<svg width="2" height="1"><path fill="url(\'#gradient\')"/></svg>')).format).toBe('svg');
    expect(() => inspectImage(ascii('<svg width="2" height="1"><style>path {animation-name:fade;animation-duration:1s}</style></svg>'))).toThrow(/animat/i);
  });
  it('rejects repeated unmatched SVG URL prefixes instead of rescanning every suffix', () => {
    const bytes=ascii(`<svg width="2" height="1"><!--${'url('.repeat(2048)}--></svg>`);
    expect(() => inspectImage(bytes)).toThrow(/url|static|self.contained/i);
  });
  it('bounds SVG text and token work before native decoding', () => {
    expect(() => inspectImage(ascii(`<svg width="2" height="1"><!--${' '.repeat(1024*1024)}--></svg>`))).toThrow(/SVG.*limit/i);
    expect(() => inspectImage(ascii(`<svg width="2" height="1">${'<path/>'.repeat(10001)}</svg>`))).toThrow(/SVG.*(token|complex|limit)/i);
  });
  it.each([
    '<path fill="u&#114;l(https://example.com/a)"/>',
    '<path style="fill:u\\72l(https://example.com/a)"/>',
    '<image href=https://example.com/a/>',
    '<image xlink:href="https://example.com/a"/>',
    '<style>path{fill:URL(https://example.com/a)}</style>',
    '<s:script xmlns:s="http://www.w3.org/2000/svg"/>',
    '<s:animate xmlns:s="http://www.w3.org/2000/svg"/>',
    '<image xml:base="https://example.com/a.svg" href="#shape"/>',
    '<path onclick="alert(1)"/>',
  ])('rejects external, encoded or active SVG content: %s', content => {
    expect(() => inspectImage(ascii(`<svg width="2" height="1">${content}</svg>`))).toThrow(/external|static|self.contained|animat/i);
  });
  it('preserves static local use references, gradients and ordinary XML entities', () => {
    const svg='<svg viewBox="0 0 2 1"><defs><path id="shape"/><linearGradient id="paint"/></defs><use href="#shape" fill="url(\'#paint\')"/><text>&amp; &lt; &gt;</text></svg>';
    expect(inspectImage(ascii(svg))).toMatchObject({format:'svg',width:2,height:1});
  });
  it('bounds TIFF dimensions and rejects cyclic page pointers before codec allocation', () => {
    const b=new Uint8Array(38); b.set([73,73,42,0,8,0,0,0,2,0]);
    const v=new DataView(b.buffer);
    v.setUint16(10,256,true); v.setUint16(12,4,true); v.setUint32(14,1,true); v.setUint32(18,100000,true);
    v.setUint16(22,257,true); v.setUint16(24,4,true); v.setUint32(26,1,true); v.setUint32(30,100000,true);
    expect(() => inspectImage(b)).toThrow(/dimension|pixel/i);
    v.setUint32(18,1,true); v.setUint32(30,1,true); v.setUint32(34,8,true);
    expect(() => inspectImage(b)).toThrow(/page|cyclic|invalid/i);
  });
});

describe('bounded JPEG EXIF orientation preflight', () => {
  it.each([true,false])('retains stored dimensions and reads all orientations with little endian=%s', littleEndian => {
    for (let orientation=1;orientation<=8;orientation++) {
      const bytes=jpegWithExif(3000,4000,orientation,littleEndian);
      // A borrowed input view must not accidentally use its backing buffer's origin.
      const padded=new Uint8Array(bytes.length+11); padded.set(bytes,7);
      expect(inspectImage(padded.subarray(7,7+bytes.length))).toMatchObject({
        format:'jpeg',width:3000,height:4000,orientation,
      });
    }
  });
  it('finds orientation after SOF and ignores unrelated APP1 payloads', () => {
    expect(inspectImage(jpegWithExif(3000,4000,8,true,true))).toMatchObject({orientation:8});
    const ordinary=new Uint8Array([255,216,255,225,0,8,88,77,80,0,0,0,255,192,0,7,8,0,3,0,4]);
    expect(inspectImage(ordinary)).toMatchObject({width:4,height:3});
  });
  it.each([
    ['invalid value', (_b: Uint8Array,v: DataView) => v.setUint16(30,9,true)],
    ['invalid type', (_b: Uint8Array,v: DataView) => v.setUint16(24,4,true)],
    ['array count', (_b: Uint8Array,v: DataView) => v.setUint32(26,0xffffffff,true)],
    ['byte order', (b: Uint8Array) => { b[12]=0 }],
    ['TIFF magic', (_b: Uint8Array,v: DataView) => v.setUint16(14,43,true)],
    ['IFD before header', (_b: Uint8Array,v: DataView) => v.setUint32(16,0,true)],
    ['IFD outside segment', (_b: Uint8Array,v: DataView) => v.setUint32(16,36,true)],
    ['excessive IFD entries', (_b: Uint8Array,v: DataView) => v.setUint16(20,65535,true)],
    ['truncated IFD', (_b: Uint8Array,v: DataView) => v.setUint16(20,2,true)],
    ['short EXIF segment', (_b: Uint8Array,v: DataView) => v.setUint16(4,8)],
    ['segment beyond input', (_b: Uint8Array,v: DataView) => v.setUint16(4,65535)],
  ] as const)('rejects malformed orientation safely: %s', (_,mutate) => {
    const b=jpegWithExif(3000,4000,6); mutate(b,new DataView(b.buffer));
    expect(() => inspectImage(b)).toThrow(/invalid|corrupt/i);
  });
  it('does not follow EXIF or next-IFD pointers into maker notes or cyclic metadata', () => {
    const b=jpegWithExif(3000,4000,6);
    // The next IFD points back to IFD0. Orientation needs only this one bounded directory.
    new DataView(b.buffer).setUint32(34,8,true);
    expect(inspectImage(b)).toMatchObject({orientation:6});
  });
  it.each([[255,216,255],[255,216,255,255,255],[255,216,255,225,0]].map(bytes => [bytes]))('rejects truncated JPEG markers without leaking RangeError', values => {
    expect(() => inspectImage(new Uint8Array(values))).toThrow(/invalid|corrupt/i);
  });
});

it('keeps storage axes when valid EXIF has no orientation and ignores unrelated pointer payloads', () => {
  const b=jpegWithExif(3000,4000,6), view=new DataView(b.buffer);
  view.setUint16(22,34665,true); view.setUint16(24,4,true); view.setUint32(30,0xffffffff,true);
  const info=inspectImage(b);
  expect(info.orientation).toBeUndefined();
  expect(info).toMatchObject({width:3000,height:4000});
});
it('rejects duplicate EXIF blocks and orientation tags instead of guessing native precedence', () => {
  const b=jpegWithExif(3000,4000,6);
  const duplicateBlock=new Uint8Array([...b.subarray(0,38),...b.subarray(2)]);
  expect(() => inspectImage(duplicateBlock)).toThrow(/invalid|corrupt/i);
  const duplicateTag=new Uint8Array([...b.subarray(0,34),...b.subarray(22,34),...b.subarray(34)]);
  const view=new DataView(duplicateTag.buffer); view.setUint16(4,46); view.setUint16(20,2,true);
  expect(() => inspectImage(duplicateTag)).toThrow(/invalid|corrupt/i);
});
it('bounds every truncated EXIF prefix to its declared JPEG segment', () => {
  const b=jpegWithExif(3000,4000,6);
  for (let length=3;length<38;length++) {
    expect(() => inspectImage(b.subarray(0,length))).toThrow(/invalid|corrupt/i);
  }
});
