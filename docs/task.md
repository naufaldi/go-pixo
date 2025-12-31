# go-pixo Task List

This document contains all tasks needed to complete the go-pixo project, organized by implementation phase. Tasks are designed to be converted into GitHub issues.

**Design Principles:**
- Each task = 1 feature = 1 PR
- Engineer can complete one task in 2-4 hours
- Task has clear start and end state
- No task depends on multiple other tasks at the same level

---

## Phase 1: PNG Minimum Valid Encoder (Correctness-First)

Goal: Output a valid PNG for small RGB/RGBA images without fancy compression yet.

### 1.1 PNG Infrastructure ✅ COMPLETED

- **[Task 1.1.1]** ✅ Create `src/png/constants.go` with PNG constants
  - Define `PNG_SIGNATURE` (8 bytes)
  - Define `ChunkType` type and constants (IHDR, IDAT, IEND)
  - Define `ColorType` constants (Grayscale=0, RGB=2, RGBA=6)
  - Define `FilterType` constants (None=0, Sub=1, Up=2, Average=3, Paeth=4)
  - Output: `src/png/constants.go`

- **[Task 1.1.2]** ✅ Create `src/png/errors.go` with error types
  - Define `Error` type implementing `error` interface
  - Add errors for: invalid signature, unknown chunk type, invalid dimensions
  - Output: `src/png/errors.go`

- **[Task 1.1.3]** ✅ Create `src/png/signature.go` with signature validation
  - Add `IsValidSignature(data []byte) bool` function
  - Add `Signature() []byte` returning PNG magic bytes
  - Test: validate real PNG files
  - Output: `src/png/signature.go`, `src/png/signature_test.go`
  - Additional: Created `docs/learning/png.md` explaining signature and constants

### 1.2 CRC32 Implementation

- **[Task 1.2.1]** Create `src/png/crc32.go` with CRC32 calculation
  - Use standard library `hash/crc32`
  - Add `CRC32(data []byte) uint32` function
  - Add `NewCRC32() hash.Hash32` for streaming
  - Test: verify against known CRC32 values
  - Output: `src/png/crc32.go`, `src/png/crc32_test.go`

### 1.3 Chunk Writing

- **[Task 1.3.1]** Create `src/png/chunk.go` with basic chunk structure
  - Define `Chunk` struct (Type string, Data []byte)
  - Add `Len() int` method
  - Add `Type() string` method
  - Output: `src/png/chunk.go`

- **[Task 1.3.2]** Add `WriteTo` method to Chunk
  - Add `WriteTo(w io.Writer) (int64, error)` method
  - Write: 4-byte length (big-endian), 4-byte type, data, 4-byte CRC
  - CRC computed over type + data
  - Test: write chunk and verify format
  - Output: `src/png/chunk.go` (updated), `src/png/chunk_test.go`

### 1.4 IHDR Chunk

- **[Task 1.4.1]** Create `src/png/ihdr_data.go` with IHDR data structure
  - Define `IHDRData` struct (Width, Height uint32, BitDepth uint8, ColorType uint8, Compression uint8, Filter uint8, Interlace uint8)
  - Add `Width()`, `Height()`, etc. accessor methods
  - Add validation (max dimensions, valid bit depth for color type)
  - Output: `src/png/ihdr_data.go`

- **[Task 1.4.2]** Add `Bytes()` method to IHDRData
  - Returns 13 bytes in standard IHDR format
  - Little-endian for Width, Height
  - Other fields as single bytes
  - Test: verify 13-byte output for known values
  - Output: `src/png/ihdr_data.go` (updated)

- **[Task 1.4.3]** Create `src/png/ihdr_writer.go` for IHDR chunk writing
  - Add `WriteIHDR(w io.Writer, data IHDRData) error` function
  - Creates chunk with type "IHDR" and IHDR data bytes
  - Test: write IHDR, verify chunk format
  - Output: `src/png/ihdr_writer.go`, `src/png/ihdr_writer_test.go`

### 1.5 IEND Chunk

- **[Task 1.5.1]** Create `src/png/iend_writer.go`
  - Add `WriteIEND(w io.Writer) error` function
  - Creates IEND chunk (type "IEND", length 0, no data, CRC of "IEND")
  - Test: verify IEND chunk format
  - Output: `src/png/iend_writer.go`, `src/png/iend_writer_test.go`

### 1.6 Adler32 for Zlib

- **[Task 1.6.1]** Create `src/compress/adler32.go`
  - Implement Adler32 checksum algorithm
  - Add `Adler32(data []byte) uint32` function
  - Add `NewAdler32() hash.Hash32` for streaming
  - Test: verify against RFC 1950 test vectors
  - Output: `src/compress/adler32.go`, `src/compress/adler32_test.go`

