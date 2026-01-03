# PNG Lossy Compression: Palette Quantization

This guide explains how lossy PNG compression works using palette quantization, the median cut algorithm, and dithering. These techniques can reduce PNG file sizes by 50-90% for images with limited color palettes.

## Why Lossy PNG?

PNG is typically **lossless**, meaning no image information is discarded during compression. However, for many images (icons, screenshots, graphics), we can achieve dramatic size reductions by reducing the number of unique colors.

### The Trade-off

| Mode | Colors | Size | Quality |
|------|--------|------|---------|
| Lossless RGBA | 16.7 million | Largest | Perfect |
| Lossy (256 colors) | 256 | 50-90% smaller | Slightly reduced |
| Lossy (16 colors) | 16 | 70-95% smaller | Noticeable artifacts |

### When to Use Lossy PNG

**Best for:**
- Icons and logos with few colors
- Screenshots with flat colors
- UI elements and graphics
- Images where small file size matters more than perfect quality

**Not suitable for:**
- Photographic images (JPEG is better)
- Medical or scientific imaging
- Any image requiring exact pixel reproduction

## Color Quantization: The Core Concept

**Color quantization** reduces the number of colors in an image by mapping each pixel to the nearest color in a smaller palette.

### The Problem: Too Many Colors

A typical RGBA image can have millions of unique colors:

```go
// 1x1 pixel: 1 color
// 100x100 image: up to 10,000 colors  
// 1000x1000 image: up to 1,000,000 colors!

// But PNG with 256 colors uses 1 byte per pixel instead of 4
// 1000x1000 lossless: 4,000,000 bytes
// 1000x1000 quantized: 1,000,000 bytes (75% smaller!)
```

### The Solution: Map to Palette

Instead of storing RGB values for each pixel, we store a **palette index**:

```go
// Original: 4 bytes per pixel
pixels := []byte{255, 0, 0, 255, 0, 255, 0, 255} // RGBA RGBA

// Quantized: 1 byte per pixel
indexed := []byte{0, 1, 2} // Index into palette[0]=red, palette[1]=green, palette[2]=blue
palette := []Color{
    {255, 0, 0},   // Index 0
    {0, 255, 0},     // Index 1
    {0, 0, 255},     // Index 2
}
```

## Color Counting: Finding the Important Colors

Before we can create a palette, we need to know which colors actually appear in the image:

```go
// CountColors counts frequency of each unique color
func CountColors(pixels []byte, colorType int) map[Color]int {
    colorMap := make(map[Color]int)
    bpp := BytesPerPixel(ColorType(colorType))

    for i := 0; i < len(pixels); i += bpp {
        c := Color{
            R: pixels[i],
            G: pixels[i+1],
            B: pixels[i+2],
        }
        colorMap[c]++
    }

    return colorMap
}
```

**Why count frequencies?**

- Common colors need more palette slots
- Rare colors can share slots with similar colors
- This preserves visual quality where it matters most

**Example output:**

```go
colorMap := CountColors(imagePixels, 6) // RGBA
// {Color{255,0,0}: 15000, Color{0,255,0}: 8000, Color{0,0,255}: 5000, ...}
// Red appears 15,000 times - very important!
// Blue appears 5,000 times - less important
```

## The Median Cut Algorithm

The **median cut algorithm** is the standard method for creating optimal color palettes. It recursively splits the color space until we have the desired number of palette entries.

### How It Works

1. **Start with all colors** in one bucket
2. **Find the color channel** (R, G, or B) with the widest range
3. **Sort colors** by that channel
4. **Split at the median** to create two buckets
5. **Repeat** until we have enough buckets
6. **Average colors** in each bucket to create palette entries

### Visual Example

```
Initial: All colors in one bucket (suppose 8 colors)
        [Red: 10-250, Green: 0-100, Blue: 0-100]

Split by Red (widest range: 10-250):
    Bucket A (red 10-130)          Bucket B (red 140-250)
    [4 colors]                    [4 colors]

Split A by Green:
    A1 (green 0-50)    A2 (green 60-100)
    [2 colors]          [2 colors]

Split B by Blue:
    B1 (blue 0-50)    B2 (blue 60-100)
    [2 colors]          [2 colors]

Palette = Average of each bucket:
    A1 -> Color{R:70, G:25, B:30}
    A2 -> Color{R:100, G:80, B:45}
    B1 -> Color{R:170, G:35, B:25}
    B2 -> Color{R:210, G:75, B:55}
```

### Implementation

