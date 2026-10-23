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
function decodeBundled(bytes: Uint8Array): Promise<RasterResult> {
  return new Promise((resolve,reject)=>{
    const worker=new Worker(new URL('./codecsDecode.worker.ts',import.meta.url),{type:'module'});
    const finish=(error?: Error, result?: RasterResult) => {
      clearTimeout(timer); worker.terminate();
      if (error) reject(error); else resolve(result!);
    };
    const timer=setTimeout(()=>finish(new Error('Image decoder timed out')),30_000);
    worker.onmessage=event=>{
      if (event.data.error) finish(new Error(event.data.error));
      else finish(undefined,event.data);
    };
    worker.onerror=()=>finish(new Error('Local image decoder failed'));
    worker.onmessageerror=()=>finish(new Error('Invalid image decoder response'));
    const buffer=new Uint8Array(bytes).buffer;
    worker.postMessage({bytes:buffer},[buffer]);
  });
}

function readFile(file: File): Promise<ArrayBuffer> {
  if (file.size>MAX_IMAGE_BYTES) return Promise.reject(new Error('Image exceeds the 64 MiB file limit'));
  return new Promise((resolve,reject)=>{
    const reader=new FileReader();
    reader.onload=()=>reader.result instanceof ArrayBuffer ? resolve(reader.result) : reject(new Error('Failed to read file'));
    reader.onerror=()=>reject(new Error('Failed to read file'));
    reader.readAsArrayBuffer(file);
  });
}

export async function decodeFile(file: File): Promise<DecodeResult> {
  const buffer=await readFile(file);
  const originalFileBytes=new Uint8Array(buffer);
  const info=inspectImage(originalFileBytes,file.type);
  if (info.format==='tiff' || info.format==='avif' || info.format==='heic') {
    const image=await decodeBundled(originalFileBytes);
    const previewUrl=URL.createObjectURL(new Blob([new Uint8Array(image.previewBytes)],{type:'image/png'}));
    return {...image,colorType:6,previewUrl,originalFileBytes,inputFormat:info.format};
  }

  // Native decoding is bounded by preflight. The caller owns bounded queueing.
  const url=URL.createObjectURL(new Blob([buffer],{type:inputMime(info.format)}));
  return new Promise((resolve,reject)=>{
    const img=new Image();
    const fail=(error: unknown) => {
      clearTimeout(timer); img.onload=null; img.onerror=null; URL.revokeObjectURL(url);
      reject(error);
    };
    const timer=setTimeout(()=>fail(new Error('Image decoder timed out')),30_000);
    img.onload=()=>{
      try {
        const width=img.naturalWidth || img.width, height=img.naturalHeight || img.height;
        assertSafeDimensions(width,height);
        const canvas=document.createElement('canvas'); canvas.width=width; canvas.height=height;
        const ctx=canvas.getContext('2d'); if (!ctx) throw new Error('Failed to create canvas context');
        ctx.drawImage(img,0,0);
        const pixels=copyImageDataPixels(ctx.getImageData(0,0,width,height).data);
        clearTimeout(timer); img.onload=null; img.onerror=null;
        resolve({pixels,width,height,colorType:6,previewUrl:url,originalFileBytes,inputFormat:info.format});
      } catch(error) { fail(error); }
    };
    img.onerror=()=>fail(new Error('Invalid or unsupported image data'));
    img.src=url;
  });
}
