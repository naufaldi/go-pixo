---
name: Performance Optimization Plan
overview: Comprehensive plan to optimize JPEG encoding (~2x improvement) and PNG encoding (~10-20x improvement) using algorithmic improvements, integer arithmetic, and optional SIMD intrinsics - all without external dependencies.
todos:
  - id: jpeg-dct
    content: Implement AAN fast DCT algorithm in src/jpeg/dct.rs (5 muls + 29 adds vs 64 muls)
    status: completed
  - id: jpeg-color
    content: Convert rgb_to_ycbcr to fixed-point integer arithmetic in src/color.rs
    status: completed
  - id: jpeg-bitwriter
    content: Optimize BitWriterMsb.write_bits to process multiple bits at once
    status: completed
  - id: simd-module
    content: Create src/simd/ module structure with feature detection and dispatch
    status: completed
  - id: png-filters-simd
    content: Implement SIMD filter_sub, filter_up, filter_average, filter_paeth
    status: completed
  - id: png-score-simd
    content: Implement SIMD score_filter using SAD instruction
    status: completed
  - id: adler32-simd
    content: Implement SIMD-accelerated Adler-32 checksum
    status: completed
  - id: crc32-hw
    content: Implement hardware-accelerated CRC32 using SSE4.2 intrinsics
    status: completed
  - id: lz77-match-simd
    content: Implement SIMD match_length using 16-byte comparison
    status: completed
  - id: lz77-hash
    content: Implement CRC32-based hash function for better distribution
    status: cancelled
  - id: lz77-lazy-tuning
    content: Add GOOD_MATCH threshold to skip lazy matching for long matches
    status: completed
---

# Performance Optimization Plan for comprs

## Current State

Based on the benchmarks, `comprs` has two main performance gaps:

- **JPEG encoding**: ~2x slower than `image` crate
- **PNG encoding**: ~20-30x slower than `image` crate

## Priority 1: JPEG Optimizations (Est. 2-3 days)

### 1a. Fast DCT using AAN Algorithm

**Current**: Naive separable DCT with 64 multiplications per row in [`src/jpeg/dct.rs`](src/jpeg/dct.rs)

```98:109:src/jpeg/dct.rs
fn dct_1d(input: &[f32], output: &mut [f32]) {
    debug_assert_eq!(input.len(), 8);
    debug_assert_eq!(output.len(), 8);

    for k in 0..8 {
        let mut sum = 0.0f32;
        for n in 0..8 {
            sum += input[n] * COS_TABLE[n][k];
        }
        output[k] = 0.5 * ALPHA[k] * sum;
    }
}
```

**Optimization**: Implement AAN (Arai-Agui-Nakajima) fast DCT with only **5 multiplications and 29 additions** per 8-point transform:

```rust
// New function in src/jpeg/dct.rs
pub fn dct_2d_fast(block: &[f32; 64]) -> [f32; 64] {
    let mut temp = [0.0f32; 64];
    let mut result = [0.0f32; 64];

    // Row transforms using AAN
    for row in 0..8 {
        let r = row * 8;
        aan_dct_1d(&block[r..r+8], &mut temp[r..r+8]);
    }

    // Column transforms
    for col in 0..8 {
        // ... transpose, apply aan_dct_1d, transpose back
    }
    result
}

fn aan_dct_1d(input: &[f32], output: &mut [f32]) {
    // Butterfly stages - only 5 multiplications needed
    let tmp0 = input[0] + input[7];
    let tmp7 = input[0] - input[7];
    let tmp1 = input[1] + input[6];
    let tmp6 = input[1] - input[6];
    // ... continue with AAN butterfly pattern
}
```

**Expected speedup**: 2-3x for DCT stage

### 1b. Integer Color Conversion

**Current**: Float arithmetic with `round()` and `clamp()` in [`src/color.rs`](src/color.rs)

```50:65:src/color.rs
pub fn rgb_to_ycbcr(r: u8, g: u8, b: u8) -> (u8, u8, u8) {
    let r = r as f32;
    let g = g as f32;
    let b = b as f32;

    // ITU-R BT.601 conversion
    let y = 0.299 * r + 0.587 * g + 0.114 * b;
    let cb = -0.168736 * r - 0.331264 * g + 0.5 * b + 128.0;
    let cr = 0.5 * r - 0.418688 * g - 0.081312 * b + 128.0;

    (
        y.round().clamp(0.0, 255.0) as u8,
        cb.round().clamp(0.0, 255.0) as u8,
        cr.round().clamp(0.0, 255.0) as u8,
    )
}
```

