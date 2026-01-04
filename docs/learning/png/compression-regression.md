# PNG Compression Regression: Understanding Why Re-compression Can Produce Larger Files

This document explains the technical reasons why re-compressing a PNG image can sometimes result in a larger file size, using `cursor-meetup.png` as a case study.

## Case Study: cursor-meetup.png

### Baseline Information
- **Original file**: `cursor-meetup.png` (727 KB / 744,704 bytes)
- **Image dimensions**: 1920x1080 pixels
- **Color type**: RGBA (32-bit)
- **Target compression**: Match or beat original size

### The Paradox

When attempting to optimize `cursor-meetup.png`, we observed that naive re-compression approaches often produce files that are **larger** than the original. This contradicts the typical expectation that optimization tools should always reduce file size.

## Root Causes

### 1. Filter Scoring Limitations

The PNG format uses filter bytes (one per scanline) to improve compressibility. The filter type is chosen per-row to maximize compression.

**Problem**: Traditional filter selection uses "sum of absolute values" (SAV) as a heuristic:

```
SAV = sum(|pixel - predictor|)
```

This metric doesn't always correlate with actual compressibility after DEFLATE encoding.

**Example**:
- Filter type 0 (None): SAV = 0 for uniform regions
- Filter type 4 (Paeth): May have higher SAV but produces better DEFLATE compression

**Our Solution**: Implement entropy-based filter scoring that considers the actual compressibility of the filtered data.

### 2. DEFLATE Iteration Requirements

DEFLATE uses LZ77 (sliding window) + Huffman coding. The standard single-pass greedy matching often misses optimal match opportunities.

**Problem**: A single pass through the data finds matches greedily but doesn't explore alternatives:

```
Input: A B C A B C A B C A B C

Greedy pass 1: Match "A B C" at positions 0 and 3
Greedy pass 2: Match "A B C" at positions 6 and 9

Optimal: Match "A B C A B C A B C" (length 9) + "A B C" (length 3)
```

**Our Solution**: Zopfli-style iterative optimization that:
- Makes multiple passes with different match strategies
- Evaluates cost of different encoding approaches
- Selects the configuration with best compression

### 3. Palette Optimization Complexity

For images with many similar colors, palette quantization can actually increase size if not done carefully.

**Problem**: When reducing to fewer colors:
1. Palette size reduces (good)
2. But dithering patterns add entropy (bad)
3. Quantization errors create additional color variations

**Example**:
- 256-color palette: 1 byte per pixel = 100 KB for palette + index data
- 128-color palette: 1 byte per pixel (still) + dithering = often larger

**Our Solution**: Quality-aware median cut with configurable accuracy trade-offs.

### 4. IDAT Chunk Structure

The IDAT chunk contains the compressed image data. Its structure affects compressibility.

**Problem**: The DEFLATE stream within IDAT has internal block boundaries that can interfere with optimal compression:

```
Block 1: Compressed with fixed Huffman tables
Block 2: Compressed with dynamic Huffman tables
...
```

Suboptimal block splitting can reduce compression ratio.

**Our Solution**: Optimize block splitting during DEFLATE encoding to maximize overall compression.

## Technical Comparison with Reference Tools

### Oxipng (Rust)

Uses the following strategies:
- Multiple filter strategies with full brute force for small images
- Zopfli-inspired iterative DEFLATE optimization
- Image transformation optimizations (sub8 bit depth reduction when applicable)

**Performance on cursor-meetup.png**: Typically achieves 3-8% reduction on well-compressed images.

### Optipng (C)

More conservative approach:
- Uses fixed optimization levels
- Limited iteration count for DEFLATE
- Focuses on correctness over aggressive optimization

**Performance on cursor-meetup.png**: Typically achieves 1-4% reduction.

### pngquant (C)

Specializes in lossy color quantization:
- High-quality median cut with weighting
- Advanced dithering algorithms
- Optional Floyd-Steinberg error diffusion

**Performance on cursor-meetup.png**: Can achieve 50-80% reduction when reducing to 128-256 colors.

## Our Implementation Strategy

### Phase 1: Filter Optimization

1. **Entropy-based scoring**: Replace SAV with estimated entropy
2. **Brute force for small images**: Try all combinations for images < 256x256
3. **Per-row optimization**: Select best filter for each row individually

### Phase 2: DEFLATE Enhancement

1. **Zopfli iterations**: 5-15 iterations for maximum compression
2. **Block splitting optimization**: Find optimal block boundaries
3. **Fixed vs Dynamic**: Auto-select based on actual data characteristics

### Phase 3: Palette Intelligence

1. **Quality parameter**: Control color accuracy vs size trade-off
2. **Alpha handling**: Special treatment for transparent regions
3. **Dithering strength**: Configurable from 0 (none) to 1.0 (maximum)

## Expected Improvements

### Conservative Goals (Phase 1-2)
- Match original size or achieve < 1% reduction
- Processing time: < 5 seconds for typical images

### Moderate Goals (Phase 1-3)
- 2-5% reduction in file size
- Processing time: < 30 seconds for typical images

### Aggressive Goals (Extreme Preset)
- 5-15% reduction in file size
- Processing time: 1-5 minutes for typical images
- Requires: High iteration counts, brute force filters

## Diagnostic Process

When compression produces larger files, analyze:

1. **Compare filter selection**: What filters did we choose vs. optimal?
2. **Analyze match opportunities**: How many LZ77 matches did we find?
3. **Evaluate Huffman tables**: Are fixed tables sufficient?
4. **Check IDAT structure**: Are there many small blocks?

## Recommendations

### For Users
1. **Start with Balanced preset**: Good balance of speed and compression
2. **Use Extreme preset** only when file size is critical
3. **Try lossy mode** for images with many similar colors
4. **Use Extreme preset with iterations** for maximum savings

### For Developers
1. Implement entropy-based filter scoring before brute force
2. Start with 5 Zopfli iterations, increase for critical cases
3. Profile filter selection time vs. compression gain
4. Consider image dimensions when choosing optimization strategy

## References

- [RFC 1951](https://tools.ietf.org/html/rfc1951) - DEFLATE specification
- [PNG Specification](http://www.w3.org/TR/PNG/) - PNG format details
- [Zopfli Whitepaper](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/43832.pdf) - Iterative DEFLATE optimization
- [Oxipng GitHub](https://github.com/shssoichiro/oxipng) - Modern PNG optimizer reference implementation

## Conclusion

PNG compression regression is a well-understood phenomenon with multiple root causes. By implementing a multi-phase optimization strategy with configurable parameters, we can achieve significant compression improvements while giving users control over the speed vs. compression trade-off.

The key insight is that **more computation is not always better** - the optimal approach depends on the specific image characteristics. Our preset system allows users to choose the appropriate level of optimization for their needs.