### 1.7 Zlib Header/Footer

- **[Task 1.7.1]** Create `src/compress/zlib_header.go`
  - Add `WriteCMF(w io.Writer, windowSize int) error` - compression method/flags
  - Add `WriteFLG(w io.Writer, checksum uint8) error` - flags byte
  - Calculate check bits for FLG
  - Test: verify CMF/FLG format
  - Output: `src/compress/zlib_header.go`, `src/compress/zlib_header_test.go`

- **[Task 1.7.2]** Create `src/compress/zlib_footer.go`
  - Add `WriteAdler32Footer(w io.Writer, checksum uint32) error`
  - Write Adler32 checksum in big-endian
  - Test: verify footer format
  - Output: `src/compress/zlib_footer.go`

### 1.8 Stored Blocks (Uncompressed DEFLATE)

- **[Task 1.8.1]** Create `src/compress/stored_block.go`
  - Add `WriteStoredBlockHeader(w io.Writer, final bool) error`
  - Add `WriteBlockData(w io.Writer, data []byte) error`
  - Add `WriteBlockFooter(w io.Writer, n uint32) error` - LEN and NLEN
  - Output: `src/compress/stored_block.go`, `src/compress/stored_block_test.go`

### 1.9 IDAT Writer

- **[Task 1.9.1]** Create `src/png/scanline.go`
  - Add `WriteScanline(w io.Writer, filter FilterType, pixels []byte) error`
  - Prepend filter byte before pixel row
  - Test: verify scanline format (filter + pixels)
  - Output: `src/png/scanline.go`, `src/png/scanline_test.go`

- **[Task 1.9.2]** Create `src/png/idat_writer.go`
  - Define `IDATWriter` struct (wraps io.Writer)
  - Add `WriteScanlines(w io.Writer, pixels []byte, width, height int, colorType int) error`
  - Call WriteScanline for each row with filter type 0
  - Output: `src/png/idat_writer.go`, `src/png/idat_writer_test.go`

### 1.10 PNG Encoder Entry Point

- **[Task 1.10.1]** Create `src/png/encoder.go`
  - Define `Encoder` struct (options IHDRData)
  - Add `Encode(pixels []byte) ([]byte, error)` method
  - Sequence: WriteSignature → WriteIHDR → WriteIDAT → WriteIEND
  - Output: `src/png/encoder.go`

- **[Task 1.10.2]** Add error handling to Encoder
  - Validate input pixel count matches width × height × bytesPerPixel
  - Handle write errors at each step
  - Output: `src/png/encoder.go` (updated)

### 1.11 Phase 1 Testing

- **[Task 1.11.1]** Create comprehensive PNG encoder tests
  - Test 1×1 RGB image
  - Test 1×1 RGBA image
  - Test 2×2 RGB image
  - Test 2×2 RGBA image
  - Verify output opens in image viewers
  - Cross-check with Go's `image/png` decoder
  - Output: `src/png/encode_test.go`

---

## Phase 2: Real DEFLATE Compression (Size Improvements)

Goal: Reduce output size without changing PNG semantics.

### 2.1 LZ77 Core

- **[Task 2.1.1]** Create `src/compress/lz77_types.go`
  - Define `Match` struct (Distance uint16, Length uint16)
  - Define `Token` type (literal or match)
  - Output: `src/compress/lz77_types.go`

- **[Task 2.1.2]** Create `src/compress/lz77_sliding_window.go`
  - Define `SlidingWindow` struct (buffer []byte, size int)
  - Add `Write(b byte)` method
  - Add `Read(pos int) byte` method
  - Output: `src/compress/lz77_sliding_window.go`

- **[Task 2.1.3]** Create `src/compress/lz77_matcher.go`
  - Add `FindMatch(window SlidingWindow, pos int) (Match, bool)` method
  - Simple greedy search for longest match
  - Output: `src/compress/lz77_matcher.go`, `src/compress/lz77_matcher_test.go`

- **[Task 2.1.4]** Create `src/compress/lz77_encoder.go`
  - Add `Encode(data []byte) []Token` method
  - Scan through data, emit literals or matches
  - Test: encode known data, verify output
  - Output: `src/compress/lz77_encoder.go`, `src/compress/lz77_encoder_test.go`

### 2.2 Huffman Basics

- **[Task 2.2.1]** Create `src/compress/huffman_types.go`
  - Define `Code` struct (Bits uint16, Length int)
  - Define `Table` struct (Codes []Code, MaxLength int)
  - Output: `src/compress/huffman_types.go`

