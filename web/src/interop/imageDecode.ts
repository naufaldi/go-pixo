import type { InputFormat } from './compressionSettings';
import { assertSafeDimensions, inspectImage, inputMime, MAX_IMAGE_BYTES } from './imageFormats';
import { copyImageDataPixels } from './pixels';

export type DecodeResult = {
  pixels: Uint8Array;
  width: number;
  height: number;
  colorType: number;
  previewUrl: string;
  originalFileBytes: Uint8Array;
  inputFormat: InputFormat;
};

type RasterResult = {pixels: Uint8Array; width: number; height: number; previewBytes: Uint8Array};
const aborted = () => new DOMException('Image decoding cancelled', 'AbortError');
function assertActive(signal?: AbortSignal): void {
  if (signal?.aborted) throw aborted();
}

function decodeBundled(bytes: Uint8Array, signal?: AbortSignal): Promise<RasterResult> {
  return new Promise((resolve,reject)=>{
    assertActive(signal);
    const worker=new Worker(new URL('./codecsDecode.worker.ts',import.meta.url),{type:'module'});
    let settled=false;
    const finish=(error?: Error, result?: RasterResult) => {
      if (settled) return;
      settled=true;
      clearTimeout(timer); signal?.removeEventListener('abort',cancel);
      worker.onmessage=null; worker.onerror=null; worker.onmessageerror=null; worker.terminate();
      if (error) reject(error); else resolve(result!);
    };
    const cancel=()=>finish(aborted());
    const timer=setTimeout(()=>finish(new Error('Image decoder timed out')),30_000);
    signal?.addEventListener('abort',cancel,{once:true});
    worker.onmessage=event=>{
      if (event.data.error) finish(new Error(event.data.error));
      else finish(undefined,event.data);
    };
    worker.onerror=()=>finish(new Error('Local image decoder failed'));
    worker.onmessageerror=()=>finish(new Error('Invalid image decoder response'));
    try {
      // The original remains owned by the caller for compression's original-byte candidate.
      const buffer=new Uint8Array(bytes).buffer;
      worker.postMessage({bytes:buffer},[buffer]);
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

export async function decodeFile(file: File, signal?: AbortSignal): Promise<DecodeResult> {
  const buffer=await readFile(file,signal);
  assertActive(signal);
  const originalFileBytes=new Uint8Array(buffer);
  const info=inspectImage(originalFileBytes,file.type);
  if (info.format==='tiff' || info.format==='avif' || info.format==='heic') {
    const image=await decodeBundled(originalFileBytes,signal);
    assertActive(signal);
    const previewUrl=URL.createObjectURL(new Blob([new Uint8Array(image.previewBytes)],{type:'image/png'}));
    return {...image,colorType:6,previewUrl,originalFileBytes,inputFormat:info.format};
  }

  // Native decoding is bounded by preflight. The caller owns bounded queueing.
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
        assertSafeDimensions(width,height);
        canvas=document.createElement('canvas'); canvas.width=width; canvas.height=height;
        const ctx=canvas.getContext('2d'); if (!ctx) throw new Error('Failed to create canvas context');
        ctx.drawImage(img,0,0);
        const pixels=copyImageDataPixels(ctx.getImageData(0,0,width,height).data);
        finish(undefined,{pixels,width,height,colorType:6,previewUrl:url,originalFileBytes,inputFormat:info.format});
      } catch(error) { finish(error); }
      finally { if (canvas) { canvas.width=0; canvas.height=0; } }
    };
    img.onerror=()=>finish(new Error('Invalid or unsupported image data'));
    try { img.src=url; } catch(error) { finish(error); }
  });
}
