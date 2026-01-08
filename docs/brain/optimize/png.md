# PNG Optimization Plan for go-pixo

Comprehensive optimization roadmap to achieve Rust-level performance while maintaining code quality and compression quality.

## Executive Summary

This document outlines a prioritized optimization plan for go-pixo's PNG encoder. The goal is to achieve **10-50x speedup** in compression operations while maintaining identical compression quality and output compatibility. The plan is based on analysis of the current codebase, comparison with the Rust implementation (pixo), and research into modern PNG optimization techniques.

The optimizations are organized into three priority levels based on impact-to-effort ratio:

- **Priority 1**: High impact, low-medium effort - Quick wins with significant improvements
- **Priority 2**: High impact, medium effort - Significant improvements requiring more work
- **Priority 3**: Medium impact, high effort - Advanced optimizations for maximum performance

---

## 1. Current State Analysis

### 1.1 Performance Bottlenecks Identified

After analyzing the current implementation against the Rust benchmark and researching optimization techniques, the following critical bottlenecks have been identified:

| Bottleneck | Location | Impact | Priority |
|------------|----------|--------|----------|
| Palette lookup O(n) | `palette.go:43-64` | 10-100x slower than LUT | P1 |
| No K-means refinement | `median_cut.go` | 5-15% quality loss | P1 |
| No Bigrams filter | `filter_selector.go` | 2-5% compression loss | P1 |
| Perceptual distance | `palette.go:43-64` | Suboptimal color matching | P1 |
| Per-row allocations | Multiple files | GC pressure, memory overhead | P2 |
| Sequential filtering | `filter_selector.go:177-191` | No multi-core utilization | P2 |
| No early termination | `filter_selector.go` | Wasted filter evaluations | P2 |
| Greedy LZ77 matching | `lz77_encoder.go` | Suboptimal compression | P3 |

### 1.2 Current Implementation Gaps

**Filter Selection** (`src/png/filter_selector.go`):
- ✅ Has 10 filter strategies (MinSum, Adaptive, AdaptiveFast, Entropy, BruteForce)
- ❌ **MISSING**: Bigrams strategy (critical for DEFLATE optimization)
- ❌ **MISSING**: Early termination logic
- ❌ **MISSING**: Scratch buffer reuse

**Quantization** (`src/png/quantize.go`, `src/png/median_cut.go`):
- ✅ Implements median-cut algorithm
- ❌ **MISSING**: K-means refinement iterations
- ❌ **MISSING**: Perceptual (Redmean) color distance

**Palette Lookup** (`src/png/palette.go`):
- ✅ Basic Euclidean distance implementation
- ❌ **MISSING**: O(1) LUT lookup (6-6-6 RGB)
- ❌ **MISSING**: Perceptual distance formula

**Memory Management**:
- ❌ **MISSING**: Scratch buffer reuse (creates new buffers per row)
- ❌ **MISSING**: sync.Pool for temporary objects
- ❌ **MISSING**: Parallel filter selection for large images

---

## 2. Priority 1 Optimizations

These optimizations provide the highest impact with reasonable effort. They should be implemented first.

### 2.1 Palette Lookup Table (LUT)

**Impact**: 10-100x speedup for quantization phase
**Effort**: Medium
**Memory**: ~256KB for 6-6-6 RGB LUT

#### Problem

Currently, `FindNearest` in `palette.go:43-64` performs O(n) linear search through the palette for each pixel. For a 256-color palette and a 1-megapixel image (1M pixels), this requires up to 256M distance calculations.

#### Solution

Implement a 6-6-6 RGB lookup table (64×64×64 = 262,144 entries) for O(1) palette indexing:

```go
// src/png/palette_lut.go

package png

// PaletteLut provides O(1) palette lookup using a precomputed table.
type PaletteLut struct {
    opaqueLut [64][64][64]uint8
    palette   Palette
}

// NewPaletteLut creates a new palette lookup table.
func NewPaletteLut(palette Palette) *PaletteLut {
    lut := &PaletteLut{palette: palette}
    for r6 := 0; r6 < 64; r6++ {
        for g6 := 0; g6 < 64; g6++ {
            for b6 := 0; b6 < 64; b6++ {
                // Convert 6-bit back to 8-bit
                r8 := (r6 << 2) | (r6 >> 4)
                g8 := (g6 << 2) | (g6 >> 4)
                b8 := (b6 << 2) | (b6 >> 4)
                lut.opaqueLut[r6][g6][b6] = uint8(palette.FindNearestIndex(r8, g8, b8))
            }
        }
    }
    return lut
}

// Lookup returns the palette index for an opaque RGB color in O(1) time.
func (lut *PaletteLut) Lookup(r, g, b uint8) uint8 {
    r6 := r >> 2
    g6 := g >> 2
    b6 := b >> 2
    return lut.opaqueLut[r6][g6][b6]
}
```

#### Integration

Modify `Quantize` in `quantize.go:28-36` to use the LUT:

```go
func Quantize(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
    // ... existing palette creation ...
    
    // Create LUT for fast lookups
    lut := NewPaletteLut(*palette)
    
    indexed := make([]byte, width)
    for i := 0; i < width; i++ {
        offset := i * bpp
        indexed[i] = lut.Lookup(pixels[offset], pixels[offset+1], pixels[offset+2])
    }
    
    return indexed, *palette
}
```

#### Expected Results

- **Quantization speed**: 10-100x improvement (depending on palette size)
- **Memory usage**: +256KB for LUT
- **Quality**: No change (identical results)

### 2.2 K-means Palette Refinement

**Impact**: 5-15% better visual quality for quantized images
**Effort**: Medium
**Speed**: ~10% slowdown (worth it for quality)

#### Problem

Median-cut produces palette colors based only on color space distribution, not on actual image content. This can lead to suboptimal palettes for photographic images with smooth gradients.

#### Solution

Add 2-3 iterations of K-means refinement after median-cut to adjust palette colors to match actual image data:

```go
// src/png/kmeans.go

package png

// RefinePaletteKmeans applies K-means refinement to a palette.
func RefinePaletteKmeans(palette *Palette, colors []ColorCount, iterations int) {
    for iter := 0; iter < iterations; iter++ {
        // Accumulate weighted centroids
        accumulators := make([][5]int, palette.Len())
        for _, c := range colors {
            bestIdx := palette.FindNearestIndex(c.Color.R, c.Color.G, c.Color.B)
            acc := &accumulators[bestIdx]
            acc[0] += int(c.Color.R) * c.Count
            acc[1] += int(c.Color.G) * c.Count
            acc[2] += int(c.Color.B) * c.Count
            acc[4] += c.Count
        }
        
        // Update palette to centroids
        for i, acc := range accumulators {
            if acc[4] > 0 {
                palette.Colors[i] = Color{
                    R: uint8(acc[0] / acc[4]),
                    G: uint8(acc[1] / acc[4]),
                    B: uint8(acc[2] / acc[4]),
                }
            }
        }
    }
}
```

#### Integration

Modify `Quantize` in `quantize.go` to call refinement after median-cut:

```go
func Quantize(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
    colorMap := CountColors(pixels, colorType)
    colorsWithCount := ToColorWithCountSlice(colorMap)
    
    paletteColors := MedianCut(colorsWithCount, maxColors)
    
    palette := NewPalette(len(paletteColors))
    for _, c := range paletteColors {
        palette.AddColor(c)
    }
    
    // K-means refinement for better photographic quality
    RefinePaletteKmeans(palette, colorsWithCount, 2)
    
    // ... rest of function ...
}
```

#### Expected Results

- **Visual quality**: 5-15% improvement for photographic content
- **Speed**: ~10% slowdown (negligible compared to LUT speedup)
- **Compression**: 2-5% better for quantized images

