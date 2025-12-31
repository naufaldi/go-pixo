# pixo Feature Reference

This document catalogs all features implemented in the pixo Rust library. Use this as a reference to compare against go-pixo implementation progress.

**Source**: `/Users/mac/WebApps/projects/pixo`  
**Last Updated**: 2025-12-31

---

## Overview

pixo is a minimal-dependency, high-performance image compression library written in pure Rust. It implements PNG and JPEG encoders from scratch without relying on C/C++ codecs.

### Key Metrics
- **WASM Binary Size**: ~159 KB (competitive compression)
- **Code Coverage**: 86%
- **Tests**: 965 tests
- **License**: MIT

### Feature Flags
- `simd` (default): SIMD-accelerated kernels with runtime detection
- `parallel` (default): Parallel row-level filtering via rayon
- `wasm`: WebAssembly bindings
- `cli`: Command-line encoder

---

## PNG Encoder Features

### Supported Color Types
| Color Type | Code | Bytes/Pixel | Notes |
|------------|------|-------------|-------|
| Grayscale | 0 | 1 | Single channel |
| Grayscale+Alpha | 1 | 2 | With transparency |
| RGB | 2 | 3 | True color |
| RGBA | 3 | 4 | True color + alpha |

### Filter Types (All 5 PNG Spec Filters)
1. **None** - No transformation (fastest)
2. **Sub** - Subtract left pixel: `b[x] - b[x-bpp]`
3. **Up** - Subtract above pixel: `b[x] - a[x]`
4. **Average** - Subtract average: `b[x] - floor((b[x-bpp]+a[x])/2)`
5. **Paeth** - Paeth predictor (best compression)

### Filter Selection Strategies
| Strategy | Description | Speed | Compression |
|----------|-------------|-------|-------------|
| `None` | Always use no filter | Fastest | Poor |
| `Sub` | Always use Sub filter | Fast | Moderate |
| `Up` | Always use Up filter | Fast | Moderate |
| `Average` | Always use Average filter | Fast | Moderate |
| `Paeth` | Always use Paeth filter | Fast | Good |
| `MinSum` | Per-row, minimum sum of absolute values | Medium | Good |
| `Adaptive` | Per-row, best filter per row | Slow | Best |
| `AdaptiveFast` | Adaptive with early cut and limited trials | Medium-Fast | Very Good |
| `Bigrams` | Per-row, minimize distinct bigrams | Slow | Best |

### Compression Levels (1-9)
- Level 1-2: Fast compression, larger files
- Level 3-5: Balanced speed/size
- Level 6-8: Slower, better compression
- Level 9: Maximum compression (Zopfli-style iterative refinement)

### Lossless Optimizations
| Optimization | Description | Effect |
|--------------|-------------|--------|
| `optimize_alpha` | Zero RGB when alpha=0 | Improves compressibility |
| `reduce_color_type` | RGB→Gray, RGBA→RGB/GrayAlpha when safe | Smaller files |
| `strip_metadata` | Remove tEXt, zTXt, iTXt, time chunks | Smaller files |
| `reduce_palette` | Optimize palette when ≤256 colors | Smaller indexed PNGs |

### Lossy PNG (Quantization)
| Feature | Description |
|---------|-------------|
| Palette Quantization | Reduce to max 256 colors using median cut |
| Auto Mode | Auto-detect when quantization is beneficial |
| Force Mode | Always quantize regardless of heuristics |
| Dithering | Floyd-Steinberg dithering for smooth gradients |
| Max Colors Configurable | 1-256 colors (default 256) |

### PNG Presets
```rust
// Preset 0: Fast - prioritizes speed
PngOptions::fast(width, height)
// - compression_level: 2
// - filter_strategy: AdaptiveFast
// - No optimizations

// Preset 1: Balanced - good tradeoff
PngOptions::balanced(width, height)
// - compression_level: 6
// - filter_strategy: Adaptive
// - All lossless optimizations enabled

// Preset 2: Max - maximum compression
PngOptions::max(width, height)
// - compression_level: 9
// - filter_strategy: Bigrams
// - optimal_compression: true (Zopfli-style)
// - All lossless optimizations enabled
```

