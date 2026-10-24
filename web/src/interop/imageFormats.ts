import type { InputFormat } from './compressionSettings';

// Bound decoded RGBA and canvas allocations as well as compressed input size.
export const MAX_IMAGE_PIXELS = 24_000_000;
export const MAX_IMAGE_DIMENSION = 16_384;
export const MAX_IMAGE_BYTES = 64 * 1024 * 1024;
const invalid = () => new Error('Invalid or corrupt image data');
const animated = () => new Error('Animated images are not supported. Export a single still frame first.');
const text = (b: Uint8Array, start: number, length: number) =>
  String.fromCharCode(...b.subarray(start, start + length));

export function assertSafeDimensions(width: number, height: number): void {
  if (!Number.isSafeInteger(width) || !Number.isSafeInteger(height) || width < 1 || height < 1 ||
      width > MAX_IMAGE_DIMENSION || height > MAX_IMAGE_DIMENSION || width * height > MAX_IMAGE_PIXELS) {
    throw new Error('Image dimensions exceed the limit: 24 megapixels and 16384 pixels per side.');
  }
}

export function inputFormatFromMime(mime: string): InputFormat {
  const value = mime.toLowerCase().split(';')[0].trim();
  const formats: Record<string, InputFormat> = {
    'image/png':'png', 'image/jpeg':'jpeg', 'image/jpg':'jpeg', 'image/webp':'webp',
    'image/avif':'avif', 'image/gif':'gif', 'image/bmp':'bmp', 'image/x-ms-bmp':'bmp',
    'image/svg+xml':'svg', 'image/heic':'heic', 'image/heif':'heic', 'image/tiff':'tiff',
  };
  return formats[value] ?? 'unknown';
}

export function sniffInputFormat(bytes: Uint8Array, _mime = ''): InputFormat {
  if (bytes.length >= 8 && text(bytes,0,8) === '\x89PNG\r\n\x1a\n') return 'png';
  if (bytes[0] === 255 && bytes[1] === 216 && bytes[2] === 255) return 'jpeg';
  if (text(bytes,0,4) === 'RIFF' && text(bytes,8,4) === 'WEBP') return 'webp';
  if (/^GIF8[79]a$/.test(text(bytes,0,6))) return 'gif';
  if (text(bytes,0,2) === 'BM') return 'bmp';
  if (text(bytes,0,4) === 'II*\0' || text(bytes,0,4) === 'MM\0*') return 'tiff';
  if (text(bytes,4,4) === 'ftyp') {
    const size = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint32(0);
    const brands: string[] = [text(bytes,8,4)];
    for (let i=16; i+4<=Math.min(size,bytes.length); i+=4) brands.push(text(bytes,i,4));
    if (brands.some(b => b === 'avif' || b === 'avis')) return 'avif';
    if (brands.some(b => /^(heic|heix|hevc|hevx|heim|heis|hevm|hevs|mif1|msf1)$/.test(b))) return 'heic';
  }
  const prefix = new TextDecoder().decode(bytes.subarray(0,4096)).replace(/^\uFEFF/, '').trimStart();
  if (/^(?:<\?xml[^>]*>\s*)?(?:<!--[\s\S]*?-->\s*)*<svg(?:\s|>)/i.test(prefix)) return 'svg';
  return 'unknown'; // A filename or MIME declaration cannot establish the actual codec.
}

export type JpegOrientation = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8;
export type ImageInfo = {
  format: InputFormat;
  // Encoded storage dimensions. Native JPEG pixels/preview apply orientation.
  width?: number; height?: number; bitDepth?: number;
  orientation?: JpegOrientation;
};

export function imageDisplayDimensions(info: ImageInfo): {width: number; height: number} {
  const {width,height}=info;
  if (width===undefined || height===undefined) throw new Error('Image needs bounded dimensions before admission');
  assertSafeDimensions(width,height);
  return info.format==='jpeg' && (info.orientation ?? 1)>=5 ? {width:height,height:width} : {width,height};
}

