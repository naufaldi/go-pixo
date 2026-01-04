# Palette Quantization and Dithering in go-pixo

This document explains the lossy compression pipeline in go-pixo, including palette quantization and dithering algorithms.

## Overview

PNG compression can be either **lossless** or **lossy**:

| Mode | Quality | Size Reduction | Use Case |
|------|---------|----------------|----------|
| Lossless | 100% | 50-70% | Graphics, screenshots, text |
| Lossy | 70-95% | 70-99% | Photos, gradients, large images |

Lossy compression reduces file size by reducing the number of colors (quantization) and optionally applying dithering to smooth the visual impact.

## When to Use Palette Quantization

Palette quantization works best when:

- Image has limited color palette (< 256 colors)
- Original is 24-bit or 32-bit (RGB/RGBA)
- Size reduction is more important than perfect quality
- Converting logos, icons, or graphics with few colors

Quantization is NOT recommended for:

- Photographs with millions of colors
- Images with smooth gradients
- Medical or scientific imaging

## Median Cut Algorithm

go-pixo uses the **median cut algorithm** for palette generation:

1. **Color Collection**: Count all unique colors in the image with their frequencies
2. **Initial Bucket**: Put all colors in a single bucket
3. **Recursive Splitting**: Find the bucket with largest color variance and split it
4. **Repeat**: Continue until reaching target color count
5. **Average**: Calculate average color for each final bucket

### Quality Parameter

The quality parameter (0.0-1.0) controls color accuracy vs palette size:

```go
// Higher quality = more colors used
palette := MedianCutWithQuality(colors, maxColors, quality)

// Quality 1.0: Use all requested colors
// Quality 0.5: Use ~75% of requested colors
// Quality 0.1: Use ~55% of requested colors
```

### Alpha Channel Support

For RGBA images, alpha information is preserved using the tRNS chunk:

```go
// RGBA quantization with alpha
palette := MedianCutRGBA(colors, maxColors)
```

## Dithering Methods

Dithering distributes quantization error to neighboring pixels, creating the illusion of more colors.

### Available Algorithms

| Method | Quality | Speed | Description |
|--------|---------|-------|-------------|
| Floyd-Steinberg | Best | Fast | Standard error diffusion |
| Jarvis-Judice-Ninke | Better | Medium | Spreads error to more pixels |
| Sierra 2-Row | Good | Fast | Faster variant of JJN |
| Stucki | Good | Medium | Clean, professional results |

### Strength Control

Dithering strength (0.0-1.0) controls intensity:

```go
// No dithering - direct palette mapping
dithered := Threshold(pixels, palette)

// Full dithering - maximum error diffusion
dithered := FloydSteinberg(pixels, palette, 1.0)

// Adjustable strength
dithered := FloydSteinbergWithStrength(pixels, palette, 0.5)
```

**Recommendations:**
- 0.0-0.25: Minimal dithering, bands visible
- 0.5: Balanced, recommended default
- 0.75-1.0: Heavy dithering, more texture

## CLI Usage Examples

### Lossless Compression (Default)

```bash
# Fast preset
./pixo -input image.png -preset fast -compare

# Balanced (default)
./pixo -input image.png -preset balanced -compare

# Maximum compression
./pixo -input image.png -preset max -compare
```

### Lossy Compression

```bash
# 256 colors, no dithering
./pixo -input image.png -lossy -max-colors 256 -compare

# 128 colors, medium dithering
./pixo -input image.png -lossy -quality 75 -max-colors 128 -dither 0.5 -compare

# 64 colors, high dithering
./pixo -input image.png -lossy -quality 50 -max-colors 64 -dither 0.75 -compare

# 8 colors for icons
./pixo -input logo.png -lossy -max-colors 8 -dither 0.5 -compare
```

### Extreme Compression with Zopfli

```bash
# Maximum compression, 15 Zopfli iterations
./pixo -input image.png -preset extreme -iterations 15 -compare

# Benchmark mode
./pixo -input image.png -preset extreme -benchmark -benchmark-runs 5
```

## Web UI Controls

The web interface provides the following controls:

### Compression Preset Slider

| Position | Preset | Description |
|----------|--------|-------------|
| Left (0) | Smaller | Maximum compression, slowest |
| Middle (1) | Balanced | Default trade-off |
| Right (2) | Faster | Fastest, larger files |

### Lossy Settings (when Lossless is unchecked)

| Control | Range | Default | Description |
|---------|-------|---------|-------------|
| Colors | 8-256 | 256 | Maximum palette size |
| Dithering | On/Off | Off | Enable error diffusion |
| Dither Strength | 0-100% | 50% | Dithering intensity |
| Quality | 0-100% | 75% | Color accuracy vs size |

### Zopfli Iterations

| Value | Effect |
|-------|--------|
| 0 | No extra optimization (fastest) |
| 5 | Good improvement (~3% better) |
| 15 | Excellent improvement (~5% better) |
| 50 | Maximum improvement (~7% better, slow) |

## Comparison with pngquant

go-pixo's lossy mode is inspired by pngquant but with different trade-offs:

| Feature | go-pixo | pngquant |
|---------|---------|----------|
| Algorithm | Median cut | Median cut + smoothing |
| Dithering | Multiple methods | Floyd-Steinberg only |
| Speed | Fast | Medium |
| Quality | Good | Excellent |
| WASM Support | Yes | No |

### When to Use Each

- **go-pixo**: Browser-based compression, WASM required, faster encoding
- **pngquant**: CLI tools, highest quality, mature toolchain

## Technical Details

### Output Format for Indexed Images

Indexed PNG (color type 3) uses:

1. **PLTE Chunk**: Palette entries (RGB for each color)
2. **tRNS Chunk**: Alpha values for each palette entry (optional)
3. **IDAT Chunk**: Indexed pixel data (1 byte per pixel)

### Memory Usage

| Mode | Memory per Megapixel |
|------|----------------------|
| Lossless RGB | ~12 MB |
| Lossless RGBA | ~16 MB |
| Lossy 256 colors | ~4 MB |
| Lossy 64 colors | ~1 MB |

## Troubleshooting

### Artifacts in Gradients

**Problem**: Banding visible in smooth gradients

**Solution**: Enable dithering with strength 0.5-0.75

### Large File Size

**Problem**: Lossy output is larger than expected

**Solution**: 
- Reduce max colors
- Lower quality target
- Enable Zopfli iterations

### Color Accuracy

**Problem**: Colors look wrong after compression

**Solution**:
- Increase max colors
- Reduce quality target
- Disable dithering

## API Reference

### Go Library

```go
import "github.com/mac/go-pixo/src/png"

// Lossy options
opts := png.LossyOptions(width, height, 128)
opts.QualityTarget = 75
opts.DitheringStrength = 0.5

encoder, _ := png.NewEncoderWithOptions(opts)
output, _ := encoder.Encode(pixels)
```

### WASM Interface

```javascript
// In browser
const result = encodePngAdvanced(
  pixels, width, height, colorType,
  preset, lossy, maxColors,
  dithering, ditherStrength,
  qualityTarget, zopfliIterations
);
```
