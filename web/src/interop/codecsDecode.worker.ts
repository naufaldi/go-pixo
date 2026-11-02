import { decodeAvif, decodeTiff, decodeHeic } from './codecs';
import { inspectImage } from './imageFormats';
import { decodeNative } from './codecsNative';

self.onmessage=async (event: MessageEvent<{bytes: ArrayBuffer; validateOnly?: boolean}>)=>{
  try {
    const bytes=new Uint8Array(event.data.bytes);
    const {format}=inspectImage(bytes);
    // Chromium's worker bitmap decoder does not accept SVG blobs. The caller's
    // bounded Image fallback handles this capability gap without hiding raster failures.
    if (format === 'svg') {
      self.postMessage({unsupportedNative:true});
      return;
    }
    if (!['tiff','avif','heic'].includes(format)) {
      if (typeof createImageBitmap !== 'function' || (!event.data.validateOnly && typeof OffscreenCanvas !== 'function')) {
        self.postMessage({unsupportedNative: true});
        return;
      }
      const image = await decodeNative(bytes, event.data.validateOnly === true);
      self.postMessage(image, {transfer: image.pixels ? [image.pixels.buffer] : []});
      return;
    }
    const image=format==='tiff' ? await decodeTiff(bytes) : format==='avif' ? await decodeAvif(bytes) : format==='heic' ? await decodeHeic(bytes) : undefined;
    if (!image) throw new Error('Unsupported bundled decoder format');
    const canvas=new OffscreenCanvas(image.width,image.height);
    try {
      const ctx=canvas.getContext('2d'); if (!ctx) throw new Error('Canvas is unavailable');
      ctx.putImageData(new ImageData(new Uint8ClampedArray(image.pixels.buffer as ArrayBuffer,image.pixels.byteOffset,image.pixels.byteLength),image.width,image.height),0,0);
      const blob=await canvas.convertToBlob({type:'image/png'});
      const previewBytes=new Uint8Array(await blob.arrayBuffer());
      self.postMessage({...image,previewBytes}, {transfer:[image.pixels.buffer,previewBytes.buffer]});
    } finally {
      canvas.width=0; canvas.height=0;
    }
  } catch(error) {
    self.postMessage({error:error instanceof Error ? error.message : 'Image decode failed'});
  }
};
