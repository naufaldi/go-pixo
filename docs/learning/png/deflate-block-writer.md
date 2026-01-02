# DEFLATE Block Writer

This document explains how DEFLATE blocks are written, covering stored, fixed, and dynamic block types.

## What is a DEFLATE Block?

A DEFLATE block is a unit of compressed data in the DEFLATE stream. Each block contains:
- A 3-bit header indicating block type and whether it's the final block
- Block-specific data (uncompressed bytes, or Huffman-encoded tokens)
- An end-of-block marker (for fixed/dynamic blocks)

## Block Types

### Stored Blocks (Type 0)

Stored blocks contain uncompressed data. They're used when:
- Data is already compressed or incompressible
- The overhead of compression would make the result larger

**Format:**
```
[Header: 1 byte] [LEN: 2 bytes] [NLEN: 2 bytes] [Data: LEN bytes]
```

- Header: `BFINAL` bit (bit 0) + block type `00` (bits 1-2)
- LEN: Little-endian length of data (max 65535)
- NLEN: One's complement of LEN (for error detection)
- Data: Raw uncompressed bytes

**Why stored blocks?** Sometimes compression doesn't help. For example, already-compressed data or random bytes won't compress further. Stored blocks avoid wasting CPU cycles trying to compress incompressible data.

### Fixed Huffman Blocks (Type 1)

Fixed blocks use predefined Huffman tables from RFC 1951. No table transmission is needed—decoders know the tables.

**Format:**
```
[Header: 3 bits] [Encoded tokens] [End-of-block: 256]
```

- Header: `BFINAL` bit + block type `01`
- Encoded tokens: Literals (0-255), lengths (257-285), distances (0-29) encoded using fixed tables
- End-of-block: Symbol 256 marks the end

**Why fixed blocks?** They're fast to encode (no table building) and decode (no table parsing). Good for small data where the overhead of dynamic tables isn't worth it.

### Dynamic Huffman Blocks (Type 2)

Dynamic blocks build custom Huffman tables from the actual data frequencies. The tables are transmitted before the data.

**Format:**
```
[Header: 3 bits] [HLIT: 5 bits] [HDIST: 5 bits] [HCLEN: 4 bits]
[Code length code lengths: HCLEN+4 * 3 bits]
[RLE-encoded literal/length code lengths]
[RLE-encoded distance code lengths]
[Encoded tokens] [End-of-block: 256]
```

- HLIT: Number of literal/length codes - 257 (range 257-286)
- HDIST: Number of distance codes - 1 (range 1-30)
- HCLEN: Number of code length codes - 4 (range 4-19)
- Code length code lengths: Huffman table for encoding code lengths
- RLE-encoded lengths: The actual code lengths for literal/length and distance symbols

**Why dynamic blocks?** They adapt to the data, often achieving better compression than fixed tables, especially for larger blocks with skewed symbol distributions.

## LSB-First Bit Ordering

DEFLATE packs bits **LSB-first** (least significant bit first) within bytes. This is different from many formats that use MSB-first.

When writing bits sequentially, the first bit goes into bit 0 (rightmost):

```
Bits written: 1, 0, 1, 1, 0
Resulting byte: 0b_???01011
                    ▲▲▲▲▲
                    │││││
                    5th bit ─┘│││
                    4th bit ──┘││
                    3rd bit ───┘│
                    2nd bit ────┘
                    1st bit ─────┘
```

Our `BitWriter` (`src/compress/bit_writer.go`) handles this automatically by accumulating bits in a buffer and writing full bytes when 8 bits are collected.

## Implementation

Block writers are in `src/compress/deflate_block.go`:
- `WriteStoredBlockDeflate`: Writes stored blocks
- `WriteFixedBlock`: Writes fixed Huffman blocks
- `WriteDynamicBlock`: Writes dynamic Huffman blocks

All block writers take a `final` parameter indicating whether this is the last block in the stream. The final block has `BFINAL=1` in its header.
