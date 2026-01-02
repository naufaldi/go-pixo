# IDAT Zlib Integration

This document explains how PNG IDAT chunks use zlib-wrapped DEFLATE compression, including the zlib header, DEFLATE payload, and Adler32 checksum.

## Zlib Format

Zlib wraps DEFLATE compression with:
1. **2-byte header** (CMF + FLG): Compression method and window size
2. **DEFLATE stream**: One or more DEFLATE blocks
3. **4-byte footer**: Adler32 checksum of uncompressed data

```
[CMF: 1 byte] [FLG: 1 byte] [DEFLATE blocks...] [Adler32: 4 bytes]
```

## Zlib Header (CMF + FLG)

The header encodes compression method and window size:

**CMF (Compression Method and Flags):**
- Bits 0-3: Compression method (8 = DEFLATE)
- Bits 4-7: Window size logarithm (CINFO)

**FLG (FLaGs):**
- Bits 0-3: FCHECK (check bits for header validation)
- Bit 4: FDICT (preset dictionary flag, usually 0)
- Bits 5-6: FLEVEL (compression level hint, 0-3)
- Bit 7: Reserved

For PNG, we use:
- CMF: `0x78` (DEFLATE, 32K window)
- FLG: `0x9C` (level 2, no dictionary, valid check bits)

**Why these values?** PNG spec recommends DEFLATE with 32K window. The check bits ensure header integrity.

## DEFLATE Payload

The DEFLATE stream contains one or more blocks compressing the scanline data:

```
[Block 1] [Block 2] ... [Final Block]
```

Each scanline is prefixed with a filter byte (0-4), then the pixel data. The entire scanline sequence is compressed as a single DEFLATE stream.

**Why compress all scanlines together?** Better compression—the encoder can find matches across scanline boundaries, not just within a single scanline.

## Adler32 Checksum

The footer contains an Adler32 checksum computed over the **uncompressed** scanline data (including filter bytes).

Adler32 is a simple checksum algorithm:
- Faster than CRC32
- Good error detection for typical data
- Used by zlib (not CRC32 like PNG chunks)

**What gets checksummed?** The raw scanline bytes before compression:
```
[Filter byte 0] [Row 0 pixels]
[Filter byte 1] [Row 1 pixels]
...
[Filter byte N] [Row N pixels]
```

**Why checksum uncompressed data?** Detects corruption in the original image data, not just in the compressed stream. If decompression succeeds but Adler32 fails, the data was corrupted before compression.

## Implementation

The IDAT writer (`src/png/idat_writer.go`) builds the zlib stream:

1. **Build scanlines**: Prepend filter byte 0 (None) to each row
2. **Compress**: Use `DeflateEncoder.EncodeAuto()` to compress all scanlines
3. **Wrap**: Prepend zlib header, append Adler32 footer
4. **Chunk**: Wrap in PNG chunk format (length + "IDAT" + data + CRC32)

```go
// Build scanlines with filter bytes
scanlineData := buildScanlines(pixels, width, height, colorType)

// Compress with DEFLATE
encoder := compress.NewDeflateEncoder()
deflateData, err := encoder.EncodeAuto(scanlineData)

// Build zlib stream
zlibHeader := compress.ZlibHeaderBytes(32768, 2)
adler32 := compress.Adler32(scanlineData)
zlibFooter := compress.ZlibFooterBytes(adler32)

zlibData := append(zlibHeader, deflateData...)
zlibData = append(zlibData, zlibFooter...)
```

## Verification

To verify a zlib stream:
1. Check header: CMF=0x78, FLG=0x9C
2. Decompress DEFLATE blocks using standard DEFLATE decoder
3. Verify Adler32 matches computed checksum of decompressed data

Our tests (`src/png/idat_writer_test.go`) do this using Go's `compress/zlib` package.

## Why Zlib Instead of Raw DEFLATE?

PNG uses zlib (not raw DEFLATE) because:
- **Checksum**: Adler32 detects data corruption
- **Standardization**: zlib is a well-known format
- **Window size**: Header specifies decompression window size
- **Compatibility**: Most PNG decoders expect zlib format

The zlib wrapper adds minimal overhead (6 bytes: 2 header + 4 footer) but provides important metadata and error detection.
