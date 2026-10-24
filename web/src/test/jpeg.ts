// Minimal metadata fixtures. Pixel decoding tests insert the segment into a native JPEG.
export function jpegExifSegment(orientation: number, littleEndian = true): Uint8Array {
  const bytes = new Uint8Array(36);
  bytes.set([255,225,0,34,69,120,105,102,0,0]);
  const view = new DataView(bytes.buffer);
  bytes.set(littleEndian ? [73,73] : [77,77], 10);
  view.setUint16(12,42,littleEndian);
  view.setUint32(14,8,littleEndian);
  view.setUint16(18,1,littleEndian);
  view.setUint16(20,274,littleEndian);
  view.setUint16(22,3,littleEndian);
  view.setUint32(24,1,littleEndian);
  view.setUint16(28,orientation,littleEndian);
  return bytes;
}

export function jpegWithExif(width: number, height: number, orientation: number, littleEndian = true, afterFrame = false): Uint8Array {
  const frame = new Uint8Array([255,192,0,7,8,height >> 8,height & 255,width >> 8,width & 255]);
  const exif = jpegExifSegment(orientation,littleEndian);
  return new Uint8Array([255,216,...(afterFrame ? frame : exif),...(afterFrame ? exif : frame),255,217]);
}
