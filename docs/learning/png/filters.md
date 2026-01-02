# PNG Filters: What, Why, and How

This guide explains PNG filters in detail: what they are, why they help DEFLATE compression, and how decoders reconstruct original pixel values from filtered data.

---

## What Are PNG Filters?

PNG filters are **predictive encoding** techniques applied to each scanline before DEFLATE compression. Instead of storing raw pixel values, filters store the **difference** between each pixel and a predicted value based on neighboring pixels.

### The Five Filter Types

| Filter | Name | Formula | Prediction Source |
|--------|------|---------|-------------------|
| 0 | None | `x` | No prediction (raw value) |
| 1 | Sub | `x - a` | Left pixel (`a`) |
| 2 | Up | `x - b` | Above pixel (`b`) |
| 3 | Average | `x - floor((a + b) / 2)` | Average of left and above |
| 4 | Paeth | `x - Paeth(a, b, c)` | Paeth predictor (left, above, upper-left) |

Where:
- `x` = current pixel value
- `a` = pixel to the left (same row)
- `b` = pixel above (previous row)
- `c` = pixel upper-left (diagonal)

### Visual Reference

```
Pixel neighborhood:
┌─────┬─────┐
│  c  │  b  │  ← Previous row
├─────┼─────┤
│  a  │  x  │  ← Current row (x is being filtered)
└─────┴─────┘
```

---

## Why Filters Help DEFLATE Compression

DEFLATE (LZ77 + Huffman) compresses data by finding **repeated patterns**. Filters transform pixel data to create more repeated values, which compress better.

### Example: Horizontal Gradient

**Without filtering** (Filter None):
```
Raw pixels: [100, 101, 102, 103, 104, 105, 106, 107]
            ↑    ↑    ↑    ↑    ↑    ↑    ↑    ↑
          All different values → poor compression
```

**With Filter Sub** (predict from left):
```
Raw pixels:  [100, 101, 102, 103, 104, 105, 106, 107]
Filtered:    [100,   1,   1,   1,   1,   1,   1,   1]
             ↑    ↑    ↑    ↑    ↑    ↑    ↑    ↑
          Base  All same! → excellent compression
```

The filtered data has **one unique value repeated 7 times**, which DEFLATE can compress very efficiently.

### Why This Works

1. **Spatial redundancy**: Adjacent pixels in images are often similar
2. **Smaller differences**: Filtering produces smaller values (often near zero)
3. **More zeros**: Many filtered bytes become zero, which compresses well
4. **Repeated patterns**: Filters create runs of identical values

---

## Modulo-256 Arithmetic

PNG filters use **modulo-256 arithmetic** (wrapping at byte boundaries). This means all operations are performed on `uint8` values, and results wrap around:

```go
// Example: Filter Sub
row := []byte{100, 150, 200}
bpp := 1

// For pixel at index 1:
left := row[0]  // 100
current := row[1]  // 150
filtered := current - left  // 150 - 100 = 50

// For pixel at index 2:
left := row[1]  // 150
current := row[2]  // 200
filtered := current - left  // 200 - 150 = 50

// If subtraction goes negative, it wraps:
// Example: 10 - 20 = 246 (modulo 256)
// uint8(10 - 20) = uint8(-10) = 246
```

### Why Modulo-256?

- PNG stores filtered values as **bytes** (0-255)
- Negative differences wrap to positive values (e.g., -10 → 246)
- Decoders can reconstruct original values using the same modulo arithmetic
- This ensures all filtered bytes fit in a single byte

---

## How Decoders Reconstruct Original Values

Reconstruction is the **inverse** of filtering. Decoders apply the same filter formula but **add** the prediction instead of subtracting:

### Reconstruction Formulas

| Filter | Reconstruction Formula |
|--------|----------------------|
| None | `x = filtered` |
| Sub | `x = filtered + a` |
| Up | `x = filtered + b` |
| Average | `x = filtered + floor((a + b) / 2)` |
| Paeth | `x = filtered + Paeth(a, b, c)` |

### Example: Sub Filter Reconstruction

**Encoding** (filtering):
```
Original: [100, 150, 200]
Filtered:  [100, 50, 50]  // 150-100=50, 200-150=50
```

**Decoding** (reconstruction):
```
Filtered:  [100, 50, 50]
Reconstructed:
  [0] = 100 (no left neighbor, use filtered value)
  [1] = 50 + 100 = 150 ✓
  [2] = 50 + 150 = 200 ✓
```

