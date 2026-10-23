import { describe, expect, it } from 'vitest';
import { assertSafeDimensions, inspectImage, sniffInputFormat } from './imageFormats';

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
