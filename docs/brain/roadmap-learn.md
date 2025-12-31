# go-pixo Learning Roadmap

A structured learning path for implementing a PNG/JPEG encoder in Go. This roadmap tells you **what to learn**, **in what order**, and **where to learn it**.

**Goal**: Build a complete understanding of image compression to implement go-pixo from scratch.

---

## Learning Phases

### Phase 1: Foundations (Week 1-2)

#### 1.1 Why Compress Images?
**Goal**: Understand the problem and high-level approaches

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [Introduction to Image Compression](../introduction-to-image-compression.md) | Internal Doc | 30 min | Required |
| [Cloudflare: What is Image Compression](https://www.cloudflare.com/learning/performance/glossary/what-is-image-compression/) | Article | 15 min | Optional |
| [A Guide to Image Compression - Penji](https://penji.co/a-guide-to-image-compression/) | Article | 20 min | Optional |

**Key Concepts**:
- Lossless vs lossy compression
- Entropy and information theory basics
- Why images compress differently than text

**After This**: You can explain why PNG works better for graphics while JPEG works better for photos.

---

#### 1.2 PNG Format Overview
**Goal**: Understand PNG file structure

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [How PNG Works - Medium](https://medium.com/@duhroach/how-png-works-f1174e3cc7b7) | Article | 20 min | Required |
| [RFC 2083 - PNG Specification](https://datatracker.ietf.org/doc/rfc2083/) | Spec | 60 min | Reference |
| [PNG: The Definitive Guide - libpng](http://www.libpng.org/pub/png/book/chapter09.html) | Book Chapter | 45 min | Reference |

**Key Concepts**:
- PNG signature (8 magic bytes)
- Chunk structure (IHDR, IDAT, IEND, PLTE, tRNS)
- Color types (Grayscale, RGB, RGBA, palette)
- Bit depth (1, 2, 4, 8, 16 bits)

**After This**: You can read a PNG file header and identify its chunks.

---

#### 1.3 JPEG Format Overview
**Goal**: Understand JPEG file structure

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [Introduction to JPEG Compression](https://www.jpeg.org/images/jpegs.jpg) | Spec | 60 min | Reference |
| [How JPEG Works - CS@UC Davis](https://www.cs.ucdavis.edu/~martel/122a/deflate.html) | Tutorial | 30 min | Optional |

**Key Concepts**:
- JPEG markers (SOI, EOI, APP0, DQT, SOF0, DHT, SOS)
- Color space conversion (RGB → YCbCr)
- Chroma subsampling (4:4:4 vs 4:2:0)
- DCT blocks (8×8)

**After This**: You can recognize JPEG structure in hex dump.

---

## Phase 2: Compression Algorithms (Week 2-4)

#### 2.1 Huffman Coding
**Goal**: Understand entropy encoding

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [Huffman Coding](../huffman-coding.md) | Internal Doc | 30 min | Required |
| [Data Compression with Huffman Encoding - David Kosbie](https://www.kosbie.net/cmu/fall-15/15-112/notes/notes-data-compression.html) | Tutorial | 45 min | Required |
| [Huffman Coding Visualization](https://www.cs.toronto.edu/~bratanov/courses/411/lecture6.pdf) | Slides | 30 min | Optional |

**Key Concepts**:
- Frequency counting
- Building Huffman tree
- Canonical codes
- Variable-length encoding

**Practice**: Manually encode a short string using Huffman coding.

---

#### 2.2 LZ77 Compression
**Goal**: Understand dictionary-based compression

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [LZ77 Compression](../lz77-compression.md) | Internal Doc | 30 min | Required |
| [Fun with Lazy Arrays: the LZ77 Algorithm](https://brandon.si/code/fun-with-lazy-arrays-the-lz77-algorithm/) | Article | 30 min | Required |
| [LZ77 Compression - GeeksforGeeks](https://www.geeksforgeeks.org/lz77-compression-algorithms/) | Tutorial | 20 min | Optional |

**Key Concepts**:
- Sliding window
- Back-references (distance, length pairs)
- Greedy vs optimal matching
- Literal/length encoding

**Practice**: Manually encode "ABABABAB" using LZ77.

---

#### 2.3 DEFLATE Algorithm
**Goal**: Understand how LZ77 and Huffman combine

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [DEFLATE Algorithm](../deflate.md) | Internal Doc | 45 min | Required |
| [An Explanation of the DEFLATE Algorithm - zlib](https://zlib.net/feldspar.html) | Tutorial | 45 min | Required |
| [The Internet's Shrink Ray: Data Compression](https://medium.com/@rutgersusacs/the-internets-shrink-ray-data-compression-with-huffman-lz77-and-deflate-04ab37f01819) | Article | 20 min | Optional |

**Key Concepts**:
- Block types (stored, fixed Huffman, dynamic Huffman)
- LZ77 output → Huffman input
- DEFLATE stream structure
- zlib wrapper (CMF/FLG, Adler32)

**After This**: You can trace how a small data sequence compresses through DEFLATE.

---

## Phase 3: PNG-Specific Concepts (Week 3-4)

#### 3.1 PNG Filters
**Goal**: Understand predictive filtering

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [PNG Encoding - Filter Section](../png-encoding.md) | Internal Doc | 30 min | Required |
| [W3C PNG Filter Specification](https://www.w3.org/TR/PNG-Filters.html) | Spec | 30 min | Required |
| [PNG: The Definitive Guide - Filtering](http://www.libpng.org/pub/png/book/chapter09.html) | Book Chapter | 30 min | Reference |

**Key Concepts**:
- Why filter before compressing
- 5 filter types: None, Sub, Up, Average, Paeth
- Filter byte per row
- Paeth predictor algorithm

**Practice**: Implement Paeth predictor in pseudocode.

```
PaethPredictor(a, b, c):
    p = a + b - c
    pa = abs(p - a)
    pb = abs(p - b)
    pc = abs(p - c)
    if pa <= pb and pa <= pc: return a
    if pb <= pc: return b
    return c
```

---

#### 3.2 PNG Chunks in Detail
**Goal**: Understand chunk writing and CRC

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [RFC 2083 - Chunk Specifications](https://datatracker.ietf.org/doc/rfc2083/) | Spec | 60 min | Required |
| [PNG Specification (Fourth Edition)](https://w3c.github.io/png/) | Spec | 60 min | Reference |

**Key Concepts**:
- Critical chunks: IHDR, PLTE, IDAT, IEND
- Ancillary chunks: tRNS, tEXt, iTXt, zTXt
- CRC32 calculation
- Chunk length/type/data structure

---

#### 3.3 Zlib/DEFLATE for PNG
**Goal**: Understand zlib wrapper

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [RFC 1950 - Zlib](https://datatracker.ietf.org/doc/rfc1950/) | Spec | 30 min | Required |
| [RFC 1951 - DEFLATE](https://datatracker.ietf.org/doc/rfc1951/) | Spec | 60 min | Reference |

**Key Concepts**:
- zlib header (CMF/FLG)
- Adler32 checksum
- DEFLATE blocks in PNG
- Final Adler32 footer

---

## Phase 4: JPEG-Specific Concepts (Week 4-5)

#### 4.1 Discrete Cosine Transform (DCT)
**Goal**: Understand frequency-domain transformation

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [DCT (Discrete Cosine Transform)](../dct.md) | Internal Doc | 45 min | Required |
| [JPEG DCT Explained](https://www.jpeg.org/images/jpegs.jpg) | Spec | 30 min | Reference |

**Key Concepts**:
- Why transform to frequency domain
- 2D DCT formula
- 8×8 block processing
- DC coefficient (average brightness)
- AC coefficients (detail)

**Practice**: Compute 1D DCT on a small array by hand.

---

#### 4.2 JPEG Quantization
**Goal**: Understand lossy compression

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [JPEG Quantization](../quantization.md) | Internal Doc | 30 min | Required |
| [Quantization Tables](https://www.jpeg.org/images/jpegs.jpg) | Spec | 20 min | Reference |

**Key Concepts**:
- Quality-based quantization tables
- Zigzag reordering
- How quantization discards information
- Quality vs size trade-off

---

#### 4.3 JPEG Encoding Pipeline
**Goal**: Understand complete JPEG encoder

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [JPEG Encoding](../jpeg-encoding.md) | Internal Doc | 45 min | Required |
| [JPEG: The Complete Standard](https://www.jpeg.org/images/jpegs.jpg) | Spec | 60 min | Reference |

**Key Concepts**:
- RGB → YCbCr conversion
- 8×8 block splitting
- DCT → Quantization
- Zigzag → Huffman encoding
- Marker insertion

---

## Phase 5: Implementation Skills (Week 5-8)

#### 5.1 Bit-Level Operations
**Goal**: Learn to work with individual bits

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [Bits Module in pixo](https://github.com/leerob/pixo/blob/main/src/bits.rs) | Code | 30 min | Required |
| [Bitwise Operations in Go](https://go.dev/ref/spec#Operators) | Spec | 15 min | Reference |

**Key Skills**:
- Reading/writing individual bits
- MSB vs LSB ordering
- Bit padding and alignment
- Big-endian byte order

**Practice**: Write a BitWriter that can write n bits to a byte stream.

---

#### 5.2 Go for Systems Programming
**Goal**: Learn Go idioms for low-level code

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [Effective Go](https://go.dev/doc/effective_go) | Official Doc | 60 min | Required |
| [Go by Example](https://gobyexample.com/) | Tutorial | 120 min | Reference |
| [syscall/js Package](https://pkg.go.dev/syscall/js) | Package Doc | 30 min | Required (WASM) |

**Key Concepts**:
- Structs and methods
- Interface satisfaction
- io.Reader/io.Writer patterns
- syscall/js for WASM
- Buffer reuse patterns

---

#### 5.3 Testing Image Codecs
**Goal**: Learn to verify encoding correctness

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [Go Testing](https://go.dev/doc/testing) | Official Doc | 30 min | Required |
| [Go Image Package](https://pkg.go.dev/image) | Package Doc | 30 min | Required |

**Key Techniques**:
- Round-trip testing (encode → decode → compare)
- Reference comparison with known-good encoders
- Golden file testing
- Property-based testing

---

## Phase 6: Advanced Topics (Week 8+)

#### 6.1 Performance Optimization
**Goal**: Make encoding fast

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [Performance Optimization](../performance-optimization.md) | Internal Doc | 30 min | Optional |
| [Go Profiling](https://go.dev/doc/pprof) | Tool Doc | 30 min | Reference |

**Topics**:
- Buffer pooling
- Escape analysis
- Concurrency patterns
- Profile-guided optimization

---

#### 6.2 WebAssembly Integration
**Goal**: Deploy to browser

| Resource | Type | Time | Priority |
|----------|------|------|----------|
| [WASM in go-pixo](wasm.md) | Internal Doc | 30 min | Required |
| [Go WASM Documentation](https://github.com/golang/go/wiki/WebAssembly) | Wiki | 45 min | Required |
| [wasm-bindgen Guide](https://rustwasm.github.io/docs/wasm-bindgen/) | Reference | 30 min | Reference |

**Topics**:
- GOOS=js GOARCH=wasm
- syscall/js package
- Memory management in WASM
- JavaScript interop

---

## Recommended Reading Order

### For PNG-Only Focus (go-pixo MVP)
```
Week 1:
  1.1 → 1.2 → 2.1 → 2.2 → 2.3

Week 2:
  3.1 → 3.2 → 3.3 → 5.1 → 5.2

Week 3-4:
  Implement PNG encoder
  5.3 (testing as you go)
```

### For Full PNG + JPEG
```
Week 1:
  1.1 → 1.2 → 1.3 → 2.1 → 2.2 → 2.3

Week 2:
  3.1 → 3.2 → 3.3 → 4.1

Week 3:
  4.2 → 4.3 → 5.1 → 5.2

Week 4-6:
  Implement PNG encoder
  Then implement JPEG encoder
```

---

## Concept Dependency Graph

```
1.1 Why Compress?
       ↓
1.2 PNG Format ───────────────→ 3.1 PNG Filters
       ↓                        ↓
2.1 Huffman ─────────────────→ 3.2 PNG Chunks
       ↓                        ↓
2.2 LZ77 ────────────────────→ 3.3 Zlib/DEFLATE
       ↓
2.3 DEFLATE ─────────────────→ PNG Complete ✓
       ↓
4.1 DCT ─────────────────────→ 4.2 Quantization
       ↓
4.3 JPEG Pipeline ───────────→ JPEG Complete ✓
       ↓
5.1 Bit Operations ──────────→ Implementation
       ↓
5.2 Go + WASM ───────────────→ Deployment
```

---

## Quick Reference: Internal Documentation

All these docs are in `/Users/mac/WebApps/projects/go-pixo/docs/`:

| Document | Phase | What It Covers |
|----------|-------|----------------|
| `introduction-to-image-compression.md` | 1.1 | Lossless vs lossy, entropy basics |
| `huffman-coding.md` | 2.1 | Huffman tree, canonical codes |
| `lz77-compression.md` | 2.2 | Sliding window, back-references |
| `deflate.md` | 2.3 | LZ77 + Huffman combination |
| `png-encoding.md` | 3.1-3.3 | Filters, chunks, zlib wrapper |
| `dct.md` | 4.1 | 2D DCT formula, 8×8 blocks |
| `quantization.md` | 4.2 | Quality tables, zigzag |
| `jpeg-encoding.md` | 4.3 | Complete JPEG pipeline |
| `performance-optimization.md` | 6.1 | SIMD, parallel, buffer reuse |
| `compression-evolution.md` | 6.1 | History, Zopfli, oxipng |

---

## External Reference Links

### Specifications (Authoritative)
| Link | What It Covers |
|------|----------------|
| [RFC 2083 - PNG](https://datatracker.ietf.org/doc/rfc2083/) | PNG format, chunks, filters |
| [RFC 1950 - Zlib](https://datatracker.ietf.org/doc/rfc1950/) | zlib wrapper format |
| [RFC 1951 - DEFLATE](https://datatracker.ietf.org/doc/rfc1951/) | DEFLATE algorithm |
| [W3C PNG Filters](https://www.w3.org/TR/PNG-Filters.html) | Filter type specifications |
| [PNG Spec (4th Ed)](https://w3c.github.io/png/) | Latest PNG specification |

### Tutorials
| Link | What It Covers |
|------|----------------|
| [How PNG Works - Medium](https://medium.com/@duhroach/how-png-works-f1174e3cc7b7) | PNG compression overview |
| [DEFLATE Explanation - zlib](https://zlib.net/feldspar.html) | DEFLATE deep dive |
| [LZ77 Algorithm](https://brandon.si/code/fun-with-lazy-arrays-the-lz77-algorithm/) | LZ77 explained |
| [Huffman Coding Tutorial](https://www.kosbie.net/cmu/fall-15/15-112/notes/notes-data-compression.html) | Huffman coding |
| [PNG: The Definitive Guide](http://www.libpng.org/pub/png/book/chapter09.html) | PNG filtering |

### Go-Specific
| Link | What It Covers |
|------|----------------|
| [Go Testing](https://go.dev/doc/testing) | Testing in Go |
| [syscall/js](https://pkg.go.dev/syscall/js) | WASM interop |
| [io.Reader](https://pkg.go.dev/io) | Streaming I/O |
| [image Package](https://pkg.go.dev/image) | Image decoding/encoding |

---

## Practice Exercises

### Exercise 1: Huffman Coding (After 2.1)
```go
// Given: "ABABABAB"
// 1. Count frequencies
// 2. Build Huffman tree
// 3. Generate canonical codes
// 4. Encode to bits
// 5. Calculate compression ratio
```

### Exercise 2: LZ77 (After 2.2)
```go
// Given: "ABABABABAB"
// 1. Find longest matches
// 2. Emit (distance, length) pairs
// 3. Calculate compression
```

### Exercise 3: Paeth Filter (After 3.1)
```go
// Given row: [255, 0, 0, 255, 0, 0]
// Previous: [0, 255, 0, 0, 255, 0]
// bpp = 3
// 1. Compute each filter type output
// 2. Calculate sum of absolute values
// 3. Choose best filter
```

### Exercise 4: DEFLATE Block (After 2.3 + 3.3)
```go
// Given: "HELLOHELLO"
// 1. Apply LZ77 → get literals/lengths
// 2. Build Huffman tree
// 3. Write stored/dynamic block
// 4. Add zlib header/footer
```

---

## Time Investment Summary

| Phase | Topics | Estimated Time |
|-------|--------|----------------|
| 1 | Foundations | 3-4 hours |
| 2 | Compression Algorithms | 5-6 hours |
| 3 | PNG-Specific | 4-5 hours |
| 4 | JPEG-Specific | 4-5 hours |
| 5 | Implementation Skills | 8-10 hours |
| 6 | Advanced Topics | 4-6 hours |
| **Total** | | **28-36 hours** |

**That's about 1-2 weeks of focused learning!**

---

## Milestones

| Milestone | What You Can Do |
|-----------|-----------------|
| ✅ After Phase 1-2 | Explain compression to someone else |
| ✅ After Phase 3 | Read a PNG file and identify all chunks |
| ✅ After Phase 4 | Read a JPEG file and identify markers |
| ✅ After Phase 5 | Implement a working PNG encoder |
| ✅ After Phase 6 | Deploy PNG encoder to browser via WASM |

---

## Next Steps

1. **Start with Phase 1.1**: Read the internal [Introduction to Image Compression](../introduction-to-image-compression.md)
2. **Follow the order**: Don't skip ahead - concepts build on each other
3. **Practice**: Don't just read - implement small exercises
4. **Reference**: Keep the [pixo-feature.md](pixo-feature.md) handy for implementation details

Good luck on your compression journey! 🚀