**Optimization**: Fixed-point arithmetic using bit shifts:

```rust
#[inline]
pub fn rgb_to_ycbcr(r: u8, g: u8, b: u8) -> (u8, u8, u8) {
    let r = r as i32;
    let g = g as i32;
    let b = b as i32;

    // Fixed-point coefficients (scaled by 256)
    // Y  = 0.299*R + 0.587*G + 0.114*B  -> (77*R + 150*G + 29*B + 128) >> 8
    // Cb = -0.169*R - 0.331*G + 0.5*B + 128 -> ((-43*R - 85*G + 128*B) >> 8) + 128
    // Cr = 0.5*R - 0.419*G - 0.081*B + 128 -> ((128*R - 107*G - 21*B) >> 8) + 128

    let y = (77 * r + 150 * g + 29 * b + 128) >> 8;
    let cb = ((-43 * r - 85 * g + 128 * b + 128) >> 8) + 128;
    let cr = ((128 * r - 107 * g - 21 * b + 128) >> 8) + 128;

    (y.clamp(0, 255) as u8, cb.clamp(0, 255) as u8, cr.clamp(0, 255) as u8)
}
```

**Expected speedup**: 1.3-1.5x for color conversion

### 1c. Batch MSB Bit Writing

**Current**: Per-bit loop in [`src/bits.rs`](src/bits.rs)

```146:164:src/bits.rs
    pub fn write_bits(&mut self, value: u32, num_bits: u8) {
        debug_assert!(num_bits <= 32);

        for i in (0..num_bits).rev() {
            let bit = ((value >> i) & 1) as u8;
            self.bit_position -= 1;
            self.current_byte |= bit << self.bit_position;

            if self.bit_position == 0 {
                self.buffer.push(self.current_byte);
                // JPEG byte stuffing: if we wrote 0xFF, add 0x00
                if self.current_byte == 0xFF {
                    self.buffer.push(0x00);
                }
                self.current_byte = 0;
                self.bit_position = 8;
            }
        }
    }
```

**Optimization**: Write multiple bits at once using bit manipulation:

```rust
#[inline]
pub fn write_bits(&mut self, value: u32, num_bits: u8) {
    let mut remaining = num_bits;
    let mut val = value;

    while remaining > 0 {
        let space = self.bit_position;
        let to_write = remaining.min(space);

        // Extract top `to_write` bits and place in current byte
        let shift = remaining - to_write;
        let bits = ((val >> shift) as u8) & ((1 << to_write) - 1);
        self.bit_position -= to_write;
        self.current_byte |= bits << self.bit_position;

        remaining -= to_write;

        if self.bit_position == 0 {
            self.flush_byte_with_stuffing();
        }
    }
}
```

**Expected speedup**: 1.2x for Huffman encoding---

## Priority 2: PNG Filter Optimizations (Est. 2-3 days)

### 2a. Vectorized Filter Implementations

**Current**: Byte-by-byte iteration in [`src/png/filter.rs`](src/png/filter.rs)

```99:103:src/png/filter.rs
fn filter_sub(row: &[u8], bpp: usize, output: &mut Vec<u8>) {
    for (i, &byte) in row.iter().enumerate() {
        let left = if i >= bpp { row[i - bpp] } else { 0 };
        output.push(byte.wrapping_sub(left));
    }
}
```

**Optimization**: Process multiple bytes at once (SIMD when available, otherwise unrolled):

```rust
fn filter_sub(row: &[u8], bpp: usize, output: &mut Vec<u8>) {
    // First bpp bytes have no left neighbor
    output.extend_from_slice(&row[..bpp]);

    // Process remaining bytes - can be vectorized
    #[cfg(all(feature = "simd", target_arch = "x86_64"))]
    {
        filter_sub_simd(&row[bpp..], &row[..row.len()-bpp], output);
    }

    #[cfg(not(all(feature = "simd", target_arch = "x86_64")))]
    {
        for i in bpp..row.len() {
            output.push(row[i].wrapping_sub(row[i - bpp]));
        }
    }
}
```

**Files to modify**: [`src/png/filter.rs`](src/png/filter.rs)

### 2b. Vectorized Filter Scoring

**Current**: Per-byte scoring with iterator:

```346:348:src/png/filter.rs
fn score_filter(filtered: &[u8]) -> u64 {
    filtered.iter().map(|&b| (b as i8).unsigned_abs() as u64).sum()
}
```

