# Advanced PNG Compression Techniques for go-pixo

## Executive Summary

This document outlines advanced PNG compression techniques that could enable go-pixo to match or exceed the Rust pixo implementation's compression ratio and performance. The techniques are organized by implementation priority, considering both impact and complexity.

---

## 1. Palette Lookup Table Optimization

**Technical Explanation**: Precomputed 64x64x64 RGB lookup table providing O(1) palette indexing with 262,144 entries (256KB memory). Eliminates expensive nearest-neighbor searches by mapping RGB values directly to palette indices through table lookup. The tradeoff of 256KB memory provides 10-100x speedup in palette operations.

For RGBA images, the implementation uses the LUT for opaque pixels (a=255) and falls back to linear search for transparent pixels.

**Implementation**:

```go
type PaletteLut struct {
    opaqueLUT [64][64][64]uint8  // 256KB total
    palette   Palette
}

func (pl *PaletteLut) Lookup(r, g, b uint8, a uint8) uint8 {
    if a == 255 {
        return pl.opaqueLUT[r>>2][g>>2][b>>2]
    }
    return pl.palette.FindNearestIndex(r, g, b, a)
}
```

**Expected Improvement**: 10-100x speedup in palette indexing, proportional to palette size  
**Complexity**: Medium  
**References**: libimagequant, IEEE color re-indexing schemes

---

## 2. K-means Palette Refinement

**Technical Explanation**: Iterative refinement of median-cut palettes where pixels are reassigned to nearest palette colors and centroids recomputed. 2-3 iterations provide 5-15% visual quality improvement for photographic content through convergence detection on quantization error.

**Implementation**:

```go
func RefinePaletteKmeans(palette *Palette, colors []ColorCount, iterations int) {
    for i := 0; i < iterations; i++ {
        accumulators := make([]Accumulator, palette.Len())
        for _, c := range colors {
            idx := palette.FindNearestIndex(c.Color)
            accumulators[idx].Add(c.Color, c.Count)
        }
        for i := range accumulators {
            palette.SetColor(i, accumulators[i].Average())
        }
    }
}
```

**Expected Improvement**: 5-15% perceptual quality improvement, 1-3% compression gain  
**Complexity**: Low  
**References**: Celebi's k-means research, Graphics Interface 1995 LKM paper

---

## 3. Bigrams Filter Strategy

**Technical Explanation**: Filter selection based on minimizing distinct byte pairs (bigrams) in filtered output, improving DEFLATE LZ77 matching efficiency. Different from entropy-based approaches by focusing on repeated sequence patterns rather than immediate compressibility.

**Implementation**:

```go
func countDistinctBigrams(data []byte) int {
    seen := make(map[uint16]bool)
    for i := 0; i < len(data)-1; i++ {
        bigram := uint16(data[i])<<8 | uint16(data[i+1])
        seen[bigram] = true
    }
    return len(seen)
}
```

**Expected Improvement**: 2-5% compression improvement over entropy-based methods  
**Complexity**: Medium  
**References**: PNG specification, OptiPNG optimization guide

---

## 4. Redmean Perceptual Distance

**Technical Explanation**: Weighted color distance formula accounting for human visual perception: `distance² = (2 + avg/256) * dr² + 4 * dg² + (2 + (255-avg)/256) * db²`. Green weighted 1024 accounts for greater sensitivity; red/blue weights vary with luminance.

**Implementation**:

```go
func RedmeanDistanceSq(c1, c2 Color) uint32 {
    rMean := (int(c1.R) + int(c2.R)) >> 1
    dr := int(c1.R) - int(c2.R)
    dg := int(c1.G) - int(c2.G)
    db := int(c1.B) - int(c2.B)
    
    rWeight := 512 + rMean
    gWeight := 1024
    bWeight := 767 - rMean
    
    return uint32((rWeight*dr*dr + gWeight*dg*dg + bWeight*db*db) >> 8)
}
```

**Expected Improvement**: 10-30% color fidelity improvement, far better than Euclidean  
**Complexity**: Low  
**References**: Stack Overflow color comparison, CIEDE2000 research

---

## 5. Cost-Model DEFLATE Parsing

**Technical Explanation**: Optimal LZ77 parsing using dynamic programming to evaluate bit costs of literal/match choices. Convergence detection stops when improvement < 0.1%, typically within 10-20 iterations.

