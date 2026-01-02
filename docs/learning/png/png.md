# PNG File Format: Signature and Constants

This guide explains the foundational elements of PNG files: the 8-byte signature and the constants used throughout PNG encoding and decoding.

## The PNG Signature: 8 Magic Bytes

Every PNG file **must** start with exactly these 8 bytes:

```text
89 50 4E 47 0D 0A 1A 0A
```

### Why These Specific Bytes?

Each byte serves a specific purpose:

#### Byte 1: `0x89` (137 in decimal)

- **Purpose**: High bit set (bit 7 = 1) prevents the file from being mistaken for a **text file**.
- **Why it matters**: Old systems used the high bit to detect binary vs text. If a PNG file is accidentally opened as text, the high bit signals "this is binary data, don't display it as ASCII."

#### Bytes 2-4: `0x50 0x4E 0x47` ("PNG" in ASCII)

- **Purpose**: Human-readable file type identifier.
- **Why it matters**: If you open a PNG file in a text editor, you'll see "PNG" starting at byte 2. This makes debugging easier.

#### Bytes 5-6: `0x0D 0x0A` (Carriage Return + Line Feed)

- **Purpose**: **Detect file transfer corruption**.
- **Why it matters**: On old systems (especially Windows/DOS), files transferred via text mode would convert line endings. If a PNG file was transferred incorrectly:
  - Unix line endings (`0x0A`) might become Windows line endings (`0x0D 0x0A`)
  - This would corrupt the signature and the decoder would reject it
- **Result**: The decoder immediately knows "this file was corrupted during transfer" and refuses to process it.

#### Byte 7: `0x1A` (26, Ctrl+Z)

- **Purpose**: **Stop text display** on old DOS systems.
- **Why it matters**: On DOS, `Ctrl+Z` was the "end of file" marker for text files. If someone tried to `type image.png` on DOS, it would stop displaying at byte 7, preventing binary garbage from flooding the screen.

#### Byte 8: `0x0A` (Line Feed)

- **Purpose**: Completes the line ending pattern and ensures Unix-style line ending detection works.

### Validation Logic

Our `IsValidSignature()` function checks these 8 bytes:

```go
func IsValidSignature(data []byte) bool {
	if len(data) < 8 {
		return false  // Can't be a PNG if it's shorter than 8 bytes
	}
	return bytes.Equal(data[:8], PNG_SIGNATURE[:])
}
```

**Why this matters**: Before we even look at pixels, chunks, or compression, we validate the signature. If it's wrong, we immediately return an error instead of wasting CPU cycles trying to decode garbage.

## PNG Constants: Preventing Magic Number Bugs

Instead of scattering "magic numbers" throughout the code, we define constants. This makes the code:

- **Readable**: `ColorRGBA` is clearer than `6`
- **Maintainable**: Change the value in one place, not 50
- **Type-safe**: Go's type system catches mistakes at compile time

### Chunk Types

PNG files are made of **chunks**. Each chunk has a 4-character ASCII type name:

```go
type ChunkType string

const (
	ChunkIHDR ChunkType = "IHDR"  // Image Header (width, height, color type)
	ChunkIDAT ChunkType = "IDAT"  // Image Data (compressed pixel rows)
	ChunkIEND ChunkType = "IEND"  // Image End (marks end of file)
)
```

**Why strings?**: PNG chunk types are 4-byte ASCII identifiers. Using `string` makes the code readable and prevents typos. For example, writing `ChunkIHDR` is safer than writing `"IHDR"` directly (typo: `"IHRD"` would compile but be wrong).

### Color Types

PNG supports different pixel formats. Each format has a numeric code:

```go
type ColorType uint8

const (
	ColorGrayscale ColorType = 0  // 1 channel: grayscale only
	ColorRGB       ColorType = 2  // 3 channels: Red, Green, Blue
	ColorRGBA      ColorType = 6  // 4 channels: Red, Green, Blue, Alpha
)
```

**Why constants?**: Instead of writing `if colorType == 6` everywhere, we write `if colorType == ColorRGBA`. This prevents bugs:

- **Bug-prone**: `if colorType == 5` (typo: 5 doesn't exist!)
- **Safe**: `if colorType == ColorRGBA` (Go compiler catches typos)

**Why these numbers?**: PNG spec defines:

- `0` = Grayscale
- `2` = RGB (no palette)
- `6` = RGBA (RGB + alpha channel)

The gaps (`1`, `3`, `4`, `5`) are reserved for palette-based modes we'll implement later.

### Filter Types

Before compression, PNG applies **filters** to each row. Each filter predicts pixel values differently:

```go
type FilterType uint8

const (
	FilterNone    FilterType = 0  // No filtering (raw pixel values)
	FilterSub     FilterType = 1  // Predict from left pixel
	FilterUp      FilterType = 2  // Predict from above pixel
	FilterAverage FilterType = 3  // Predict from average of left + above
	FilterPaeth   FilterType = 4  // Predict using Paeth algorithm
)
```

**Why constants?**: When we write filter selection logic, we use `FilterPaeth` instead of `4`. This makes the code self-documenting:

- **Unclear**: `if filter == 4 { ... }`
- **Clear**: `if filter == FilterPaeth { ... }`

## How Constants Prevent Bugs

### Example 1: Magic Number Bug

**Without constants** (bug-prone):

```go
func encodeColorType(ct uint8) error {
	if ct == 6 {  // What does 6 mean? Is this RGBA or something else?
		return encodeRGBA()
	}
	return nil
}
```

**With constants** (safe):

```go
func encodeColorType(ct ColorType) error {
	if ct == ColorRGBA {  // Clear: we're handling RGBA
		return encodeRGBA()
	}
	return nil
}
```

### Example 2: Typo Detection

**Without constants**:

```go
if chunkType == "IHRD" {  // Typo! Should be "IHDR"
	// This compiles but is wrong!
}
```

**With constants**:

```go
	if chunkType == ChunkIHDR {  // Go compiler catches typos
		// Safe!
	}
```

## IDAT Compression Pipeline

PNG image data flows through multiple compression layers:

```
PNG Pixels → Scanlines → Filters → DEFLATE → Zlib → IDAT Chunk
```

### The Complete Flow

1. **Raw Pixels**: Image data as RGB/RGBA bytes
2. **Scanlines**: Each row gets a filter byte (Phase 1 uses filter 0 = None)
3. **DEFLATE Compression** (Phase 2):
   - **LZ77**: Finds repeated patterns, emits tokens (literals + matches)
   - **Huffman Coding**: Assigns variable-length codes to tokens
   - **Bit Stream**: Codes written LSB-first
4. **Zlib Wrapper**: Adds CMF/FLG header and Adler32 footer
5. **IDAT Chunk**: Wraps compressed data with chunk structure (length + "IDAT" + data + CRC32)

See [IDAT Zlib Integration](idat-zlib-integration.md) for details on how zlib wraps DEFLATE in PNG IDAT chunks.
See `docs/learning/png/zlib.md` for details on LZ77 and Huffman coding internals.

## Summary: Signature, Constants, and Validation

1. **PNG Signature**: 8 bytes that identify the file format and detect corruption.

   - `0x89`: High bit prevents text file confusion
   - `"PNG"`: Human-readable identifier
   - `0x0D 0x0A`: Detects transfer corruption
   - `0x1A`: Stops DOS text display
   - `0x0A`: Completes line ending pattern

2. **Constants**: Type-safe, readable identifiers for chunk types, color types, and filter types.

   - Prevents magic number bugs
   - Makes code self-documenting
   - Catches typos at compile time

3. **Validation**: Always check the signature first. If it's wrong, reject the file immediately.

4. **Compression**: IDAT chunks use DEFLATE (LZ77 + Huffman) wrapped in zlib format for efficient storage.

## The IEND Chunk: End of File Marker

Every valid PNG file **must** end with an IEND chunk. This chunk marks the end of the image data and tells the decoder "there is nothing more to read."

### IEND Chunk Structure

```text
4 bytes: Length = 0x00000000 (no data)
4 bytes: Type  = "IEND" (0x49 0x45 0x4E 0x44)
4 bytes: CRC32 of "IEND" (0xAE426082)
```

### Implementation

The IEND chunk is simple - it has no data, only the chunk type and CRC:

```go
func WriteIEND(w io.Writer) error {
	chunk := Chunk{chunkType: ChunkIEND, Data: nil}
	_, err := chunk.WriteTo(w)
	return err
}
```

**Why no data?**: The IEND chunk serves only as a marker. The PNG spec requires it, but it carries no information. The CRC32 covers just the type "IEND" to detect corruption.

**Why is it required?**: The IEND chunk provides a clear "end of file" boundary. Without it, the decoder wouldn't know if there are more chunks or if the file was truncated.

**Total size**: 12 bytes (4 length + 4 type + 4 CRC)

## Adler32 Checksum for Zlib

PNG uses **zlib** format (RFC 1950) to wrap compressed image data. Unlike PNG chunks which use CRC32, zlib uses **Adler32** for its checksum.

### Why Adler32 Instead of CRC32?

| Feature         | CRC32     | Adler32              |
| --------------- | --------- | -------------------- |
| Speed           | Slower    | Faster               |
| Error Detection | Excellent | Very Good            |
| Streaming       | Good      | Better for streaming |
| RFC 1950 (zlib) | Not used  | Required             |
| PNG chunks      | Required  | Not used             |

Adler32 is faster to compute and works better for streaming scenarios, which is why zlib (and thus PNG) uses it.

### Adler32 Algorithm

The algorithm maintains two running sums, s1 and s2, modulo 65521:

```go
const adler32Mod = 65521

func Adler32(data []byte) uint32 {
	if len(data) == 0 {
		return 1  // Empty data returns 1
	}

	s1 := uint32(1)
	s2 := uint32(0)

	for _, b := range data {
		s1 = (s1 + uint32(b)) % adler32Mod
		s2 = (s2 + s1) % adler32Mod
	}

	return s2<<16 | s1
}
```

**How it works**:

- `s1`: Sum of all bytes + 1 (mod 65521)
- `s2`: Sum of all s1 values (mod 65521)
- Final value: `(s2 << 16) | s1`

**Why modulo 65521?**: 65521 is the largest prime less than 2^16 (65536). This provides good distribution for the checksum.

### Streaming Interface

For large data, we implement `hash.Hash32`:

```go
type adler32Writer struct {
	s1 uint32
	s2 uint32
}

func NewAdler32() hash.Hash32 {
	return &adler32Writer{s1: 1, s2: 0}
}

func (a *adler32Writer) Write(p []byte) (n int, err error) {
	for _, b := range p {
		a.s1 = (a.s1 + uint32(b)) % adler32Mod
		a.s2 = (a.s2 + a.s1) % adler32Mod
	}
	return len(p), nil
}
```

This allows streaming computation without loading all data into memory.

### Test Vectors (RFC 1950)

| Input      | Adler32    |
| ---------- | ---------- |
| "" (empty) | 0x00000001 |
| "A"        | 0x00420042 |
| "ABC"      | 0x02280121 |

## Zlib Header and Footer

The zlib format wraps DEFLATE-compressed data with a specific header and footer structure:

```text
[CMF byte] [FLG byte] [DEFLATE data] [Adler32 checksum (4 bytes)]
```

### Zlib Header: CMF Byte

The **Compression Method and Flags** byte (CMF) specifies the compression method and window size:

```
bits 0-3:  Compression method (8 = DEFLATE, only valid value)
bits 4-7:  Window size as power of 2 (wlog: 0=1, 1=2, ..., 15=32768)
```

```go
func WriteCMF(w io.Writer, windowSize int) error {
	cm := 8  // DEFLATE

	var wlog int
	switch windowSize {
	case 1:   wlog = 0
	case 2:   wlog = 1
	case 4:   wlog = 2
	case 8:   wlog = 3
	case 16:  wlog = 4
	case 32:  wlog = 5
	case 64:  wlog = 6
	case 128: wlog = 7
	case 256: wlog = 8
	case 512:  wlog = 9
	case 1024: wlog = 10
	case 2048: wlog = 11
	case 4096: wlog = 12
	case 8192: wlog = 13
	case 16384: wlog = 14
	case 32768: wlog = 15
	default: return ErrInvalidWindowSize
	}

	cmf := byte((cm & 0xF) | ((wlog & 0xF) << 4))
	return binary.Write(w, binary.BigEndian, cmf)
}
```

**Common values**:

- `0x78`: Window=32, DEFLATE (most common)
- `0x1B`: Window=8, DEFLATE

### Zlib Header: FLG Byte

The **Flags** byte (FLG) contains error detection and compression settings:

```
bits 0-4:  Check bits for CMF+FLG validation
bit 5:     Dict flag (0 = no preset dictionary)
bits 6-7:  Compression level (0=none, 1=fastest, 2=fast, 3=best)
```

```go
func WriteFLG(w io.Writer, checksum uint8) error {
	dictFlag := 0   // No preset dictionary
	level := 2      // Default compression level

	flg := byte((checksum & 0x1F) | ((dictFlag & 1) << 5) | ((level & 3) << 6))
	return binary.Write(w, binary.BigEndian, flg)
}
```

**Check bits**: Bits 0-4 are computed so that `(CMF * 256 + FLG) % 31 == 0`. This allows decoders to detect CMF/FLG corruption.

### Zlib Footer: Adler32

The footer contains the 4-byte Adler32 checksum of the original (uncompressed) data:

```go
func WriteAdler32Footer(w io.Writer, checksum uint32) error {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], checksum)
	_, err := w.Write(buf[:])
	return err
}
```

**Big-endian**: The checksum is stored most-significant byte first.

### Complete Zlib Stream

A complete zlib stream looks like:

```text
Byte 0:   0x78      (CMF: DEFLATE, window=32)
Byte 1:   0x9C      (FLG: check bits=0, no dict, level=2)
Bytes 2+: [DEFLATE compressed data]
Last 4:   [Adler32 checksum of original data]
```

**Common zlib header**: `78 9C` is the most common zlib header (window=32, level=2, valid check bits).

## Summary

1. **IEND Chunk**: 12-byte end marker (length=0, type="IEND", CRC)

   - Required by PNG spec to mark end of file
   - No data, just type and CRC

2. **Adler32 Checksum**: Used by zlib for data integrity

   - Two 16-bit sums (s1, s2) modulo 65521
   - Faster than CRC32, better for streaming
   - Final value: `(s2 << 16) | s1`

3. **Zlib Format**: Wrapper around DEFLATE compression
   - CMF byte: compression method (8=DEFLATE) + window size
   - FLG byte: check bits + dict flag + compression level
   - Footer: 4-byte Adler32 of original data

## Implementation Summary

This document covers three essential components for PNG and zlib:

| Component   | File                          | Purpose                                           |
| ----------- | ----------------------------- | ------------------------------------------------- |
| IEND Chunk  | `src/png/iend_writer.go`      | Writes 12-byte end marker (length=0, "IEND", CRC) |
| Adler32     | `src/compress/adler32.go`     | Computes checksum per RFC 1950                    |
| Zlib Header | `src/compress/zlib_header.go` | Writes CMF (method+window) and FLG (flags)        |
| Zlib Footer | `src/compress/zlib_footer.go` | Writes 4-byte Adler32 checksum                    |

### Test Coverage

- **IEND**: Verifies 12-byte format, length=0, type="IEND", CRC=0xAE426082
- **Adler32**: Tests empty data, single byte, "ABC" vector (0x018D00C7), streaming
- **Zlib Header**: Tests CMF for all valid window sizes (1-32768), invalid sizes rejected
- **Zlib Footer**: Tests big-endian output for various checksum values

### Key Formulas

**Adler32**:

```
s1 = (s1 + byte) % 65521
s2 = (s2 + s1) % 65521
checksum = (s2 << 16) | s1
```

**CMF byte**:

```
CMF = compression_method | (window_log << 4)
     = 8 | (log2(window_size) << 4)
```

**FLG byte**:

```
FLG = check_bits | (dict_flag << 5) | (level << 6)
```

## Next Steps

- Learn about [PNG Chunks](../png-encoding.md#file-structure) (IHDR, IDAT, IEND)
- Understand [PNG Filters](../png-encoding.md#the-five-png-filters) (Sub, Up, Average, Paeth)
- Explore [PNG Encoding Pipeline](../png-encoding.md#the-png-pipeline)
