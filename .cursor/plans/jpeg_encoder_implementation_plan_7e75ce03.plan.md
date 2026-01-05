---
name: JPEG Encoder Implementation Plan
overview: Implement a complete JPEG encoder in Go (go-pixo) based on the Rust reference implementation (pixo), covering baseline JPEG encoding, progressive encoding, advanced optimizations (trellis quantization, optimized Huffman tables), and full integration with WASM and CLI. The implementation will match PNG's feature set including presets, options builder, and advanced compression techniques.
todos:
  - id: jpeg-6.1-infra
    content: "Create JPEG infrastructure: constants.go (markers, types), errors.go (error types), color.go (RGB to YCbCr conversion)"
    status: pending
  - id: jpeg-6.2-blocks
    content: "Implement block extraction: blocks.go (8x8 blocks for 4:4:4), mcu.go (16x16 MCU for 4:2:0 subsampling)"
    status: pending
    dependencies:
      - jpeg-6.1-infra
  - id: jpeg-6.3-dct
    content: "Implement DCT: dct.go with ForwardDCT and InverseDCT using AAN algorithm"
    status: pending
    dependencies:
      - jpeg-6.2-blocks
  - id: jpeg-6.4-quantize
    content: "Implement quantization: quantize.go with quality scaling and QuantizeBlock function"
    status: pending
    dependencies:
      - jpeg-6.3-dct
  - id: jpeg-6.5-zigzag
    content: "Implement zigzag reordering: zigzag.go with ZigzagReorder and Dezigzag functions"
    status: pending
    dependencies:
      - jpeg-6.4-quantize
  - id: jpeg-6.6-dc
    content: "Implement DC encoding: dc.go with EncodeDC function for differential DC encoding"
    status: pending
    dependencies:
      - jpeg-6.5-zigzag
  - id: jpeg-6.7-ac
    content: "Implement AC encoding: ac.go with RunLengthEncode for zero runs and non-zero coefficients"
    status: pending
    dependencies:
      - jpeg-6.6-dc
  - id: jpeg-6.8-huffman
    content: "Implement standard Huffman tables: huffman.go with standard JPEG Huffman tables and encoding functions"
    status: pending
    dependencies:
      - jpeg-6.7-ac
  - id: jpeg-6.9-bitwriter
    content: "Implement JPEG bit writer: bit_writer.go with MSB-first bit ordering and byte stuffing"
    status: pending
    dependencies:
      - jpeg-6.8-huffman
  - id: jpeg-6.10-markers
    content: "Implement JPEG markers: markers.go with SOI, EOI, APP0, DQT, SOF0/SOF2, DHT, SOS, DRI writing functions"
    status: pending
    dependencies:
      - jpeg-6.9-bitwriter
  - id: jpeg-6.11-encoder
    content: "Create baseline JPEG encoder: encoder.go with NewEncoder and Encode methods, comprehensive tests, WASM bridge integration"
    status: pending
    dependencies:
      - jpeg-6.10-markers
  - id: jpeg-7.1-subsample
    content: "Implement chroma subsampling: subsample.go with 4:2:0 subsampling support"
    status: pending
    dependencies:
      - jpeg-6.11-encoder
  - id: jpeg-7.2-optimized-huffman
    content: "Implement optimized Huffman tables: huffman_optimized.go with frequency-based table generation"
    status: pending
    dependencies:
      - jpeg-7.1-subsample
  - id: jpeg-7.3-progressive
    content: "Implement progressive JPEG: progressive.go with scan scripts and multi-scan encoding"
    status: pending
    dependencies:
      - jpeg-7.2-optimized-huffman
  - id: jpeg-7.4-presets
    content: "Implement presets and options: options.go, options_builder.go with Fast/Balanced/Max presets"
    status: pending
    dependencies:
      - jpeg-7.3-progressive
  - id: jpeg-8.1-trellis
    content: "Implement trellis quantization: trellis.go with Viterbi algorithm for rate-distortion optimization"
    status: pending
    dependencies:
      - jpeg-7.4-presets
  - id: jpeg-8.2-quality
    content: Enhance quality-based optimizations in quantize.go
    status: pending
    dependencies:
      - jpeg-8.1-trellis
  - id: jpeg-8.3-progressive-scans
    content: Optimize progressive scan scripts in progressive.go
    status: pending
    dependencies:
      - jpeg-8.2-quality
  - id: jpeg-8.4-wasm
    content: "Update WASM bridge and web UI: bridge.go with EncodeJpegAdvanced, worker.ts and UI components updated"
    status: pending
    dependencies:
      - jpeg-8.3-progressive-scans
  - id: jpeg-8.5-cli
    content: "Update CLI for JPEG: main.go with -format jpeg, -quality, -preset, -progressive, -subsampling, -trellis flags"
    status: pending
    dependencies:
      - jpeg-8.4-wasm
  - id: jpeg-9-docs
    content: "Create JPEG documentation: jpeg.md (overview), encoder.md (architecture), dct.md (DCT theory), quantization.md (quality scaling), progressive.md (progressive encoding), trellis.md (R-D optimization), huffman.md (Huffman tables), subsampling.md (chroma subsampling), index.md (documentation index), update png-vs-jpeg.md"
    status: pending
    dependencies:
      - jpeg-8.5-cli
