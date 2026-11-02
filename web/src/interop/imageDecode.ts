import type { InputFormat } from './compressionSettings';
import { HEIC_UNAVAILABLE_MESSAGE, inspectImage, inputMime, MAX_IMAGE_BYTES } from './imageFormats';
import { ownedImageDataPixels } from './pixels';
import { imageOperation, type OperationSettings } from './imageOperation';
import { assertDecodedDimensions } from './codecsNative';

export type DecodeResult = {
  pixels?: Uint8Array;
  // Display dimensions, matching native oriented pixels and the original preview.
  width: number;
  height: number;
  colorType: number;
  previewUrl: string;
  originalFileBytes: Uint8Array;
  inputFormat: InputFormat;
};

type RasterResult = {pixels?: Uint8Array; width: number; height: number; previewBytes?: Uint8Array};
export const MAX_FALLBACK_PIXELS = 1_000_000;
const aborted = () => new DOMException('Image decoding cancelled', 'AbortError');
function assertActive(signal?: AbortSignal): void {
  if (signal?.aborted) throw aborted();
}

function decodeInWorker(bytes: Uint8Array, validateOnly: boolean, signal?: AbortSignal): Promise<RasterResult | undefined> {
  return new Promise((resolve,reject)=>{
    assertActive(signal);
    const worker=new Worker(new URL('./codecsDecode.worker.ts',import.meta.url),{type:'module'});
    let settled=false;
    const finish=(error?: Error, result?: RasterResult) => {
      if (settled) return;
      settled=true;
      clearTimeout(timer); signal?.removeEventListener('abort',cancel);
      worker.onmessage=null; worker.onerror=null; worker.onmessageerror=null; worker.terminate();
      if (error) reject(error); else resolve(result);
    };
    const cancel=()=>finish(aborted());
    const timer=setTimeout(()=>finish(new Error('Image decoder timed out')),30_000);
    signal?.addEventListener('abort',cancel,{once:true});
    worker.onmessage=event=>{
      const response = event.data;
      if (!response || typeof response !== 'object') finish(new Error('Invalid image decoder response'));
      else if (response.unsupportedNative === true) finish();
      else if (typeof response.error === 'string') finish(new Error(response.error));
      else if (typeof response.width !== 'number' || typeof response.height !== 'number' ||
        (response.pixels !== undefined && !(response.pixels instanceof Uint8Array)) ||
        (response.previewBytes !== undefined && !(response.previewBytes instanceof Uint8Array))) finish(new Error('Invalid image decoder response'));
      else finish(undefined,response);
    };
    worker.onerror=()=>finish(new Error('Local image decoder failed'));
    worker.onmessageerror=()=>finish(new Error('Invalid image decoder response'));
    try {
      // The original remains owned by the caller for compression's original-byte candidate.
      const buffer=new Uint8Array(bytes).buffer;
      worker.postMessage({bytes:buffer,validateOnly},[buffer]);
    } catch(error) { finish(error instanceof Error ? error : new Error('Failed to start image decoder')); }
  });
}

function readFile(file: File, signal?: AbortSignal): Promise<ArrayBuffer> {
  if (file.size>MAX_IMAGE_BYTES) return Promise.reject(new Error('Image exceeds the 64 MiB file limit'));
  return new Promise((resolve,reject)=>{
    assertActive(signal);
    const reader=new FileReader();
    let settled=false;
    const finish=(error?: Error, result?: ArrayBuffer) => {
      if (settled) return;
      settled=true;
      clearTimeout(timer); signal?.removeEventListener('abort',cancel);
      reader.onload=null; reader.onerror=null; reader.onabort=null;
      if (reader.readyState===1) reader.abort();
      if (error) reject(error); else resolve(result!);
    };
    const cancel=()=>finish(aborted());
    const timer=setTimeout(()=>finish(new Error('File read timed out')),30_000);
    signal?.addEventListener('abort',cancel,{once:true});
    reader.onload=()=>reader.result instanceof ArrayBuffer ? finish(undefined,reader.result) : finish(new Error('Failed to read file'));
    reader.onerror=()=>finish(new Error('Failed to read file'));
    reader.onabort=cancel;
    try { reader.readAsArrayBuffer(file); }
    catch(error) { finish(error instanceof Error ? error : new Error('Failed to read file')); }
  });
}

