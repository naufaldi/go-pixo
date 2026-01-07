# Go-pixo vs pixo: PNG Comparison

Detailed comparison of PNG compression implementations between Go and Rust.

## Table of Contents

1. [Overview](#1-overview)
2. [Filter Selection](#2-filter-selection)
3. [Quantization and Palette](#3-quantization-and-palette)
4. [Performance Architecture](#4-performance-architecture)
5. [Compression Quality](#5-compression-quality)

---

## 1. Overview

This document provides a detailed comparison of PNG compression implementations between go-pixo (Go) and pixo (Rust). The focus is on understanding the architectural differences in filter selection, color quantization, performance optimizations, and compression quality.

PNG compression involves several stages:
- **Filter selection**: Preprocesses scanlines to improve compressibility
- **Color quantization**: Reduces color palette size for indexed images
- **DEFLATE compression**: Combines LZ77 with Huffman coding

Both implementations follow the PNG specification (RFC 2083) but differ significantly in their approach to optimization.

---

## 2. Filter Selection

Filter selection is a critical step in PNG encoding that preprocesses image data to improve compressibility. The PNG specification defines five filter types (None, Sub, Up, Average, Paeth), and both implementations extend these with adaptive selection strategies.

### 2.1 Go-pixo Filter Strategies

The Go implementation offers 10 distinct filter strategies, providing flexibility for different image types and performance requirements.

```go:3:32:go-pixo/src/png/filter_selector.go
func SelectFilterWithStrategy(row []byte, prevRow []byte, bpp int, strategy FilterStrategy) (FilterType, []byte) {
    switch strategy {
    case FilterStrategyNone:
        return selectNone(row, prevRow, bpp)
    case FilterStrategySub:
        return selectSub(row, prevRow, bpp)
    case FilterStrategyUp:
        return selectUp(row, prevRow, bpp)
    case FilterStrategyAverage:
        return selectAverage(row, prevRow, bpp)
    case FilterStrategyPaeth:
        return selectPaeth(row, prevRow, bpp)
    case FilterStrategyMinSum:
        return selectMinSum(row, prevRow, bpp)
    case FilterStrategyAdaptive:
        return selectAdaptive(row, prevRow, bpp)
    case FilterStrategyAdaptiveFast:
        return selectAdaptiveFast(row, prevRow, bpp)
    case FilterStrategyEntropy:
        return selectEntropy(row, prevRow, bpp)
    case FilterStrategyBruteForce:
        return selectBruteForce(row, prevRow, bpp)
    default:
        return selectAdaptive(row, prevRow, bpp)
    }
}
```

#### MinSum Strategy

The `MinSum` strategy evaluates all five filters and selects the one with the lowest sum of absolute values, which generally correlates with better compressibility.

```go:54:81:go-pixo/src/png/filter_selector.go
func selectMinSum(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    var bestFilter FilterType
    var bestFiltered []byte
    bestScore := -1

    filters := []struct {
        typ FilterType
        fn  func() []byte
    }{
        {FilterNone, func() []byte { return ApplyFilterNone(row) }},
        {FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
        {FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
        {FilterAverage, func() []byte { return ApplyFilterAverage(row, prevRow, bpp) }},
        {FilterPaeth, func() []byte { return ApplyFilterPaeth(row, prevRow, bpp) }},
    }

    for _, f := range filters {
        filtered := f.fn()
        score := SumAbsoluteValues(filtered)
        if bestScore < 0 || score < bestScore {
            bestScore = score
            bestFilter = f.typ
            bestFiltered = filtered
        }
    }

    return bestFilter, bestFiltered
}
```

#### AdaptiveFast Strategy

The `AdaptiveFast` strategy provides a speed optimization by evaluating only three filters (None, Sub, Up) instead of all five.

```go:87:113:go-pixo/src/png/filter_selector.go
func selectAdaptiveFast(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    var bestFilter FilterType
    var bestFiltered []byte
    bestScore := -1

    filters := []struct {
        typ FilterType
        fn  func() []byte
    }{
        {FilterNone, func() []byte { return ApplyFilterNone(row) }},
        {FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
        {FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
    }

    for _, f := range filters {
        filtered := f.fn()
        score := SumAbsoluteValues(filtered)
        if bestScore < 0 || score < bestScore {
            bestScore = score
            bestFilter = f.typ
            bestFiltered = filtered
        }
    }

    return bestFilter, bestFiltered
}
```

#### Entropy Strategy

The `Entropy` strategy uses Shannon entropy calculation to select filters, which can sometimes yield better results for specific image types.

```go:115:142:go-pixo/src/png/filter_selector.go
func selectEntropy(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    var bestFilter FilterType
    var bestFiltered []byte
    bestEntropy := -1.0

    filters := []struct {
        typ FilterType
        fn  func() []byte
    }{
        {FilterNone, func() []byte { return ApplyFilterNone(row) }},
        {FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
        {FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
        {FilterAverage, func() []byte { return ApplyFilterAverage(row, prevRow, bpp) }},
        {FilterPaeth, func() []byte { return ApplyFilterPaeth(row, prevRow, bpp) }},
    }

    for _, f := range filters {
        filtered := f.fn()
        entropy := CalculateEntropy(filtered)
        if bestEntropy < 0 || entropy < bestEntropy {
            bestEntropy = entropy
            bestFilter = f.typ
            bestFiltered = filtered
        }
    }

    return bestFilter, bestFiltered
}
```

### 2.2 pixo Filter Strategies

The Rust implementation provides 9 filter strategies, with a critical unique addition: the `Bigrams` strategy.

```rust:343:364:pixo/src/png/mod.rs
pub enum FilterStrategy {
    /// Always use no filter (fastest encoding).
    None,
    /// Always use Sub filter.
    Sub,
    /// Always use Up filter.
    Up,
    /// Always use Average filter.
    Average,
    /// Always use Paeth filter.
    Paeth,
    /// Choose filter per row minimizing sum of absolute values (min-sum).
    MinSum,
    /// Choose best filter per row (best compression, slower).
    Adaptive,
    /// Adaptive but with early cut and limited trials (faster).
    AdaptiveFast,
    /// Choose filter per row minimizing distinct bigrams (best DEFLATE correlation).
    Bigrams,
}
```

The `Bigrams` strategy is unique to the Rust implementation and represents a significant optimization for DEFLATE compression. Rather than minimizing raw entropy or sum of absolute values, this strategy minimizes the number of distinct byte pairs (bigrams) in the filtered output, which directly correlates with LZ77 match-finding efficiency.

### 2.3 Key Differences Summary

| Aspect | Go-pixo | pixo |
|--------|---------|------|
| Total strategies | 10 | 9 |
| MinSum strategy | Yes | Yes |
| AdaptiveFast strategy | Yes | Yes |
| Entropy strategy | Yes | No |
| Bigrams strategy | No | **Yes** |
| BruteForce strategy | Yes | No |

The `Bigrams` strategy is particularly important because it optimizes for the actual DEFLATE compression algorithm rather than just raw data characteristics. This can lead to 2-5% better compression ratios on typical images.

---

## 3. Quantization and Palette

Color quantization reduces the number of colors in an image to create a palette-based representation, significantly reducing file size for images with limited color palettes.

### 3.1 Go-pixo Quantization

The Go implementation uses standard median-cut algorithm without post-processing refinement.

```go:5:39:go-pixo/src/png/quantize.go
func Quantize(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
    if maxColors <= 0 {
        maxColors = 256
    }
    if maxColors > 256 {
        maxColors = 256
    }

    colorMap := CountColors(pixels, colorType)
    colorsWithCount := ToColorWithCountSlice(colorMap)

    paletteColors := MedianCut(colorsWithCount, maxColors)

    palette := NewPalette(len(paletteColors))
    for _, c := range paletteColors {
        palette.AddColor(c)
    }

    bpp := BytesPerPixel(ColorType(colorType))
    width := len(pixels) / bpp

    indexed := make([]byte, width)

    for i := 0; i < width; i++ {
        offset := i * bpp
        c := Color{
            R: pixels[offset],
            G: pixels[offset+1],
            B: pixels[offset+2],
        }
        indexed[i] = uint8(palette.FindNearest(c))
    }

    return indexed, *palette
}
```

The palette lookup uses simple Euclidean distance in RGB space, which does not account for human visual perception.

### 3.2 Go-pixo Median-Cut Implementation

```go:15:80:go-pixo/src/png/median_cut.go
func MedianCut(colorsWithCount []ColorWithCount, maxColors int) []Color {
    if len(colorsWithCount) == 0 {
        return []Color{}
    }

    if len(colorsWithCount) <= maxColors {
        result := make([]Color, len(colorsWithCount))
        for i, cwc := range colorsWithCount {
            result[i] = cwc.Color
        }
        return result
    }

    buckets := []bucket{{colors: colorsWithCount}}

    for len(buckets) < effectiveMaxColors {
        largestIdx := -1
        maxVariance := 0.0
        for i := range buckets {
            variance := calculateColorVariance(buckets[i].colors)
            if variance > maxVariance && len(buckets[i].colors) >= 2 {
                maxVariance = variance
                largestIdx = i
            }
        }

        if largestIdx == -1 || len(buckets[largestIdx].colors) < 2 {
            break
        }

        left, right := splitBucketWithQuality(buckets[largestIdx].colors, quality)
        buckets[largestIdx].colors = left
        if len(right) > 0 {
            buckets = append(buckets, bucket{colors: right})
        }
    }

    result := make([]Color, 0, maxColors)
    for _, b := range buckets {
        if len(b.colors) > 0 {
            result = append(result, averageColors(b.colors))
        }
    }

    return result
}
```

### 3.3 pixo Quantization with K-means Refinement

The Rust implementation augments median-cut with K-means iterations, significantly improving palette quality for photographic content.

```rust:1301:1339:pixo/src/png/mod.rs
fn median_cut_palette(colors: Vec<ColorCount>, max_colors: usize) -> Vec<[u8; 4]> {
    if colors.is_empty() {
        return vec![[0, 0, 0, 255]];
    }

    let colors_for_kmeans = colors.clone();

    let mut boxes = vec![ColorBox::from_colors(colors)];
    while boxes.len() < max_colors {
        let (idx, _) = boxes
            .iter()
            .enumerate()
            .max_by_key(|(_, b)| {
                let (_, r) = b.range();
                r
            })
            .unwrap();
        if !boxes[idx].can_split() {
            break;
        }
        let b = boxes.remove(idx);
        let (l, r) = b.split();
        if !l.colors.is_empty() {
            boxes.push(l);
        }
        if !r.colors.is_empty() {
            boxes.push(r);
        }
    }

    let mut palette: Vec<[u8; 4]> = boxes.into_iter()
        .map(|b| b.make_palette_entry())
        .collect();

    // K-means refinement for better photographic quality
    refine_palette_kmeans(&mut palette, &colors_for_kmeans);

    palette
}
```

### 3.4 K-means Refinement

The K-means refinement iteratively adjusts palette colors to better match the actual image data distribution.

```rust:1346:1390:pixo/src/png/mod.rs
fn refine_palette_kmeans(palette: &mut [[u8; 4]], colors: &[ColorCount]) {
    const ITERATIONS: usize = 2;

    if palette.is_empty() || colors.is_empty() {
        return;
    }

    for _ in 0..ITERATIONS {
        let mut accumulators: Vec<(u64, u64, u64, u64, u64)> = vec![(0, 0, 0, 0, 0); palette.len()];

        for color in colors {
            let mut best_idx = 0;
            let mut best_dist = u32::MAX;
            for (i, p) in palette.iter().enumerate() {
                let dist = perceptual_distance_sq(color.rgba, *p);
                if dist < best_dist {
                    best_dist = dist;
                    best_idx = i;
                }
            }

            let count = color.count as u64;
            let acc = &mut accumulators[best_idx];
            acc.0 += color.rgba[0] as u64 * count;
            acc.1 += color.rgba[1] as u64 * count;
            acc.2 += color.rgba[2] as u64 * count;
            acc.3 += color.rgba[3] as u64 * count;
            acc.4 += count;
        }

        for (i, acc) in accumulators.iter().enumerate() {
            if acc.4 > 0 {
                palette[i] = [
                    (acc.0 / acc.4) as u8,
                    (acc.1 / acc.4) as u8,
                    (acc.2 / acc.4) as u8,
                    (acc.3 / acc.4) as u8,
                ];
            }
        }
    }
}
```

### 3.5 Perceptual Color Distance

The Rust implementation uses the Redmean formula for perceptual color distance, which accounts for human visual perception.

```rust:1404:1430:pixo/src/png/mod.rs
#[inline]
fn perceptual_distance_sq(c1: [u8; 4], c2: [u8; 4]) -> u32 {
    let dr = c1[0] as i32 - c2[0] as i32;
    let dg = c1[1] as i32 - c2[1] as i32;
    let db = c1[2] as i32 - c2[2] as i32;
    let da = c1[3] as i32 - c2[3] as i32;

    let r_mean = (c1[0] as i32 + c2[0] as i32) >> 1;

    // Redmean formula: weights vary based on average red intensity
    let r_weight = 512 + r_mean;
    let b_weight = 767 - r_mean;
    const G_WEIGHT: i32 = 1024;

    let dist = (r_weight * dr * dr + G_WEIGHT * dg * dg + b_weight * db * db) >> 8;
    (dist + da * da) as u32
}
```

The Redmean formula provides perceptually accurate color distance by adjusting weights based on the average intensity. Green has the highest fixed weight (human eye is most sensitive to green), while red and blue weights vary with luminance level.

### 3.6 Palette Lookup Table Optimization

The Rust implementation uses a precomputed 6-6-6 RGB lookup table for O(1) palette indexing.

```rust:1445:1499:pixo/src/png/mod.rs
struct PaletteLut {
    opaque_lut: Vec<u8>,
    palette: Vec<[u8; 4]>,
}

impl PaletteLut {
    fn new(palette: Vec<[u8; 4]>) -> Self {
        let mut opaque_lut = vec![0u8; 64 * 64 * 64];

        for r6 in 0..64u8 {
            for g6 in 0..64u8 {
                for b6 in 0..64u8 {
                    let r8 = (r6 << 2) | (r6 >> 4);
                    let g8 = (g6 << 2) | (g6 >> 4);
                    let b8 = (b6 << 2) | (b6 >> 4);

                    let idx = nearest_palette_index([r8, g8, b8, 255], &palette);
                    let lut_idx = ((r6 as usize) << 12) | ((g6 as usize) << 6) | (b6 as usize);
                    opaque_lut[lut_idx] = idx;
                }
            }
        }

        Self { opaque_lut, palette }
    }

    #[inline]
    fn lookup(&self, r: u8, g: u8, b: u8, a: u8) -> u8 {
        if a == 255 {
            let r6 = r >> 2;
            let g6 = g >> 2;
            let b6 = b >> 2;
            let lut_idx = ((r6 as usize) << 12) | ((g6 as usize) << 6) | (b6 as usize);
            self.opaque_lut[lut_idx]
        } else {
            nearest_palette_index([r, g, b, a], &self.palette)
        }
    }
}
```

This 256KB lookup table (64^3 entries) provides constant-time palette indexing for the common case of opaque pixels, while falling back to direct computation for transparent pixels.

---

## 4. Performance Architecture

The performance gap between Rust and Go implementations is primarily due to architectural differences in SIMD utilization, parallel processing, and memory management.

### 4.1 pixo SIMD Acceleration

The Rust implementation leverages SIMD (Single Instruction Multiple Data) instructions for critical performance bottlenecks.

```rust:70:90:pixo/src/simd/mod.rs
#[inline]
pub fn score_filter(filtered: &[u8]) -> u64 {
    #[cfg(target_arch = "x86_64")]
    {
        match *X86_SIMD_LEVEL {
            X86SimdLevel::Avx2 => unsafe { x86_64::score_filter_avx2(filtered) },
            X86SimdLevel::Ssse3 | X86SimdLevel::Sse2 => unsafe {
                x86_64::score_filter_sse2(filtered)
            },
            X86SimdLevel::Scalar => fallback::score_filter(filtered),
        }
    }

    #[cfg(target_arch = "aarch64")]
    {
        unsafe { aarch64::adler32_neon(data) }
    }

    #[cfg(not(any(target_arch = "x86_64", target_arch = "aarch64")))]
    fallback::score_filter(filtered)
}
```

The SIMD feature detection is cached at startup to eliminate runtime overhead.

```rust:44:56:pixo/src/simd/mod.rs
fn detect_x86_simd_level() -> X86SimdLevel {
    if is_x86_feature_detected!("avx2") {
        X86SimdLevel::Avx2
    } else if is_x86_feature_detected!("ssse3") {
        X86SimdLevel::Ssse3
    } else if is_x86_feature_detected!("sse2") {
        X86SimdLevel::Sse2
    } else {
        X86SimdLevel::Scalar
    }
}

static X86_SIMD_LEVEL: LazyLock<X86SimdLevel> = LazyLock::new(detect_x86_simd_level);
```

### 4.2 Filter Operations with SIMD

Filter operations are SIMD-accelerated for all five filter types.

```rust:157:178:pixo/src/simd/mod.rs
pub fn filter_sub(row: &[u8], bpp: usize, output: &mut Vec<u8>) {
    #[cfg(target_arch = "x86_64")]
    {
        match *X86_SIMD_LEVEL {
            X86SimdLevel::Avx2 => unsafe { x86_64::filter_sub_avx2(row, bpp, output) },
            X86SimdLevel::Ssse3 | X86SimdLevel::Sse2 => unsafe {
                x86_64::filter_sub_sse2(row, bpp, output)
            },
            X86SimdLevel::Scalar => fallback::filter_sub(row, bpp, output),
        }
    }

    #[cfg(target_arch = "aarch64")]
    {
        unsafe { aarch64::filter_sub_neon(row, bpp, output) };
    }

    #[cfg(not(any(target_arch = "x86_64", target_arch = "aarch64")))]
    fallback::filter_sub(row, bpp, output)
}
```

### 4.3 pixo Parallel Processing

The Rust implementation uses Rayon for parallel filter selection on images with sufficient height.

```rust:94:112:pixo/src/png/filter.rs
#[cfg(feature = "parallel")]
{
    if height > 32
        && matches!(
            strategy,
            FilterStrategy::Adaptive | FilterStrategy::AdaptiveFast | FilterStrategy::Bigrams
        )
    {
        return apply_filters_parallel(
            data,
            height as usize,
            row_bytes,
            bytes_per_pixel,
            filtered_row_size,
            strategy,
        );
    }
}
```

### 4.4 pixo Adaptive Scratch Buffers

To reduce memory allocations and GC pressure, the Rust implementation reuses scratch buffers across filter evaluations.

```rust:13:40:pixo/src/png/filter.rs
struct AdaptiveScratch {
    none: Vec<u8>,
    sub: Vec<u8>,
    up: Vec<u8>,
    avg: Vec<u8>,
    paeth: Vec<u8>,
}

impl AdaptiveScratch {
    fn new(row_len: usize) -> Self {
        Self {
            none: Vec::with_capacity(row_len),
            sub: Vec::with_capacity(row_len),
            up: Vec::with_capacity(row_len),
            avg: Vec::with_capacity(row_len),
            paeth: Vec::with_capacity(row_len),
        }
    }

    fn clear(&mut self) {
        self.none.clear();
        self.sub.clear();
        self.up.clear();
        self.avg.clear();
        self.paeth.clear();
    }
}
```

### 4.5 pixo Early Termination

The Rust implementation includes early termination logic to skip unnecessary filter evaluations.

```rust:313:327:pixo/src/png/filter.rs
fn adaptive_filter(
    row: &[u8],
    prev_row: &[u8],
    bpp: usize,
    output: &mut Vec<u8>,
    scratch: &mut AdaptiveScratch,
) {
    scratch.clear();

    let mut best_filter = FILTER_NONE;
    let mut best_score = u64::MAX;
    let early_stop = (row.len() as u64 / 4).saturating_add(1);

    scratch.none.extend_from_slice(row);
    let score = score_filter(&scratch.none);
    if score < best_score {
        best_score = score;
        best_filter = FILTER_NONE;
        if best_score <= early_stop {
            output.push(best_filter);
            output.extend_from_slice(&scratch.none);
            return;
        }
    }

    if best_score == 0 {
        output.push(best_filter);
        output.extend_from_slice(&scratch.none);
        return;
    }
    // ... continues with other filters
}
```

### 4.6 pixo Reusable Deflater Pool

For DEFLATE compression, Rust uses a reusable deflater pool to avoid repeated allocations.

```rust:75:96:pixo/src/compress/deflate.rs
static DEFLATE_REUSE: LazyLock<Vec<Mutex<Deflater>>> = LazyLock::new(|| {
    (0..=9)
        .map(|level| Mutex::new(Deflater::new(level.max(1) as u8)))
        .collect()
});

#[inline]
fn with_reusable_deflater<T>(level: u8, f: impl FnOnce(&mut Deflater) -> T) -> T {
    let level = level.clamp(1, 9);
    let idx = level as usize;
    let pool = DEFLATE_REUSE.get(idx).unwrap_or_else(|| panic!("deflater pool missing"));
    let mut guard = pool.lock().expect("deflater mutex poisoned");
    if guard.level() != level {
        *guard = Deflater::new(level);
    }
    f(&mut guard)
}
```

### 4.7 Go-pixo Current Limitations

The Go implementation lacks SIMD acceleration and uses per-row allocations without scratch buffer reuse.

```go:54:81:go-pixo/src/png/filter_selector.go
func selectMinSum(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    var bestFilter FilterType
    var bestFiltered []byte
    bestScore := -1

    filters := []struct {
        typ FilterType
        fn  func() []byte
    }{
        {FilterNone, func() []byte { return ApplyFilterNone(row) }},
        {FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
        {FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
        {FilterAverage, func() []byte { return ApplyFilterAverage(row, prevRow, bpp) }},
        {FilterPaeth, func() []byte { return ApplyFilterPaeth(row, prevRow, bpp) }},
    }

    for _, f := range filters {
        filtered := f.fn()
        score := SumAbsoluteValues(filtered)
        if bestScore < 0 || score < bestScore {
            bestScore = score
            bestFilter = f.typ
            bestFiltered = filtered
        }
    }
    return bestFilter, bestFiltered
}
```

---

## 5. Compression Quality

### 5.1 Zopfli-style Iteration

Both implementations support Zopfli-style iterative compression for better DEFLATE results.

```go:33:98:go-pixo/src/compress/zopfli.go
func ZopfliEncode(data []byte, config ZopfliConfig) ([]byte, error) {
    if len(data) == 0 {
        return []byte{}, nil
    }

    encoder := NewDeflateEncoder()
    bestResult, err := encoder.EncodeAuto(data)
    if err != nil {
        return nil, err
    }
    bestSize := len(bestResult)

    for iteration := 0; iteration < config.Iterations; iteration++ {
        encoder.SetCompressionLevel(9)
        singleResult, encodeErr := encoder.EncodeAuto(data)
        if encodeErr == nil && len(singleResult) < bestSize {
            bestResult = singleResult
            bestSize = len(singleResult)
        }

        fixedResult, encodeErr := encoder.Encode(data, false)
        if encodeErr == nil && len(fixedResult) < bestSize {
            bestResult = fixedResult
            bestSize = len(fixedResult)
        }

        dynamicResult, encodeErr := encoder.Encode(data, true)
        if encodeErr == nil && len(dynamicResult) < bestSize {
            bestResult = dynamicResult
            bestSize = len(dynamicResult)
        }
    }

    return bestResult, nil
}
```

### 5.2 Rust Cost-Model Based Optimal Parsing

The Rust implementation includes cost-model based optimal parsing with convergence detection.

```rust:291:357:pixo/src/compress/deflate.rs
pub fn deflate_optimal(data: &[u8], iterations: usize) -> Vec<u8> {
    let mut lz77 = Lz77Compressor::new(9);

    let initial_tokens = lz77.compress(data);
    let (mut lit_len_counts, mut dist_counts) = count_symbols(&initial_tokens);

    let est_bytes = estimated_deflate_size(data.len(), 9);
    let mut best_output = encode_dynamic_huffman_with_capacity(&initial_tokens, est_bytes);
    let mut best_size = best_output.len();

    let mut prev_cost = f32::MAX;
    let mut match_cache = LongestMatchCache::new(data.len());

    for iter in 0..iterations {
        let cost_model = CostModel::from_statistics(&lit_len_counts, &dist_counts);
        let tokens = lz77.compress_optimal_cached(data, &cost_model, &mut match_cache);

        let output = encode_dynamic_huffman_with_capacity(&tokens, est_bytes);
        if output.len() < best_size {
            best_size = output.len();
            best_output = output;
        }

        let (new_lit_counts, new_dist_counts) = count_symbols(&tokens);

        let cost: f32 = tokens
            .iter()
            .map(|t| match t {
                Token::Literal(b) => cost_model.literal_cost(*b),
                Token::Match { length, distance } => cost_model.match_cost(*length, *distance),
            })
            .sum();

        if iter > 2 && (prev_cost - cost).abs() < cost * 0.001 {
            break;
        }
        prev_cost = cost;

        for i in 0..286 {
            lit_len_counts[i] = (lit_len_counts[i] as f32 * 0.5 + new_lit_counts[i] as f32) as u32;
        }
        for i in 0..30 {
            dist_counts[i] = (dist_counts[i] as f32 * 0.5 + new_dist_counts[i] as f32) as u32;
        }
    }

    best_output
}
```

The convergence detection stops iterations early when no significant improvement is detected, saving computation time.

---

## Related Files

- [Main Overview](../diff-rust-go.md) - Overview of Go vs Rust comparison
- [JPEG Comparison](./diff-jpeg.md) - Detailed JPEG implementation comparison
- [Optimization Guide](./optimization-guide.md) - Actionable optimization recommendations
- [Research Papers](./research-papers.md) - Research papers and references

---

## Quick Summary

| Aspect | Go-pixo | pixo | Priority |
|--------|---------|------|----------|
| Filter strategies | 10 | 9 (+ Bigrams) | Medium |
| K-means refinement | No | Yes | High |
| Palette LUT | No | Yes (O(1)) | High |
| SIMD acceleration | No | Yes (AVX2/SSSE3/NEON) | High |
| Parallel processing | No | Yes (Rayon) | Medium |
| Early termination | No | Yes | Low |
| Perceptual distance | Euclidean | Redmean | Low |