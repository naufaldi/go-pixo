# Dynamic Huffman: Corrupt Stream Postmortem

This is a “learn from our failure” note for `go-pixo`: dynamic Huffman encoding sometimes produced a byte stream that **looked** successful (no encoder error), but decoders failed with errors like:

- `unexpected EOF`
- `flate: corrupt input ...`

These failures happened in `src/compress` unit tests and also surfaced at the PNG layer because PNG IDAT uses zlib-wrapped DEFLATE.

## Symptom

- Fixed Huffman blocks decompressed fine.
- Dynamic Huffman blocks sometimes failed decompression, even for small token streams.
- `EncodeAuto` did not always fall back to fixed blocks, because the dynamic path returned bytes without returning an error.

## Root causes (what actually corrupted output)

### 1) Dynamic header invariants were violated

The DEFLATE dynamic header (RFC 1951) is extremely strict. Two key issues:

- **HLIT must include symbol 256** (End-of-Block).
  - If HLIT doesn’t cover index 256, the decoder can’t read the EOB symbol and the block can terminate incorrectly.

- **HCLEN must be at least 4**.
  - Even if only a couple code-length codes are non-zero, the header must still transmit at least 4 code-length code lengths in the `CodeLengthOrder` sequence.

### 2) Code-length table mismatch (encoder vs header)

Dynamic blocks use a *third* Huffman alphabet (the “code-length alphabet”, 0–18) to encode the RLE stream of literal/length + distance code lengths.

The decoder builds that code-length Huffman tree from the **code-length code lengths you transmitted**. The encoder must use the **same** canonical table implied by those lengths when it writes the RLE stream.

Previously, `go-pixo` built a code-length table by rebuilding a Huffman tree from “presence” frequency (1/0), which can produce a different canonical assignment than the table implied by the transmitted lengths. That makes the decoder misread the RLE stream and everything after becomes garbage.

### 3) RLE (16/17/18) encoding bugs

The RLE rules for encoding code lengths have tight ranges:

- `16`: repeat previous non-zero length 3–6 times (extra = repeat-3, 2 bits)
- `17`: repeat zero 3–10 times (extra = repeat-3, 3 bits)
- `18`: repeat zero 11–138 times (extra = repeat-11, 7 bits)

Previously we had:
- an off-by-one for symbol `16` (repeat count handling)
- missing splitting of long runs (e.g. >138 zeros)

Both can desynchronize the decoder.

### 4) LZ77 state leak across `EncodeAuto` attempts

`DeflateEncoder` reuses an internal `LZ77Encoder` with a sliding window. Each independent DEFLATE stream must start with an **empty** history window.

`EncodeAuto` encodes twice (fixed then dynamic). If the LZ77 window isn’t reset per encode attempt, the dynamic attempt can emit matches that reference bytes from the previous attempt — producing invalid DEFLATE output even though the encoder “succeeded”.

## Fix summary

The fixes landed in:

- `src/compress/huffman_header.go`
  - enforce HLIT/EOB minimum and HCLEN minimum
  - ensure code-length table is built canonically from transmitted lengths
  - correct RLE splitting and off-by-one behavior

- `src/compress/lz77_encoder.go` and `src/compress/lz77_sliding_window.go`
  - reset the sliding window per `Encode()` call

## How to detect this class of bug early

1) Always validate encoder output by **decoding it** in tests:
   - DEFLATE: `compress/flate.NewReader`
   - zlib wrapper: `compress/zlib.NewReader`

2) Treat “bytes returned” as *not sufficient* for success:
   - A dynamic encoder can emit bytes and still be invalid.
   - In auto-mode, you may want a decode-verify step (or rely on tests) before trusting the dynamic result.

## Related docs

- [DEFLATE Block Writer](deflate-block-writer.md) (block format + where pitfalls happen)
- [DEFLATE Encoder](deflate-encoder.md) (pipeline + auto mode considerations)
- [IDAT Zlib Integration](idat-zlib-integration.md) (how DEFLATE errors surface as PNG decode failures)