isProject: false
---

# JPEG Encoder Implementation Plan for go-pixo

## User Stories

**US-1: Baseline JPEG Encoding**

As a developer, I want to encode RGB/Grayscale images to JPEG format so that I can compress photos with standard JPEG compatibility.

**US-2: Quality Control**

As a user, I want to control JPEG quality (1-100) so that I can balance file size and image quality.

**US-3: Advanced Compression**

As a user, I want to use advanced JPEG features (progressive encoding, trellis quantization, optimized Huffman) so that I can achieve better compression ratios.

**US-4: Preset System**

As a user, I want to use presets (Fast, Balanced, Max) so that I can quickly choose compression settings without detailed configuration.

**US-5: Web Integration**

As a web developer, I want to use JPEG encoding via WASM so that I can compress images client-side in the browser.

**US-6: CLI Support**

As a user, I want to compress JPEG files via command line so that I can batch process images.

## Acceptance Criteria

- AC-1: Encode valid baseline JPEG files that open in standard image viewers
- AC-2: Support quality levels 1-100 with proper quantization scaling
- AC-3: Support both RGB and Grayscale color types
- AC-4: Implement progressive JPEG encoding with multiple scans
- AC-5: Implement trellis quantization for 5-15% better compression
- AC-6: Generate optimized Huffman tables from image data
- AC-7: Support chroma subsampling (4:2:0 and 4:4:4)
- AC-8: Provide Fast, Balanced, and Max presets matching PNG behavior
- AC-9: Expose all features via WASM bridge
- AC-10: Add CLI support with quality and preset flags
- AC-11: All tests pass with `go test ./...`
- AC-12: All linting passes with `golangci-lint run`

## Architecture Overview

The JPEG encoder follows the same architectural patterns as the PNG encoder:

```
src/jpeg/
├── constants.go          # JPEG markers, constants
├── color.go              # RGB to YCbCr conversion
├── blocks.go             # 8x8 block extraction
├── dct.go                # Discrete Cosine Transform
├── quantize.go           # Quantization tables and operations
├── zigzag.go             # Zigzag reordering
├── dc.go                 # DC coefficient encoding
├── ac.go                 # AC coefficient run-length encoding
├── huffman.go            # JPEG Huffman tables (standard + optimized)
├── bit_writer.go         # MSB-first bit writer (JPEG format)
├── markers.go            # JPEG marker writing (SOI, EOI, APP0, DQT, SOF0/SOF2, DHT, SOS, DRI)
├── progressive.go        # Progressive scan encoding
├── trellis.go            # Trellis quantization (R-D optimization)
├── subsample.go          # Chroma subsampling (4:2:0)
├── options.go            # Options struct and presets
├── options_builder.go    # Fluent options builder
└── encoder.go            # Main encoder entry point
```

**Key Differences from PNG:**

