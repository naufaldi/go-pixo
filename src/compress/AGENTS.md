# Compression Core Package

Core compression algorithms: DEFLATE, LZ77, Huffman coding, zlib integration.

## Package Identity

- **Purpose**: Low-level compression primitives shared by PNG and JPEG encoders
- **Technology**: Pure Go, no dependencies
- **Tests**: Table-driven with `t.Run`, descriptive names

## Setup & Run

```bash
# Run tests
go test ./src/compress/...

# Run specific test
go test -run TestHuffman ./src/compress/...

# Lint
golangci-lint run ./src/compress/...

# Benchmark
go test -bench=. ./src/compress/...
```

## Patterns & Conventions

**File Organization**:
```
src/compress/
├── *_types.go      # Type definitions
├── *_encoder.go     # Encoders
├── *_decoder.go     # Decoders (if any)
├── *_test.go        # Tests
└── *_constants.go  # Constants
```

**Key Patterns**:
- ✅ DO: Use descriptive test names with `t.Run("name", func(t *testing.T){...})`
- ❌ DON'T: Suppress errors with `_` or `panic`
- ✅ DO: Export core algorithms (PascalCase), keep helpers private (camelCase)
- ❌ DON'T: Mix concerns (e.g., encoding logic in type definition files)

**Testing**:
- ✅ DO: Use table-driven tests for multiple scenarios
- Example: `src/compress/huffman_codes_test.go:67-89`
- ✅ DO: Test edge cases (empty data, max values, boundary conditions)
- ✅ DO: Benchmark performance-critical code with `go test -bench=.`

**Documentation**:
- ✅ DO: Add Godoc comments to all exported functions
- Example: `src/compress/deflate_encoder.go:23-35`

## Touch Points / Key Files

- **LZ77**: `src/compress/lz77_encoder.go` (sequential compression)
- **Huffman**: `src/compress/huffman_codes.go` (canonical codes)
- **Bit Writer**: `src/compress/bit_writer.go` (LSB-first bit streaming)
- **DEFLATE**: `src/compress/deflate_encoder.go` (main encoder)

## JIT Index Hints

```bash
# Find encoder implementation
rg -n "Encode.*func" src/compress/

# Find test for specific algorithm
rg -n "func Test.*Huffman" src/compress/*_test.go

# Find constants
rg -n "const.*Block" src/compress/

# Find benchmarks
rg -n "func Benchmark" src/compress/*_test.go
```

## Common Gotchas

- **Bit Order**: DEFLATE uses LSB-first bit ordering (bit 0 = LSB)
- **Integer Sizes**: Respect DEFLATE limits (max distance 32K, max length 258)
- **Memory**: Sliding window is 32KB circular buffer (see `lz77_sliding_window.go`)
- **Zero Runs**: AC encoding requires careful zero run-length tracking

## Pre-PR Checks

```bash
go test ./src/compress/... && golangci-lint run ./src/compress/...
```