```go
func MedianCut(colorsWithCount []ColorWithCount, maxColors int) []Color {
    if len(colorsWithCount) <= maxColors {
        // Fewer colors than max - return all
        result := make([]Color, len(colorsWithCount))
        for i, cwc := range colorsWithCount {
            result[i] = cwc.Color
        }
        return result
    }

    buckets := []bucket{{colors: colorsWithCount}}

    // Keep splitting until we have enough buckets
    for len(buckets) < maxColors {
        // Find largest bucket
        largestIdx := -1
        maxSize := 0
        for i := range buckets {
            if len(buckets[i].colors) > maxSize {
                maxSize = len(buckets[i].colors)
                largestIdx = i
            }
        }

        if largestIdx == -1 || maxSize < 2 {
            break // Can't split further
        }

        // Split the largest bucket
        left, right := splitBucket(buckets[largestIdx].colors)
        buckets[largestIdx].colors = left
        if len(right) > 0 {
            buckets = append(buckets, bucket{colors: right})
        }
    }

    // Average colors in each bucket
    result := make([]Color, 0, maxColors)
    for _, b := range buckets {
        if len(b.colors) > 0 {
            result = append(result, averageColors(b.colors))
        }
    }

    return result
}

func splitBucket(colors []ColorWithCount) ([]ColorWithCount, []ColorWithCount) {
    // Find channel with widest range
    minR, maxR := uint8(255), uint8(0)
    minG, maxG := uint8(255), uint8(0)
    minB, maxB := uint8(255), uint8(0)

    for _, c := range colors {
        if c.R < minR { minR = c.R }
        if c.R > maxR { maxR = c.R }
        if c.G < minG { minG = c.G }
        if c.G > maxG { maxG = c.G }
        if c.B < minB { minB = c.B }
        if c.B > maxB { maxB = c.B }
    }

    // Choose channel with widest range
    rangeR := int(maxR) - int(minR)
    rangeG := int(maxG) - int(minG)
    rangeB := int(maxB) - int(minB)

    sortBy := 0 // 0=R, 1=G, 2=B
    maxRange := rangeR
    if rangeG > maxRange {
        maxRange = rangeG
        sortBy = 1
    }
    if rangeB > maxRange {
        maxRange = rangeB
        sortBy = 2
    }

    // Sort by chosen channel
    sorted := make([]ColorWithCount, len(colors))
    copy(sorted, colors)
    sort.Slice(sorted, func(i, j int) bool {
        switch sortBy {
        case 0: return sorted[i].R < sorted[j].R
        case 1: return sorted[i].G < sorted[j].G
        default: return sorted[i].B < sorted[j].B
        }
    })

    // Split at median
    mid := len(sorted) / 2
    return sorted[:mid], sorted[mid:]
}
```

### Why Median Cut?

| Algorithm | Quality | Speed | Memory |
|-----------|---------|-------|--------|
| Median Cut | Good | Fast | Low |
| K-Means | Better | Slow | Medium |
| Octree | Good | Medium | High |

Median cut is the standard choice for PNG quantization because it provides good quality with low memory usage and fast execution.

## Palette Data Structures

```go
// Color represents an RGB color
type Color struct {
    R, G, B uint8
}

// ColorWithCount includes frequency information
type ColorWithCount struct {
    Color
    Count int
}

// Palette represents an indexed color palette
type Palette struct {
    Colors    []Color  // Max 256 colors
    NumColors int      // Actual number used
}

// FindNearest returns the palette index of the closest color
func (p *Palette) FindNearest(c Color) int {
    bestIdx := 0
    bestDist := uint64(^uint64(0)) // Max value

    for i := 0; i < p.NumColors; i++ {
        dr := int64(c.R) - int64(p.Colors[i].R)
        dg := int64(c.G) - int64(p.Colors[i].G)
        db := int64(c.B) - int64(p.Colors[i].B)

        dist := uint64(dr*dr + dg*dg + db*db)
        if dist < bestDist {
            bestDist = dist
            bestIdx = i
        }
    }

    return bestIdx
}
```

**Euclidean distance** in RGB space measures color similarity. Colors closer together look more similar:

```
distance = sqrt((R1-R2)^2 + (G1-G2)^2 + (B1-B2)^2)
```

## Dithering: Hiding Quantization Artifacts

When we reduce colors, some pixels get mapped to slightly different colors. This causes **banding** - smooth gradients become stepped. **Dithering** spreads this error to neighboring pixels, creating the illusion of more colors.

### Without Dithering (Threshold)

Each pixel maps directly to the nearest palette color:

