# Code Reading Guide: PNG Serialization

This guide explains how to read the serialization code, with direct links to the implementation.

---

## Reading Order (Bottom-Up Approach)

Read in this order to understand the serialization flow:

```
1. CRC32 (foundation)
   ↓
2. Chunk structure (data container)
   ↓
3. Chunk serialization (how chunks become bytes)
   ↓
4. IHDR data structure (specific chunk data)
   ↓
5. IHDR chunk writer (puts it all together)
```

---

## Step 1: Understand CRC32 (Foundation)

**File:** [`src/compress/crc32.go`](src/compress/crc32.go)

**What to read:**
- Lines 8-10: `CRC32()` function - computes checksum
- Lines 12-14: `NewCRC32()` function - streaming interface

**Key understanding:**
- Takes bytes → returns 32-bit checksum
- Used to verify data integrity
- Used later when writing chunks

**Why read this first:** Everything else depends on CRC32 for chunk validation.

---

## Step 2: Understand Chunk Structure (Data Container)

**File:** [`src/png/chunk.go`](src/png/chunk.go)

**What to read:**
- Lines 10-13: `Chunk` struct definition

**Key understanding:**
- A chunk has a type (4 bytes) and data (variable length)
- This is the container before serialization

**Why read this:** This is what we're serializing.

---

## Step 3: Understand Chunk Serialization (Core Logic)

**File:** [`src/png/chunk.go`](src/png/chunk.go)

**What to read (in order):**

1. **Lines 23-27:** `CRC()` method
   - Computes CRC32 over Type + Data
   - Uses `compress.CRC32()` from Step 1

2. **Lines 29-41:** `Bytes()` method ⭐ **MOST IMPORTANT**
   - Creates byte array: `[Length][Type][Data][CRC]`
   - Length: 4 bytes (big-endian)
   - Type: 4 bytes (e.g., "IHDR")
   - Data: N bytes (variable)
   - CRC: 4 bytes (big-endian)
   - Total: 4 + 4 + N + 4 bytes

3. **Lines 43-47:** `WriteTo()` method
   - Writes chunk bytes to `io.Writer`
   - Uses `Bytes()` internally

**Visual representation:**
```
Bytes() output:
┌─────────┬─────────┬──────────┬─────────┐
│ Length  │  Type   │   Data   │   CRC   │
│ 4 bytes │ 4 bytes │ N bytes  │ 4 bytes │
└─────────┴─────────┴──────────┴─────────┘
```

**Why read this:** This is the core serialization logic - everything else builds on this.

---

## Step 4: Understand IHDR Data Structure

**File:** [`src/png/ihdr.go`](src/png/ihdr.go)

**What to read:**

1. **Lines 9-17:** `IHDRData` struct
   - Contains 7 fields: Width, Height, BitDepth, ColorType, Compression, Filter, Interlace

2. **Lines 37-47:** `Bytes()` method ⭐
   - Returns exactly 13 bytes
   - Width/Height: 4 bytes each (little-endian!)
   - Other fields: 1 byte each
   - **Note:** Width/Height use little-endian (unlike chunk length/CRC which use big-endian)

**Visual representation:**
```
IHDRData.Bytes() output (13 bytes):
┌──────┬──────┬─────┬─────┬─────┬─────┬─────┐
│Width │Height│BitD │ColT │Comp │Filt │Intl │
│4 bytes│4 bytes│1│1│1│1│1│
└──────┴──────┴─────┴─────┴─────┴─────┴─────┘
```

**Why read this:** This is the data that goes inside an IHDR chunk.

---

## Step 5: Understand IHDR Chunk Writer (Putting It All Together)

**File:** [`src/png/ihdr.go`](src/png/ihdr.go)

**What to read:**
- **Lines 96-104:** `WriteIHDR()` function ⭐ **ENTRY POINT**

**Key understanding:**
1. Creates a `Chunk` with type "IHDR" and IHDRData bytes as data
2. Uses `Chunk.WriteTo()` to serialize (from Step 3)
3. This is the entry point for writing an IHDR chunk