### PNG API
```rust
// Basic encoding
png::encode(data: &[u8], options: &PngOptions) -> Result<Vec<u8>>

// Buffer reuse (avoids allocations)
png::encode_into(output: &mut Vec<u8>, data: &[u8], options: &PngOptions) -> Result<()>

// Builder pattern
PngOptions::builder(width, height)
    .color_type(ColorType::Rgba)
    .compression_level(6)
    .filter_strategy(FilterStrategy::Adaptive)
    .optimize_alpha(true)
    .reduce_color_type(true)
    .strip_metadata(true)
    .reduce_palette(true)
    .lossy(true)  // Enable quantization
    .build()
```

---

## JPEG Encoder Features

### Supported Color Types
| Color Type | Code | Bytes/Pixel | Notes |
|------------|------|-------------|-------|
| Grayscale | 0 | 1 | Single channel (Y only) |
| RGB | 2 | 3 | YCbCr conversion |

> Note: JPEG does not support alpha channel. RGBA must be converted to RGB first.

### DCT Implementations
| Type | Description | Quality | Speed |
|------|-------------|---------|-------|
| Integer DCT | Fixed-point arithmetic | Good | Fast |
| Floating-point DCT | Full precision | Best | Slow |

### Quantization
- Quality-based scaling (1-100)
- Standard luminance table
- Standard chrominance table
- Custom table support

### Chroma Subsampling
| Mode | Description | Compression | Quality Impact |
|------|-------------|-------------|----------------|
| S444 | 4:4:4, no subsampling | Baseline | Best |
| S420 | 4:2:0, 2×2 chroma downsample | ~30% smaller | Minimal |

### Huffman Coding
| Feature | Description |
|---------|-------------|
| Fixed Huffman | Standard JPEG tables (fast) |
| Dynamic Huffman | Custom tables from symbol frequencies |
| Optimized Huffman | Image-dependent (like mozjpeg optimize_coding) |

### Progressive JPEG
- Multiple scan encoding
- Spectral selection
- Successive approximation
- DC first, then AC in bands

### Trellis Quantization
- Rate-distortion optimization
- Per-block quantization tuning
- Significant size reduction at same quality

### JPEG Presets
```rust
// Preset 0: Fast - standard Huffman, 4:4:4, baseline
JpegOptions::fast(width, height, quality)

// Preset 1: Balanced - optimized Huffman, 4:4:4, baseline
JpegOptions::balanced(width, height, quality)

// Preset 2: Max - all optimizations enabled
JpegOptions::max(width, height, quality)
// - subsampling: S420
// - optimize_huffman: true
// - progressive: true
// - trellis_quant: true
```

### JPEG API
```rust
// Basic encoding
jpeg::encode(data: &[u8], options: &JpegOptions) -> Result<Vec<u8>>

// Buffer reuse
jpeg::encode_into(output: &mut Vec<u8>, data: &[u8], options: &JpegOptions) -> Result<()>

// Builder pattern
JpegOptions::builder(width, height)
    .color_type(ColorType::Rgb)
    .quality(85)
    .subsampling(Subsampling::S420)
    .optimize_huffman(true)
    .progressive(true)
    .trellis_quant(true)
    .restart_interval(Some(8))
    .build()
```

### JPEG Markers Implemented
- SOI (Start of Image) - 0xFFD8
- APP0 (JFIF) - 0xFFE0
- DQT (Define Quantization Table) - 0xFFDB
- SOF0 (Start of Frame - baseline) - 0xFFC0
- SOF2 (Start of Frame - progressive) - 0xFFC2
- DHT (Define Huffman Table) - 0xFFC4
- DRI (Define Restart Interval) - 0xFFDD
- SOS (Start of Scan) - 0xFFDA
- EOI (End of Image) - 0xFFD9

---

## Compression Algorithms

### LZ77
- Sliding window (32KB default)
- Greedy match finding
- Back-references (distance, length pairs)
- Configurable window size

### Huffman Coding
- Canonical Huffman table construction
- Dynamic block-type support
- Bit-level writing (MSB first)
- JPEG DC/AC encoding

### DEFLATE
- Stored blocks (uncompressed)
- Fixed Huffman blocks
- Dynamic Huffman blocks
- Optimal DEFLATE (Zopfli-style iterative)