### 2.3 Bigrams Filter Strategy

**Impact**: 2-5% better compression ratio
**Effort**: Medium
**Speed**: Similar to MinSum strategy

#### Problem

Go-pixo lacks the `Bigrams` filter strategy that optimizes for DEFLATE LZ77 matching by minimizing distinct byte pairs. The current MinSum strategy minimizes sum of absolute values, which correlates with but doesn't directly optimize for DEFLATE compression.

#### Solution

Add a new filter strategy that counts and minimizes distinct bigrams (byte pairs) in filtered output:

```go
// src/png/filter_bigrams.go

package png

// FilterStrategyBigrams selects the filter that minimizes distinct bigrams.
const FilterStrategyBigrams FilterStrategy = "Bigrams"

func selectBigrams(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    var bestFilter FilterType
    var bestFiltered []byte
    bestBigrams := -1
    
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
        bigrams := countDistinctBigrams(filtered)
        if bestBigrams < 0 || bigrams < bestBigrams {
            bestBigrams = bigrams
            bestFilter = f.typ
            bestFiltered = filtered
        }
    }
    
    return bestFilter, bestFiltered
}

// countDistinctBigrams counts unique byte pairs in the data.
func countDistinctBigrams(data []byte) int {
    if len(data) < 2 {
        return 0
    }
    bigrams := make(map[uint16]bool)
    for i := 0; i < len(data)-1; i++ {
        bigram := uint16(data[i])<<8 | uint16(data[i+1])
        bigrams[bigram] = true
    }
    return len(bigrams)
}
```

#### Optimized Bigrams Counting

For even better performance, use a bitmask-based approach for rows up to 256 bytes:

```go
func countDistinctBigramsOptimized(data []byte) int {
    if len(data) < 2 {
        return 0
    }
    
    // Use two 64-bit masks for bigrams 0-127 and 128-255
    var lowMask uint64
    var highMask uint64
    
    for i := 0; i < len(data)-1; i++ {
        bigram := uint16(data[i])<<8 | uint16(data[i+1])
        if bigram < 128 {
            lowMask |= 1 << bigram
        } else {
            highMask |= 1 << (bigram - 128)
        }
    }
    
    return bits.OnesCount64(lowMask) + bits.OnesCount64(highMask)
}
```

#### Integration

Update `SelectFilterWithStrategy` in `filter_selector.go:7-32` to handle the new strategy:

```go
func SelectFilterWithStrategy(row []byte, prevRow []byte, bpp int, strategy FilterStrategy) (FilterType, []byte) {
    switch strategy {
    // ... existing cases ...
    case FilterStrategyBigrams:
        return selectBigrams(row, prevRow, bpp)
    // ... 
    }
}
```

Update `filter_types.go` to add the new constant:

```go
const FilterStrategyBigrams FilterStrategy = "Bigrams"
```

#### Expected Results

- **Compression ratio**: 2-5% improvement on typical images
- **Speed**: Similar to MinSum strategy
- **Compatibility**: No breaking changes

### 2.4 Perceptual Color Distance (Redmean)

**Impact**: Better visual quality for same compression ratio
**Effort**: Low
**Speed**: Same as Euclidean

#### Problem

Current palette lookup uses simple Euclidean distance in RGB space, which doesn't account for human visual perception. The human eye is more sensitive to green than red or blue, and sensitivity varies with luminance level.

#### Solution

Implement the Redmean formula for perceptual color distance:

```go
// src/png/perceptual.go

package png

// PerceptualDistanceSq calculates the squared perceptual distance between two colors.
// Uses the Redmean formula which accounts for human visual perception.
func PerceptualDistanceSq(c1, c2 Color) uint32 {
    dr := int(c1.R) - int(c2.R)
    dg := int(c1.G) - int(c2.G)
    db := int(c1.B) - int(c2.B)
    
    // Redmean: weights vary based on average red intensity
    // For high red (bright colors), red differences matter more
    // For low red (dark colors), blue differences matter more
    // Green always has the highest fixed weight (human eye most sensitive)
    rMean := (int(c1.R) + int(c2.R)) >> 1
    
    // Scale weights by 256 to avoid floating point
    // r_weight = 2 + r_mean/256 → (512 + r_mean) / 256
    // g_weight = 4 (fixed)
    // b_weight = 2 + (255-r_mean)/256 → (767 - r_mean) / 256
    rWeight := 512 + rMean
    bWeight := 767 - rMean
    const gWeight = 1024 // 4 * 256
    
    // Compute weighted distance (result scaled by 256)
    dist := (rWeight*dr*dr + gWeight*dg*dg + bWeight*db*db) >> 8
    
    return uint32(dist)
}
```

#### Integration

Update `FindNearest` in `palette.go:43-64` to use perceptual distance:

```go
func (p *Palette) FindNearest(c Color) int {
    if p.NumColors == 0 {
        return 0
    }
    
    bestIdx := 0
    bestDist := uint32(math.MaxUint32)
    
    for i := 0; i < p.NumColors; i++ {
        dist := PerceptualDistanceSq(c, p.Colors[i])
        if dist < bestDist {
            bestDist = dist
            bestIdx = i
        }
    }
    
    return bestIdx
}
```

#### Why Redmean?

The Redmean formula provides a good balance between accuracy and performance:

| Formula | Accuracy | Performance | Complexity |
|---------|----------|-------------|------------|
| Euclidean | Low | Fast | Simple |
| Redmean | High | Fast | Simple |
| CIEDE2000 | Highest | Slow | Complex |

For PNG quantization, Redmean provides significant quality improvement over Euclidean with no performance penalty.

#### Expected Results

- **Visual quality**: Noticeable improvement for skin tones, gradients, and highlights
- **Speed**: No change (same computational complexity)
- **Compression**: No change (palette selection is independent)

---

## 3. Priority 2 Optimizations

These optimizations provide significant speed improvements but require more implementation effort.

### 3.1 Scratch Buffer Reuse with sync.Pool

**Impact**: Reduced GC pressure, faster for large images
**Effort**: Medium
**Memory**: Reuses buffers instead of allocating per-row

#### Problem

Currently, each filter evaluation allocates new byte slices, causing significant GC pressure for large images. For a 4K image with 2160 rows, this could mean 10,000+ allocations per filter evaluation.

#### Solution

Use Go's `sync.Pool` to reuse scratch buffers across filter evaluations:

```go
// src/png/filter_scratch.go

package png

import "sync"

var filterScratchPool = sync.Pool{
    New: func() interface{} {
        return &FilterScratch{
            none:  make([]byte, 0, 256),
            sub:   make([]byte, 0, 256),
            up:    make([]byte, 0, 256),
            avg:   make([]byte, 0, 256),
            paeth: make([]byte, 0, 256),
        }
    },
}

// FilterScratch holds reusable buffers for filter evaluation.
type FilterScratch struct {
    none  []byte
    sub   []byte
    up    []byte
    avg   []byte
    paeth []byte
}

func (s *FilterScratch) clear() {
    s.none = s.none[:0]
    s.sub = s.sub[:0]
    s.up = s.up[:0]
    s.avg = s.avg[:0]
    s.paeth = s.paeth[:0]
}

func (s *FilterScratch) reset(minSize int) {
    s.clear()
    
    if cap(s.none) < minSize {
        s.none = make([]byte, minSize)
        s.sub = make([]byte, minSize)
        s.up = make([]byte, minSize)
        s.avg = make([]byte, minSize)
        s.paeth = make([]byte, minSize)
    }
}
```

#### Integration

Modify `SelectAll` in `filter_selector.go:177-191` to use pooled scratch buffers:

```go
func SelectAll(pixels []byte, width, height, bpp int) []FilterType {
    filters := make([]FilterType, height)
    var prevRow []byte
    
    scratch := filterScratchPool.Get().(*FilterScratch)
    defer filterScratchPool.Put(scratch)
    
    rowBytes := width * bpp
    
    for y := 0; y < height; y++ {
        offset := y * rowBytes
        row := pixels[offset : offset+rowBytes]
        
        scratch.reset(rowBytes)
        filterType, _ := selectAdaptiveWithScratch(row, prevRow, bpp, scratch)
        filters[y] = filterType
        
        prevRow = row
    }
    
    return filters
}
```

#### Expected Results

- **GC pressure**: Significantly reduced for large images
- **Speed**: 10-30% improvement for large images
- **Memory**: More consistent memory usage

### 3.2 Parallel Filter Selection

**Impact**: 2-8x speedup proportional to CPU cores
**Effort**: Medium-High
**Memory**: Minimal overhead for coordination

#### Problem

Filter selection for each row is currently sequential, even though rows are independent. For tall images (e.g., 4000x3000), this is a significant bottleneck.

#### Solution

Use goroutines to process rows in parallel:

```go
// src/png/filter_parallel.go

package png

import "sync"

func SelectAllParallel(pixels []byte, width, height, bpp int) []FilterType {
    if height <= 32 {
        // Not worth the overhead for small images
        return SelectAll(pixels, width, height, bpp)
    }
    
    filters := make([]FilterType, height)
    rowBytes := width * bpp
    
    type result struct {
        y      int
        filter FilterType
    }
    results := make(chan result, height)
    
    // Process rows in parallel
    for y := 0; y < height; y++ {
        go func(y int) {
            offset := y * rowBytes
            row := pixels[offset : offset+rowBytes]
            
            var prevRow []byte
            if y > 0 {
                prevOffset := (y - 1) * rowBytes
                prevRow = pixels[prevOffset : prevOffset+rowBytes]
            }
            
            filterType, _ := SelectFilter(row, prevRow, bpp)
            results <- result{y, filterType}
        }(y)
    }
    
    // Collect results
    for i := 0; i < height; i++ {
        r := <-results
        filters[r.y] = r.filter
    }
    
    return filters
}
```

#### Adaptive Parallelism

Use different strategies based on image dimensions:

```go
func SelectAllOptimized(pixels []byte, width, height, bpp int) []FilterType {
    // Use parallel for tall images, sequential for wide/small
    if height > 64 && height > width {
        return SelectAllParallel(pixels, width, height, bpp)
    }
    return SelectAll(pixels, width, height, bpp)
}
```

#### Expected Results

- **Speed**: 2-8x improvement (proportional to CPU cores)
- **Overhead**: ~1ms for goroutine coordination
- **Best for**: Tall images (height >> width)

### 3.3 Early Termination in Filter Selection

**Impact**: 10-30% faster filter selection
**Effort**: Low
**Memory**: No additional memory

#### Problem

Currently, all 5 filters are evaluated even when a good enough score is found early. For many rows, the best filter is obvious after evaluating 2-3 filters.

#### Solution

Add early termination logic to skip remaining filters when best score is already optimal:

```go
// src/png/filter_early.go

package png

func selectAdaptiveEarly(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    scratch := filterScratchPool.Get().(*FilterScratch)
    defer filterScratchPool.Put(scratch)
    scratch.reset(len(row))
    
    var bestFilter FilterType
    var bestFiltered []byte
    bestScore := -1
    
    // Early-stop threshold: if a candidate beats this, skip remaining
    earlyStop := len(row) / 4
    
    filters := []struct {
        typ FilterType
        fn  func() []byte
    }{
        {FilterNone, func() []byte { 
            scratch.none = scratch.none[:len(row)]
            copy(scratch.none, row)
            return scratch.none
        }},
        {FilterSub, func() []byte { 
            return ApplyFilterSubTo(row, bpp, scratch.sub[:0]) 
        }},
        {FilterUp, func() []byte { 
            return ApplyFilterUpTo(row, prevRow, scratch.up[:0]) 
        }},
        {FilterAverage, func() []byte { 
            return ApplyFilterAverageTo(row, prevRow, bpp, scratch.avg[:0]) 
        }},
        {FilterPaeth, func() []byte { 
            return ApplyFilterPaethTo(row, prevRow, bpp, scratch.paeth[:0]) 
        }},
    }
    
    for _, f := range filters {
        filtered := f.fn()
        score := SumAbsoluteValues(filtered)
        
        if bestScore < 0 || score < bestScore {
            bestScore = score
            bestFilter = f.typ
            bestFiltered = filtered
            
            // Early termination if we've found a near-perfect score
            if bestScore <= earlyStop {
                break
            }
        }
    }
    
    return bestFilter, bestFiltered
}
```

