# DEFLATE Encoder

This document explains the DEFLATE encoder pipeline: how raw data flows through LZ77 tokenization and Huffman encoding to produce compressed DEFLATE blocks.

## Pipeline Overview

```
Raw Data → LZ77 Encoder → Tokens → Huffman Encoder → DEFLATE Block → Compressed Data
```

1. **LZ77 Encoder**: Finds repeated sequences, emits literals or matches
2. **Token Frequency Counting**: Counts how often each symbol appears
3. **Huffman Table Building**: Builds optimal codes from frequencies (fixed or dynamic)
4. **Block Writing**: Encodes tokens using Huffman codes, writes DEFLATE block

## LZ77 Tokenization

The LZ77 encoder (`src/compress/lz77_encoder.go`) scans the input data and emits tokens:

- **Literal tokens**: Single bytes that don't match previous data
- **Match tokens**: Back-references (distance, length) to previously seen data

Example:
```
Input: "ABCABC"
Tokens: [Literal('A'), Literal('B'), Literal('C'), Match(distance=3, length=3)]
```

The sliding window (32KB) tracks recently seen data to find matches.

## Token Frequency Counting

Before building Huffman tables, we count symbol frequencies:

- **Literal/length frequencies**: Count occurrences of:
  - Literals 0-255 (byte values)
  - Length codes 257-285 (for matches)
  - End-of-block symbol 256
- **Distance frequencies**: Count occurrences of distance codes 0-29

This happens in `countTokenFrequencies` (`src/compress/deflate_block.go`).

## Huffman Table Building

### Fixed Tables

Fixed tables are predefined in RFC 1951. No computation needed—just use `LiteralLengthTable()` and `DistanceTable()`.

**When to use:** Small data, speed-critical encoding, or when dynamic tables wouldn't help.

### Dynamic Tables

Dynamic tables are built from actual frequencies:

1. Build Huffman tree from frequencies (`BuildTree`)
2. Extract code lengths from tree (`GenerateCodes`)
3. Convert to canonical codes (`Canonicalize`)
4. Ensure table has entries for all possible symbols (0-286 for literal/length, 0-29 for distance)

**When to use:** Larger data where custom tables improve compression.

## Auto Mode

`EncodeAuto` tries both fixed and dynamic compression and returns the smaller result:

```go
encoder := NewDeflateEncoder()
compressed, err := encoder.EncodeAuto(data)
```

**Why auto mode?** Fixed tables are faster but may not compress as well. Dynamic tables compress better but have overhead. Auto mode picks the best trade-off automatically.

**Trade-offs:**
- Fixed: Fast encoding, no table overhead, predictable size
- Dynamic: Better compression for larger/repetitive data, but table overhead (HLIT/HDIST/HCLEN + code lengths)

### Important: encoder state must not leak across streams

Our `DeflateEncoder` reuses an internal `LZ77Encoder`. LZ77 has a **sliding window history** (up to 32KiB). For DEFLATE, that history must start empty for each independent DEFLATE stream.

This matters in two places:

1) **Encoding multiple streams with the same encoder instance**

If you call `enc.Encode(...)` multiple times, you must reset the LZ77 window each time or you can emit matches that reference bytes from a previous stream (corrupt output).

2) **`EncodeAuto` (fixed then dynamic)**

`EncodeAuto` runs *two* encodes back-to-back (fixed, then dynamic). If the LZ77 window isn’t reset per encode attempt, the dynamic attempt can be corrupted even though it “succeeds” (no error returned). This showed up as decompression errors like **`unexpected EOF`**.

We fix this by resetting the sliding window at the start of `LZ77Encoder.Encode()`.

For a concrete write-up of the failure mode and the dynamic header pitfalls, see:
- [Dynamic Huffman: Corrupt Stream Postmortem](dynamic-huffman-corrupt-stream.md)

## Implementation

The encoder is in `src/compress/deflate_encoder.go`:

- `NewDeflateEncoder()`: Creates a new encoder
- `Encode(data, useDynamic)`: Compresses with fixed or dynamic tables
- `EncodeAuto(data)`: Automatically chooses the better method

The encoder uses `LZ77Encoder` for tokenization and `WriteFixedBlock`/`WriteDynamicBlock` for block writing.

## Example Usage

```go
encoder := compress.NewDeflateEncoder()

// Fixed compression (faster)
fixed, err := encoder.Encode(data, false)

// Dynamic compression (better ratio)
dynamic, err := encoder.Encode(data, true)

// Auto (best of both)
best, err := encoder.EncodeAuto(data)
```

All three produce valid DEFLATE streams that can be decompressed by standard DEFLATE decoders.
