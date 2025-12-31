# Rust, WASM, and the Web Demo (how this works)

This repo contains **two related things**:

1. **`pixo` (Rust library)**: the actual PNG/JPEG encoders + resizing code.
2. **`web/` (SvelteKit app)**: a demo UI that uses the Rust library **compiled to WebAssembly (WASM)** so it can run inside a browser.

If it feels confusing (“it’s Rust, so how can it run on a website?”), the key idea is:

> **The browser isn’t running Rust source code.**  
> It’s running a **WASM binary** that was **compiled from Rust**.

---

## 1) What “WASM” means in this project

**WebAssembly** is a portable, low-level binary format (like a tiny virtual CPU). Modern browsers can execute WASM efficiently and safely inside the page.

Rust can compile to WASM by targeting `wasm32-unknown-unknown`. That produces a `.wasm` file, which the browser can load.

In this repo, WASM is used so the exact same PNG/JPEG implementation written in Rust can run:

- **natively** (as a normal Rust library or CLI), and
- **in the browser** (via WASM), without rewriting compression code in JavaScript.

---

## 2) How the Rust library connects to JavaScript

WASM by itself is just a binary module with functions and linear memory. JavaScript needs a way to:

- load the module,
- pass data in/out (pixel bytes, encoded output bytes),
- and call exported functions.

This project uses `wasm-bindgen` to generate that “bridge”.

### The Rust side: exported functions

The file `src/wasm.rs` defines a **small API** that is easy to call from JS:

- `encodePng(...) -> Vec<u8>`
- `encodeJpeg(...) -> Vec<u8>`
- `resizeImage(...) -> Vec<u8>`
- `bytesPerPixel(...) -> u8`

Those functions are marked with `#[wasm_bindgen]`, which tells `wasm-bindgen`:

- “export this function to JS”
- “convert types like `&[u8]` / `Vec<u8>` to JS `Uint8Array`”

This is the correlation you’re asking about:

**Rust image compression code** (PNG/JPEG encoders)  
→ wrapped by a **thin WASM API layer** (`src/wasm.rs`)  
→ called by the **web app**.

---

## 3) How the WASM build is produced

There are three main build steps:

### Step A: Compile Rust to `.wasm`

Rust can produce a WASM module when you build for the WASM target:

- Target: `wasm32-unknown-unknown`
- Features: `wasm` (enables the WASM API module)

You can see the feature flag in `Cargo.toml`:

- `wasm = ["wasm-bindgen", "talc"]`

And the crate is configured to produce a suitable artifact:

- `crate-type = ["cdylib", "rlib"]`

### Step B: Run `wasm-bindgen` to generate JS glue

`wasm-bindgen` creates:

- a JS module (wrapper code that loads the WASM + marshals types), and
- a “bindgen output” `.wasm` (often named `*_bg.wasm`)

In this repo, those outputs go to:

- `web/src/lib/pixo-wasm/`

### Step C (optional): Optimize the WASM binary

The script optionally runs `wasm-opt -Oz` (from Binaryen) to reduce binary size.

### The automated script

Instead of running all steps manually, the repo provides:

- `web/scripts/build-wasm.mjs`

That script:

- runs `cargo build ... --target wasm32-unknown-unknown ...`
- runs `wasm-bindgen --target web ...`
- runs `wasm-opt` if available

---

## 4) How the web app uses the WASM module

The web demo lives in `web/` and is a normal SvelteKit/Vite app.

The important “bridge” file on the web side is:

- `web/src/lib/wasm.ts`

What it does:

1. **Dynamically imports** the generated module `web/src/lib/pixo-wasm/pixo.js`
2. Calls its default export to initialize the WASM module (`await init()`)
3. Stores the module so the app can call exported functions like `encodePng` / `encodeJpeg`

Then the UI code (e.g. `web/src/routes/+page.svelte`) does roughly:

1. Read an image file in the browser
2. Draw it to a canvas (or decode it)
3. Extract raw pixels (`ImageData`)
4. Call Rust-via-WASM to compress:
   - `encodePng(pixelBytes, width, height, ...)`
   - or `encodeJpeg(rgbBytes, width, height, ...)`
5. Wrap the returned bytes into a browser `Blob` so it can be downloaded / previewed

### Mental model (data flow)

```
User picks file
  -> Browser decodes to pixels (RGBA bytes)
     -> JS calls WASM function (encodePng/encodeJpeg)
        -> Rust encodes into PNG/JPEG bytes
           -> JS receives Uint8Array result
              -> Blob/ObjectURL for download + preview
```

---

## 5) Why there is a `pixo-web` Rust binary

In `Cargo.toml`, there is a binary target:

- `src/bin/web.rs` (named `pixo-web`)

This exists mainly as a **dedicated entrypoint for WASM builds**. It re-exports the WASM functions when the `wasm` feature is enabled.

In practice, the web build script compiles the crate for WASM and then uses `wasm-bindgen` to generate the JS wrapper used by the SvelteKit app.

---

## 6) What is “special” about running in the browser

Some important differences vs normal native Rust:

- **No direct filesystem/network access** inside WASM (unless JS provides it).
- **No DOM access** from Rust: JS owns the UI; Rust just computes bytes.
- **Memory is managed differently**: this repo uses `talc` as a global allocator on wasm32
  to keep the WASM binary small and manage heap allocations correctly.

So the web app is still “a JavaScript app”, but it uses Rust/WASM as a high-performance
compression engine.

---

## One question for you (so I can tailor the explanation)

When you say “confuse”, what part is the most confusing?

1. “How can Rust compile into something browsers can run?”
2. “How can JS call functions inside WASM and pass arrays back/forth?”
3. “How is the web demo wired up in this repo (which files do what)?”