- **[Task 2.2.2]** Create `src/compress/frequency.go`
  - Add `CountFrequencies(data []byte) []int` - count literal/length frequencies
  - Add `CountDistanceFrequencies(matches []Match) []int` - count distance frequencies
  - Output: `src/compress/frequency.go`

- **[Task 2.2.3]** Create `src/compress/huffman_tree.go`
  - Add `BuildTree(frequencies []int) *Node` - Huffman tree from frequencies
  - Define `Node` struct (Left, Right *Node, Symbol int, Weight int)
  - Output: `src/compress/huffman_tree.go`

- **[Task 2.2.4]** Create `src/compress/huffman_codes.go`
  - Add `GenerateCodes(node *Node) map[int]Code` - canonical codes
  - Add `Canonicalize(codes map[int]Code) ([]Code, []int)` - canonical form
  - Test: generate codes, verify prefix-free
  - Output: `src/compress/huffman_codes.go`, `src/compress/huffman_codes_test.go`

### 2.3 Fixed Huffman Tables

- **[Task 2.3.1]** Create `src/compress/fixed_huffman_tables.go`
  - Define literal/length code table (RFC 1951 Table 1)
  - Define distance code table (RFC 1951 Table 2)
  - Add `LiteralLengthTable() Table` getter
  - Add `DistanceTable() Table` getter
  - Output: `src/compress/fixed_huffman_tables.go`

### 2.4 Bit Writer

- **[Task 2.4.1]** Create `src/compress/bit_writer.go`
  - Define `BitWriter` struct (wraps io.Writer)
  - Add `Write(bits uint16, n int) error` - write n bits
  - Add `Flush() error` - write remaining bits (with padding)
  - Test: write bits, verify byte output
  - Output: `src/compress/bit_writer.go`, `src/compress/bit_writer_test.go`

### 2.5 Dynamic Huffman Tables

- **[Task 2.5.1]** Create `src/compress/huffman_header.go`
  - Add `WriteHLIT(w io.Writer, n int) error` - number of literal codes
  - Add `WriteHDIST(w io.Writer, n int) error` - number of distance codes
  - Add `WriteHCLEN(w io.Writer, lengths []int) error` - code length order
  - Output: `src/compress/huffman_header.go`

- **[Task 2.5.2]** Create `src/compress/dynamic_tables.go`
  - Add `BuildDynamicTables(litFreq []int, distFreq []int) (litTable, distTable Table)`
  - Build custom Huffman tables from actual frequencies
  - Output: `src/compress/dynamic_tables.go`

### 2.6 DEFLATE Block Writer

- **[Task 2.6.1]** Create `src/compress/deflate_constants.go`
  - Define block type constants (00=stored, 01=fixed, 10=dynamic, 11=invalid)
  - Define length/distance extra bit counts (RFC 1951 Table 1, 2)
  - Output: `src/compress/deflate_constants.go`

- **[Task 2.6.2]** Create `src/compress/deflate_literal_encoder.go`
  - Add `EncodeLiteral(w *BitWriter, symbol int, table Table) error`
  - Add `EncodeLength(w *BitWriter, length int, table Table) error`
  - Add `EncodeDistance(w *BitWriter, distance int, table Table) error`
  - Output: `src/compress/deflate_literal_encoder.go`

- **[Task 2.6.3]** Create `src/compress/deflate_block.go`
  - Add `WriteStoredBlock(w io.Writer, final bool, data []byte) error`
  - Add `WriteFixedBlock(w io.Writer, final bool, tokens []Token) error`
  - Add `WriteDynamicBlock(w io.Writer, final bool, tokens []Token) error`
  - Test: write blocks, verify format
  - Output: `src/compress/deflate_block.go`, `src/compress/deflate_block_test.go`

### 2.7 DEFLATE Encoder

- **[Task 2.7.1]** Create `src/compress/deflate_encoder.go`
  - Define `DeflateEncoder` struct
  - Add `Encode(data []byte, useDynamic bool) ([]byte, error)`
  - Sequence: LZ77 → Huffman → blocks
  - Test: compress data, verify smaller output
  - Output: `src/compress/deflate_encoder.go`, `src/compress/deflate_encoder_test.go`

### 2.8 Zlib Integration

- **[Task 2.8.1]** Update `src/png/idat_writer.go` to use DEFLATE
  - Replace stored blocks with DeflateEncoder
  - Keep zlib header (CMF/FLG) and footer (Adler32)
  - Test: verify PNG size reduction
  - Output: `src/png/idat_writer.go` (updated), `src/png/idat_writer_test.go`