**Optimization**: Use SIMD SAD (Sum of Absolute Differences) instruction or unrolled loop:

```rust
#[inline]
fn score_filter(filtered: &[u8]) -> u64 {
    #[cfg(all(feature = "simd", target_arch = "x86_64"))]
    unsafe {
        use std::arch::x86_64::*;
        // _mm_sad_epu8 computes sum of absolute differences - perfect for scoring
    }

    #[cfg(not(all(feature = "simd", target_arch = "x86_64")))]
    {
        // Unrolled scalar fallback
        filtered.chunks(8).map(|chunk| {
            chunk.iter().map(|&b| (b as i8).unsigned_abs() as u64).sum::<u64>()
        }).sum()
    }
}
```

---

## Priority 3: DEFLATE/LZ77 Optimizations (Est. 3-4 days)

### 3a. SIMD Match Length Comparison

**Current**: u64-based comparison in [`src/compress/lz77.rs`](src/compress/lz77.rs) (already optimized from byte-by-byte):

```189:228:src/compress/lz77.rs
    fn match_length(&self, data: &[u8], pos1: usize, pos2: usize) -> usize {
        let max_len = (data.len() - pos2).min(MAX_MATCH_LENGTH);
        let mut length = 0;

        // Compare 8 bytes at a time using u64
        while length + 8 <= max_len {
            let a = u64::from_ne_bytes(data[pos1 + length..pos1 + length + 8].try_into().unwrap());
            let b = u64::from_ne_bytes(data[pos2 + length..pos2 + length + 8].try_into().unwrap());
            if a != b {
                let xor = a ^ b;
                #[cfg(target_endian = "little")]
                { length += (xor.trailing_zeros() / 8) as usize; }
                #[cfg(target_endian = "big")]
                { length += (xor.leading_zeros() / 8) as usize; }
                return length;
            }
            length += 8;
        }
        // ... remaining bytes
    }
```

**Optimization**: Use SIMD to compare 16-32 bytes at once:

```rust
#[cfg(all(feature = "simd", target_arch = "x86_64"))]
#[target_feature(enable = "sse2")]
unsafe fn match_length_simd(data: &[u8], pos1: usize, pos2: usize, max_len: usize) -> usize {
    use std::arch::x86_64::*;
    let mut length = 0;

    while length + 16 <= max_len {
        let a = _mm_loadu_si128(data[pos1 + length..].as_ptr() as *const _);
        let b = _mm_loadu_si128(data[pos2 + length..].as_ptr() as *const _);
        let cmp = _mm_cmpeq_epi8(a, b);
        let mask = _mm_movemask_epi8(cmp) as u32;

        if mask != 0xFFFF {
            return length + mask.trailing_ones() as usize;
        }
        length += 16;
    }
    // Handle remaining bytes...
    length
}
```

### 3b. Better Hash Function

**Current**: Simple multiplicative hash:

```34:43:src/compress/lz77.rs
fn hash3(data: &[u8], pos: usize) -> usize {
    if pos + 2 >= data.len() { return 0; }
    let h = (data[pos] as u32)
        | ((data[pos + 1] as u32) << 8)
        | ((data[pos + 2] as u32) << 16);
    ((h.wrapping_mul(2654435769)) >> 17) as usize & (HASH_SIZE - 1)
}
```

**Optimization**: Use CRC32 intrinsic for better distribution (when available):

```rust
#[cfg(all(feature = "simd", target_arch = "x86_64"))]
#[target_feature(enable = "sse4.2")]
unsafe fn hash3_crc(data: &[u8], pos: usize) -> usize {
    use std::arch::x86_64::_mm_crc32_u32;
    let val = u32::from_le_bytes([data[pos], data[pos+1], data[pos+2], 0]);
    (_mm_crc32_u32(0, val) as usize) & (HASH_SIZE - 1)
}
```

### 3c. Lazy Match Threshold Tuning

**Current**: Lazy match emits literal if next match is only 2 longer:

```105:117:src/compress/lz77.rs
                if self.lazy_matching && pos + 1 < data.len() {
                    self.update_hash(data, pos);
                    if let Some((next_length, _)) = self.find_best_match(data, pos + 1) {
                        if next_length > length + 1 {
                            tokens.push(Token::Literal(data[pos]));
                            pos += 1;
                            continue;
                        }
                    }
                }
```

**Optimization**: Skip lazy matching for "good enough" matches (like zlib):