### Checksums
| Algorithm | Used For |
|-----------|----------|
| CRC32 | PNG chunk integrity |
| Adler32 | Zlib/DEFLATE integrity |

---

## WebAssembly Features

### Exported Functions
| Function | Signature | Description |
|----------|-----------|-------------|
| `encodePng` | `(data, width, height, colorType, preset, lossy) -> Uint8Array` | Encode PNG |
| `encodeJpeg` | `(data, width, height, colorType, quality, preset, subsampling420) -> Uint8Array` | Encode JPEG |
| `bytesPerPixel` | `(colorType) -> u8` | Get bytes per pixel |
| `resizeImage` | `(data, srcW, srcH, dstW, dstH, colorType, algorithm) -> Uint8Array` | Resize image |

### WASM Build
```bash
cargo build --target wasm32-unknown-unknown --release --features wasm
wasm-bindgen --target web --out-dir web/src/lib/pixo-wasm \
  target/wasm32-unknown-unknown/release/pixo.wasm
```

### WASM Binary Size
- ~159 KB with competitive compression
- Uses talc allocator for smaller binary and proper memory management

---

## CLI Features

### Commands
```bash
# Encode PNG
pixo input.png -o output.png --preset balanced

# Encode JPEG
pixo input.jpg -o output.jpg --quality 85 --progressive

# Resize
pixo input.png --resize 800x600 -o output.png

# Verbose output
pixo input.png -v  # Shows filter usage histogram
```

### CLI Options
| Option | Description |
|--------|-------------|
| `--preset 0/1/2` | Fast/Balanced/Max |
| `--quality 1-100` | JPEG quality (default 75) |
| `--lossy` | Enable PNG quantization |
| `--progressive` | Progressive JPEG |
| `--subsampling` | JPEG chroma subsampling |
| `--strip` | Strip metadata |
| `-v` | Verbose (filter statistics) |

---

## Image Resizing

### Supported Algorithms
| Algorithm | Code | Quality | Speed |
|-----------|------|---------|-------|
| Nearest | 0 | Lowest | Fastest |
| Bilinear | 1 | Medium | Medium |
| Lanczos3 | 2 | Highest | Slowest |

### Resize API
```rust
resize::resize(
    data: &[u8],
    options: &ResizeOptions
) -> Result<Vec<u8>>

// Builder
ResizeOptions::builder(src_width, src_height)
    .dst(dst_width, dst_height)
    .color_type(ColorType::Rgba)
    .algorithm(ResizeAlgorithm::Lanczos3)
    .build()
```

---

## Performance Features

### SIMD Acceleration
| Architecture | Instruction Sets |
|--------------|------------------|
| x86_64 | AVX2, SSE |
| aarch64 | NEON |

### Parallel Processing
- Rayon-based parallel row filtering
- Configurable via `parallel` feature flag
- Automatic work division across cores

### Runtime Detection
- SIMD features detected at runtime
- Falls back to scalar implementation when unavailable
- No runtime overhead when SIMD unavailable

### Buffer Reuse
- `encode_into` variants reuse output buffers
- Reduces allocations for batch processing
- Important for WASM memory management

---

## Codebase Structure

```
src/
├── bits.rs              # Bit-level utilities
├── color.rs             # Color type, RGB<->YCbCr
├── compress/
│   ├── adler32.rs       # Adler32 checksum
│   ├── crc32.rs         # CRC32 checksum
│   ├── deflate.rs       # DEFLATE implementation
│   ├── huffman.rs       # Huffman coding
│   └── lz77.rs          # LZ77 matcher
├── decode/              # Decoding (PNG/JPEG) - CLI feature
├── error.rs             # Error types
├── jpeg/
│   ├── dct.rs           # DCT implementation
│   ├── huffman.rs       # JPEG Huffman
│   ├── mod.rs           # JPEG encoder
│   ├── progressive.rs   # Progressive encoding
│   ├── quantize.rs      # Quantization
│   └── trellis.rs       # Trellis quantization
├── lib.rs               # Library root, exports
├── png/
│   ├── bit_depth.rs     # Bit depth handling
│   ├── chunk.rs         # PNG chunk writing
│   ├── filter.rs        # Filter implementations
│   └── mod.rs           # PNG encoder
├── resize.rs            # Image resizing
├── simd/                # SIMD implementations
│   ├── aarch64.rs       # NEON kernels
│   ├── fallback.rs      # Scalar fallback
│   ├── mod.rs           # SIMD dispatch
│   └── x86_64.rs        # AVX2/SSE kernels
└── wasm.rs              # WASM bindings
```

