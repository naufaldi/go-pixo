declare module 'utif' {
  export type Ifd = Record<string, number[]> & {width: number; height: number; data: Uint8Array};
  export function decode(buffer: ArrayBuffer): Ifd[];
  export function decodeImage(buffer: ArrayBuffer, ifd: Ifd): void;
  export function toRGBA8(ifd: Ifd): Uint8Array;
  export function encodeImage(rgba: ArrayBuffer, width: number, height: number): ArrayBuffer;
  export function encode(ifds: Record<string, number[]>[]): ArrayBuffer;
}