---

## Phase 3: PNG Filters (Compression Ratio Improvements)

Goal: Improve size with filter byte per row optimization.

### 3.1 Filter Implementations

- **[Task 3.1.1]** Create `src/png/filter_types.go`
  - Define filter type constants
  - Add documentation for each filter type
  - Output: `src/png/filter_types.go`

- **[Task 3.1.2]** Create `src/png/filter_none.go`
  - Add `FilterNone(b []byte, prev []byte) []byte` - identity
  - Output: `src/png/filter_none.go`

- **[Task 3.1.3]** Create `src/png/filter_sub.go`
  - Add `FilterSub(b []byte) []byte` - b[x] - b[x-bpp]
  - Output: `src/png/filter_sub.go`

- **[Task 3.1.4]** Create `src/png/filter_up.go`
  - Add `FilterUp(b []byte, prev []byte) []byte` - b[x] - prev[x]
  - Output: `src/png/filter_up.go`

- **[Task 3.1.5]** Create `src/png/filter_average.go`
  - Add `FilterAverage(b []byte, prev []byte, bpp int) []byte` - b[x] - floor((b[x-bpp]+prev[x])/2)
  - Output: `src/png/filter_average.go`

### 3.2 Paeth Predictor

- **[Task 3.2.1]** Create `src/png/paeth.go`
  - Add `PaethPredictor(a, b, c int) int` function
  - Implement algorithm per PNG spec
  - Test: verify against PNG spec examples
  - Output: `src/png/paeth.go`, `src/png/paeth_test.go`

- **[Task 3.2.2]** Create `src/png/filter_paeth.go`
  - Add `FilterPaeth(b []byte, prev []byte, bpp int) []byte`
  - Use PaethPredictor for each byte
  - Output: `src/png/filter_paeth.go`

### 3.3 Filter Reconstruction

- **[Task 3.3.1]** Create `src/png/filter_reconstruct.go`
  - Add `ReconstructNone(b []byte) []byte`
  - Add `ReconstructSub(b []byte, bpp int) []byte`
  - Add `ReconstructUp(b, prev []byte) []byte`
  - Add `ReconstructAverage(b, prev []byte, bpp int) []byte`
  - Add `ReconstructPaeth(b, prev []byte, bpp int) []byte`
  - Test: encode then decode, verify matches original
  - Output: `src/png/filter_reconstruct.go`, `src/png/filter_reconstruct_test.go`

### 3.4 Filter Selection

- **[Task 3.4.1]** Create `src/png/filter_score.go`
  - Add `SumAbsoluteValues(b []byte) int` function
  - Test: verify sum calculation
  - Output: `src/png/filter_score.go`

- **[Task 3.4.2]** Create `src/png/filter_selector.go`
  - Add `SelectFilter(row []byte, prevRow []byte, bpp int) FilterType`
  - Try all 5 filters, pick one with minimum sum
  - Add `SelectAll(pixels []byte, width, height, bpp int) []FilterType`
  - Output: `src/png/filter_selector.go`, `src/png/filter_selector_test.go`

- **[Task 3.4.3]** Update `src/png/idat_writer.go` to use filter selection
  - Replace filter type 0 with intelligent selection
  - Test: verify size improvement
  - Output: `src/png/idat_writer.go` (updated)

### 3.5 Phase 3 Testing

- **[Task 3.5.1]** Create filter effectiveness tests
  - Test on sample images
  - Compare size with filter none vs all filters
  - Output: `src/png/filter_test.go`

---

## Phase 4: PNG Lossless Optimizations

Goal: Add preset system with configurable optimization options.

### 4.1 Options Structure

- **[Task 4.1.1]** Create `src/png/options.go`
  - Define `Options` struct with optimization flags
  - Define `Preset` type (Fast, Balanced, Max)
  - Add `DefaultOptions() Options` function
  - Output: `src/png/options.go`

- **[Task 4.1.2]** Create `src/png/options_builder.go`
  - Define `OptionsBuilder` struct
  - Add chainable methods: `Fast()`, `Balanced()`, `Max()`, `WithFilterSelection(bool)`, etc.
  - Add `Build() Options` method
  - Test: verify preset configurations
  - Output: `src/png/options_builder.go`, `src/png/options_builder_test.go`

### 4.2 Alpha Optimization

- **[Task 4.2.1]** Create `src/png/alpha.go`
  - Add `HasAlpha(pixels []byte, colorType int) bool` function
  - Add `ZeroRgbWhenAlphaZero(pixels []byte, colorType int) []byte` function
  - Output: `src/png/alpha.go`, `src/png/alpha_test.go`