- JPEG uses **MSB-first** bit ordering (vs PNG's LSB-first for DEFLATE)
- JPEG uses **YCbCr** color space (vs PNG's RGB/RGBA)
- JPEG is **lossy by nature** (DCT + quantization)
- JPEG has **progressive encoding** built-in (vs PNG's sequential)

## Implementation Phases

### Phase 6: Baseline JPEG Encoder (Correctness-First)

**Testing Requirements for Phase 6:**

- **Each task must include its own unit tests** (`*_test.go` files)
- After completing each task, run: `go test ./src/jpeg/...` (must pass)
- After completing each task, run: `golangci-lint run ./src/jpeg/...` (must pass, no warnings)
- Encoder output must decode correctly with Go's `image/jpeg` decoder
- Output must open in standard image viewers

#### 6.1 JPEG Infrastructure

**Task 6.1.1**: Create `src/jpeg/constants.go`

- Define JPEG marker constants (SOI=0xFFD8, EOI=0xFFD9, APP0=0xFFE0, DQT=0xFFDB, SOF0=0xFFC0, SOF2=0xFFC2, DHT=0xFFC4, SOS=0xFFDA, DRI=0xFFDD)
- Define `ColorType` constants (Grayscale=1, RGB=3)
- Define `Subsampling` type (S444, S420)
- Test: verify constants are correct values
- Output: `src/jpeg/constants.go`, `src/jpeg/constants_test.go`

**Task 6.1.2**: Create `src/jpeg/errors.go`

- Define JPEG-specific error types
- Errors: invalid quality, invalid dimensions, unsupported color type, invalid data length
- Test: verify error messages are descriptive
- Output: `src/jpeg/errors.go`, `src/jpeg/errors_test.go`

**Task 6.1.3**: Create `src/jpeg/color.go`

- Add `RGBToYCbCr(r, g, b uint8) (y, cb, cr uint8)` function
- Implement ITU-R BT.601 conversion using fixed-point arithmetic
- Formula: Y = (77*R + 150*G + 29*B + 128) >> 8
- Add `YCbCrToRGB(y, cb, cr uint8) (r, g, b uint8)` for testing
- Test: round-trip conversion accuracy
- Output: `src/jpeg/color.go`, `src/jpeg/color_test.go`

#### 6.2 Block Splitting

**Task 6.2.1**: Create `src/jpeg/blocks.go`

- Add `ExtractBlock(data []byte, width, height, blockX, blockY int, colorType ColorType) ([64]float32, [64]float32, [64]float32)` function
- Extract 8x8 block from image data
- Handle edge padding (replicate last pixel)
- Convert RGB to YCbCr during extraction
- Level-shift to -128..127 range for DCT
- Output: `src/jpeg/blocks.go`, `src/jpeg/blocks_test.go`

**Task 6.2.2**: Create `src/jpeg/mcu.go` for 4:2:0 subsampling

- Add `ExtractMCU420(data []byte, width, height, mcuX, mcuY int) ([4][64]float32, [64]float32, [64]float32)` function
- Extract 16x16 MCU with 4 Y blocks and 1 Cb/Cr block each
- Average chroma over 2x2 regions
- Output: `src/jpeg/mcu.go`, `src/jpeg/mcu_test.go`

#### 6.3 DCT Implementation

**Task 6.3.1**: Create `src/jpeg/dct.go`

- Add `ForwardDCT(block [64]float32) [64]float32` function
- Implement 2D DCT using AAN (Arai-Agui-Nakajima) algorithm
- Process 8x8 blocks (row-wise then column-wise)
- Use floating-point for accuracy (integer version can be added later)
- Add `InverseDCT(block [64]float32) [64]float32` for testing
- Test: IDCT(DCT(x)) ≈ x (within tolerance)
- Output: `src/jpeg/dct.go`, `src/jpeg/dct_test.go`

#### 6.4 Quantization

**Task 6.4.1**: Create `src/jpeg/quantize.go`

- Define standard JPEG quantization tables (luminance and chrominance)
- Add `QuantizationTables` struct with quality scaling
- Add `NewQuantizationTables(quality uint8) *QuantizationTables` constructor
- Implement quality scaling: scale = (quality < 50) ? 5000/quality : 200 - 2*quality
- Scale tables and clamp to 1-255 range
- Store tables in both zigzag and natural order
- Output: `src/jpeg/quantize.go`, `src/jpeg/quantize_test.go`

**Task 6.4.2**: Add quantization operations

- Add `QuantizeBlock(dct [64]float32, table [64]float32) [64]int16` function
- Round DCT coefficients divided by quantization values
- Test: verify quantization produces expected values, test edge cases (zero, negative, large values)
- Output: `src/jpeg/quantize.go` (updated), `src/jpeg/quantize_test.go` (updated)

#### 6.5 Zigzag Reordering

**Task 6.5.1**: Create `src/jpeg/zigzag.go`

- Define zigzag scan order array `[64]int`
- Add `ZigzagReorder(block [64]int16) [64]int16` function
- Reorder quantized coefficients to zigzag order
- Add `Dezigzag(coeffs [64]int16) [64]int16` for testing
- Test: zigzag then dezigzag = original
- Output: `src/jpeg/zigzag.go`, `src/jpeg/zigzag_test.go`

#### 6.6 DC Encoding

**Task 6.6.1**: Create `src/jpeg/dc.go`

- Add `EncodeDC(dc int16, prevDC int16) (category uint8, diffBits uint16, bitLen uint8)` function
- Compute DC difference: diff = dc - prevDC
- Calculate category (bit length needed): category = bits needed for |diff|
- Encode category using Huffman table
- Encode diff value in two's complement
- Add `DecodeDC` for testing
- Output: `src/jpeg/dc.go`, `src/jpeg/dc_test.go`

#### 6.7 AC Encoding

**Task 6.7.1**: Create `src/jpeg/ac.go`

- Add `RunLengthEncode(coeffs [64]int16) []ACRun` function
- Define `ACRun` struct: `{RunLength uint8, Size uint8, Value int16}`
- Encode zero runs and non-zero coefficients
- Handle EOB (End of Block) marker
- Handle ZRL (Zero Run Length) for runs >= 16
- Add `RunLengthDecode` for testing
- Output: `src/jpeg/ac.go`, `src/jpeg/ac_test.go`

#### 6.8 Huffman Tables

**Task 6.8.1**: Create `src/jpeg/huffman.go`

- Define standard JPEG Huffman tables (DC luminance, DC chrominance, AC luminance, AC chrominance)
- Add `HuffmanTables` struct with lookup tables
- Add `NewHuffmanTables() *HuffmanTables` constructor with standard tables
- Build code lookup tables for fast encoding
- Add encoding functions: `EncodeDC(category uint8, isLuminance bool) (code uint16, length uint8)`
- Add encoding functions: `EncodeAC(run, size uint8, isLuminance bool) (code uint16, length uint8)`
- Output: `src/jpeg/huffman.go`, `src/jpeg/huffman_test.go`

#### 6.9 Bit Writer for JPEG

**Task 6.9.1**: Create `src/jpeg/bit_writer.go`

- Define `BitWriter` struct (MSB-first, different from DEFLATE's LSB-first)
- Add `Write(bits uint16, n int) error` method
- Add `WriteByte(b byte) error` method
- Handle byte stuffing: 0xFF → 0xFF 0x00
- Add `Flush() error` method
- Add `Finish() []byte` method
- Test: write bits, verify byte output, byte stuffing
- Output: `src/jpeg/bit_writer.go`, `src/jpeg/bit_writer_test.go`

#### 6.10 Markers

**Task 6.10.1**: Create `src/jpeg/markers.go`

- Add `WriteSOI(w io.Writer) error` - Start of Image (0xFFD8)
- Add `WriteEOI(w io.Writer) error` - End of Image (0xFFD9)
- Add `WriteAPP0(w io.Writer) error` - JFIF header
- Add `WriteDQT(w io.Writer, tableID uint8, table [64]uint8) error` - Quantization table
- Add `WriteSOF0(w io.Writer, width, height uint16, colorType ColorType, subsampling Subsampling) error` - Baseline frame
- Add `WriteSOF2(w io.Writer, width, height uint16, colorType ColorType, subsampling Subsampling) error` - Progressive frame
- Add `WriteDHT(w io.Writer, tableID uint8, bits [16]uint8, vals []uint8) error` - Huffman table
- Add `WriteSOS(w io.Writer, colorType ColorType) error` - Start of Scan (baseline)
- Add `WriteSOSProgressive(w io.Writer, scan *ScanSpec, colorType ColorType) error` - Progressive scan
- Add `WriteDRI(w io.Writer, interval uint16) error` - Restart interval
- Test: write markers, verify format, marker lengths
- Output: `src/jpeg/markers.go`, `src/jpeg/markers_test.go`

#### 6.11 JPEG Encoder Entry Point

**Task 6.11.1**: Create `src/jpeg/encoder.go`

- Define `Encoder` struct (width, height, colorType, quality)
- Add `NewEncoder(width, height int, colorType ColorType, quality uint8) (*Encoder, error)` constructor
- Add `Encode(pixels []byte) ([]byte, error)` method
- Sequence: RGB→YCbCr → blocks → DCT → quantize → zigzag → Huffman → markers
- Write: SOI → APP0 → DQT → SOF0 → DHT → SOS → scan data → EOI
- Validate input dimensions and pixel count
- Test: 1×1 RGB image, 1×1 Grayscale image, 8×8 RGB image, 16×16 RGB image, various quality levels (1, 25, 50, 75, 100), non-multiple-of-8 dimensions (edge padding), verify output decodes with Go's `image/jpeg` decoder
- Output: `src/jpeg/encoder.go`, `src/jpeg/encoder_test.go`

**Task 6.11.2**: Update WASM bridge

- Add `EncodeJpeg` function to `src/wasm/bridge.go`
- Support quality parameter
- Support RGB and Grayscale color types
- Test: verify WASM bridge function works correctly
- Output: `src/wasm/bridge.go` (updated), `src/wasm/bridge_test.go` (updated)

### Phase 7: Advanced JPEG Features

**Testing Requirements for Phase 7:**

- **Each task must include its own unit tests** (`*_test.go` files)
- After completing each task, run: `go test ./src/jpeg/...` (must pass)
- After completing each task, run: `golangci-lint run ./src/jpeg/...` (must pass, no warnings)
- All Phase 6 tests must continue to pass
- New feature tests for subsampling, optimized Huffman, progressive encoding
- Verify optimized features produce smaller files than baseline
- Progressive JPEG must decode correctly

#### 7.1 Chroma Subsampling

**Task 7.1.1**: Create `src/jpeg/subsample.go`

- Add `Subsample420(cb, cr []byte, width, height int) ([]byte, []byte)` function
- Average every 2×2 block of chroma samples
- Update encoder to support 4:2:0 subsampling
- Update MCU extraction for 4:2:0
- Test: verify subsampling reduces chroma data by 4x
- Output: `src/jpeg/subsample.go`, `src/jpeg/subsample_test.go`

#### 7.2 Optimized Huffman Tables

**Task 7.2.1**: Create `src/jpeg/huffman_optimized.go`

- Add `BuildOptimizedTables(data []byte, width, height int, colorType ColorType, subsampling Subsampling, quantTables *QuantizationTables) *HuffmanTables` function
- Process all blocks to count symbol frequencies
- Build custom Huffman tables from frequencies
- Use same Huffman tree building logic as PNG (reuse `compress/huffman_tree.go` concepts)
- Generate canonical codes
- Test: verify optimized tables produce smaller files than standard tables
- Output: `src/jpeg/huffman_optimized.go`, `src/jpeg/huffman_optimized_test.go`

#### 7.3 Progressive JPEG

**Task 7.3.1**: Create `src/jpeg/progressive.go`

- Define `ScanSpec` struct: `{Components []uint8, SS uint8, SE uint8, AH uint8, AL uint8}`
- Add `DefaultProgressiveScript() []ScanSpec` function
- Add `SimpleProgressiveScript() []ScanSpec` function
- Add `EncodeDCScan(writer *BitWriter, scan *ScanSpec, coeffs [][64]int16, huffTables *HuffmanTables)` function
- Add `EncodeACFirstScan(writer *BitWriter, scan *ScanSpec, coeffs [][64]int16, huffTables *HuffmanTables)` function
- Add `EncodeACRefineScan(writer *BitWriter, scan *ScanSpec, coeffs [][64]int16, huffTables *HuffmanTables)` function
- Implement spectral selection and successive approximation
- Test: verify progressive JPEG opens in browsers
- Output: `src/jpeg/progressive.go`, `src/jpeg/progressive_test.go`

**Task 7.3.2**: Update encoder for progressive mode

- Modify `encoder.go` to support progressive encoding
- Compute all DCT coefficients first
- Encode multiple scans using progressive script
- Write SOF2 marker instead of SOF0
- Test: verify progressive JPEG encodes and decodes correctly, test with various image sizes
- Output: `src/jpeg/encoder.go` (updated), `src/jpeg/encoder_test.go` (updated)

#### 7.4 JPEG Presets and Options

**Task 7.4.1**: Create `src/jpeg/options.go`

- Define `Options` struct: `{Width, Height int, ColorType ColorType, Quality uint8, Subsampling Subsampling, OptimizeHuffman bool, Progressive bool, TrellisQuant bool, RestartInterval *uint16}`
- Define `Preset` type (Fast, Balanced, Max)
- Add `FastOptions(width, height int, quality uint8) Options` function
- Add `BalancedOptions(width, height int, quality uint8) Options` function
- Add `MaxOptions(width, height int, quality uint8) Options` function
- Test: verify preset configurations are correct, test all preset options
- Output: `src/jpeg/options.go`, `src/jpeg/options_test.go`

**Task 7.4.2**: Create `src/jpeg/options_builder.go`

- Define `OptionsBuilder` struct
- Add chainable methods: `Quality()`, `Subsampling()`, `OptimizeHuffman()`, `Progressive()`, `TrellisQuant()`, `RestartInterval()`, `Preset()`
- Add `Build() Options` method
- Test: verify preset configurations
- Output: `src/jpeg/options_builder.go`, `src/jpeg/options_builder_test.go`

**Task 7.4.3**: Update encoder to use Options

- Modify `NewEncoder` to accept `Options`
- Update `Encode` to use options for all settings
- Test: verify encoder works with all option combinations
- Output: `src/jpeg/encoder.go` (updated), `src/jpeg/encoder_test.go` (updated)

### Phase 8: Advanced JPEG Optimizations

**Testing Requirements for Phase 8:**

- **Each task must include its own unit tests** (`*_test.go` files)
- After completing each task, run: `go test ./src/jpeg/...` (must pass)
- After completing each task, run: `golangci-lint run ./src/jpeg/...` (must pass, no warnings)
- All previous tests must continue to pass
- Trellis quantization must show 5-15% compression improvement
- WASM bridge tests must pass
- CLI tests must pass
- Web UI must work correctly

#### 8.1 Trellis Quantization

**Task 8.1.1**: Create `src/jpeg/trellis.go`

- Add `TrellisQuantize(dct [64]float32, quantTable [64]float32, lambda float32) [64]int16` function
- Implement Viterbi algorithm for rate-distortion optimization
- Consider multiple candidate quantized values per coefficient
- Track zero runs for accurate EOB prediction
- Use cost model: cost = rate + lambda * distortion
- Test: verify trellis produces 5-15% better compression
- Output: `src/jpeg/trellis.go`, `src/jpeg/trellis_test.go`

**Task 8.1.2**: Integrate trellis into encoder

- Add trellis option to `Options`
- Use trellis quantization when enabled
- Test: verify trellis integration works, compare file sizes with/without trellis
- Output: `src/jpeg/encoder.go` (updated), `src/jpeg/encoder_test.go` (updated)

#### 8.2 Quality-Based Optimizations

**Task 8.2.1**: Enhance quantization for quality control

- Improve quality scaling algorithm
- Add quality presets (low, medium, high, maximum)
- Test: verify quality levels produce expected file sizes, test edge cases
- Output: `src/jpeg/quantize.go` (updated), `src/jpeg/quantize_test.go` (updated)

#### 8.3 Enhanced Progressive Scans

**Task 8.3.1**: Optimize progressive scan scripts

- Create multiple scan script presets
- Fine-tune spectral selection ranges
- Optimize for different image types
- Test: verify optimized scan scripts produce better compression
- Output: `src/jpeg/progressive.go` (updated), `src/jpeg/progressive_test.go` (updated)

#### 8.4 WASM Integration

**Task 8.4.1**: Update WASM bridge for advanced features

- Add `EncodeJpegAdvanced` function with all options
- Support quality, subsampling, progressive, trellis, optimized Huffman
- Add progress callback support
- Map presets: Smaller→MaxOptions, Balanced→BalancedOptions, Faster→FastOptions
- Test: verify all advanced options work via WASM bridge
- Output: `src/wasm/bridge.go` (updated), `src/wasm/bridge_test.go` (updated)

**Task 8.4.2**: Update web UI for JPEG

- Add JPEG support to `web/src/worker.ts`
- Add JPEG controls to `web/src/App.res` and `BottomBar.res`
- Support quality slider, progressive toggle, subsampling options
- Output: `web/src/worker.ts` (updated), `web/src/App.res` (updated), `web/src/components/BottomBar.res` (updated)

#### 8.5 CLI Integration

**Task 8.5.1**: Update CLI for JPEG

- Add `-format jpeg` flag to `src/cmd/cli/main.go`
- Add `-quality` flag (1-100)
- Add `-preset` flag (fast, balanced, max)
- Add `-progressive` flag
- Add `-subsampling` flag (444, 420)
- Add `-trellis` flag
- Add `-optimize-huffman` flag
- Test: verify all flags work correctly, test CLI with various combinations
- Output: `src/cmd/cli/main.go` (updated), `src/cmd/cli/main_test.go` (updated)

### Phase 9: Documentation

**Task 9.1**: Create JPEG overview documentation

- Create `docs/learning/jpg/jpeg.md`
- Explain JPEG format basics
- Compare JPEG vs PNG (reference existing `png-vs-jpeg.md`)
- Explain lossy compression concept
- Document JPEG markers and structure
- Output: `docs/learning/jpg/jpeg.md`

**Task 9.2**: Create JPEG encoder documentation

- Create `docs/learning/jpg/encoder.md`
- Document encoder architecture and pipeline
- Explain RGB to YCbCr conversion
- Document block extraction and DCT process
- Explain quantization and quality scaling
- Document encoding flow: blocks → DCT → quantize → zigzag → Huffman → markers
- Output: `docs/learning/jpg/encoder.md`

**Task 9.3**: Create DCT documentation

- Create `docs/learning/jpg/dct.md`
- Explain Discrete Cosine Transform theory
- Document AAN algorithm implementation
- Explain why DCT is used for image compression
- Show DCT coefficient visualization
- Output: `docs/learning/jpg/dct.md`

**Task 9.4**: Create quantization documentation

- Create `docs/learning/jpg/quantization.md`
- Explain quantization tables and quality scaling
- Document standard JPEG quantization tables
- Explain how quality affects file size and visual quality
- Show quality level comparisons
- Output: `docs/learning/jpg/quantization.md`

**Task 9.5**: Create progressive JPEG documentation

- Create `docs/learning/jpg/progressive.md`
- Explain progressive vs baseline JPEG
- Document spectral selection and successive approximation
- Explain progressive scan scripts
- Show compression benefits of progressive encoding
- Output: `docs/learning/jpg/progressive.md`

**Task 9.6**: Create trellis quantization documentation

- Create `docs/learning/jpg/trellis.md`
- Explain rate-distortion optimization
- Document Viterbi algorithm for trellis quantization
- Explain how trellis improves compression (5-15%)
- Show compression ratio comparisons
- Output: `docs/learning/jpg/trellis.md`

**Task 9.7**: Create Huffman tables documentation

- Create `docs/learning/jpg/huffman.md`
- Explain JPEG Huffman encoding (DC and AC)
- Document standard JPEG Huffman tables
- Explain optimized Huffman table generation
- Show compression benefits of optimized tables
- Output: `docs/learning/jpg/huffman.md`

**Task 9.8**: Create chroma subsampling documentation

- Create `docs/learning/jpg/subsampling.md`
- Explain 4:2:0 vs 4:4:4 subsampling
- Document why chroma subsampling works (human vision)
- Show visual quality comparison
- Explain file size impact
- Output: `docs/learning/jpg/subsampling.md`

**Task 9.9**: Create JPEG index documentation

- Create `docs/learning/jpg/index.md`
- Organize all JPEG documentation with links
- Add quick reference tables (quality levels, presets, subsampling)
- Add getting started guide
- Link to related PNG documentation
- Output: `docs/learning/jpg/index.md`

**Task 9.10**: Update main documentation

- Update `docs/learning/png-vs-jpeg.md` with implementation details
- Add links to new JPEG documentation
- Update any cross-references
- Output: `docs/learning/png-vs-jpeg.md` (updated)

## File Structure

```
src/jpeg/
├── constants.go              # Markers, constants, types
├── errors.go                 # Error types
├── color.go                  # RGB to YCbCr conversion
├── blocks.go                 # 8x8 block extraction (4:4:4)
├── mcu.go                    # MCU extraction (4:2:0)
├── dct.go                    # Discrete Cosine Transform
├── quantize.go               # Quantization tables and operations
├── zigzag.go                 # Zigzag reordering
├── dc.go                     # DC coefficient encoding
├── ac.go                     # AC run-length encoding
├── huffman.go                # Standard Huffman tables
├── huffman_optimized.go      # Optimized Huffman table generation
├── bit_writer.go             # MSB-first bit writer
├── markers.go                # JPEG marker writing
├── progressive.go            # Progressive scan encoding
├── trellis.go                # Trellis quantization
├── subsample.go              # Chroma subsampling
├── options.go                # Options struct and presets
├── options_builder.go        # Fluent builder
└── encoder.go                 # Main encoder
```

## Code Reuse Opportunities

1. **Huffman Tree Building**: Reuse concepts from `src/compress/huffman_tree.go` for optimized Huffman tables (adapt for JPEG symbol space)

2. **Options Pattern**: Follow same pattern as `src/png/options.go` and `src/png/options_builder.go`

3. **Error Handling**: Follow same error pattern as PNG encoder

4. **WASM Bridge Pattern**: Follow same pattern as PNG bridge in `src/wasm/bridge.go`

5. **CLI Pattern**: Follow same pattern as PNG CLI support

## Testing Strategy

**Important: Each task includes its own unit tests. Tests are not deferred to a separate phase.**

- **Unit tests for each component**: Each task creates `*_test.go` files alongside implementation files
- **Test after each task**: Run `go test ./src/jpeg/...` and `golangci-lint run ./src/jpeg/...` after completing each task
- **Integration tests**: Encoder tests verify full pipeline (various sizes, quality levels, color types)
- **Conformance tests**: Encoder tests verify output decodes with Go's `image/jpeg` decoder
- **Edge case tests**: Included in relevant component tests (1×1 images, non-multiple-of-8 dimensions, extreme quality values)
- **All tests must pass**: `go test ./...` before moving to next task
- **All linting must pass**: `golangci-lint run` with no warnings before moving to next task

## Integration Points

1. **WASM Bridge** (`src/wasm/bridge.go`): Add `EncodeJpeg` and `EncodeJpegAdvanced` functions
2. **CLI** (`src/cmd/cli/main.go`): Add JPEG format support with quality and preset flags
3. **Web UI** (`web/src/`): Add JPEG compression UI controls

## Dependencies

- No external dependencies (pure Go implementation)
- Reuse existing `compress` package concepts where applicable
- Follow Go standard library patterns (`image/jpeg` for reference, not dependency)

## Notes

- JPEG bit writer uses **MSB-first** (opposite of DEFLATE's LSB-first)
- JPEG uses **YCbCr** color space (convert from RGB input)
- JPEG is **lossy by nature** (DCT + quantization always loses information)
- Progressive JPEG requires computing all DCT coefficients first, then encoding in multiple scans
- Trellis quantization is computationally expensive but provides 5-15% compression improvement