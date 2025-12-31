# PNG Infrastructure: CRC32 and Chunks

This document explains the fundamental building blocks of PNG files: **chunks** and **CRC32**. Understanding these concepts is essential before implementing the PNG encoder.

---

## 1. PNG File Structure

A PNG file is organized as a sequence of **chunks**, like a sandwich with layers:

```
┌─────────────────────────────────────────────────────────────┐
│                     PNG File                                 │
├─────────────────────────────────────────────────────────────┤
│  8 bytes  │  Chunk  │  Chunk  │  ...  │  Chunk  │           │
│ Signature │  (IHDR) │  (IDAT) │       │  (IEND) │           │
└─────────────────────────────────────────────────────────────┘
```

### The Order Matters

A valid PNG file must contain chunks in this specific order:

| Position | Chunk | Purpose | Required? |
|----------|-------|---------|-----------|
| 1 | IHDR | Image header (dimensions, color type, etc.) | Yes - must be first |
| 2+ | IDAT | Compressed image data | Yes - can be multiple |
| Last | IEND | End marker | Yes - must be last |
| ... | PLTE | Color palette | No - only for indexed color |
| ... | tEXt, zTXt | Text metadata | No |
| ... | gAMA | Gamma information | No |

---

## 2. What is a PNG Chunk?

A chunk is a **self-contained data packet** with a specific structure:

```
┌─────────────────────────────────────────────────────────────┐
│  4 bytes   │  4 bytes   │  N bytes   │  4 bytes   │        │
│  Length    │   Type     │    Data    │   CRC32    │        │
└─────────────────────────────────────────────────────────────┘
```

### Chunk Fields Explained

| Field | Size | Description |
|-------|------|-------------|
| Length | 4 bytes | Size of Data field in bytes (big-endian). Does NOT include Type or CRC. |
| Type | 4 bytes | Four-character identifier (e.g., "IHDR", "IDAT", "IEND"). Case-sensitive. |
| Data | N bytes | Chunk-specific content. Size = Length field. |
| CRC32 | 4 bytes | Checksum computed over Type + Data fields (NOT including Length). |

### Chunk Type Naming Convention

Chunk types use a specific naming convention:

- **Capital letters** = critical chunk (must be supported, file invalid if unknown)
- **Lowercase letters** = ancillary chunk (optional, can be ignored)
- **IHDR** - all caps = critical
- **iTXt** - mixed case = ancillary
- **IEND** - all caps = critical

---

## 3. Why CRC32?

CRC32 stands for **Cyclic Redundancy Check, 32-bit**. It is a checksum algorithm used to detect data corruption.

### The Problem: Data Corruption

When files are transferred over networks, copied between devices, or stored on disks, bits can accidentally flip due to:

- Electrical interference
- Disk sector errors
- Network transmission errors
- Memory corruption

Without detection, you might load a corrupted image and see garbage or nothing at all.

### How CRC32 Works

CRC32 generates a **fingerprint** for data. The same input always produces the same 32-bit output.

```
Data: "IHDR" + [13 bytes of image header]
                    ↓
           CRC32 Algorithm
                    ↓
         Result: 0x1A2B3C4D  (32-bit fingerprint)
                    ↓
           Store with chunk
```

### Verification Process

When reading a PNG chunk:

```
1. Read: "IHDR" + [13 bytes] + CRC=0x1A2B3C4D
2. Calculate CRC32 on "IHDR" + [13 bytes]
3. Compare calculated CRC with stored CRC
4. Match?   → Data is intact, proceed
5. No match → Data corrupted, reject or warn
```

### Why Not Simpler Checksums?

| Method | Size | Collision Resistance | Use Case |
|--------|------|---------------------|----------|
| XOR | 1 byte | Very low | Simple parity |
| Sum | 4 bytes | Low | Basic error detection |
| CRC32 | 4 bytes | High | File integrity, networking, compression |
| MD5 | 16 bytes | Very high | Cryptographic verification |
| SHA-256 | 32 bytes | Extremely high | Security-critical |

CRC32 provides excellent collision resistance for its small size (4 bytes), making it ideal for PNG chunks and DEFLATE compression.

### CRC32 in PNG Specification

The PNG spec requires CRC32 for **every chunk**:

- IHDR chunk: CRC over "IHDR" + 13 bytes
- IDAT chunk: CRC over "IDAT" + compressed data
- IEND chunk: CRC over "IEND" (no data)
- Any other chunk: CRC over chunk type + data

If any chunk has an invalid CRC, the PNG file is considered corrupt.

---

## 4. Common PNG Chunks

### IHDR - Image Header (Critical)

The IHDR chunk must appear **first** and contains image dimensions and format:

