# Rust reference: `pixo` PNG compression

This note links the Rust implementation in `../pixo` (outside this repo) to the PNG/DEFLATE concepts documented here, so you can cross-check behavior while building `go-pixo`.

Primary Rust files:
- `../pixo/src/png/mod.rs` (PNG encoder pipeline)
- `../pixo/src/decode/png.rs` (PNG decoder; IDAT → zlib inflate → unfilter)
- `../pixo/src/compress/deflate.rs` (DEFLATE + zlib wrapper)
- `../pixo/src/compress/lz77.rs` (matchfinding + greedy/lazy + optimal parsing)
- `../pixo/src/compress/huffman.rs` (Huffman + canonical codes; fixed tables)

## PNG compression in `pixo` (encoder)

High level:

```
pixels → (optional transforms) → per-row PNG filter → zlib(deflate(filtered_scanlines)) → IDAT
```

The “filtered scanlines” buffer is *exactly* what PNG specifies for IDAT before compression:
- Each row starts with a 1-byte filter type (0–4)
- Followed by the row’s raw bytes (packed if bit depth < 8)

In `pixo/src/png/mod.rs`, this shows up as:
- `filter::apply_filters_with_row_bytes(...)` producing `filtered`
- Compression as either:
  - `deflate_zlib_packed(&filtered, options.compression_level)` (fast path)
  - `deflate_optimal_zlib(&filtered, 5)` (Zopfli-style, slower/smaller)
- `write_idat_chunks(output, &compressed)` writing the zlib stream into one or more `IDAT` chunks.

## zlib wrapper vs raw DEFLATE

PNG uses a *zlib* stream (RFC 1950) inside IDAT, not “raw DEFLATE” (RFC 1951).

In `pixo/src/compress/deflate.rs`:
- `deflate_*` functions return **raw DEFLATE**
- `deflate_zlib*` / `deflate_optimal_zlib` return **zlib-wrapped DEFLATE**:
  - zlib header (CMF/FLG)
  - DEFLATE bitstream
  - Adler-32 of the *uncompressed* data

This matches the integration described in `docs/learning/png/idat-zlib-integration.md` and `docs/learning/png/zlib.md`.

## DEFLATE internals (how `compress/` supports PNG)

### LZ77 tokenization (`pixo/src/compress/lz77.rs`)

`Lz77Compressor` scans the filtered byte stream and emits:
- `Token::Literal(u8)` for raw bytes
- `Token::Match { length, distance }` for back-references within a 32KiB window

PNG-filtered data often contains long runs of zeros (especially with Sub/Up). `pixo` has a dedicated “same-byte run” detector to cheaply turn these into distance=1 matches, which both speeds up searching and usually improves compression.

For max compression, `compress_optimal` does dynamic programming over “literal vs match length” choices, using a `CostModel`:
- Start with a baseline parse to get symbol stats
- Build costs from entropy
- Re-parse optimally with those costs
- Iterate (Zopfli-style refinement), reusing a `LongestMatchCache` so it doesn’t recompute matches each iteration

### Huffman coding (`pixo/src/compress/huffman.rs`)

`build_codes(frequencies, max_length)`:
- Builds a Huffman tree
- Extracts code lengths
- Generates **canonical** codes (DEFLATE requirement)

It also precomputes fixed Huffman tables (`fixed_literal_codes`, `fixed_distance_codes`) for block type 1.

### Block writing (`pixo/src/compress/deflate.rs`)

The encoder chooses between:
- stored blocks (uncompressed) for high-entropy/incompressible data
- fixed Huffman blocks (small/simple)
- dynamic Huffman blocks (best ratio for large/repetitive data like scanlines)

Dynamic blocks are written in `write_dynamic_huffman_block(...)`:
- Count literal/length + distance frequencies from tokens
- Build Huffman codes via `huffman::build_codes`
- RLE-encode code lengths, write HLIT/HDIST/HCLEN + code-length alphabet
- Encode token stream and end-of-block symbol

Bit packing is LSB-first (DEFLATE requirement), same as described in `docs/learning/png/deflate-block-writer.md`.

## Decoder cross-check (`pixo/src/decode/png.rs`)

On the decode side, `pixo` does the inverse of the encoder pipeline:
- Parse chunks, concatenate IDAT payload bytes
- Inflate with a zlib-aware inflater (`inflate_zlib_with_size(...)`)
- Unfilter rows (None/Sub/Up/Average/Paeth)
- Expand to pixels (palette, tRNS, bit-depth unpacking, etc.)

This is useful for validating that the output `go-pixo` produces is standards-compliant: if `pixo` can decode it, you’re very likely writing correct IDAT/zlib/filters.
