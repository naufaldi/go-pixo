# PNG Learning Resources

Comprehensive documentation for understanding PNG encoding and compression.

---

## Getting Started

- [PNG Overview](png.md) - Introduction to PNG format and go-pixo implementation
- [PNG Encoder](encoder.md) - How the encoder works end-to-end
- [Scanlines](scanlines.md) - How image data is organized into scanlines

---

## Filter System

Filters are applied to each scanline to improve compressibility.

- [PNG Filters](filters.md) - Filter types and how they work
- [Filter Selection](filter-selection.md) - Choosing the best filter per row
- [Entropy-Based Filtering](entropy-filtering.md) - Advanced entropy scoring algorithm

---

## Compression Pipeline

Understanding the DEFLATE/zlib compression stack.

- [Compression Overview](advanced-compression.md) - Full compression optimization guide
- [Zlib Integration](zlib.md) - zlib header/footer handling
- [DEFLATE Algorithm](deflate.md) - LZ77 + Huffman coding explanation
- [IDAT and Zlib](idat-zlib-integration.md) - Image data chunk integration

---

## DEFLATE Implementation

Detailed documentation of the DEFLATE encoder.

- [DEFLATE Encoder](deflate-encoder.md) - Core encoder implementation
- [DEFLATE Block Writer](deflate-block-writer.md) - Block type handling
- [Stored Blocks](stored-blocks.md) - Uncompressed block handling

---

## Quantization and Palette

Reducing colors for smaller file sizes.

- [Quantization](quantization.md) - Color quantization overview
- [Palette Quantization and Dithering](quantization-dithering.md) - Lossy compression guide
- [Median Cut Algorithm](quantization.md#median-cut) - Palette generation algorithm

---

## Advanced Topics

- [PNG Infrastructure](png-infra.md) - Core data structures
- [Rust PNG Implementation](rust-png.md) - Reference implementation insights

---

## Quick Reference

### Filter Types

| Filter | Description | Best For |
|--------|-------------|----------|
| 0 (None) | No transformation | Already compressible data |
| 1 (Sub) | Predict from left pixel | Horizontal gradients |
| 2 (Up) | Predict from above pixel | Vertical patterns |
| 3 (Average) | Average of left and above | Diagonal patterns |
| 4 (Paeth) | Linear combination | General purpose |

### Color Types

| Color Type | Description | Bytes/Pixel |
|------------|-------------|-------------|
| 0 | Grayscale | 1 |
| 2 | RGB | 3 |
| 3 | Palette | 1 (index) |
| 4 | Grayscale + Alpha | 2 |
| 6 | RGBA | 4 |

### Compression Strategies

| Strategy | Speed | Compression | Use Case |
|----------|-------|-------------|----------|
| Fast | Fastest | Good | Real-time encoding |
| Balanced | Medium | Better | General use |
| Max | Slow | Best | Offline compression |
| Extreme | Slowest | Maximum | Archival |

---

## Related Documentation

- [API Documentation](../api.md) - Complete function reference
- [CLI Usage](../../CLI.md) - Command-line interface guide
- [WASM Integration](../../web/README.md) - Browser-based compression