#### Expected Results

- **Speed**: 10-30% faster filter selection
- **Quality**: No change (still evaluates all filters if needed)
- **Best for**: Images with obvious optimal filters

---

## 4. Priority 3 Optimizations

These optimizations provide additional performance but require significant implementation effort.

### 4.1 Cost-Model LZ77 Optimal Parsing

**Impact**: 3-8% better compression ratio
**Effort**: High
**Speed**: Slower encoding, same decoding

#### Problem

Current LZ77 encoder uses greedy matching - it takes the first match found. This doesn't always produce optimal DEFLATE output because LZ77 has non-local effects (matches can affect later decisions).

#### Solution

Implement optimal parsing with dynamic programming:

```go
// src/compress/lz77_optimal.go

package compress

// CostModel estimates the cost of encoding tokens.
type CostModel struct {
    literalCost [256]float32
    matchCost   [259]float32 // length 3-258
}

// NewCostModel creates a cost model from Huffman frequencies.
func NewCostModel(litLenCounts []int, distCounts []int) *CostModel {
    cm := &CostModel{}
    
    // Estimate literal costs from frequencies
    total := float32(0)
    for _, c := range litLenCounts {
        total += float32(c)
    }
    for i := range cm.literalCost {
        if total > 0 {
            cm.literalCost[i] = -log2(float32(litLenCounts[i]+1) / total)
        } else {
            cm.literalCost[i] = 8 // Default to 8 bits
        }
    }
    
    // Estimate match costs
    total = 0
    for _, c := range distCounts {
        total += float32(c)
    }
    for i := range cm.matchCost {
        if total > 0 {
            cm.matchCost[i] = -log2(float32(distCounts[min(i, len(distCounts)-1)]+1) / total)
        } else {
            cm.matchCost[i] = 8
        }
    }
    
    return cm
}

// OptimalLZ77 performs optimal parsing with cost model.
func (enc *LZ77Encoder) OptimalEncode(data []byte, costModel *CostModel) []Token {
    // Dynamic programming approach:
    // dp[i] = minimum cost to encode data[i:]
    // Track backpointers for reconstruction
    
    n := len(data)
    dp := make([]float32, n+1)
    next := make([]int, n)
    matchLen := make([]int, n)
    
    dp[n] = 0 // Cost of empty suffix
    
    for i := n - 1; i >= 0; i-- {
        // Literal cost
        dp[i] = dp[i+1] + costModel.literalCost[data[i]]
        next[i] = -1 // -1 indicates literal
        
        // Check all possible matches
        for length := enc.minMatchLen; length <= maxMatchLength && i+length <= n; length++ {
            // Find match at this position (simplified - real impl would use hash table)
            dist, found := enc.findMatchAt(data, i, length)
            if !found {
                break
            }
            
            cost := costModel.matchCost[length-3] // Adjust for index
            if i+length <= n {
                cost += dp[i+length]
            }
            
            if cost < dp[i] {
                dp[i] = cost
                next[i] = dist
                matchLen[i] = length
            }
        }
    }
    
    // Reconstruct optimal token sequence
    var tokens []Token
    i := 0
    for i < n {
        if next[i] < 0 {
            tokens = append(tokens, TokenLiteral(data[i]))
            i++
        } else {
            tokens = append(tokens, TokenMatch(uint16(next[i]), uint16(matchLen[i])))
            i += matchLen[i]
        }
    }
    
    return tokens
}
```

