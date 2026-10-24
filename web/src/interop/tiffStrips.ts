import { Inflate, type ZStream } from 'pako';
import type { Ifd } from 'utif';
import { MAX_IMAGE_PIXELS } from './imageFormats';

// UTIF's DEFLATE branch retains the entire inflated strip before copying it.
// Decode only methods with explicit output bounds here, then use UTIF solely
// for the bounded sample-to-RGBA conversion. No dependency globals are patched.
export function decodeTiffStrips(bytes: Uint8Array, page: Ifd, width: number, height: number): Uint8Array {
  const compression=page.t259?.[0] ?? 1;
  if (![1,8,32946].includes(compression)) {
    throw new Error('Unsupported TIFF compression. Export stored or DEFLATE strip-based TIFF.');
  }
  const samples=page.t277?.[0] ?? 1, bits=page.t258?.[0] ?? 1;
  const rowBytes=Math.ceil(width*samples*bits/8), total=rowBytes*height;
  if (total>MAX_IMAGE_PIXELS*4) throw new Error('TIFF decoded byte budget exceeded');
  const rows=Math.min(page.t278?.[0] ?? height,height);
  const offsets=page.t273, lengths=page.t279, count=Math.ceil(height/rows);
  if (!Number.isSafeInteger(rows) || rows<1 || offsets?.length!==count || lengths?.length!==count) {
    throw new Error('Invalid TIFF strip count or row layout');
  }
  const predictor=page.t317?.[0] ?? 1;
  if (![1,2].includes(predictor) || (predictor===2 && bits!==8) || (page.t266?.[0] ?? 1)!==1) {
    throw new Error('Unsupported TIFF predictor or fill order');
  }
  // Validate every source range and stored length before allocating output.
  let encodedTotal=0;
  for (let i=0;i<count;i++) {
    const offset=offsets[i], length=lengths[i];
    const expected=rowBytes*Math.min(rows,height-i*rows);
    encodedTotal+=length;
    if (!Number.isSafeInteger(offset) || !Number.isSafeInteger(length) || offset<0 || length<1 ||
        offset>bytes.length-length || encodedTotal>bytes.length || (compression===1 && length!==expected)) {
      throw new Error('Invalid TIFF strip range or length');
    }
  }
  const output=new Uint8Array(total);
  let written=0;
  for (let i=0;i<count;i++) {
    const expected=rowBytes*Math.min(rows,height-i*rows);
    const source=bytes.subarray(offsets[i],offsets[i]+lengths[i]);
    if (compression===1) output.set(source,written);
    else {
      // A tiny image gets a tiny chunk. A large strip can overrun its budget
      // by at most one 16 KiB scratch chunk before the callback aborts inflate.
      const inflate=new Inflate({chunkSize:Math.min(16*1024,expected+1),windowBits:15});
      let stripWritten=0, complete=false;
      // Keep the public stream reference to verify every encoded byte was consumed.
      let stream: ZStream | undefined;
      inflate.onStart=strm=>{ stream=strm; };
      inflate.onData=chunk=>{
        if (!(chunk instanceof Uint8Array)) throw new Error('Invalid TIFF strip output');
        if (chunk.length>expected-stripWritten || chunk.length>total-written-stripWritten) {
          throw new Error('TIFF strip exceeds its decoded byte budget limit');
        }
        output.set(chunk,written+stripWritten);
        stripWritten+=chunk.length;
      };
      inflate.onEnd=status=>{ complete=status===0; };
      inflate.push(source,true);
      // onEnd is called only at a real stream end (or error). Reject truncation,
      // checksum errors and trailing data as well as short decoded strips.
      const remaining=stream?.avail_in;
      if (!complete || remaining!==0 || stripWritten!==expected) {
        throw new Error('Invalid or corrupt TIFF DEFLATE strip length or stream');
      }
    }
    written+=expected;
  }
  if (predictor===2) {
    for (let row=0;row<height;row++) {
      const start=row*rowBytes;
      for (let x=samples;x<rowBytes;x++) output[start+x]=(output[start+x]+output[start+x-samples])&255;
    }
  }
  return output;
}