### Important: Sequential Reconstruction

Reconstruction must be done **sequentially** (left-to-right) because:
- **Sub filter** needs the reconstructed left pixel (`a`)
- **Average filter** needs both left (`a`) and above (`b`)
- **Paeth filter** needs left (`a`), above (`b`), and upper-left (`c`)

```go
func ReconstructSub(filtered []byte, bpp int) []byte {
    result := make([]byte, len(filtered))
    for i := 0; i < len(filtered); i++ {
        var left byte
        if i >= bpp {
            left = result[i-bpp]  // Use RECONSTRUCTED left pixel
        }
        result[i] = filtered[i] + left
    }
    return result
}
```

---

## Filter Selection Strategy

Encoders should **try all 5 filters** for each row and choose the one that produces the smallest compressed size. However, computing DEFLATE compression for each filter is expensive, so encoders use a **heuristic**: choose the filter with the minimum **sum of absolute values**.

### Why Sum of Absolute Values?

- **Correlates with compression**: Smaller absolute values → more zeros → better compression
- **Fast to compute**: O(n) per filter, no compression needed
- **Good approximation**: Usually picks the same filter as full compression would

```go
func SumAbsoluteValues(filtered []byte) int {
    sum := 0
    for _, b := range filtered {
        signed := int(int8(b))  // Interpret as signed byte
        if signed < 0 {
            sum -= signed  // abs(negative)
        } else {
            sum += signed  // abs(positive)
        }
    }
    return sum
}
```

### Selection Algorithm

```go
func SelectFilter(row, prev []byte, bpp int) (FilterType, []byte) {
    bestFilter := FilterNone
    bestFiltered := FilterNone(row)
    bestScore := SumAbsoluteValues(bestFiltered)

    // Try all filters
    for _, filter := range []FilterType{FilterSub, FilterUp, FilterAverage, FilterPaeth} {
        filtered := applyFilter(filter, row, prev, bpp)
        score := SumAbsoluteValues(filtered)
        if score < bestScore {
            bestScore = score
            bestFilter = filter
            bestFiltered = filtered
        }
    }

    return bestFilter, bestFiltered
}
```

---

## Edge Cases

### First Row

- **No previous row**: `prev` is empty or all zeros
- **Sub filter**: First `bpp` bytes have no left neighbor (use raw value)
- **Up filter**: All differences are relative to zero
- **Average/Paeth**: Treat missing neighbors as zero

### First Pixel Group

- For filters that use left neighbor (Sub, Average, Paeth):
  - First `bpp` bytes have no left neighbor
  - Use raw value (no prediction)

### Byte Boundaries

- Filters operate **per-byte**, not per-pixel
- For RGB (bpp=3), each color channel is filtered independently
- This allows different channels to benefit from different patterns

---

## Implementation Notes

### Filter Application

```go
func FilterSub(row []byte, bpp int) []byte {
    result := make([]byte, len(row))
    for i := 0; i < len(row); i++ {
        var left byte
        if i >= bpp {
            left = row[i-bpp]
        }
        result[i] = row[i] - left  // Modulo-256 automatically
    }
    return result
}
```

### Reconstruction

```go
func ReconstructSub(filtered []byte, bpp int) []byte {
    result := make([]byte, len(filtered))
    for i := 0; i < len(filtered); i++ {
        var left byte
        if i >= bpp {
            left = result[i-bpp]  // Use reconstructed value
        }
        result[i] = filtered[i] + left  // Modulo-256 automatically
    }
    return result
}
```

---

## Summary

1. **PNG filters** predict pixel values from neighbors and store differences
2. **Modulo-256 arithmetic** ensures all values fit in bytes (wraps negatives)
3. **Reconstruction** is the inverse: add prediction to filtered value
4. **Sequential processing** required (left-to-right, top-to-bottom)
5. **Filter selection** uses sum of absolute values as a compression heuristic
6. **Filters help DEFLATE** by creating repeated patterns and more zeros

---

## Related Documentation

- [PNG Scanlines](scanlines.md) - How scanlines are structured with filter bytes
- [Paeth Predictor](paeth.md) - Detailed explanation of the Paeth algorithm
- [Filter Selection](filter-selection.md) - Why and how filters are chosen per row
- [PNG Encoding Pipeline](../png-encoding.md) - How filters fit into the complete encoding process