// Read only IFD0's inline SHORT orientation. Never follow EXIF, thumbnail, maker
// note or next-IFD pointers, nor allocate arrays from metadata-provided counts.
function jpegExifOrientation(bytes: Uint8Array, start: number, end: number): JpegOrientation | undefined {
  if (end-start<8) throw invalid();
  const view=new DataView(bytes.buffer,bytes.byteOffset+start,end-start);
  const order=view.getUint16(0);
  if (order!==0x4949 && order!==0x4d4d) throw invalid();
  const le=order===0x4949;
  if (view.getUint16(2,le)!==42) throw invalid();
  const ifd=view.getUint32(4,le);
  if (ifd<8 || ifd>view.byteLength-2) throw invalid();
  const count=view.getUint16(ifd,le);
  if (count>1024 || ifd+2+count*12+4>view.byteLength) throw invalid();
  let orientation: JpegOrientation | undefined;
  for (let i=0;i<count;i++) {
    const entry=ifd+2+i*12;
    if (view.getUint16(entry,le)!==274) continue;
    if (orientation!==undefined || view.getUint16(entry+2,le)!==3 || view.getUint32(entry+4,le)!==1) throw invalid();
    const value=view.getUint16(entry+8,le);
    if (value<1 || value>8) throw invalid();
    orientation=value as JpegOrientation;
  }
  return orientation;
}

