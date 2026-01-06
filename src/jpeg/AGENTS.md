# JPEG Encoder Package

Complete JPEG encoder with baseline and progressive modes, trellis quantization, and optimization options.

## Package Identity

- **Purpose**: Production-ready JPEG encoder for photographs and images
- **Technology**: Pure Go, depends on `src/compress` for Huffman
- **Tests**: Round-trip validation with `image/jpeg` decoder

## Setup & Run

```bash
# Run JPEG tests
go test ./src/jpeg/...

# Test specific feature
go test -run TestProgressive ./src/jpeg/...

# Benchmark encoding
go test -bench=. ./src/jpeg/...

# Lint
golangci-lint run ./src/jpeg/...
```

## Patterns & Conventions

**File Organization**:
```
src/jpeg/
├── encoder.go        # Main encoder (baseline + progressive)
├── dct.go          # Discrete Cosine Transform
├── quantize.go     # Standard + Trellis quantization
├── huffman.go      # Standard Huffman tables
├── *_optimized.go  # Optimized algorithms
└── *_test.go      # Tests
```

**Key Patterns**:
- ✅ DO: Follow JPEG spec markers exactly (SOI, DQT, SOF, DHT, SOS, EOI)
- ✅ DO: Use YCbCr color space (not RGB) for DCT
- ✅ DO: Implement zigzag reordering for AC coefficients
- ❌ DON'T: Skip byte stuffing (0xFF → 0xFF 0x00)
- ❌ DON'T: Use wrong quantization tables for luminance/chrominance

**Encoding Pipeline**:
```
RGB → YCbCr → Blocks → DCT → Quantize → Zigzag → Huffman → Markers
```

**Baseline vs Progressive** (see `src/jpeg/encoder.go:120-203`):
- ✅ DO: Use `SimpleProgressiveScript()` for web delivery
- ✅ DO: Use `HighQualityProgressiveScript()` for max quality
- ❌ DON'T: Mix scan parameters (SS/SE ranges)

**Trellis Quantization** (see `src/jpeg/trellis.go:23-67`):
- ✅ DO: Enable with `TrellisQuant: true` in options
- ✅ DO: Use `CalculateLambda()` based on quality (1-100)
- ❌ DON'T: Apply trellis to DC coefficient (index 0)
- Expected: 5-15% compression improvement

## Touch Points / Key Files

- **Main Encoder**: `src/jpeg/encoder.go:30-95` (encoding entry)
- **DCT**: `src/jpeg/dct.go:34-67` (AAN algorithm)
- **Progressive**: `src/jpeg/progressive.go:26-89` (scan scripts)
- **Trellis**: `src/jpeg/trellis.go:45-123` (rate-distortion)

## JIT Index Hints

```bash
# Find encoder modes
rg -n "Progressive.*bool" src/jpeg/options.go

# Find scan scripts
rg -n "ProgressiveScript" src/jpeg/progressive.go

# Find quantization
rg -n "Quantize" src/jpeg/quantize.go

# Find marker writing
rg -n "WriteSOI\|WriteEOI" src/jpeg/markers.go
```

## Common Gotchas

- **Bit Order**: JPEG uses MSB-first (opposite of DEFLATE)
- **YCbCr Conversion**: Use ITU-R BT.601 formula (see `src/jpeg/color.go:23-45`)
- **MCU Structure**: 4:2:0 subsampling = 4 Y + 1 Cb + 1 Cr per 16×16 block
- **Progressive Memory**: Must store ALL coefficients before writing first scan
- **Quality Scaling**: Q > 50 uses different formula than Q ≤ 50

## Pre-PR Checks

```bash
go test ./src/jpeg/... && golangci-lint run ./src/jpeg/...
```
