# Advanced PNG Compression Techniques

This document analyzes why re-compressing PNG images can sometimes result in larger files, and explores advanced compression techniques that can be implemented to achieve better optimization.

## The Problem: Re-compression Size Increase

When we attempted to re-compress `cursor-meetup.png`, we encountered an unexpected result:

| Metric                     | Value   |
| -------------------------- | ------- |
| Original file size         | 727 KB  |
| Raw pixel data             | 14.3 MB |
| Our compressed output      | 1.04 MB |
| Compression ratio (vs raw) | 7.30%   |

While our output achieves **93% reduction** from raw pixels, it's **43% larger** than the original file. This reveals a fundamental challenge in PNG compression.

### Why This Happens

The original PNG file was created with optimizations that our current encoder doesn't support:

1. **Pre-optimized DEFLATE**: The original may have used Zopfli or similar iterative compression that finds better Huffman tables
2. **Better filter selection**: Original may have tried all 5 filter types per row with different strategies
3. **Color type optimization**: The original may have detected and used palette mode (PNG8) for images with limited colors
4. **Bit depth reduction**: Original may have reduced from 8-bit to 4-bit or less for grayscale images

## Advanced Compression Techniques

### 1. Zopfli-style Iterative Compression

Zopfli is a DEFLATE encoder that produces 3-8% smaller output than standard zlib by:

- Iteratively trying many Huffman table configurations
- Using a cost model to evaluate compression efficiency
- Taking much longer but producing better results

**Implementation approach**: Modify `deflate_encoder.go` to implement iterative refinement similar to `EncodeOptimal()`, but with better heuristics for trying different table configurations.

**Reference**: [Zopfli Optimization: Literally Free Bandwidth](https://blog.codinghorror.com/zopfli-optimization-literally-free-bandwidth/)

### 2. OxiPNG-style Multi-Pass Filter Optimization

OxiPNG tries all filter types (0-4) for each row and selects the optimal combination:

```bash
# OxiPNG usage example
oxipng --filters 0me --optimize image.png
```

- **0**: No filter
- **m**: Minimum sum filtering
- **e**: Entropy filtering
- **b**: Brute force (try all)

**Implementation approach**: Extend `filter_selector.go` to:

1. Try all 5 filter types per row (currently does this)
2. Use entropy-based scoring in addition to sum of absolute values
3. Consider global optimization across rows

**Reference**: [oxipng GitHub](https://github.com/oxipng/oxipng)

### 3. Color Type and Bit Depth Reduction

OptiPNG performs these optimizations:

| Transformation    | Description                            |
| ----------------- | -------------------------------------- |
| RGBA → RGB        | Remove alpha channel if fully opaque   |
| RGB → Grayscale   | Convert if R=G=B for all pixels        |
| 16-bit → 8-bit    | Reduce bit depth if values fit         |
| Palette reduction | Reduce to 256 colors with quantization |

**Implementation approach**: Already partially implemented in `color_reduce.go` and `color_analysis.go`. Enhancement needed:

- Better palette generation using median cut
- Dithering support for smooth gradients
- Automatic detection of reduction opportunities

**Reference**: [OptiPNG Manual](https://optipng.sourceforge.net/optipng-7.9.1.man1.html)

### 4. Lossy PNG Compression (pngquant)

For images where some quality loss is acceptable, pngquant provides significant size reduction:

```bash
# pngquant usage
pngquant --quality=70-90 image.png
```

- Reduces 24/32-bit images to 8-bit indexed color (PNG8)
- Uses advanced quantization algorithms (median cut with dithering)
- Typically achieves 60-70% size reduction

**Trade-off**: Lossy compression means some quality loss

**Reference**: [pngquant](https://pngquant.org/)

### 5. Palette Optimization (PLTE/tRNS Chunks)

For images with ≤256 unique colors, palette mode can dramatically reduce size:

| Mode           | Bytes/Pixel | Best For                     |
| -------------- | ----------- | ---------------------------- |
| RGBA           | 4           | Photos, gradients            |
| RGB            | 3           | Opaque images                |
| Indexed (PNG8) | 1           | Graphics, icons, screenshots |

**Implementation approach**: Already implemented in Phase 5 (quantization). Enhancement needed:

- Better quality palette generation
- Alpha channel support (tRNS chunk)
- Automatic detection of palette suitability

### 6. Stored Block Optimization

Currently implemented as fallback when DEFLATE doesn't help:

- Stored blocks add only 5 bytes overhead
- Useful for already-compressed data (JPEG artifacts, random noise)
- Must not be used when DEFLATE provides savings

## Roadmap for Implementation

### Phase 1: Quick Wins (Easy to implement)

1. **Better filter scoring**: Use entropy-based scoring in addition to sum of absolute values
2. **Improved Zopfli iteration**: Increase iterations in `EncodeOptimal()` with better heuristics
3. **Better palette detection**: Enhanced color reduction analysis

### Phase 2: Medium Effort

1. **Advanced filter strategies**: Implement OxiPNG-style brute force for small images
2. **Dithering support**: Add Floyd-Steinberg dithering for palette reduction
3. **Parallel compression**: Try multiple strategies in parallel

### Phase 3: Advanced (Complex)

1. **Full Zopfli implementation**: Complete rewrite of DEFLATE with iterative optimization
2. **WASM optimization**: Ensure all advanced features work in browser
3. **Machine learning**: Use ML to predict optimal compression parameters

## Comparison with Existing Tools

| Tool          | Type     | Compression | Speed  | Language |
| ------------- | -------- | ----------- | ------ | -------- |
| **go-pixo**   | Lossless | Good        | Fast   | Go       |
| **OptiPNG**   | Lossless | Better      | Medium | C        |
| **OxiPNG**    | Lossless | Best        | Fast   | Rust     |
| **pngquant**  | Lossy    | Excellent   | Medium | C        |
| **zopflipng** | Lossless | Better      | Slow   | C++      |

## Recommendations

### For go-pixo Improvement Priority

1. **Immediate**: Improve filter selection with entropy scoring
2. **Short-term**: Better palette quantization (median cut enhancement)
3. **Medium-term**: Zopfli-style iterative compression
4. **Long-term**: Full OxiPNG feature parity

### Alternative: Lossy Mode

Consider adding lossy compression option:

- Use pngquant-style palette quantization
- Quality parameter (0-100)
- Automatic vs manual mode
- WebAssembly compatible

## Conclusion

The cursor-meetup.png case reveals that PNG compression is a complex optimization problem. The original file was pre-optimized with techniques that go-pixo doesn't yet implement.

To achieve competitive compression ratios, we need to implement:

1. Better filter optimization (entropy-based, brute force for small images)
2. Improved palette generation (median cut, dithering)
3. Iterative DEFLATE optimization (Zopfli-style)
4. Optional lossy mode for maximum compression

## References

- [OxiPNG: Multithreaded PNG Optimizer](https://github.com/oxipng/oxipng)
- [OptiPNG PNG Optimizer](https://optipng.sourceforge.net/)
- [pngquant: Lossy PNG Compressor](https://pngquant.org/)
- [Zopfli Optimization](https://blog.codinghorror.com/zopfli-optimization-literally-free-bandwidth/)
- [Everything You Need to Know About PNG](https://convertft.com/blogs/everything-you-need-to-know-about-png-features-compression-and-optimization-tips)
- [A Guide to PNG Optimization - OptiPNG](https://optipng.sourceforge.net/pngtech/optipng.html)