export function inspectImage(bytes: Uint8Array, mime = ''): ImageInfo {
  if (!bytes.length) throw new Error('Image is empty');
  if (bytes.length > MAX_IMAGE_BYTES) throw new Error('Image exceeds the 64 MiB file limit');
  const format = sniffInputFormat(bytes, mime);
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  let width: number | undefined, height: number | undefined;
  let bitDepth: number | undefined;
  let orientation: JpegOrientation | undefined;
  const u32 = (p: number, le = false) => view.getUint32(p, le);
  if (format === 'unknown') throw new Error('Unsupported or invalid image format');
  if (format === 'png') {
    if (bytes.length < 33 || text(bytes,12,4) !== 'IHDR') throw invalid();
    width=u32(16); height=u32(20); bitDepth=bytes[24];
    for (let p=8; p+12<=bytes.length;) {
      const n=u32(p), tag=text(bytes,p+4,4);
      if (n > bytes.length-p-12) throw invalid();
      if (tag === 'acTL') throw animated();
      p+=n+12;
    }
  } else if (format === 'gif') {
    if (bytes.length < 13) throw invalid();
    width=view.getUint16(6,true); height=view.getUint16(8,true);
    let p=13 + ((bytes[10]&128) ? 3*(1<<((bytes[10]&7)+1)) : 0), frames=0;
    const skipBlocks = () => {
      while (p<bytes.length) { const n=bytes[p++]; if (!n) return; p+=n; }
      throw invalid();
    };
    while (p<bytes.length) {
      const tag=bytes[p++];
      if (tag===59) break;
      if (tag===33) { p++; skipBlocks(); }
      else if (tag===44) {
        if (++frames>1) throw animated();
        if (p+9>bytes.length) throw invalid();
        const flags=bytes[p+8]; p+=9;
        if (flags&128) p+=3*(1<<((flags&7)+1));
        p++; skipBlocks();
      } else throw invalid();
    }
    if (frames!==1) throw invalid();
  } else if (format === 'webp') {
    for (let p=12; p+8<=bytes.length;) {
      const tag=text(bytes,p,4), n=u32(p+4,true), s=p+8;
      if (tag==='ANIM' || tag==='ANMF' || (tag==='VP8X' && (bytes[s]&2))) throw animated();
      if (n>bytes.length-s) throw invalid();
      if (tag==='VP8X' && n>=10) {
        width=1+bytes[s+4]+(bytes[s+5]<<8)+(bytes[s+6]<<16);
        height=1+bytes[s+7]+(bytes[s+8]<<8)+(bytes[s+9]<<16);
      } else if (tag==='VP8 ' && n>=10 && text(bytes,s+3,3)==='\x9d\x01\x2a') {
        width=view.getUint16(s+6,true)&16383; height=view.getUint16(s+8,true)&16383;
      } else if (tag==='VP8L' && n>=5 && bytes[s]===47) {
        const bits=u32(s+1,true); width=(bits&16383)+1; height=((bits>>>14)&16383)+1;
      }
      p=s+n+(n&1);
    }
    if (width===undefined) throw invalid();
  } else if (format === 'jpeg') {
    let p=2, exifSeen=false;
    while (p<bytes.length) {
      if (bytes[p++]!==255) throw invalid();
      while (bytes[p]===255) p++;
      if (p>=bytes.length) throw invalid();
      const marker=bytes[p++];
      if (marker===217 || marker===218) break; // Metadata ends before scan data.
      if (marker===1 || (marker>=208 && marker<=215)) continue;
      if (p+2>bytes.length) throw invalid();
      const n=view.getUint16(p);
      if (n<2 || n>bytes.length-p) throw invalid();
      if (marker===225 && n>=8 && text(bytes,p+2,6)==='Exif\0\0') {
        // Ambiguous duplicate EXIF blocks can disagree with native orientation.
        if (exifSeen) throw invalid();
        exifSeen=true;
        orientation=jpegExifOrientation(bytes,p+8,p+n);
      }
      if ([192,193,194,195,197,198,199,201,202,203,205,206,207].includes(marker)) {
        if (n<7) throw invalid();
        height=view.getUint16(p+3); width=view.getUint16(p+5);
      }
      p+=n;
    }
    if (width===undefined) throw invalid();
  } else if (format === 'bmp') {
    if (bytes.length<26) throw invalid();
    if (u32(14,true)===12) { width=view.getUint16(18,true); height=view.getUint16(20,true); }
    else { width=view.getInt32(18,true); height=Math.abs(view.getInt32(22,true)); }
  } else if (format === 'tiff') {
    if (bytes.length<8) throw invalid();
    const le=bytes[0]===73, p=u32(4,le);
    if (p<8 || p+2>bytes.length) throw invalid();
    const count=view.getUint16(p,le), end=p+2+count*12;
    if (count>1024 || end+4>bytes.length) throw invalid();
    for (let i=0;i<count;i++) {
      const s=p+2+i*12, tag=view.getUint16(s,le), type=view.getUint16(s+2,le), n=u32(s+4,le);
      const size=([0,1,1,2,4,8,1,1,2,4,8,4,8] as number[])[type];
      if (!size || n>1_000_000 || n*size>bytes.length) throw invalid();
      const value=n*size<=4 ? s+8 : u32(s+8,le);
      if (value>bytes.length-n*size) throw invalid();
      if (tag===330 || tag===50706 || tag===33421) throw new Error('RAW / DNG TIFF images are not supported');
      if (tag===256 || tag===257) {
        if (n!==1 || (type!==3 && type!==4)) throw invalid();
        const v=type===3 ? view.getUint16(value,le) : u32(value,le);
        if (tag===256) width=v; else height=v;
      }
    }
    if (width===undefined || height===undefined) throw invalid();
    assertSafeDimensions(width,height);
    if (u32(end,le)!==0) throw new Error('Multiple-page TIFF images are not supported. Export a single page first.');
  } else if (format === 'avif' || format === 'heic') {
    // Parse ISO BMFF boxes rather than searching compressed payload for magic text.
    const walk = (start: number, end: number, depth: number) => {
      if (depth>8) throw invalid();
      for (let p=start; p+8<=end;) {
        let n=u32(p); const tag=text(bytes,p+4,4);
        let header=8;
        if (n===1) { if (p+16>end) throw invalid(); n=Number(view.getBigUint64(p+8)); header=16; }
        if (n===0) n=end-p;
        if (!Number.isSafeInteger(n) || n<header || n>end-p) throw invalid();
        const s=p+header;
        if (tag==='ftyp') {
          const brands=[text(bytes,s,4)];
          for (let i=s+8;i+4<=p+n;i+=4) brands.push(text(bytes,i,4));
          if (brands.some(b=>['avis','msf1','hevc','hevx','hevs','hevm'].includes(b))) throw animated();
        }
        if (tag==='ispe') {
          if (n<header+12) throw invalid();
          const w=u32(s+4), h=u32(s+8); assertSafeDimensions(w,h);
          if (width===undefined || w*h>width*height!) { width=w; height=h; }
        }
        if (tag==='pixi') {
          if (n<header+5 || bytes[s+4]>n-header-5) throw invalid();
          for (let i=0;i<bytes[s+4];i++) bitDepth=Math.max(bitDepth ?? 0,bytes[s+5+i]);
        }
        if (tag==='av1C') {
          if (n<header+4) throw invalid();
          bitDepth=Math.max(bitDepth ?? 0,bytes[s+2]&64 ? bytes[s+2]&32 ? 12 : 10 : 8);
        }
        if (tag==='hvcC') {
          if (n<header+23) throw invalid();
          bitDepth=Math.max(bitDepth ?? 0,8+(bytes[s+17]&7),8+(bytes[s+18]&7));
        }
        if (['meta','iprp','ipco','moov','trak','mdia','minf','stbl'].includes(tag)) walk(s+(tag==='meta'?4:0),p+n,depth+1);
        p+=n;
      }
    };
    walk(0,bytes.length,0);
    // No decoder should allocate from a container with no bounded spatial header.
    if (width===undefined) throw invalid();
  } else if (format === 'svg') {
    const svg=new TextDecoder().decode(bytes);
    if (/<(?:animate\w*|set)\b|\banimation(?:-\w+)?\s*[:=]|@keyframes/i.test(svg)) throw animated();
    if (/<!DOCTYPE|<!ENTITY|<script\b|<foreignObject\b|\bon\w+\s*=/i.test(svg)) throw new Error('SVG must be a self-contained static image');
    const refs=[...svg.matchAll(/\b(?:href|src)\s*=\s*["']([^"']*)["']/gi)].map(m=>m[1]);
    const urls=[...svg.matchAll(/url\(([^)]*)\)/gi)].map(m=>m[1].trim().replace(/^["']|["']$/g,''));
    if (refs.concat(urls).some(ref=>!ref.trim().startsWith('#')) || /@import/i.test(svg)) {
      throw new Error('SVG external resources are not supported; use a self-contained image');
    }
    const root=svg.match(/<svg\b([^>]*)>/i)?.[1]; if (root===undefined) throw invalid();
    const attr=(name: string) => root.match(new RegExp(`\\b${name}\\s*=\\s*["']([^"']*)["']`,'i'))?.[1];
    const dimension=(value?: string) => value && /^\d+(?:\.\d+)?(?:px)?$/i.test(value) ? Math.ceil(parseFloat(value)) : undefined;
    width=dimension(attr('width')); height=dimension(attr('height'));
    const box=attr('viewBox')?.trim().split(/[\s,]+/).map(Number);
    if (box?.length===4 && box.every(Number.isFinite) && box[2]>0 && box[3]>0) {
      width??=Math.ceil(height ? height*box[2]/box[3] : box[2]);
      height??=Math.ceil(width*box[3]/box[2]);
    }
    if (!width || !height) throw new Error('SVG needs bounded width/height or a viewBox');
  }
  if (width!==undefined && height!==undefined) assertSafeDimensions(width,height);
  return {format,width,height,bitDepth,orientation}; // TIFF headers are checked by UTIF before decodeImage.
}

export function inputMime(format: InputFormat): string {
  return ({jpeg:'image/jpeg',svg:'image/svg+xml',heic:'image/heic',tiff:'image/tiff'} as Record<string,string>)[format] ?? `image/${format}`;
}
