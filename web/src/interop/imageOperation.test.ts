import {expect, it} from 'vitest';
import {imageOperation} from './imageOperation';
it('budgets the larger output before upscaling instead of only admitting source pixels', () => {
  expect(imageOperation({format: 'png',width: 1000,height: 1000}, {lossy: true,targetWidth: 4000}).workspaceBytes).toBe(512_000_000);
});
it('budgets native JPEG orientation before a single-axis resize can increase the output area', () => {
  // Native decode can rotate 4000x3000 into 3000x4000. Width 4200 then
  // produces 4200x5600 (23.52 MP), rather than the header's 4200x3150.
  expect(imageOperation({format: 'jpeg',width: 4000,height: 3000}, {lossy: true,targetWidth: 4200}).workspaceBytes).toBe(752_640_000);
});
it('treats explicit unchanged dimensions as byte-only but requires pixels for changed dimensions', () => {
  expect(imageOperation({format: 'jpeg',width: 2,height: 1}, {lossy: false,targetWidth: 2}).kind).toBe('original');
  expect(imageOperation({format: 'jpeg',width: 2,height: 1}, {lossy: false,targetWidth: 1}).kind).toBe('pixels');
});
