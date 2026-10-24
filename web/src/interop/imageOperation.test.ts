import {expect, it} from 'vitest';
import {imageOperation} from './imageOperation';
import {inspectImage} from './imageFormats';
import {jpegWithExif} from '../test/jpeg';
it('budgets the larger output before upscaling instead of only admitting source pixels', () => {
  expect(imageOperation({format: 'png',width: 1000,height: 1000}, {lossy: true,targetWidth: 4000}).workspaceBytes).toBe(512_000_000);
});
it('budgets native JPEG orientation before a single-axis resize can increase the output area', () => {
  // Native decode can rotate 4000x3000 into 3000x4000. Width 4200 then
  // produces 4200x5600 (23.52 MP), rather than the header's 4200x3150.
  expect(imageOperation(inspectImage(jpegWithExif(4000,3000,6)), {lossy: true,targetWidth: 4200}).workspaceBytes).toBe(752_640_000);
});
it('treats explicit unchanged dimensions as byte-only but requires pixels for changed dimensions', () => {
  expect(imageOperation({format: 'jpeg',width: 2,height: 1}, {lossy: false,targetWidth: 2}).kind).toBe('original');
  expect(imageOperation({format: 'jpeg',width: 2,height: 1}, {lossy: false,targetWidth: 1}).kind).toBe('pixels');
});

it('accepts the reviewer 3000x4000 rotated JPEG resized to displayed width 5000', () => {
  expect(imageOperation(inspectImage(jpegWithExif(3000,4000,6)), {lossy:true,targetWidth:5000})).toMatchObject({
    kind:'pixels',width:5000,height:3750,resized:true,workspaceBytes:600_000_000,
  });
});
it('does not silently keep displayed width 4000 when lossless JPEG requests width 3000', () => {
  expect(imageOperation(inspectImage(jpegWithExif(3000,4000,6)), {lossy:false,targetWidth:3000})).toMatchObject({
    kind:'pixels',width:3000,height:2250,resized:true,workspaceBytes:384_000_000,
  });
});
it('uses displayed dimensions for unchanged JPEG eligibility', () => {
  expect(imageOperation(inspectImage(jpegWithExif(3000,4000,6)), {lossy:false,targetWidth:4000})).toMatchObject({
    kind:'original',width:4000,height:3000,resized:false,workspaceBytes:0,
  });
});
it('rejects an oversized display resize even when stored-axis planning would fit', () => {
  expect(() => imageOperation(inspectImage(jpegWithExif(4000,3000,6)), {lossy:true,targetWidth:5000})).toThrow(/dimension|pixel/i);
});

it.each([
  [1,3000,4000],[2,3000,4000],[3,3000,4000],[4,3000,4000],
  [5,4000,3000],[6,4000,3000],[7,4000,3000],[8,4000,3000],
])('plans orientation %i using display dimensions %ix%i', (orientation,width,height) => {
  expect(imageOperation(inspectImage(jpegWithExif(3000,4000,orientation)), {lossy:true})).toMatchObject({
    width,height,resized:false,workspaceBytes:384_000_000,
  });
});