### 4.3 Color Type Analysis

- **[Task 4.3.1]** Create `src/png/color_analysis.go`
  - Add `IsGrayscale(pixels []byte, colorType int) bool` function
  - Add `CanReduceToGrayscale(pixels []byte) bool` function
  - Add `CanReduceToRGB(pixels []byte) bool` function
  - Output: `src/png/color_analysis.go`

- **[Task 4.3.2]** Create `src/png/color_reduce.go`
  - Add `ReduceGrayscale(pixels []byte, width, height int) ([]byte, error)` function
  - Add `ReduceRGBAtoRGB(pixels []byte, width, height int) ([]byte, error)` function
  - Output: `src/png/color_reduce.go`, `src/png/color_reduce_test.go`

### 4.4 Metadata Stripping

- **[Task 4.4.1]** Update chunk writer to skip ancillary chunks
  - Modify `WriteTo` to only write required chunks (IHDR, IDAT, IEND)
  - Test: verify no tEXt, zTXt, etc. chunks written
  - Output: `src/png/chunk.go` (updated)

### 4.5 Encoder Integration

- **[Task 4.5.1]** Update `src/png/encoder.go` to use Options
  - Modify `Encode` to accept `Options` parameter
  - Apply optimizations before encoding
  - Output: `src/png/encoder.go` (updated)

### 4.6 Phase 4 Testing

- **[Task 4.6.1]** Create preset tests
  - Test Fast preset (minimal processing)
  - Test Balanced preset (filters only)
  - Test Max preset (all optimizations)
  - Measure size differences
  - Output: `src/png/preset_test.go`

---

## Phase 5: PNG Lossy Mode (Quantization)

Goal: Optional lossy PNG with palette quantization.

### 5.1 Palette Quantization Core

- **[Task 5.1.1]** Create `src/png/palette.go`
  - Define `Palette` struct (Colors []Color, NumColors int)
  - Define `Color` struct (R, G, B uint8)
  - Add `NewPalette(maxColors int) *Palette` function
  - Output: `src/png/palette.go`

- **[Task 5.1.2]** Create `src/png/color_count.go`
  - Add `CountColors(pixels []byte, colorType int) map[Color]int` function
  - Count frequency of each unique color
  - Output: `src/png/color_count.go`

- **[Task 5.1.3]** Create `src/png/median_cut.go`
  - Add `MedianCut(colors []ColorWithCount, maxColors int) []Color` function
  - Recursively split color space
  - Output: `src/png/median_cut.go`, `src/png/median_cut_test.go`

- **[Task 5.1.4]** Create `src/png/quantize.go`
  - Add `Quantize(pixels []byte, colorType int, maxColors int) ([]byte, Palette)` function
  - Build palette from colors
  - Map each pixel to nearest palette color
  - Test: verify color count ≤ 256
  - Output: `src/png/quantize.go`, `src/png/quantize_test.go`

### 5.2 Dithering

- **[Task 5.2.1]** Create `src/png/dither.go`
  - Define `Ditherer` struct (error []int)
  - Add `FloydSteinberg(pixels []byte, palette Palette) []byte` function
  - Add `Threshold(pixels []byte, palette Palette) []byte` function (no dithering)
  - Output: `src/png/dither.go`, `src/png/dither_test.go`

### 5.3 PLTE Chunk

- **[Task 5.3.1]** Create `src/png/plte_writer.go`
  - Add `WritePLTE(w io.Writer, palette Palette) error` function
  - Write palette as PLTE chunk (before IDAT)
  - Output: `src/png/plte_writer.go`, `src/png/plte_writer_test.go`

### 5.4 tRNS Chunk

- **[Task 5.3.2]** Create `src/png/trns_writer.go`
  - Add `WriteTRNS(w io.Writer, palette Palette) error` function
  - Write alpha values for palette entries (after PLTE)
  - Output: `src/png/trns_writer.go`, `src/png/trns_writer_test.go`

### 5.5 Lossy API Integration

- **[Task 5.5.1]** Update `src/png/encoder.go` for lossy mode
  - Add `QuantizeBeforeEncoding(pixels []byte, colorType int, options Options) ([]byte, Palette)` function
  - Modify `Encode` to handle quantized data
  - Output: `src/png/encoder.go` (updated)

### 5.6 Phase 5 Testing

- **[Task 5.6.1]** Create lossy PNG tests
  - Test quantization on various images
  - Test dithering on/off
  - Verify output < lossless size
  - Output: `src/png/lossy_test.go`