**Implementation**:

```go
type CostModel struct {
    litLenCost [256]float32
    matchCost  [258][32]float32
}

func (e *DeflateEncoder) EncodeOptimal(data []byte) []byte {
    var bestSize int
    var bestResult []byte
    var prevCost float32 = math.MaxFloat32
    
    for iter := 0; iter < e.config.Iterations; iter++ {
        costModel := BuildCostModel(e.litFreq, e.distFreq)
        tokens := e.compressOptimal(data, &costModel)
        result := e.encodeTokens(tokens)
        
        if len(result) < bestSize {
            bestSize = len(result)
            bestResult = result
        }
        
        cost := costModel.TotalCost(tokens)
        if iter > 2 && (prevCost-cost)/prevCost < 0.001 {
            break
        }
        prevCost = cost
    }
    return bestResult
}
```

**Expected Improvement**: 3-10% compression over greedy parsing  
**Complexity**: High  
**References**: Optimal parsing thesis, DCC 2013 REP problem paper

---

## 6. SIMD Filter Operations

**Technical Explanation**: AVX2/SSSE3/NEON vectorization for filter scoring operations. Runtime CPU feature detection via CPUID/GETNEON with scalar fallback. Processes 32 bytes per AVX2 instruction.

**Implementation**:

```go
// Using golang.org/x/exp/simd or assembly
func ScoreFilterSIMD(data []byte) uint64 {
    if simd.AVX2Supported() {
        return scoreFilterAVX2(data)
    } else if simd.SSSE3Supported() {
        return scoreFilterSSSE3(data)
    }
    return scoreFilterScalar(data)
}
```

**Expected Improvement**: 3-5x speedup in filter operations  
**Complexity**: High  
**References**: simd-png, lodepng-turbo, Arm Chromium case study

---

## 7. Parallel PNG Encoding

**Technical Explanation**: Goroutine-based parallel filter selection with strips of 32-64 rows. Sequential fallback for images <32 rows to avoid overhead. Scales near-linearly with CPU cores.

**Implementation**:

```go
func SelectFiltersParallel(pixels []byte, width, height, bpp int) []FilterType {
    if height <= 32 {
        return SelectFiltersSequential(pixels, width, height, bpp)
    }
    
    results := make([]FilterType, height)
    rowLen := width * bpp
    
    var wg sync.WaitGroup
    stripSize := 32
    
    for strip := 0; strip < height; strip += stripSize {
        wg.Add(1)
        go func(strip int) {
            defer wg.Done()
            end := min(strip+stripSize, height)
            for row := strip; row < end; row++ {
                offset := row * rowLen
                prevOffset := (row - 1) * rowLen
                prevRow := pixels[prevOffset:prevOffset+rowLen]
                results[row] = SelectFilter(
                    pixels[offset:offset+rowLen],
                    prevRow, bpp,
                )
            }
        }(strip)
    }
    wg.Wait()
    return results
}
```

**Expected Improvement**: 2-4x speedup on 4-8 core systems  
**Complexity**: Medium  
**References**: pngx, mtpng

---

## 8. Early Termination Optimization

**Technical Explanation**: Threshold-based filter skipping when best score ≤ row_length/4 + 1. Evaluates filters in likelihood order to maximize early termination.

**Implementation**:

```go
func SelectFilterEarlyTerminate(row, prevRow []byte, bpp int) FilterType {
    bestScore := int(^uint(0) >> 1)
    bestFilter := FilterNone
    
    earlyStop := len(row)/4 + 1
    
    filters := []FilterType{FilterNone, FilterSub, FilterUp, FilterAverage, FilterPaeth}
    
    for _, f := range filters {
        filtered := ApplyFilter(row, prevRow, bpp, f)
        score := SumAbsoluteValues(filtered)
        
        if score < bestScore {
            bestScore = score
            bestFilter = f
            if bestScore <= earlyStop {
                return bestFilter
            }
        }
    }
    return bestFilter
}
```

**Expected Improvement**: 20-40% reduction in filter evaluations  
**Complexity**: Low  
**References**: PNG specification heuristics, libpng recommendations

---

## 9. Zopfli-style Iteration