export type DecodeOptions = OperationSettings & {admit?: (workspaceBytes: number) => Promise<void>};

export async function decodeFile(file: File, signal?: AbortSignal, options?: DecodeOptions): Promise<DecodeResult> {
  const buffer=await readFile(file,signal);
  assertActive(signal);
  const originalFileBytes=new Uint8Array(buffer);
  const info=inspectImage(originalFileBytes,file.type);
  if (info.format === 'heic') throw new Error(HEIC_UNAVAILABLE_MESSAGE);
  const operation = imageOperation(info, options);
  await options?.admit?.(operation.workspaceBytes);
  assertActive(signal);
  const validateOnly = operation.kind !== 'pixels';
  const image = typeof Worker === 'function' ? await decodeInWorker(originalFileBytes,validateOnly,signal) : undefined;
  if (image) {
    assertActive(signal);
    assertDecodedDimensions(info,image.width,image.height);
    if (!validateOnly && (!image.pixels || image.pixels.length !== image.width * image.height * 4)) throw new Error('Invalid image decoder pixel response');
    const previewUrl=image.previewBytes
      ? URL.createObjectURL(new Blob([image.previewBytes as Uint8Array<ArrayBuffer>],{type:'image/png'}))
      : URL.createObjectURL(new Blob([buffer],{type:inputMime(info.format)}));
    return {pixels:validateOnly ? undefined : image.pixels,width:image.width,height:image.height,colorType:6,previewUrl,originalFileBytes,inputFormat:info.format};
  }

  if (info.format === 'tiff' || info.format === 'avif') throw new Error('This image requires worker decoding support');
  // A capability-only fallback keeps older browsers useful for small images.
  // Decoder failures never retry on the main thread, and large raster copies
  // require worker support rather than doing up to 96 MB of synchronous work.
  if (!validateOnly && info.width! * info.height! > MAX_FALLBACK_PIXELS) {
    throw new Error('Large images require worker image decoding support in this browser. Try a supported browser or an image under 1 megapixel.');
  }
  const url=URL.createObjectURL(new Blob([buffer],{type:inputMime(info.format)}));
  return new Promise((resolve,reject)=>{
    const img=new Image();
    let settled=false;
    const finish=(error?: unknown, result?: DecodeResult) => {
      if (settled) return;
      settled=true;
      clearTimeout(timer); signal?.removeEventListener('abort',cancel);
      img.onload=null; img.onerror=null; img.src='';
      if (error) { URL.revokeObjectURL(url); reject(error); }
      else resolve(result!);
    };
    const cancel=()=>finish(aborted());
    const timer=setTimeout(()=>finish(new Error('Image decoder timed out')),30_000);
    signal?.addEventListener('abort',cancel,{once:true});
    img.onload=()=>{
      if (settled) return;
      let canvas: HTMLCanvasElement | undefined;
      try {
        const width=img.naturalWidth || img.width, height=img.naturalHeight || img.height;
        assertDecodedDimensions(info,width,height);
        if (validateOnly) {
          finish(undefined,{width,height,colorType:6,previewUrl:url,originalFileBytes,inputFormat:info.format});
          return;
        }
        canvas=document.createElement('canvas'); canvas.width=width; canvas.height=height;
        const ctx=canvas.getContext('2d'); if (!ctx) throw new Error('Failed to create canvas context');
        ctx.drawImage(img,0,0);
        const pixels=ownedImageDataPixels(ctx.getImageData(0,0,width,height).data);
        finish(undefined,{pixels,width,height,colorType:6,previewUrl:url,originalFileBytes,inputFormat:info.format});
      } catch(error) { finish(error); }
      finally { if (canvas) { canvas.width=0; canvas.height=0; } }
    };
    img.onerror=()=>finish(new Error('Invalid or unsupported image data'));
    try { img.src=url; } catch(error) { finish(error); }
  });
}