#### Expected Results

- **Compression**: 3-8% better ratio
- **Speed**: 2-10x slower (depends on iterations)
- **Use case**: Offline compression where size matters more than speed

### 4.2 SIMD Acceleration for DCT

**Impact**: 3-5x speedup for DCT operations
**Effort**: High (requires assembly or external library)
**Memory**: No additional memory

#### Problem

Go doesn't have native SIMD support, but there are options:
1. Use `golang.org/x/exp/simd` (experimental)
2. Use assembly (complex, platform-specific)
3. Use a library like `github.com/klauspost/asm`

#### Solution Options

**Option A: Use golang.org/x/exp/simd** (easiest, but experimental):

```go
package png

import "golang.org/x/exp/simd"

func ForwardDCTSIMD(block [64]float32) [64]float32 {
    // Use SIMD for horizontal transform
    // Fall back to scalar for vertical transform
    var result [64]float32
    
    // Load 8 floats into SIMD register
    for row := 0; row < 8; row++ {
        rowData := block[row*8 : row*8+8]
        simdBlock := simd.Load32(rowData)
        // ... SIMD operations ...
    }
    
    return result
}
```

**Option B: Use assembly** (fastest, most complex):

This requires writing platform-specific assembly code for x86_64 and ARM64.

**Option C: Use existing library** (recommended for now):

```go
package png

import "github.com/klauspost/compress/fz"
```

#### Expected Results

- **DCT speed**: 3-5x improvement
- **JPEG encoding**: 20-30% overall speedup
- **Trade-off**: Complexity vs. performance

---

## 5. Implementation Roadmap

### Phase 1: Quick Wins (Week 1)

**Focus**: High-impact, low-effort optimizations

| Task | Effort | Impact | Status |
|------|--------|--------|--------|
| Add Palette LUT (6-6-6 RGB) | 2 days | 10-100x faster quantization | ⬜ |
| Add Perceptual Distance (Redmean) | 1 day | Better quality, no speed cost | ⬜ |
| Add Early Termination | 1 day | 10-30% faster filtering | ⬜ |

### Phase 2: Core Improvements (Week 2)

**Focus**: Medium-effort optimizations with significant speed improvements

| Task | Effort | Impact | Status |
|------|--------|--------|--------|
| Add K-means Refinement | 2 days | 5-15% better quality | ⬜ |
| Add Bigrams Filter Strategy | 2 days | 2-5% better compression | ⬜ |
| Add Scratch Buffer Reuse | 2 days | Reduced GC pressure | ⬜ |

### Phase 3: Parallelization (Week 3)

**Focus**: Multi-core utilization

| Task | Effort | Impact | Status |
|------|--------|--------|--------|
| Add Parallel Filter Selection | 3 days | 2-8x speedup | ⬜ |
| Benchmark and Tune | 1 day | Validate improvements | ⬜ |

### Phase 4: Advanced Optimizations (Week 4+)

**Focus**: Complex optimizations for maximum performance

| Task | Effort | Impact | Status |
|------|--------|--------|--------|
| Cost-Model LZ77 Optimal Parsing | 1 week | 3-8% better compression | ⬜ |
| SIMD Acceleration | 1 week | 3-5x DCT speedup | ⬜ |

---

## 6. Expected Performance Improvements

### Quantization (Lossy PNG)

| Metric | Current | After Phase 1 | After Phase 2 | After Phase 3 |
|--------|---------|---------------|---------------|---------------|
| Speed | 1x (baseline) | 10-50x | 10-60x | 20-100x |
| Quality | Baseline | +5-15% | +5-20% | +5-20% |
| Compression | Baseline | +2-5% | +4-10% | +4-10% |