---

## Phase 6: JPEG Baseline Encoder

Goal: Implement JPEG encoding for photos.

### 6.1 Color Conversion

- **[Task 6.1.1]** Create `src/jpeg/constants.go`
  - Define JPEG marker constants (SOI, EOI, APP0, DQT, SOF0, DHT, SOS)
  - Define `Component` struct (ID, H, V, QuantTable, DCTable, ACTable)
  - Output: `src/jpeg/constants.go`

- **[Task 6.1.2]** Create `src/jpeg/ycbcr.go`
  - Define `YCbCr` struct (Y, Cb, Cr []byte)
  - Add `RGBToYCbCr(r, g, b []byte) (y, cb, cr []byte)` function
  - Add `YCbCrToRGB(y, cb, cr []byte) (r, g, b []byte)` function
  - Test: round-trip conversion
  - Output: `src/jpeg/ycbcr.go`, `src/jpeg/ycbcr_test.go`

### 6.2 Block Splitting

- **[Task 6.2.1]** Create `src/jpeg/blocks.go`
  - Add `SplitIntoBlocks(data []byte, width, height int) [][][]int8` function
  - Handle edge padding (replicate last pixel)
  - Output: `src/jpeg/blocks.go`, `src/jpeg/blocks_test.go`

### 6.3 DCT Implementation

- **[Task 6.3.1]** Create `src/jpeg/dct.go`
  - Add `ForwardDCT(block [][]int) [][]int` function
  - Implement integer DCT (not floating point)
  - Add `InverseDCT(block [][]int) [][]int` function (for verification)
  - Test: IDCT(InverseDCT(x)) ≈ x
  - Output: `src/jpeg/dct.go`, `src/jpeg/dct_test.go`

### 6.4 Quantization

- **[Task 6.4.1]** Create `src/jpeg/quantization_tables.go`
  - Define standard luminance table (quality 50)
  - Define standard chrominance table (quality 50)
  - Add `ScaleTable(table []int, quality int) []int` function
  - Output: `src/jpeg/quantization_tables.go`

- **[Task 6.4.2]** Create `src/jpeg/quantize.go`
  - Add `Quantize(block [][]int, table []int) [][]int` function
  - Round(DCT / table)
  - Output: `src/jpeg/quantize.go`

### 6.5 Zigzag

- **[Task 6.5.1]** Create `src/jpeg/zigzag.go`
  - Define zigzag order array [64]int
  - Add `Zigzag(block [][]int) []int` function
  - Add `Dezigzag(coeffs []int) [][]int` function
  - Test: zigzag then dezigzag = original
  - Output: `src/jpeg/zigzag.go`, `src/jpeg/zigzag_test.go`

### 6.6 DC Encoding

- **[Task 6.6.1]** Create `src/jpeg/dc.go`
  - Add `EncodeDC(dc int) (bits []int, size int)` function
  - Compute difference from previous DC
  - Size-category encoding
  - Add `DecodeDC(coeffs []int) int` function
  - Output: `src/jpeg/dc.go`, `src/jpeg/dc_test.go`

### 6.7 AC Encoding

- **[Task 6.7.1]** Create `src/jpeg/ac.go`
  - Add `RunLengthEncode(coeffs []int) []Tuple` function
  - Tuple = (runLength, size) for zeros, then (0, value) for non-zeros
  - Add `RunLengthDecode(tuples []Tuple) []int` function
  - Output: `src/jpeg/ac.go`, `src/jpeg/ac_test.go`

### 6.8 Huffman Tables

- **[Task 6.8.1]** Create `src/jpeg/huffman_tables.go`
  - Define standard DC luminance table
  - Define standard DC chrominance table
  - Define standard AC luminance table
  - Define standard AC chrominance table
  - Output: `src/jpeg/huffman_tables.go`

- **[Task 6.8.2]** Create `src/jpeg/huffman_encoder.go`
  - Add `HuffmanEncode(symbol int, table []Code) (bits []int)` function
  - Look up code in table, return bits
  - Output: `src/jpeg/huffman_encoder.go`

### 6.9 Bit Writer for JPEG

- **[Task 6.9.1]** Create `src/jpeg/bit_writer.go`
  - Add `WriteByte(b byte) error` function
  - Add `WriteBits(bits int, n int) error` function
  - Handle byte stuffing (0xFF → 0xFF 0x00)
  - Output: `src/jpeg/bit_writer.go`

### 6.10 Markers