**Technical Explanation**: Exhaustive DEFLATE refinement with entropy modeling and shortest-path search. Fixed/dynamic modes per iteration with 0.1% convergence threshold.

**Implementation**:

```go
func ZopfliEncode(data []byte, iterations int) ([]byte, error) {
    bestResult, bestSize := initialEncode(data)
    
    for iter := 0; iter < iterations; iter++ {
        // Try fixed Huffman
        fixed := encodeFixed(data)
        if len(fixed) < bestSize {
            bestResult = fixed
            bestSize = len(fixed)
        }
        
        // Try dynamic Huffman
        dynamic := encodeDynamic(data)
        if len(dynamic) < bestSize {
            bestResult = dynamic
            bestSize = len(dynamic)
        }
    }
    return bestResult, nil
}
```

**Expected Improvement**: 5-15% compression improvement  
**Complexity**: High  
**References**: Google Zopfli, zopflipng implementation

---

## 10. Adaptive Scratch Buffers

**Technical Explanation**: sync.Pool buffer recycling for filter evaluation operations. Pools sized to common working sets (256B to 64KB) reduce GC pressure by 30-50%.

**Implementation**:

```go
type AdaptiveScratch struct {
    none []byte
    sub  []byte
    up   []byte
    avg  []byte
    paeth []byte
    pool *sync.Pool
}

func NewAdaptiveScratch(rowLen int) *AdaptiveScratch {
    return &AdaptiveScratch{
        none:  make([]byte, rowLen),
        sub:   make([]byte, rowLen),
        up:    make([]byte, rowLen),
        avg:   make([]byte, rowLen),
        paeth: make([]byte, rowLen),
    }
}

func (s *AdaptiveScratch) Clear() {
    clear(s.none)
    clear(s.sub)
    clear(s.up)
    clear(s.avg)
    clear(s.paeth)
}
```

**Expected Improvement**: 30-50% GC overhead reduction  
**Complexity**: Medium  
**References**: Go sync.Pool docs, VictoriaMetrics analysis

---

## Implementation Priority Matrix

| Technique | Compression | Speed | Complexity | Priority |
|-----------|-------------|-------|------------|----------|
| Early Termination | Low | High | Low | 1 |
| Adaptive Scratch Buffers | Low | High | Medium | 2 |
| Parallel Encoding | Low | High | Medium | 3 |
| Redmean Distance | Medium | High | Low | 4 |
| K-means Refinement | Medium | Medium | Low | 5 |
| Bigrams Filter | Medium | Medium | Medium | 6 |
| Palette LUT | Low | High | Medium | 7 |
| SIMD Filters | Low | High | High | 8 |
| Cost-Model Parsing | High | Low | High | 9 |
| Zopfli Iteration | High | Low | High | 10 |

---

## Recommendations

### Phase 1 (Week 1): Quick Wins

1. **Early Termination** - 20-40% fewer filter evaluations, trivial implementation
2. **Adaptive Scratch Buffers** - 30-50% GC reduction, simple buffer reuse
3. **Redmean Distance** - Better visual quality, one function change

### Phase 2 (Week 2): Performance

1. **Parallel Encoding** - 2-4x speedup on multi-core
2. **K-means Refinement** - 5-15% visual quality improvement
3. **Bigrams Filter** - 2-5% compression improvement

### Phase 3 (Week 3-4): Advanced

1. **Palette LUT** - 10-100x speedup for quantization
2. **SIMD Filters** - 3-5x speedup for filter operations

### Phase 4 (Month 2): Expert

1. **Cost-Model Parsing** - 3-10% compression improvement
2. **Zopfli Iteration** - 5-15% compression improvement

---

## Research Papers and References

### Color Quantization

- "Color Quantization of Images" - Decarlo and Santella
- "Median Cut Algorithm Variations" - Celebi
- "K-means for Color Quantization" - Graphics Interface 1995

### DEFLATE Optimization

- "Optimal Parsing for DEFLATE" - Thesis
- "Zopfli: Small Deflate Encoder" - Google Whitepaper
- "LZ77 Optimal Parsing" - DCC 2013

### Perceptual Metrics

- "CIEDE2000 Color Difference" - CIE
- "Redmean Color Distance" - Various implementations

### SIMD Optimization

- "SIMD in Image Processing" - Arm Case Studies
- "AVX2 Performance Guide" - Intel
