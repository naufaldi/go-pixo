// WASM Bridge for Go-Pixo

export interface GoWasmInstance {
  encodePng(pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean): Uint8Array | string;
  bytesPerPixel(colorType: number): number;
}

declare global {
  interface Window {
    Go: any;
    encodePng: GoWasmInstance['encodePng'];
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
  lossy: boolean = false
): Uint8Array => {
  if (!window.encodePng) {
    throw new Error('WASM not initialized');
  }

  const result = window.encodePng(pixels, width, height, colorType, preset, lossy);
  
  if (typeof result === 'string' && result.startsWith('error:')) {
    throw new Error(result);
  }

  return result as Uint8Array;
};

export const getBytesPerPixel = (colorType: number): number => {
  if (!window.bytesPerPixel) {
    return 4; // Default to RGBA
  }
  return window.bytesPerPixel(colorType);
};
