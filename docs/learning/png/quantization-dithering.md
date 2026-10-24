# Palette Quantization and Dithering in go-pixo

This document explains the lossy compression pipeline in go-pixo, including palette quantization and dithering algorithms.

## Overview

PNG compression can be either **lossless** or **lossy**:

| Mode | Pixel behavior | Size behavior |
|------|----------------|---------------|
| Lossless | Preserves every sample, including RGB under zero alpha | Byte recompression returns the original when the candidate is larger |
| Palette quantization | May change RGB colors | Savings depend on the image and palette overhead |

Lossy compression reduces file size by reducing the number of colors (quantization) and optionally applying dithering to smooth the visual impact.

## Encoder option contract

`Encoder.EncodeWithOptions` uses `QuantizeWithOptions` when `MaxColors` is 1..256.
`EncodePngAdvanced` exposes the same settings; existing argument positions are unchanged.
These controls have distinct meanings:

- `MaxColors` is a hard palette cap. Zero disables quantization.
- `QualityTarget` (0..100) drives the existing median-cut heuristic. When the input
  exceeds the cap, it uses roughly `MaxColors * (0.5 + QualityTarget/200)` entries
  and adjusts bucket splits. The one-color cap always stays one. Inputs already
  within the cap retain their exact RGB colors, regardless of quality.
- Quality 100 uses the full cap. It still permits lossy quantization. This is not
  a measured fidelity target, a minimum-quality guarantee, or JPEG's quality scale.
  There is no extra quality reduction to meet a file-size target, including for
  tiny images.
- Empty `Options{}` retains the legacy full-quality helper behavior. To request
  quality zero in a Go struct literal, set `QualityTargetSet: true`. Presets and
  `ApplyLossy` set this presence flag automatically. Nonzero quality values always
  take effect. The WASM argument zero explicitly requests the lowest quality.
- `UsePerceptualDistance=true` or `DistanceMetricRedmean` selects Redmean instead of
  Euclidean distance for palette indices. It affects both direct and dithered
  lookups; it does not change how median cut generates the palette.
- `Dithering` enables horizontal error diffusion. `DitheringStrength` scales the
  signed error sent to the next pixel by `7/16`, from zero to full strength. Error
  resets at each row. This path is not full two-dimensional Floyd-Steinberg.

The production mapper uses exact linear palette searches, at most 256 entries per
pixel. Diffusion carries three integers and allocates no image-sized scratch or
full RGB lookup table. Color counting still uses the existing histogram. The
optional LUT argument on standalone quantizer helpers retains its legacy behavior.

The current RGB palette cannot represent alpha. Palette requests reject grayscale
and non-opaque RGBA inputs instead of panicking or silently discarding transparency.
Both public PNG encode entry points disable quantization, dithering, and hidden-RGB
optimization when `lossy=false`. `RecompressPNGBytesLossless` also overrides lossy
template settings.

### Automatic compression versus a required palette

`EncodePng(..., lossy=true, maxColors=256)` treats the full default palette budget
as opportunistic. For non-opaque RGBA it compresses the original RGBA samples,
including hidden RGB, instead of requiring an unsupported RGB palette. Filters,
compression level, and DEFLATE effort still apply. Opaque input still uses the
selected quantizer; a reduced budget (1..255) always requires a palette.

Explicit low-level `Options.MaxColors` requests remain strict even at 256 colors.
Basic Go/JS callers can also require a full palette by passing `false` as the new
optional final `allowRGBAFallback` argument. Advanced calls remain strict by
default, so explicit quality, dithering, and distance requests are not silently
bypassed. An advanced caller can opt into the same fallback by appending `true`
after the progress callback (JS argument 18). The progress callback stays at JS
argument 17. Existing calls do not need new arguments.

The app marks its preset-derived worker requests with `pngAllowRGBAFallback=true`.
This records compression intent without identifying a preset by its numeric values.
For transparent full-budget input, palette-only settings are inapplicable and RGBA
samples are preserved. Reduced budgets, including the Ultra preset's 128 colors,
continue to reject transparency. Custom worker requests omit the flag to require
quantization. No alpha-aware palette or dithering implementation is implied.

This raw-pixel fallback does not guarantee smaller files. The worker still compares
successful same-format candidates with the original bytes and returns the original
when it is no larger. Resized images must use the encoded candidate.

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

The encoder's palette path supports opaque RGBA only. The standalone RGBA palette
generation helpers below are not an alpha-preserving encoder pipeline:

```go
// RGBA quantization with alpha
palette := MedianCutRGBA(colors, maxColors)
```

## Dithering Methods

Dithering distributes quantization error to neighboring pixels, creating the illusion of more colors.

### Standalone helpers

These helpers are separate from the production option path described above:

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
