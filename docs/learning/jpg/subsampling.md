# Chroma Subsampling

Chroma subsampling is a technique that reduces the resolution of color information (Chrominance) while keeping the brightness information (Luminance) at full resolution. This exploits the fact that the human eye is much more sensitive to changes in brightness than in color.

## Supported Modes

`go-pixo` supports two subsampling modes:

### 1. 4:4:4 (No Subsampling)
In this mode, every pixel has its own Y, Cb, and Cr value. No data is discarded before compression. This provides the highest quality but results in larger file sizes.

- **Component Ratio**: 1:1:1
- **MCU Size**: 8x8 pixels

### 2. 4:2:0 (Standard Subsampling)
In 4:2:0 mode, the chroma components (Cb and Cr) are downsampled by a factor of 2 in both horizontal and vertical directions. A single chroma sample is shared by a 2x2 block of luminance samples.

- **Component Ratio**: 4:1:1 (effective)
- **MCU Size**: 16x16 pixels (contains 4 Y blocks, 1 Cb block, and 1 Cr block)
- **Space Saving**: Reduces the amount of raw data by 50% before entropy coding.

## Implementation in go-pixo

The subsampling is handled during the block extraction phase in `src/jpeg/mcu.go`.

### `ExtractMCU420`
This function extracts a 16x16 region from the image:
1. It extracts 4 luminance blocks (8x8 each).
2. It averages the Cb and Cr values over each 2x2 pixel area to produce a single 8x8 Cb block and a single 8x8 Cr block.

```go
func ExtractMCU420(data []byte, width, height, mcuX, mcuY int) (yBlocks [4][64]float32, cbBlock, crBlock [64]float32) {
    // ...
    // Chroma is subsampled 2:1 in each dimension
    cbCrX := mcuInternalX / 2
    cbCrY := mcuInternalY / 2
    // ...
    cbAccum[cbCrIdx] += float32(cb)
    // ...
    // Average over 2x2 regions
    cbBlock[i] = (cbAccum[i] * 0.25) - 128.0
}
```

## When to use 4:2:0?
- **Always for photographs**: The quality loss is almost imperceptible to humans.
- **Web delivery**: Significantly reduces file size without meaningful quality impact.
- **Mobile/Bandwidth-constrained**: Essential for fast loading.

Use **4:4:4** only when you need absolute color precision (e.g., high-end editing or specific technical graphics).