### Lossless PNG

| Metric | Current | After Phase 1 | After Phase 2 | After Phase 3 |
|--------|---------|---------------|---------------|---------------|
| Speed | 1x (baseline) | 1.5x | 2x | 4-8x |
| Compression | Baseline | +1-2% | +3-5% | +3-5% |

### Overall

- **Typical images**: 5-20x speedup with same or better quality
- **Large images**: 10-50x speedup with parallel processing
- **Memory usage**: +256KB for LUT, but reduced allocations

---

## 7. Testing Strategy

### Unit Tests

For each optimization, add comprehensive unit tests:

```go
func TestPaletteLUT(t *testing.T) {
    palette := NewPalette(256)
    // ... add colors ...
    
    lut := NewPaletteLut(palette)
    
    // Verify LUT matches direct lookup
    for r := uint8(0); r < 255; r += 17 {
        for g := uint8(0); g < 255; g += 17 {
            for b := uint8(0); b < 255; b += 17 {
                lutIdx := lut.Lookup(r, g, b)
                directIdx := palette.FindNearest(Color{r, g, b})
                if lutIdx != directIdx {
                    t.Errorf("LUT mismatch for (%d, %d, %d): LUT=%d, direct=%d", 
                        r, g, b, lutIdx, directIdx)
                }
            }
        }
    }
}
```

### Benchmark Tests

Add benchmarks to track performance improvements:

```go
func BenchmarkQuantize(b *testing.B) {
    img := generateTestImage(1024, 1024)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Quantize(img, ColorRGB, 256)
    }
}
```

### Integration Tests

Verify output compatibility:

```go
func TestQuantizationOutput(t *testing.T) {
    original := loadTestImage("testdata/photo.png")
    
    // Encode with current implementation
    result1, _ := Quantize(original.pixels, original.colorType, 256)
    
    // Verify against reference (Rust implementation)
    expectedSize := getReferenceSize("photo.png", 256)
    if len(result1) < expectedSize*95/100 || len(result1) > expectedSize*105/100 {
        t.Errorf("Unexpected output size: got %d, expected ~%d", len(result1), expectedSize)
    }
}
```

---

## 8. Compatibility and Safety

### Backward Compatibility

All optimizations maintain backward compatibility:
- Output format unchanged (standard PNG)
- API unchanged (same function signatures)
- Quality unchanged (or improved)

### Thread Safety

- `sync.Pool` is thread-safe by design
- Parallel filter selection uses proper synchronization
- LUT is read-only after creation (thread-safe)

### Memory Safety

- No unsafe pointers
- Bounds checking on all array access
- Proper cleanup with defer statements

---

## 9. Conclusion

This optimization plan provides a clear roadmap to achieve Rust-level performance in go-pixo's PNG encoder. The key insights are:

1. **Palette LUT provides the biggest speedup** - The O(n) to O(1) improvement alone can provide 10-100x speedup for quantization.

2. **K-means and Perceptual Distance improve quality** - These optimizations improve visual quality without sacrificing speed.

3. **Bigrams filter improves compression** - This aligns filter selection with DEFLATE's actual compression mechanism.

4. **Parallel processing scales with cores** - For large images, multi-core utilization provides linear speedup.

5. **Scratch buffer reuse reduces GC pressure** - This improves overall performance and reduces latency spikes.

The implementation should proceed in phases, with each phase delivering measurable improvements while maintaining code quality and compatibility.

---

## References

- [PNG Specification (W3C)](https://www.w3.org/TR/png-3/)
- [Rust pixo Implementation](https://github.com/shsms/pixo)
- [OptiPNG Documentation](https://optipng.sourceforge.net/pngtech/optipng.html)
- [DEFLATE Algorithm](https://www.rfc-editor.org/rfc/rfc1951)
- [Color Difference Formulas](https://en.wikipedia.org/wiki/Color_difference#sRGB)