- **[Task 6.10.1]** Create `src/jpeg/markers.go`
  - Add `WriteSOI(w io.Writer) error` function
  - Add `WriteEOI(w io.Writer) error` function
  - Add `WriteAPP0(w io.Writer) error` function (JFIF header)
  - Add `WriteDQT(w io.Writer, tableID int, table []int) error` function
  - Add `WriteSOF0(w io.Writer, width, height int, components []Component) error` function
  - Add `WriteDHT(w io.Writer, tableID int, bits []int, vals []int) error` function
  - Add `WriteSOS(w io.Writer, components []Component) error` function
  - Test: write markers, verify format
  - Output: `src/jpeg/markers.go`, `src/jpeg/markers_test.go`

### 6.11 JPEG Encoder Entry Point

- **[Task 6.11.1]** Create `src/jpeg/encoder.go`
  - Define `Encoder` struct (width, height, quality int)
  - Add `Encode(pixels []byte) ([]byte, error)` method
  - Sequence: RGB→YCbCr → blocks → DCT → quantize → zigzag → Huffman → markers
  - Output: `src/jpeg/encoder.go`

- **[Task 6.11.2]** Test JPEG encoder
  - Encode 1×1, 8×8, 16×16 images
  - Verify output opens in browsers
  - Test various quality settings
  - Output: `src/jpeg/encoder_test.go`

---

## Phase 7: JPEG Features and Presets

Goal: Advanced JPEG features after baseline works.

### 7.1 Chroma Subsampling

- **[Task 7.1.1]** Create `src/jpeg/subsample.go`
  - Add `Subsample420(cb, cr []byte, width, height int) ([]byte, []byte)` function
  - Average every 2×2 block
  - Update encoder to use subsampled Cb/Cr
  - Output: `src/jpeg/subsample.go`, `src/jpeg/subsample_test.go`

### 7.2 Optimized Huffman Tables

- **[Task 7.2.1]** Create `src/jpeg/optimized_huffman.go`
  - Add `BuildOptimizedTables(blocks [][][]int, dcTable, acTable []int) (dcBits, dcVals, acBits, acVals []int)`
  - Count symbol frequencies from actual data
  - Build custom tables
  - Output: `src/jpeg/optimized_huffman.go`

### 7.3 Progressive JPEG

- **[Task 7.3.1]** Create `src/jpeg/progressive.go`
  - Add `ProgressiveOrder(coeffIndex int) int` function
  - Add `WriteProgressiveScan(w io.Writer, blocks [][][]int, dcTable, acTable []int, start, end int) error`
  - Split coefficients into multiple scans
  - Output: `src/jpeg/progressive.go`

### 7.4 JPEG Presets

- **[Task 7.4.1]** Create `src/jpeg/presets.go`
  - Define `Preset` type (Fast, Balanced, Max)
  - Add `FastEncoder() Encoder` function (subsampling 420, standard tables)
  - Add `BalancedEncoder() Encoder` function (subsampling 420, optimized tables)
  - Add `MaxEncoder() Encoder` function (subsampling 444, optimized tables)
  - Output: `src/jpeg/presets.go`

---

## Phase 8: Web Product Polish

Goal: Make the product easy to use.

### 8.1 Drag and Drop

- **[Task 8.1.1]** Update `web/src/main.ts` with visual drag feedback
  - Add dragenter/dragleave event handlers
  - Show visual indicator when file is over drop zone
  - Output: `web/src/main.ts`

- **[Task 8.1.2]** Support multiple file drop
  - Add `handleDrop` for multiple files
  - Process files one at a time
  - Output: `web/src/main.ts`

### 8.2 Progress Indicator

- **[Task 8.2.1]** Add progress bar for compression
  - Create `ProgressBar` component
  - Show progress during WASM execution
  - Output: `web/src/main.ts`

### 8.3 Batch Processing

- **[Task 8.3.1]** Implement batch file list UI
  - Create file list component
  - Show status (pending, processing, done, error)
  - Allow download all
  - Output: `web/src/main.ts`

### 8.4 Before/After Preview

- **[Task 8.4.1]** Create side-by-side preview component
  - Show original image on left
  - Show compressed image on right
  - Display size comparison
  - Output: `web/src/main.ts`

### 8.5 Preset UI

- **[Task 8.5.1]** Update preset selector with plain language
  - "Smallest (more compression)"
  - "Balanced"
  - "Best Quality"
  - Show estimated size trade-off
  - Output: `web/src/main.ts`

### 8.6 Privacy Messaging

- **[Task 8.6.1]** Add privacy indicator
  - "Runs locally on your device"
  - "No data sent to servers"
  - Visual badge
  - Output: `web/src/main.ts`

### 8.7 Web Worker