```go
func Threshold(pixels []byte, palette Palette) []byte {
    bpp := 3 // RGB
    width := len(pixels) / bpp
    indexed := make([]byte, width)

    for i := 0; i < width; i++ {
        offset := i * bpp
        c := Color{
            R: pixels[offset],
            G: pixels[offset+1],
            B: pixels[offset+2],
        }
        indexed[i] = uint8(palette.FindNearest(c))
    }

    return indexed
}
```

### With Floyd-Steinberg Dithering

Error is diffused to neighboring pixels using specific weights:

```
        Current pixel
            |
    7/16 -> right neighbor
    3/16 -> bottom-left
    5/16 -> bottom
    1/16 -> bottom-right
```

```go
func FloydSteinberg(pixels []byte, palette Palette) []byte {
    bpp := 3 // RGB
    width := len(pixels) / bpp

    // Convert to mutable format
    pixelData := make([][3]int, width)
    for i := 0; i < width; i++ {
        offset := i * bpp
        pixelData[i] = [3]int{
            int(pixels[offset]),
            int(pixels[offset+1]),
            int(pixels[offset+2]),
        }
    }

    indexed := make([]byte, width)
    errors := make([][3]int, width+2)

    for i := 0; i < width; i++ {
        // Apply accumulated error
        r := clamp(pixelData[i][0] + errors[i][0])
        g := clamp(pixelData[i][1] + errors[i][1])
        b := clamp(pixelData[i][2] + errors[i][2])

        // Find nearest palette color
        c := Color{uint8(r), uint8(g), uint8(b)}
        paletteIdx := palette.FindNearest(c)
        paletteColor := palette.Colors[paletteIdx]

        // Calculate error
        errR := r - int(paletteColor.R)
        errG := g - int(paletteColor.G)
        errB := b - int(paletteColor.B)

        indexed[i] = uint8(paletteIdx)

        // Distribute error to neighbors
        if i+1 < width {
            errors[i+1][0] += errR * 7 / 16
            errors[i+1][1] += errG * 7 / 16
            errors[i+1][2] += errB * 7 / 16
        }
    }

    return indexed
}

func clamp(v int) int {
    if v < 0 { return 0 }
    if v > 255 { return 255 }
    return v
}
```

### Visual Comparison

```
Without dithering:          With dithering:
R R R R R R R R           R R r r r R R R
R R R R R R R R    vs     r r r r r r r r
G G G G G G G G           g g g g g g g g
G G G G G G G G           G G g g g G G G
```

Dithering adds "noise" but prevents visible banding.

## PLTE Chunk: Storing the Palette

The **PLTE** chunk stores the palette in a PNG file:

```text
4 bytes: Length (3 * numColors)
4 bytes: Type = "PLTE"
3 * numColors: RGB values (R0,G0,B0, R1,G1,B1, ...)
4 bytes: CRC32 of "PLTE" + RGB data
```

```go
func WritePLTE(w io.Writer, palette Palette) error {
    if palette.NumColors < 1 || palette.NumColors > 256 {
        return ErrInvalidChunkData
    }

    data := make([]byte, 3*palette.NumColors)
    for i := 0; i < palette.NumColors; i++ {
        data[i*3+0] = palette.Colors[i].R
        data[i*3+1] = palette.Colors[i].G
        data[i*3+2] = palette.Colors[i].B
    }

    // Write length
    length := uint32(len(data))
    if err := binary.Write(w, binary.BigEndian, length); err != nil {
        return err
    }

    // Write type
    if err := binary.Write(w, nil, []byte("PLTE")); err != nil {
        return err
    }

    // Write palette data
    if _, err := w.Write(data); err != nil {
        return err
    }

    // Write CRC
    crc := compress.CRC32(append([]byte("PLTE"), data...))
    return binary.Write(w, binary.BigEndian, crc)
}
```

**Example:** A 256-color palette is 768 bytes (256 x 3), plus 12 bytes for chunk header/footer = 780 bytes total.

## tRNS Chunk: Palette Transparency

The **tRNS** chunk adds alpha values to a palette:

```text
4 bytes: Length (numAlphaValues)
4 bytes: Type = "tRNS"
numAlphaValues: Alpha values (0=transparent, 255=opaque)
4 bytes: CRC32 of "tRNS" + alpha data
```