```
Width:              4 bytes (little-endian)
Height:             4 bytes (little-endian)
Bit Depth:          1 byte  (1, 2, 4, 8, or 16)
Color Type:         1 byte  (0=grayscale, 2=RGB, 3=indexed, 4=grayscale+alpha, 6=RGBA)
Compression:        1 byte  (0=DEFLATE)
Filter:             1 byte  (0=none)
Interlace:          1 byte  (0=no interlace, 1=Adam7)

Total: 13 bytes
```

### IDAT - Image Data (Critical)

Contains the actual compressed image data. The image is stored as **scanlines**, where each scanline is one row of pixels:

```
Scanline = [1 byte filter type] + [pixel data]

For RGB image (8-bit):
Scanline = 0x00 + R,G,B,R,G,B,... (width times)

All scanlines concatenated, then DEFLATE compressed.
```

### IEND - End of PNG (Critical)

Marks the end of the PNG file. Has **zero data**:

```
Length:  0x00000000
Type:    "IEND"
Data:    (none)
CRC32:   CRC of "IEND" = 0xAE426082
```

---

## 5. PNG Encoder Workflow

Understanding chunks and CRC32, the PNG encoder follows this sequence:

```mermaid
flowchart TD
    A[Start] --> B[Write 8-byte PNG Signature<br/>89 50 4E 47 0D 0A 1A 0A]
    B --> C[Write IHDR Chunk<br/>13 bytes + CRC32]
    C --> D[Write IDAT Chunk(s)<br/>Compressed data + CRC32]
    D --> E[Write IEND Chunk<br/>0 bytes + CRC32]
    E --> F[Complete PNG File]
    
    style B fill:#c8e6c9
    style C fill:#ffe0b2
    style D fill:#ffccbc
    style E fill:#cfd8dc
```

### Step-by-Step

1. **Signature**: Write the 8-byte PNG magic number to identify the file
2. **IHDR**: Create 13-byte header, wrap in chunk with CRC32
3. **IDAT**: Convert image to scanlines, apply filters, compress with DEFLATE, wrap in chunk(s) with CRC32
4. **IEND**: Write empty chunk with CRC32 of "IEND"

---

## 6. Analogy: Shipping Package

Think of a PNG chunk like a **tamper-evident shipping package**:

| Concept | PNG Chunk | Shipping Package |
|---------|-----------|------------------|
| Contents | Data field | Package contents |
| Label | Type field | Shipping label |
| Weight | Length field | Package weight |
| Seal | CRC32 | Tamper-evident seal |

**If the seal is broken when it arrives, you know someone tampered with it.**

Similarly, if the CRC32 doesn't match when reading a PNG, the data is corrupted and should not be used.

---

## 7. Implementation Notes

### CRC32 in Go

Go provides CRC32 in the standard library:

```go
import "hash/crc32"

data := []byte("IHDR" + ihdrBytes)
checksum := crc32.ChecksumIEEE(data)
```

The `hash/crc32` package also provides an `io.Writer` interface for streaming:

```go
w := crc32.NewIEEE()
w.Write([]byte("IHDR"))
w.Write(ihdrBytes)
checksum := w.Sum32()
```

### Chunk Writing Order

Must write fields in this exact order:

1. Length (4 bytes, big-endian)
2. Type (4 bytes, e.g., "IHDR")
3. Data (N bytes)
4. CRC32 of Type + Data (4 bytes, big-endian)

```go
func (c *Chunk) WriteTo(w io.Writer) (int64, error) {
    var total int64
    
    // 1. Length (big-endian)
    lengthBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(lengthBuf, uint32(len(c.Data)))
    if n, err := w.Write(lengthBuf); err != nil {
        return total, err
    }
    total += int64(n)
    
    // 2. Type
    if n, err := w.Write([]byte(c.Type)); err != nil {
        return total, err
    }
    total += int64(n)
    
    // 3. Data
    if n, err := w.Write(c.Data); err != nil {
        return total, err
    }
    total += int64(n)
    
    // 4. CRC32 of Type + Data
    crc := crc32.ChecksumIEEE(append([]byte(c.Type), c.Data...))
    crcBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(crcBuf, crc)
    if n, err := w.Write(crcBuf); err != nil {
        return total, err
    }
    total += int64(n)
    
    return total, nil
}
```

---

## 8. Summary

| Concept | Key Point |
|---------|-----------|
| **Chunk** | Self-contained data packet with type, data, and CRC32 |
| **Chunk Structure** | Length (4) + Type (4) + Data (N) + CRC32 (4) |
| **CRC32** | Detects corruption by generating a fingerprint of Type + Data |
| **IHDR** | First chunk, contains image dimensions and format |
| **IDAT** | Contains compressed image data (scanlines + filters) |
| **IEND** | Last chunk, marks end of PNG file |
| **Required Order** | Signature → IHDR → IDAT(s) → IEND |

Understanding chunks and CRC32 is the foundation for implementing a valid PNG encoder. Without proper chunk handling and CRC32 calculation, the output will not be recognized as a valid PNG file.