```rust
const GOOD_MATCH_LENGTH: usize = 32;
const MAX_LAZY_MATCH: usize = 258;

if length >= GOOD_MATCH_LENGTH {
    // Skip lazy matching - current match is good enough
} else if self.lazy_matching && pos + 1 < data.len() {
    // ... existing logic
}
```

---

## Priority 4: Checksum Optimizations (Est. 1-2 days)

### 4a. SIMD Adler-32

**Current**: Byte-by-byte with deferred modulo in [`src/compress/adler32.rs`](src/compress/adler32.rs):

```8:29:src/compress/adler32.rs
pub fn adler32(data: &[u8]) -> u32 {
    const MOD_ADLER: u32 = 65_521;
    const NMAX: usize = 5552;

    let mut s1: u32 = 1;
    let mut s2: u32 = 0;

    for chunk in data.chunks(NMAX) {
        for &b in chunk {
            s1 += b as u32;
            s2 += s1;
        }
        s1 %= MOD_ADLER;
        s2 %= MOD_ADLER;
    }

    (s2 << 16) | s1
}
```

**Optimization**: Use SIMD to process 16-32 bytes at once with weighted sums:

```rust
#[cfg(all(feature = "simd", target_arch = "x86_64"))]
#[target_feature(enable = "ssse3")]
unsafe fn adler32_simd(data: &[u8]) -> u32 {
    use std::arch::x86_64::*;
    // Process 16 bytes at a time using _mm_sad_epu8 for s1
    // and weighted multiplication for s2
}
```

**Expected speedup**: 4-8x for Adler-32

### 4b. SIMD CRC32 (Hardware Acceleration)

**Current**: Table-based byte-by-byte in [`src/compress/crc32.rs`](src/compress/crc32.rs):

```29:35:src/compress/crc32.rs
pub fn crc32(data: &[u8]) -> u32 {
    let mut crc = 0xFFFFFFFF_u32;
    for &byte in data {
        let index = ((crc ^ byte as u32) & 0xFF) as usize;
        crc = (crc >> 8) ^ CRC_TABLE[index];
    }
    crc ^ 0xFFFFFFFF
}
```

**Optimization**: Use hardware CRC32 instruction (processes 8 bytes per instruction):

```rust
#[cfg(all(feature = "simd", target_arch = "x86_64"))]
#[target_feature(enable = "sse4.2")]
unsafe fn crc32_hw(data: &[u8]) -> u32 {
    use std::arch::x86_64::_mm_crc32_u64;
    let mut crc = !0u64;

    for chunk in data.chunks_exact(8) {
        let val = u64::from_le_bytes(chunk.try_into().unwrap());
        crc = _mm_crc32_u64(crc, val);
    }
    // Handle remaining bytes...
    !crc as u32
}
```

**Expected speedup**: 8-10x for CRC32---

## Implementation Structure

New file structure under `src/simd/`:

```javascript
src/
├── simd/
│   ├── mod.rs           # Feature detection, dispatch macros
│   ├── x86_64.rs        # SSE2/SSE4.2/AVX2 implementations
│   ├── aarch64.rs       # NEON implementations (future)
│   └── fallback.rs      # Scalar fallbacks (current implementations)
```

**Feature gate in `Cargo.toml`**:

```toml
[features]
default = []
simd = []  # Enable SIMD optimizations
parallel = ["rayon"]  # Already exists
```

---

## Expected Results

| Component | Current | After Optimization | Notes || -------------- | --------------- | ------------------ | ------------------------- || JPEG DCT | baseline | 2-3x faster | AAN algorithm || JPEG Color | baseline | 1.3-1.5x faster | Integer math || JPEG BitWriter | baseline | 1.2x faster | Batch writes || **JPEG Total** | **~2x slower** | **~parity** | || PNG Filters | baseline | 2-4x faster | SIMD (with feature) || PNG Checksums | baseline | 4-10x faster | SIMD/HW (with feature) || LZ77 Matching | baseline | 1.5-2x faster | SIMD + algorithm || **PNG Total** | **~20x slower** | **~3-5x slower** | Without deps, gap remains |---

## Caveats

Without using optimized dependencies like `miniz_oxide`, the PNG encoder will still have a performance gap compared to `image` crate. The gap can be reduced to ~3-5x through pure algorithmic and SIMD optimizations, but matching C-level DEFLATE performance in pure Rust without battle-tested compression libraries is challenging.The JPEG encoder, however, can achieve near-parity through the DCT and color conversion optimizations since those are mathematically well-defined transforms.those are mathematically well-defined transforms.
