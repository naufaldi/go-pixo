// WASM bindings for Go-Pixo
// These will be used by ReScript components

@val external window: 'a = "window"

type wasmState =
  | NotReady
  | Ready
  | Error(string)

let initWasm = (): Promise.t<unit> => {
  %raw("
    new Promise((resolve, reject) => {
      const go = new window.Go();
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
    })
  ")
}

@val external encodePng: ('a, int, int, int, int, bool, int) => array<int> = "window.encodePng"
@val
external encodePngAdvanced: (
  'a,
  int,
  int,
  int,
  int,
  bool,
  int,
  bool,
  float,
  int,
  int,
  int,
  bool,
  int,
  bool,
  bool,
  option<(string, int) => unit>,
) => array<int> = "window.encodePngAdvanced"
@val external encodeJpeg: ('a, int, int, int, int) => array<int> = "window.encodeJpeg"
@val
external encodeJpegAdvanced: (
  'a,
  int,
  int,
  int,
  int,
  int,
  bool,
  bool,
  bool,
  int,
) => array<int> = "window.encodeJpegAdvanced"
@val
external recompressPngLossless: (
  'a,
  int,
  int,
  option<(string, int) => unit>,
) => array<int> = "window.recompressPngLossless"
@val external bytesPerPixel: int => int = "window.bytesPerPixel"

let encodePngImage = (pixels: 'a, width: int, height: int, colorType: int): array<int> => {
  encodePng(pixels, width, height, colorType, 1, false, 0)
}

let encodePngImageWithOptions = (
  pixels: 'a,
  width: int,
  height: int,
  colorType: int,
  preset: int,
  lossy: bool,
  maxColors: int,
): array<int> => {
  encodePng(pixels, width, height, colorType, preset, lossy, maxColors)
}
