// WASM Bridge for Go-Pixo
import { unwrapWasmResult } from './interop/wasmResult';


export interface GoWasmInstance {
  encodePng(
    pixels: Uint8Array,
    width: number,
    height: number,
    colorType: number,
    preset: number,
    lossy: boolean,
    maxColors: number
  ): Uint8Array | string;
  encodePngAdvanced(
    pixels: Uint8Array,
    width: number,
    height: number,
    colorType: number,
    preset: number,
    lossy: boolean,
    maxColors: number,
    dithering: boolean,
    ditherStrength: number,
    qualityTarget: number,
    zopfliIterations: number,
    filterStrategy: number,
    usePerceptual: boolean,
    distanceMetric: number,
    optimalDeflate: boolean,
    filterEarlyTerm: boolean,
    progressCallback?: (phase: string, progress: number) => void
  ): Uint8Array | string;
  encodeJpeg(
    pixels: Uint8Array,
    width: number,
    height: number,
    colorType: number,
    quality: number
  ): Uint8Array | string;
  encodeJpegAdvanced(
    pixels: Uint8Array,
    width: number,
    height: number,
    colorType: number,
    quality: number,
    subsampling: number,
    progressive: boolean,
    trellis: boolean,
    optimizeHuffman: boolean,
    preset: number
  ): Uint8Array | string;
  recompressPngLossless(
    pngBytes: Uint8Array,
    preset: number,
    zopfliIterations: number,
    progressCallback?: (phase: string, progress: number) => void
  ): Uint8Array | string;
  recompressJpeg(
    jpegBytes: Uint8Array,
    preset: number,
    quality: number,
    progressive: boolean,
    trellis: boolean,
    optimizeHuffman: boolean
  ): Uint8Array | string;
  bytesPerPixel(colorType: number): number;
}

declare global {
  interface Window {
    Go: any;
    encodePng: GoWasmInstance['encodePng'];
    encodePngAdvanced: GoWasmInstance['encodePngAdvanced'];
    encodeJpeg: GoWasmInstance['encodeJpeg'];
    encodeJpegAdvanced: GoWasmInstance['encodeJpegAdvanced'];
    recompressPngLossless: GoWasmInstance['recompressPngLossless'];
    recompressJpeg: GoWasmInstance['recompressJpeg'];
    bytesPerPixel: GoWasmInstance['bytesPerPixel'];
    goWasmInit: () => void;
  }
}

export const initWasm = async (): Promise<void> => {
  return new Promise((resolve, reject) => {
    // @ts-ignore
    const go = new window.Go();

    // Set up initialization callback called from Go's main()
    window.goWasmInit = () => {
      console.log('Go WASM initialized');
      resolve();
    };

    fetch('/main.wasm')
      .then(response => response.arrayBuffer())
      .then(buffer => WebAssembly.instantiate(buffer, go.importObject))
      .then(result => {
        go.run(result.instance);
      })
      .catch(err => {
        console.error('Failed to load WASM:', err);
        reject(err);
      });
  });
};

export const encodePng = (
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 6, // RGBA
  preset: number = 1, // Balanced
  lossy: boolean = false,
  maxColors: number = 0
): Uint8Array => {
  if (!window.encodePng) {
    throw new Error('WASM not initialized');
  }

  const result = window.encodePng(
    pixels,
    width,
    height,
    colorType,
    preset,
    lossy,
    maxColors
  );
  
  return unwrapWasmResult(result);
};

export const encodePngAdvanced = (
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 6,
  preset: number = 1,
  lossy: boolean = false,
  maxColors: number = 0,
  dithering: boolean = false,
  ditherStrength: number = 0.5,
  qualityTarget: number = 75,
  zopfliIterations: number = 0,
  filterStrategy: number = 0,
  usePerceptual: boolean = false,
  distanceMetric: number = 0,
  optimalDeflate: boolean = false,
  filterEarlyTerm: boolean = false,
  progressCallback?: (phase: string, progress: number) => void
): Uint8Array => {
  if (!window.encodePngAdvanced) {
    throw new Error('WASM not initialized');
  }

  const result = window.encodePngAdvanced(
    pixels,
    width,
    height,
    colorType,
    preset,
    lossy,
    maxColors,
    dithering,
    ditherStrength,
    qualityTarget,
    zopfliIterations,
    filterStrategy,
    usePerceptual,
    distanceMetric,
    optimalDeflate,
    filterEarlyTerm,
    progressCallback
  );

  return unwrapWasmResult(result);
};

export const encodeJpeg = (
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 3,
  quality: number = 85
): Uint8Array => {
  if (!window.encodeJpeg) {
    throw new Error('WASM not initialized');
  }

  const result = window.encodeJpeg(
    pixels,
    width,
    height,
    colorType,
    quality
  );

  return unwrapWasmResult(result);
};

export const encodeJpegAdvanced = (
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 3,
  quality: number = 85,
  subsampling: number = 0,
  progressive: boolean = false,
  trellis: boolean = false,
  optimizeHuffman: boolean = false,
  preset: number = 1
): Uint8Array => {
  if (!window.encodeJpegAdvanced) {
    throw new Error('WASM not initialized');
  }

  const result = window.encodeJpegAdvanced(
    pixels,
    width,
    height,
    colorType,
    quality,
    subsampling,
    progressive,
    trellis,
    optimizeHuffman,
    preset
  );

  return unwrapWasmResult(result);
};

export const recompressPngLossless = (
  pngBytes: Uint8Array,
  preset: number,
  zopfliIterations: number = 0,
  progressCallback?: (phase: string, progress: number) => void
): Uint8Array => {
  if (!window.recompressPngLossless) {
    throw new Error('WASM not initialized');
  }

  const result = window.recompressPngLossless(
    pngBytes,
    preset,
    zopfliIterations,
    progressCallback
  );

  return unwrapWasmResult(result);
};

export const recompressJpeg = (
  jpegBytes: Uint8Array,
  preset: number,
  quality: number = 85,
  progressive: boolean = false,
  trellis: boolean = false,
  optimizeHuffman: boolean = false
): Uint8Array => {
  if (!window.recompressJpeg) {
    throw new Error('recompressJpeg not available');
  }

  const result = window.recompressJpeg(
    jpegBytes,
    preset,
    quality,
    progressive,
    trellis,
    optimizeHuffman
  );

  return unwrapWasmResult(result);
};

export const getBytesPerPixel = (colorType: number): number => {
  if (!window.bytesPerPixel) {
    return 4; // Default to RGBA
  }
  return window.bytesPerPixel(colorType);
};
