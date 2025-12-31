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

## Summary

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

## Next Steps

- Learn about [PNG Chunks](../png-encoding.md#file-structure) (IHDR, IDAT, IEND)
- Understand [PNG Filters](../png-encoding.md#the-five-png-filters) (Sub, Up, Average, Paeth)
- Explore [PNG Encoding Pipeline](../png-encoding.md#the-png-pipeline)