---

## go-pixo Feature Gap Analysis

### PNG Features Status

| Feature | pixo | go-pixo | Priority |
|---------|------|---------|----------|
| PNG signature/chunks | ✓ | Phase 1 | High |
| CRC32 | ✓ | Phase 1 | High |
| Filter types (5) | ✓ | Phase 3 | High |
| Filter strategies (5) | ✓ | Phase 3 | Medium |
| Compression levels (1-9) | ✓ | Phase 2 | Medium |
| Optimize alpha | ✓ | Phase 4 | Medium |
| Reduce color type | ✓ | Phase 4 | Medium |
| Strip metadata | ✓ | Phase 4 | Low |
| Palette reduction | ✓ | Phase 5 | Medium |
| Quantization | ✓ | Phase 5 | Medium |
| Dithering | ✓ | Phase 5 | Low |
| Optimal compression | ✓ | Future | Low |

### JPEG Features Status

| Feature | pixo | go-pixo | Priority |
|---------|------|---------|----------|
| Baseline DCT | ✓ | Phase 6 | High |
| RGB→YCbCr | ✓ | Phase 6 | High |
| Quantization tables | ✓ | Phase 6 | High |
| Zigzag reordering | ✓ | Phase 6 | Medium |
| DC differential | ✓ | Phase 6 | Medium |
| AC RLE | ✓ | Phase 6 | Medium |
| Fixed Huffman | ✓ | Phase 6 | Medium |
| Dynamic Huffman | ✓ | Phase 7 | Low |
| Chroma subsampling | ✓ | Phase 7 | Medium |
| Optimized Huffman | ✓ | Phase 7 | Low |
| Progressive | ✓ | Phase 7 | Low |
| Trellis quantization | ✓ | Phase 7 | Low |

### Compression Features Status

| Feature | pixo | go-pixo | Priority |
|---------|------|---------|----------|
| LZ77 matcher | ✓ | Phase 2 | High |
| Huffman coding | ✓ | Phase 2 | High |
| DEFLATE blocks | ✓ | Phase 2 | High |
| Adler32 | ✓ | Phase 1 | High |
| Optimal DEFLATE | ✓ | Future | Low |

### Platform Features Status

| Feature | pixo | go-pixo | Priority |
|---------|------|---------|----------|
| WASM bindings | ✓ | Phase 0 | Done |
| Buffer reuse | ✓ | Planning | Medium |
| SIMD | ✓ | Not planned | N/A |
| Parallel processing | ✓ | Not planned | N/A |
| CLI | ✓ | Future | Low |

---

## API Comparison

### pixo API (Rust)
```rust
// PNG
png::encode(data, &PngOptions) -> Result<Vec<u8>>
png::encode_into(&mut Vec, data, &PngOptions) -> Result<()>

// JPEG
jpeg::encode(data, &JpegOptions) -> Result<Vec<u8>>
jpeg::encode_into(&mut Vec, data, &JpegOptions) -> Result<()>

// Resize
resize::resize(data, &ResizeOptions) -> Result<Vec<u8>>

// Color types
ColorType::Gray, GrayAlpha, Rgb, Rgba
ColorType::bytes_per_pixel()
```

### go-pixo Target API (Go)
```go
// PNG
EncodePng(pixels []byte, width, height, colorType, preset int, lossy bool) ([]byte, error)

// JPEG
EncodeJpeg(pixels []byte, width, height, colorType, quality, preset int, subsampling420 bool) ([]byte, error)

// Utility
BytesPerPixel(colorType int) int
```

### go-pixo Target WASM API (JavaScript)
```javascript
// From src/wasm/bridge.go
encodePng(pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean): Uint8Array
bytesPerPixel(colorType: number): number
```

---

## References

- pixo crates.io: https://crates.io/crates/pixo
- pixo docs.rs: https://docs.rs/pixo
- pixo playground: https://pixo.leerob.com
- pixo GitHub: https://github.com/leerob/pixo
