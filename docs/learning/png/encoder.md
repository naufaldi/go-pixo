# PNG Encoder

The PNG encoder converts raw pixel data into a valid PNG file by orchestrating chunk writing, zlib compression, and DEFLATE encoding.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      PNG Encoder                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   IHDR      │ -> │   IDAT      │ -> │   IEND      │     │
│  │   Chunk     │    │   Chunks    │    │   Chunk     │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
│        │                  │                  │              │
│        v                  v                  v              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │Dimensions   │    │Zlib Header  │    │  Empty      │     │
│  │Bit Depth    │    │Stored       │    │  Chunk      │     │
│  │Color Type   │    │Blocks       │    │             │     │
│  └─────────────┘    │Adler32      │    └─────────────┘     │
│                     └─────────────┘                         │
└─────────────────────────────────────────────────────────────┘
```

## Encoder API

### NewEncoder

```go
func NewEncoder(width, height int, colorType ColorType) (*Encoder, error)
```

Creates a new encoder with specified dimensions and color type. Validates that dimensions are positive.

**Parameters:**
- `width`: Image width in pixels (must be > 0)
- `height`: Image height in pixels (must be > 0)
- `colorType`: One of `ColorGrayscale`, `ColorRGB`, `ColorRGBA`

**Returns:**
- Encoder instance
- Error if dimensions are invalid or color type is unsupported

### Encode

```go
func (e *Encoder) Encode(pixels []byte) ([]byte, error)
```

Encodes raw pixel data into a complete PNG file.

**Parameters:**
- `pixels`: Raw pixel data (row-major order, no filter bytes)

**Validation:**
- Pixel count must exactly match `width * height * bytesPerPixel`
- For RGB: 3 bytes per pixel
- For RGBA: 4 bytes per pixel
- For Grayscale: 1 byte per pixel

**Returns:**
- Complete PNG file as bytes
- Error if pixel count doesn't match

## Encoding Process

### 1. PNG Signature

8-byte magic number that identifies PNG files:

```
89 50 4E 47 0D 0A 1A 0A
```

### 2. IHDR Chunk

Image Header chunk containing:

| Field       | Size  | Description                              |
|-------------|-------|------------------------------------------|
| Width       | 4 B   | Image width (big-endian)                 |
| Height      | 4 B   | Image height (big-endian)                |
| Bit Depth   | 1 B   | Bits per sample (always 8 for Phase 1)   |
| Color Type  | 1 B   | 2=RGB, 6=RGBA                            |
| Compression | 1 B   | Compression method (0 = DEFLATE)         |
| Filter      | 1 B   | Filter method (0)                        |
| Interlace   | 1 B   | Interlace method (0 = no interlace)      |

### 3. IDAT Chunks

Image Data chunk containing compressed pixel data:

```
┌─────────────────────────────────────────┐
│              zlib Wrapper                │
├─────────────────────────────────────────┤
│  CMF (Compression Method/Flags)   2 B   │
│  FLG (Flags)                      1 B   │
│  ┌───────────────────────────────┐      │
│  │     DEFLATE Data              │      │
│  │  ┌─────────────────────────┐  │      │
│  │  │  Stored Block Header    │  │      │
│  │  │  LEN / NLEN             │  │      │
│  │  │  Filter + Pixel Data    │  │      │
│  │  └─────────────────────────┘  │      │
│  │         ... (one per row)     │      │
│  └───────────────────────────────┘      │
│  Adler32 Checksum                 4 B   │
└─────────────────────────────────────────┘
```

**Key Points:**
- Each scanline gets its own stored block
- Filter byte 0 (None) prepended to each scanline
- Zlib header uses CMF=0x78 (DEFLATE, 32K window)
- Zlib footer contains Adler32 checksum of uncompressed data

### 4. IEND Chunk

Image End chunk - marks end of PNG file. Always:

```
Length: 0
Type:   IEND
CRC:    AE 42 60 82
```

## Usage Example

```go
// Create encoder for 100x100 RGB image
enc, err := png.NewEncoder(100, 100, png.ColorRGB)
if err != nil {
    log.Fatal(err)
}

// Prepare pixel data (100 * 100 * 3 = 30000 bytes)
pixels := make([]byte, 100*100*3)
// ... fill pixels ...

// Encode to PNG
pngData, err := enc.Encode(pixels)
if err != nil {
    log.Fatal(err)
}

// Write to file
os.WriteFile("output.png", pngData, 0644)
```

## Color Types

| Constant     | Value | Description           | Bytes/Pixel |
|--------------|-------|-----------------------|-------------|
| ColorGrayscale | 0     | Grayscale             | 1           |
| ColorRGB       | 2     | RGB                   | 3           |
| ColorRGBA      | 6     | RGBA                  | 4           |

## Error Handling

The encoder returns descriptive errors for:

- `ErrInvalidDimensions`: Width or height is <= 0
- Pixel count mismatch: `png: pixel count mismatch: got X bytes, want Y`

## Limitations (Phase 1)

- Only 8-bit depth supported
- Only RGB and RGBA color types supported
- Only filter type 0 (None) used
- No interlacing support
- Stored blocks only (no Huffman or dynamic blocks)
