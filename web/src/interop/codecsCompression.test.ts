import { expect, it } from 'vitest';
import { compressWithCandidates } from './codecsCompression';

function png(w: number, h: number, length = 45) {
  const b=new Uint8Array(length); b.set([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82]);
  const v=new DataView(b.buffer); v.setUint32(16,w); v.setUint32(20,h); b.set([8,6],24); return b;
}
function webp() {
  return new Uint8Array([82,73,70,70,18,0,0,0,87,69,66,80,86,80,56,76,5,0,0,0,47,1,0,0,0,0]); // 2x1 VP8L header
}
const base={format:'png' as const,outputFormat:'auto' as const,lossy:true,width:2,height:1,originalFileBytes:png(2,1,81)};

it('auto compares the original PNG with fast WebP without invoking PNG quantization', async () => {
  const result=await compressWithCandidates(base, async format => {
    if (format!=='webp') throw new Error('Unnecessary current-format encode');
    return webp();
  });
  expect(result.outputFormat).toBe('webp');
  expect(result.compressedBytes).toEqual(webp());
});
it('keeps requested resized pixels even when the encoded image is larger', async () => {
  const result=await compressWithCandidates({...base,outputFormat:'png',width:1,height:1,resized:true}, async () => png(1,1,93));
  expect(result.compressedBytes.length).toBe(93);
  expect(new DataView(result.compressedBytes.buffer).getUint32(16)).toBe(1);
});
it('does not label silently returned PNG bytes as AVIF', async () => {
  await expect(compressWithCandidates({...base,outputFormat:'avif'}, async () => png(2,1))).rejects.toThrow(/format|signature/i);
});
it('lossless JPEG keeps original bytes instead of running a lossy recompressor', async () => {
  const jpeg=new Uint8Array([255,216,255,192,0,11,8,0,1,0,2,1,1,17,0,255,217]);
  const result=await compressWithCandidates({...base,format:'jpeg',outputFormat:'same',lossy:false,originalFileBytes:jpeg}, async () => { throw new Error('Lossy encode must not run'); });
  expect(result.compressedBytes).toBe(jpeg);
  expect(result.outputFormat).toBe('jpeg');
});
it('auto retains a valid original when the optional modern codec fails', async () => {
  const result=await compressWithCandidates(base, async () => { throw new Error('Codec unavailable'); });
  expect(result.compressedBytes).toBe(base.originalFileBytes);
  expect(result.outputFormat).toBe('png');
});
it('preserves high-bit-depth input and rejects lossless conversion through 8-bit pixels', async () => {
  const bytes=png(2,1); bytes[24]=16;
  const request={...base,lossy:false,originalFileBytes:bytes};
  const result=await compressWithCandidates(request,async () => png(2,1));
  expect(result.compressedBytes).toBe(bytes);
  await expect(compressWithCandidates({...request,outputFormat:'avif'},async () => png(2,1))).rejects.toThrow(/bit.depth|precision/i);
});