```go
func WriteTRNS(w io.Writer, alphaValues []uint8) error {
    if len(alphaValues) == 0 || len(alphaValues) > 256 {
        return ErrInvalidChunkData
    }

    data := make([]byte, len(alphaValues))
    for i, a := range alphaValues {
        data[i] = a
    }

    // Write length
    length := uint32(len(data))
    if err := binary.Write(w, binary.BigEndian, length); err != nil {
        return err
    }

    // Write type
    if err := binary.Write(w, nil, []byte("tRNS")); err != nil {
        return err
    }

    // Write alpha data
    if _, err := w.Write(data); err != nil {
        return err
    }

    // Write CRC
    crc := compress.CRC32(append([]byte("tRNS"), data...))
    return binary.Write(w, binary.BigEndian, crc)
}
```

**Chunk Order:** PLTE must come before tRNS, and both must come before IDAT:

```
IHDR -> PLTE -> [tRNS] -> IDAT -> IEND
```

## Complete Quantization Pipeline

```go
func Quantize(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
    if maxColors <= 0 || maxColors > 256 {
        maxColors = 256
    }

    // Step 1: Count colors
    colorMap := CountColors(pixels, colorType)

    // Step 2: Sort by frequency
    colorsWithCount := ToColorWithCountSlice(colorMap)

    // Step 3: Generate palette with median cut
    paletteColors := MedianCut(colorsWithCount, maxColors)

    // Step 4: Build palette
    palette := NewPalette(len(paletteColors))
    for _, c := range paletteColors {
        palette.AddColor(c)
    }

    // Step 5: Map pixels to palette indices
    bpp := BytesPerPixel(ColorType(colorType))
    width := len(pixels) / bpp
    indexed := make([]byte, width)

    for i := 0; i < width; i++ {
        offset := i * bpp
        c := Color{
            R: pixels[offset],
            G: pixels[offset+1],
            B: pixels[offset+2],
        }
        indexed[i] = uint8(palette.FindNearest(c))
    }

    return indexed, *palette
}
```

## Integration with PNG Encoder

When lossy mode is enabled, the encoder quantizes before writing:

```go
func (e *Encoder) EncodeWithOptions(pixels []byte, opts Options) ([]byte, error) {
    // ...

    // Lossy mode: Quantize to palette
    if opts.MaxColors > 0 && opts.MaxColors < 256 {
        indexedPixels, palette := Quantize(
            processedPixels,
            int(colorType),
            opts.MaxColors,
        )

        // Write PLTE chunk
        if err := WritePLTE(&buf, palette); err != nil {
            return nil, err
        }

        // Write IDAT with indexed pixels
        if err := WriteIDATWithOptions(&buf, indexedPixels,
            opts.Width, opts.Height, ColorIndexed, opts); err != nil {
            return nil, err
        }

        return buf.Bytes(), nil
    }

    // Lossless mode: continue as normal
    // ...
}
```

## File Size Comparison

| Image Type | Original | Quantized (256) | Reduction |
|------------|----------|-----------------|-----------|
| Simple icon | 50 KB | 12 KB | 76% |
| Screenshot | 500 KB | 180 KB | 64% |
| Logo | 25 KB | 8 KB | 68% |
| Photo | 2 MB | 1.8 MB | 10% |

**Photos don't benefit much** because they use many colors throughout the image. For photos, JPEG is a better choice.

## Summary

1. **Color Quantization** reduces colors from millions to 256 or fewer

2. **Median Cut Algorithm** creates optimal palettes by recursively splitting color space

3. **Dithering** hides quantization artifacts by diffusing color error

4. **PLTE Chunk** stores the palette RGB values (3 bytes per color)

5. **tRNS Chunk** adds alpha values for palette transparency

6. **Indexing** replaces RGB values with single-byte palette indices

7. **Best for:** Icons, logos, screenshots, UI graphics

8. **Not for:** Photos (use JPEG instead)

## Implementation Reference

| Component | File | Purpose |
|-----------|------|---------|
| Palette | `src/png/palette.go` | Color, Palette, FindNearest |
| Color Counting | `src/png/color_count.go` | CountColors, frequency analysis |
| Median Cut | `src/png/median_cut.go` | MedianCut algorithm |
| Quantization | `src/png/quantize.go` | Quantize, QuantizeWithDithering |
| Dithering | `src/png/dither.go` | Threshold, FloydSteinberg |
| PLTE Writer | `src/png/plte_writer.go` | WritePLTE |
| tRNS Writer | `src/png/trns_writer.go` | WriteTRNS |
| Encoder | `src/png/encoder.go` | Lossy mode integration |

## Next Steps

- Explore [PNG Filters](filters.md) for lossless compression improvements
- Understand [DEFLATE Compression](deflate.md) for how compressed data is stored
- Learn about [Color Type Analysis](../png/color_analysis.md) for lossless color reduction
