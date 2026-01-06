# JPEG Encoding Documentation

Welcome to the JPEG learning materials. This directory contains deep dives into the implementation of the `go-pixo` JPEG encoder.

## Topics

- [Chroma Subsampling](subsampling.md) - How we reduce color data to save space.
- [Optimized Huffman Tables](huffman-optimized.md) - Custom entropy coding based on image content.
- [Progressive JPEG](progressive.md) - Multiple scans for better compression and web delivery.
- [Trellis Quantization](trellis.md) - Rate-distortion optimization for better compression.
- [Encoder Options](options.md) - Configuring the encoder with presets and builders.

## JPEG Pipeline Overview

The JPEG encoder follows these steps:

1. **Color Conversion**: Convert RGB pixels to YCbCr (Luminance, Blue-Difference, Red-Difference).
2. **Chroma Subsampling**: Optionally downsample Cb and Cr components (e.g., 4:2:0).
3. **Block Splitting**: Divide the image into 8x8 blocks (or 16x16 MCUs for 4:2:0).
4. **Forward DCT**: Apply Discrete Cosine Transform to each block to convert spatial data to frequency data.
5. **Quantization**: Divide DCT coefficients by a quantization table and round to the nearest integer. This is the primary source of quality loss.
6. **Zigzag Reordering**: Reorder coefficients in a zigzag pattern to group zeros together.
7. **Entropy Coding**: Use Huffman coding to compress the coefficients.
8. **Marker Writing**: Wrap the compressed data in standard JPEG markers (SOI, DQT, SOF, DHT, SOS, EOI).

## Comparison with PNG

Unlike PNG, which is lossless, JPEG is **lossy**. It is optimized for photographs where exact pixel values are less important than visual perception.

| Feature | PNG | JPEG |
| --- | --- | --- |
| Compression | Lossless (DEFLATE) | Lossy (DCT + Quantization) |
| Transparency | Supported (Alpha channel) | Not supported |
| Best For | Text, Icons, Graphics | Photographs, Natural scenes |
| Speed | Fast to Slow (Zopfli) | Generally very fast |
| Compression Ratio | Lower | Higher (10:1 or more) |
