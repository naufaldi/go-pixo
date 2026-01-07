# Go-pixo vs pixo: Comprehensive Comparison and Optimization Guide

This document provides an overview of the detailed comparison between the Go implementation (go-pixo) and Rust implementation (pixo) of PNG and JPEG compression algorithms. The goal is to understand the architectural differences, performance characteristics, and quality trade-offs between the two implementations, ultimately providing actionable recommendations to optimize the Go implementation.

## Quick Navigation

Choose the topic that interests you:

| Topic | Description | File |
|-------|-------------|------|
| **PNG Comparison** | Detailed PNG filter selection, quantization, and performance analysis | [Read PNG comparison](./diff-png.md) |
| **JPEG Comparison** | Detailed JPEG DCT, quantization, Huffman, trellis, and progressive encoding | [Read JPEG comparison](./diff-jpeg.md) |
| **Optimization Guide** | Actionable recommendations to optimize Go implementation | [View optimization guide](./optimization-guide.md) |
| **Research Papers** | Curated list of research papers and references for learning | [Browse papers](./research-papers.md) |

## Summary Comparison

### PNG Implementation

| Aspect | Go-pixo | pixo | Priority |
|--------|---------|------|----------|
| Filter strategies | 10 (including Entropy, BruteForce) | 9 (including unique Bigrams) | Medium |
| K-means refinement | No | Yes | High |
| Palette LUT | No | Yes (O(1) lookup) | High |
| SIMD acceleration | No | Yes (AVX2/SSSE3/NEON) | High |
| Parallel processing | No | Yes (Rayon) | Medium |
| Early termination | No | Yes | Low |
| Perceptual distance | Euclidean | Redmean | Low |

**Key findings**:
- pixo's Bigrams strategy provides 2-5% better compression
- Palette LUT provides 10-100x speedup for quantization
- K-means refinement improves visual quality by 5-15% for photographic content

### JPEG Implementation

| Aspect | Go-pixo | pixo | Priority |
|--------|---------|------|----------|
| DCT | 3.5KB, floating-point | 42KB, SIMD integer | High |
| Quantization | Basic | Enhanced + RDO | Medium |
| Huffman | 5KB, basic | 26KB, optimized | Medium |
| Trellis | 4.5KB, basic | 19KB, complete | High |
| Progressive | 6.5KB, basic | 29KB, optimized | Low |

**Key findings**:
- pixo's SIMD DCT provides 3-5x speedup
- Complete trellis optimization provides 5-10% better quality
- Optimized Huffman reduces encoding time by 10-20%

## Key Performance Differences

### Why pixo is Faster

1. **SIMD Acceleration**: Uses AVX2/SSSE3/SSE2 for filter operations and DCT, processing 256-512 bits per cycle vs 64 bits for Go
2. **Parallel Processing**: Uses Rayon for parallel filter selection on images with >32 rows
3. **Palette LUT**: 256K-entry lookup table provides O(1) palette indexing vs O(n) linear search in Go
4. **Zero-Cost Abstractions**: Rust's iterators and generics compile to optimal machine code
5. **Memory Layout**: Better memory locality through scratch buffer reuse

### Why pixo Has Better Quality

1. **K-means refinement**: Post-median-cut optimization produces better palettes for actual image content
2. **Bigrams filter**: Optimizes for DEFLATE's LZ77 mechanism rather than just raw entropy
3. **Redmean distance**: Better matches human vision for palette assignment
4. **Cost-model parsing**: Optimal LZ77 parsing with convergence detection

## Recommended Reading Order

### For Understanding the Comparison

1. Start with this overview to understand the big picture
2. Read [PNG comparison](./diff-png.md) if you work on PNG encoding
3. Read [JPEG comparison](./diff-jpeg.md) if you work on JPEG encoding

### For Implementation

1. Read [Optimization guide](./optimization-guide.md) for actionable recommendations
2. Check [Research papers](./research-papers.md) for theoretical background
3. Implement Priority 1 optimizations first for maximum impact

## Optimization Priorities

### Priority 1: High-Impact, Medium-Effort

1. **Add Palette LUT** - 10-100x speedup for quantization
2. **Add K-means refinement** - 5-15% better visual quality
3. **Implement Bigrams filter** - 2-5% better compression
4. **Add SIMD for DCT** - 3-5x speedup for JPEG
5. **Optimize Huffman caching** - 10-20% faster JPEG encoding

### Priority 2: Medium-Impact, High-Effort

1. **Add parallel filter selection** - 2-8x speedup (CPU cores)
2. **Implement full trellis** - 5-10% better quality
3. **Add progressive streaming** - 50% memory reduction
4. **Implement cost-model parsing** - 3-8% better compression

### Priority 3: Low-Impact, Low-Effort

1. **Add scratch buffers** - Reduced GC pressure
2. **Add early termination** - 10-30% faster filtering
3. **Switch to Redmean distance** - Better visual quality

## File Structure

```
docs/learning/optimize/
├── diff-rust-go.md        # This file (overview)
├── diff-png.md            # Detailed PNG comparison
├── diff-jpeg.md           # Detailed JPEG comparison
├── optimization-guide.md  # Actionable recommendations
└── research-papers.md     # Research paper references
```

## Related Documentation

- [PNG encoding in go-pixo](../../png/index.md)
- [JPEG encoding in go-pixo](../../jpg/index.md)
- [Compression algorithms](../../compression-evolution.md)
- [Performance optimization](../../performance-optimization.md)

---

## Document Statistics

| Metric | Value |
|--------|-------|
| Total comparison documents | 4 |
| PNG comparison | ~500 lines |
| JPEG comparison | ~400 lines |
| Optimization recommendations | ~400 lines |
| Research papers | ~300 lines |

This modular structure allows you to focus on specific topics without reading through the entire comparison at once.