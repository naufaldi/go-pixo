# PNG Encoder Package

Complete PNG encoder with filters, DEFLATE compression, and optimization options.

## Package Identity

- **Purpose**: Production-ready PNG encoder with lossless and lossy modes
- **Technology**: Pure Go, depends on `src/compress` for DEFLATE
- **Tests**: Integration tests with valid PNG output verification

## Setup & Run

```bash
# Run PNG tests
go test ./src/png/...

# Test specific feature
go test -run TestFilter ./src/png/...

# Benchmark encoding
go test -bench=. ./src/png/...

# Lint
golangci-lint run ./src/png/...
```

## Patterns & Conventions

**File Organization**:
```
src/png/
├── encoder.go       # Main Encoder struct
├── *_writer.go     # Chunk/scanline writers
├── *_test.go       # Tests
├── options.go      # Configuration presets
└── *_reduce.go     # Color reduction algorithms
```

**Key Patterns**:
- ✅ DO: Follow PNG spec strictly (chunk order, CRC32, zlib format)
- ✅ DO: Use `FilterType` constants for filter selection
- ✅ DO: Test with real PNG files from `images/` directory
- ❌ DON'T: Skip CRC32 validation on chunks
- ❌ DON'T: Write ancillary chunks (tEXt, zTXt) unless needed

**Encoding Pipeline**:
```
Pixels → Filter Selection → IDAT Writer → Zlib → PNG
```

**Filter Selection** (see `src/png/filter_selector.go:45-67`):
- ✅ DO: Use `SumAbsoluteValues()` for quick filtering
- ✅ DO: Use `SelectFilter()` for optimal per-row selection
- ❌ DON'T: Always use filter type 0 (None)

**Color Reduction** (see `src/png/color_reduce.go:23-45`):
- ✅ DO: Check `CanReduceToGrayscale()` before conversion
- ✅ DO: Verify `HasAlpha()` before alpha optimization
- ❌ DON'T: Reduce colors without checking if image qualifies

## Touch Points / Key Files

- **Main Encoder**: `src/png/encoder.go:67-134` (encoding pipeline)
- **Options**: `src/png/options.go:23-89` (preset system)
- **Filter Selection**: `src/png/filter_selector.go:34-78` (adaptive filtering)
- **IDAT Writer**: `src/png/idat_writer.go:56-123` (zlib integration)

## JIT Index Hints

```bash
# Find encoder usage
rg -n "NewEncoder" src/png/*

# Find filter implementations
rg -n "Filter.*func" src/png/

# Find optimization options
rg -n "Options.*struct" src/png/options.go

# Find test images
ls images/*.png
```

## Common Gotchas

- **Scanline Order**: PNG is bottom-to-top (different from many formats)
- **Filter Bytes**: Each scanline starts with filter type byte (0-4)
- **Zlib Integration**: Must use DEFLATE with proper header/footer
- **Color Types**: PNG has 6 color types, ensure proper conversion
- **Memory**: Large images may need streaming (not fully implemented)

## Pre-PR Checks

```bash
go test ./src/png/... && golangci-lint run ./src/png/...
```