**Complete flow:**
```
WriteIHDR(writer, ihdrData)
    ↓
1. ihdrData.Bytes() → Returns 13 bytes
    ↓
2. Create Chunk{Type: "IHDR", Data: [13 bytes]}
    ↓
3. chunk.WriteTo(writer)
    ↓
4. chunk.Bytes() → Creates: [Length=13][Type="IHDR"][Data=[13 bytes]][CRC=[4 bytes]]
    ↓
5. Write to io.Writer → Final output: 4 + 4 + 13 + 4 = 25 bytes
```

**Why read this:** This shows how everything connects together.

---

## Complete Serialization Flow

When you call `WriteIHDR()`, here's what happens:

```
WriteIHDR(writer, ihdrData)
    ↓
[src/png/ihdr.go:37-47] ihdrData.Bytes() 
   → Returns 13 bytes: [Width][Height][BitDepth][ColorType][Compression][Filter][Interlace]
    ↓
[src/png/ihdr.go:97-100] Create Chunk{Type: "IHDR", Data: [13 bytes]}
    ↓
[src/png/chunk.go:43-47] chunk.WriteTo(writer)
    ↓
[src/png/chunk.go:29-41] chunk.Bytes()
   → [src/png/chunk.go:23-27] Computes CRC: CRC32("IHDR" + [13 bytes])
   → Creates: [Length=13][Type="IHDR"][Data=[13 bytes]][CRC=[4 bytes]]
    ↓
[src/png/chunk.go:44-45] Write to io.Writer
   → Final output: 4 + 4 + 13 + 4 = 25 bytes
```

---

## Quick Reference: Key Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| [`src/compress/crc32.go`](src/compress/crc32.go) | CRC32 checksum | `CRC32()`, `NewCRC32()` |
| [`src/png/chunk.go`](src/png/chunk.go) | Chunk serialization | `CRC()`, `Bytes()`, `WriteTo()` |
| [`src/png/ihdr.go`](src/png/ihdr.go) | IHDR data & writer | `Bytes()`, `WriteIHDR()` |

---

## Reading Tips

1. **Start with `WriteIHDR()`** - It's the entry point, read [`src/png/ihdr.go:96-104`](src/png/ihdr.go#L96-L104)
2. **Trace backwards** - See what it calls
3. **Read `Chunk.Bytes()`** - Core serialization logic, read [`src/png/chunk.go:29-41`](src/png/chunk.go#L29-L41)
4. **Understand byte layout** - Draw it out as you read
5. **Check endianness** - Big-endian for chunk length/CRC, little-endian for IHDR width/height

---

## Visual Summary

```
┌─────────────────────────────────────────────────┐
│ WriteIHDR(writer, ihdrData)                     │
│ [src/png/ihdr.go:96-104]                        │
└─────────────────────────────────────────────────┘
                    ↓
    ┌───────────────────────────┐
    │ ihdrData.Bytes()          │
    │ [src/png/ihdr.go:37-47]   │
    │ → 13 bytes                │
    └───────────────────────────┘
                    ↓
    ┌───────────────────────────┐
    │ Create Chunk              │
    │ [src/png/ihdr.go:97-100]  │
    │ Type: "IHDR"              │
    │ Data: [13 bytes]           │
    └───────────────────────────┘
                    ↓
    ┌───────────────────────────┐
    │ chunk.Bytes()             │
    │ [src/png/chunk.go:29-41]  │
    │ → [4][4][13][4] = 25 bytes│
    └───────────────────────────┘
                    ↓
    ┌───────────────────────────┐
    │ Write to io.Writer         │
    │ [src/png/chunk.go:43-47]  │
    └───────────────────────────┘
```

---

## Testing the Code

After reading, verify your understanding:

```bash
# Run tests to see it in action
go test ./src/png/... -v -run TestWriteIHDR

# Check test implementation
# File: src/png/ihdr_test.go (lines 179-210)
```

---

## Related Documentation

- **PNG Specification:** See [`docs/learning/png/png-infra.md`](docs/learning/png/png-infra.md) for detailed explanation of chunks and CRC32
- **Task Plan:** See [`docs/task.md`](docs/task.md) for implementation tasks