- **[Task 8.7.1]** Create `web/src/worker.ts`
  - Move WASM calls to Web Worker
  - Post messages for progress
  - Update main thread UI
  - Output: `web/src/worker.ts`

- **[Task 8.7.2]** Update main.ts to use worker
  - Replace direct WASM calls with worker messages
  - Show live progress from worker
  - Output: `web/src/main.ts`

### 8.8 Memory Optimization

- **[Task 8.8.1]** Create buffer pool in `web/src/wasm.ts`
  - Pool `Uint8Array` buffers
  - Reuse buffers between compressions
  - Output: `web/src/wasm.ts`

---

## Infrastructure Tasks (Cross-Cutting)

### Build and Testing

- **[Infra 1]** Update `AGENTS.md` with test commands
  - Add `go test ./...` command
  - Add `go fmt ./...` command
  - Add `go vet ./...` command
  - Output: `AGENTS.md`

- **[Infra 2]** Add Makefile with common commands
  - `make test` → `go test ./...`
  - `make fmt` → `go fmt ./...`
  - `make vet` → `go vet ./...`
  - `make build` → `GOOS=js GOARCH=wasm go build`
  - Output: `Makefile`

### Documentation

- **[Doc 1]** Add Go doc comments to all exported functions
  - `src/png/*.go` (each file)
  - `src/compress/*.go` (each file)
  - `src/jpeg/*.go` (each file)
  - `src/wasm/bridge.go`

- **[Doc 2]** Update `README.md` with usage examples
  - Go library usage
  - Web usage
  - API reference

---

## Task Dependencies

```
Phase 1 (PNG Encoder)
  ├─ 1.1 PNG Infrastructure
  ├─ 1.2 CRC32
  ├─ 1.3-1.5 Chunks (IHDR, IEND)
  ├─ 1.6-1.7 Zlib (Adler32, Header/Footer)
  ├─ 1.8 Stored Blocks
  ├─ 1.9 Scanlines + IDAT
  └─ 1.10-1.11 Encoder + Tests

Phase 2 (DEFLATE) → depends on Phase 1
  ├─ 2.1 LZ77
  ├─ 2.2 Huffman
  ├─ 2.3-2.5 Tables + Headers
  └─ 2.6-2.8 Blocks + Encoder + Integration

Phase 3 (Filters) → depends on Phase 1
  ├─ 3.1 Filter Types
  ├─ 3.2 Paeth
  ├─ 3.3 Reconstruction
  └─ 3.4-3.5 Selection + Tests

Phase 4 (Optimizations) → depends on Phase 1
  ├─ 4.1 Options
  ├─ 4.2-4.4 Optimizations
  └─ 4.5-4.6 Integration + Tests

Phase 5 (Lossy) → depends on Phase 1
  ├─ 5.1-5.2 Quantization
  ├─ 5.3-5.4 PLTE/tRNS
  └─ 5.5-5.6 Integration + Tests

Phase 6 (JPEG) → independent of PNG phases
  ├─ 6.1-6.5 Core (YCbCr, Blocks, DCT, Quant, Zigzag)
  ├─ 6.6-6.7 Encoding (DC, AC)
  ├─ 6.8-6.9 Huffman + BitWriter
  └─ 6.10-6.11 Markers + Encoder + Tests

Phase 7 (JPEG Features) → depends on Phase 6
  ├─ 7.1 Subsampling
  ├─ 7.2 Optimized Tables
  └─ 7.3-7.4 Progressive + Presets

Phase 8 (Web Polish) → depends on Phase 1+
  └─ All tasks can start after Phase 1 completes
```

---

## Quick Reference

| Phase | Tasks | Primary Output |
|-------|-------|----------------|
| 1 | 11 | Valid PNG encoder |
| 2 | 8 | DEFLATE compression |
| 3 | 5 | Filter selection |
| 4 | 6 | Preset system |
| 5 | 6 | Lossy PNG |
| 6 | 11 | JPEG encoder |
| 7 | 4 | JPEG features |
| 8 | 8 | Web UI polish |
| Infra | 4 | Build/test/docs |

---

## Implementation Order for MVP

For fastest path to working product:

1. **Phase 1** (all 11 tasks) - Complete PNG encoder
2. **Phase 3** (all 5 tasks) - Add filters for compression
3. **Phase 2** (all 8 tasks) - Add DEFLATE (can do after filters)
4. **Phase 4** (all 6 tasks) - Add presets (optional)
5. **Phase 6-7** (JPEG) - Later phase
6. **Phase 5** (Lossy PNG) - Optional
7. **Phase 8** (Web polish) - After core works
