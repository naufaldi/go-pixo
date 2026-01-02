# Paeth Predictor: The PNG Filter Algorithm

The **Paeth predictor** (Filter Type 4) is PNG's most sophisticated filter. It uses three neighboring pixels to predict the current pixel value, providing the best compression for most image types.

---

## What is the Paeth Predictor?

The Paeth predictor chooses the "closest" neighbor pixel (left, above, or upper-left) based on a simple distance calculation. It's named after Alan Paeth, who designed the algorithm.

### The Algorithm

```go
func PaethPredictor(a, b, c int) int {
    // a = left pixel
    // b = above pixel
    // c = upper-left pixel (diagonal)
    
    p := a + b - c
    pa := abs(p - a)
    pb := abs(p - b)
    pc := abs(p - c)
    
    if pa <= pb && pa <= pc {
        return a  // Left is closest
    }
    if pb <= pc {
        return b  // Above is closest
    }
    return c  // Upper-left is closest
}
```

---

## Intuition: Why This Works

The Paeth predictor estimates what the current pixel **should be** based on its neighbors, then picks the neighbor that best matches that estimate.

### The Prediction Value `p`

```
p = a + b - c
```

This formula comes from **linear interpolation**:
- If the image has a **smooth gradient**, `a + b - c` approximates the current pixel
- Think of it as: "move right from `a`, move down from `b`, but subtract `c` to avoid double-counting"

### Visual Example

```
Neighbor positions:
┌─────┬─────┐
│  c  │  b  │  ← Previous row
├─────┼─────┤
│  a  │  x  │  ← Current pixel (x)
└─────┴─────┘

Example values:
┌─────┬─────┐
│ 50  │ 100 │
├─────┼─────┤
│ 80  │  x  │
└─────┴─────┘

p = 80 + 100 - 50 = 130
pa = |130 - 80| = 50
pb = |130 - 100| = 30
pc = |130 - 50| = 80

pb is smallest → return b = 100
```

---

## Why Paeth is Effective

### 1. Handles Multiple Patterns

- **Horizontal edges**: Chooses `a` (left)
- **Vertical edges**: Chooses `b` (above)
- **Diagonal patterns**: Chooses `c` (upper-left)
- **Smooth gradients**: Uses `p` approximation

### 2. Adaptive Selection

Unlike fixed filters (Sub always uses left, Up always uses above), Paeth **adapts** to the local image structure.

### 3. Best Overall Performance

For most images, Paeth provides the best compression because it can handle:
- Horizontal lines
- Vertical lines
- Diagonal lines
- Smooth gradients
- Mixed patterns

---

## Step-by-Step Example

### Input

```
Pixel neighborhood:
┌─────┬─────┐
│ 10  │ 20  │  ← Previous row
├─────┼─────┤
│ 15  │  x  │  ← Current pixel (x = 25)
└─────┴─────┘

a = 15 (left)
b = 20 (above)
c = 10 (upper-left)
x = 25 (current, to be filtered)
```

### Step 1: Calculate Prediction

```
p = a + b - c
p = 15 + 20 - 10
p = 25
```

### Step 2: Calculate Distances

```
pa = |p - a| = |25 - 15| = 10
pb = |p - b| = |25 - 20| = 5
pc = |p - c| = |25 - 10| = 15
```

### Step 3: Choose Closest

```
pa = 10
pb = 5  ← smallest
pc = 15

pb <= pc → return b = 20
```

### Step 4: Filter

```
Filtered value = x - predictor
Filtered value = 25 - 20 = 5
```

### Step 5: Reconstruction (Decoder)

```
Filtered value = 5
Predictor = PaethPredictor(15, 20, 10) = 20
Reconstructed = 5 + 20 = 25 ✓
```

---

## Edge Cases

### First Row (no previous row)

```
┌─────┬─────┐
│  -  │  -  │  ← No previous row
├─────┼─────┤
│  a  │  x  │
└─────┴─────┘

b = 0 (treat as zero)
c = 0 (treat as zero)
p = a + 0 - 0 = a
pa = |a - a| = 0
pb = |a - 0| = |a|
pc = |a - 0| = |a|

pa is smallest → return a
```

Result: Paeth falls back to Sub filter (predict from left) for the first row.

### First Pixel Group (no left neighbor)

```
┌─────┬─────┐
│  c  │  b  │
├─────┼─────┤
│  -  │  x  │  ← No left neighbor
└─────┴─────┘

a = 0 (treat as zero)
b = b
c = 0 (treat as zero)
p = 0 + b - 0 = b
pa = |b - 0| = |b|
pb = |b - b| = 0
pc = |b - 0| = |b|

pb is smallest → return b
```

Result: Paeth falls back to Up filter (predict from above) for the first pixel group.

---

## Comparison with Other Filters

| Filter | Prediction | Best For |
|--------|-----------|----------|
| None | 0 | Random/noisy data |
| Sub | `a` | Horizontal patterns |
| Up | `b` | Vertical patterns |
| Average | `(a+b)/2` | Smooth gradients |
| **Paeth** | **Adaptive** | **Mixed patterns (best overall)** |

---

## Implementation Details

### Filter Application

```go
func FilterPaeth(row []byte, prev []byte, bpp int) []byte {
    result := make([]byte, len(row))
    for i := 0; i < len(row); i++ {
        var a, b, c int
        
        // Get left pixel
        if i >= bpp {
            a = int(row[i-bpp])
        }
        
        // Get above pixel
        if len(prev) > 0 && i < len(prev) {
            b = int(prev[i])
        }
        
        // Get upper-left pixel
        if i >= bpp && len(prev) > 0 && i < len(prev) {
            c = int(prev[i-bpp])
        }
        
        predictor := PaethPredictor(a, b, c)
        result[i] = row[i] - byte(predictor)
    }
    return result
}
```

### Reconstruction

```go
func ReconstructPaeth(filtered []byte, prev []byte, bpp int) []byte {
    result := make([]byte, len(filtered))
    for i := 0; i < len(filtered); i++ {
        var a, b, c int
        
        // Get reconstructed left pixel
        if i >= bpp {
            a = int(result[i-bpp])
        }
        
        // Get above pixel (from previous row, already reconstructed)
        if len(prev) > 0 && i < len(prev) {
            b = int(prev[i])
        }
        
        // Get upper-left pixel
        if i >= bpp && len(prev) > 0 && i < len(prev) {
            c = int(prev[i-bpp])
        }
        
        predictor := PaethPredictor(a, b, c)
        result[i] = filtered[i] + byte(predictor)
    }
    return result
}
```

---

## Why Not Always Use Paeth?

While Paeth is usually the best filter, encoders still try all 5 filters because:

1. **Some images** compress better with simpler filters (e.g., pure horizontal patterns → Sub)
2. **Filter overhead**: The filter byte itself is part of the compressed data
3. **Heuristic limitations**: Sum of absolute values doesn't always match actual compression
4. **Edge cases**: First row/column may benefit from None/Sub/Up

---

## Summary

1. **Paeth predictor** uses three neighbors (`a`, `b`, `c`) to predict current pixel
2. **Formula**: `p = a + b - c`, then choose neighbor closest to `p`
3. **Adaptive**: Automatically selects best neighbor based on local pattern
4. **Best overall**: Usually provides best compression for mixed patterns
5. **Edge cases**: Falls back to Sub (first row) or Up (first pixel group)

---

## Related Documentation

- [PNG Filters](filters.md) - Overview of all five filter types
- [Filter Selection](filter-selection.md) - How encoders choose filters
- [PNG Scanlines](scanlines.md) - How filters are applied to scanlines
