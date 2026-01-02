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

@val external encodePng: ('a, int, int, int, int, bool) => array<int> = "window.encodePng"

let encodePngImage = (pixels: 'a, width: int, height: int, colorType: int): array<int> => {
  encodePng(pixels, width, height, colorType, 1, false)
}

let encodePngImageWithOptions = (pixels: 'a, width: int, height: int, colorType: int, preset: int, lossy: bool): array<int> => {
  encodePng(pixels, width, height, colorType, preset, lossy)
}
