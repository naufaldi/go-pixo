import { expect, it } from 'vitest';
import { compressionCandidates, resolveResizeDimensions, selectCompressionResult } from './compressionSettings';

it('auto uses a fast modern candidate and an encodable current format', () => {
  expect(compressionCandidates('png', 'auto', false)).toEqual(['webp', 'png']);
  expect(compressionCandidates('gif', 'auto', false)).toEqual(['webp', 'png']);
  expect(compressionCandidates('jpeg', 'auto', true)).toEqual(['avif', 'png']);
  expect(compressionCandidates('avif', 'same', true)).toEqual(['avif']);
  expect(() => compressionCandidates('png', 'jpeg', true)).toThrow(/lossy/i);
  expect(() => compressionCandidates('png', 'webp', true)).toThrow(/lossy/i);
});
it('selects the smallest actual candidate without undoing resize', () => {
  const original = {compressedBytes: new Uint8Array(10), outputFormat: 'png' as const};
  const resized = {compressedBytes: new Uint8Array(20), outputFormat: 'png' as const};
  expect(selectCompressionResult([resized], original, false)).toBe(original);
  expect(selectCompressionResult([resized], original, true)).toBe(resized);
  const webp = {compressedBytes: new Uint8Array(5), outputFormat: 'webp' as const};
  expect(selectCompressionResult([resized, webp], original, false)).toBe(webp);
});
it('bounds resize dimensions before allocating and retains aspect ratio', () => {
  expect(resolveResizeDimensions(400, 200, 100)).toEqual({width:100,height:50,resized:true});
  expect(resolveResizeDimensions(400, 200, undefined, 100)).toEqual({width:200,height:100,resized:true});
  expect(resolveResizeDimensions(400, 200, 400)).toEqual({width:400,height:200,resized:false});
  expect(() => resolveResizeDimensions(400,200,0)).toThrow();
  expect(() => resolveResizeDimensions(400,200,1e9)).toThrow();
  expect(() => resolveResizeDimensions(400,200,NaN)).toThrow();
});
