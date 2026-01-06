# Optimized Huffman Tables

Standard JPEG encoders often use "standard" Huffman tables defined in the JPEG specification (Annex K). While these tables work well for many images, they are not optimal for any specific image. `go-pixo` provides an option to generate custom Huffman tables tailored to each individual image.

## How it Works

Optimizing Huffman tables is a two-pass process:

1. **Pass 1: Frequency Counting**: The encoder performs a "dry run" of the DCT, quantization, and zigzag reordering. Instead of writing bits to the output, it counts how often each symbol (DC category or AC run/size) appears in the actual image data.
2. **Pass 2: Tree Construction**: The encoder builds a Huffman tree from these frequencies using the standard Huffman algorithm. It then extracts canonical code lengths and builds the final `DHT` (Define Huffman Table) markers.

## Benefits

- **Better Compression**: Optimized tables typically reduce file size by **5-10%** compared to standard tables at the same quality level.
- **Image-Specific Efficiency**: Images with very few colors or repetitive patterns benefit significantly from custom tables.

## Implementation in go-pixo

The logic is contained in `src/jpeg/huffman_optimized.go`.

### `BuildOptimizedTables`
This function is the main entry point. It iterates over all MCUs, counts frequencies, and returns a `HuffmanTables` struct containing the custom codes.

```go
func BuildOptimizedTables(pixels []byte, width, height int, ...) *HuffmanTables {
    // 1. Count frequencies via dry run
    // 2. Build tree and generate codes
    // 3. Return customized tables
}
```

### Huffman Tree Building
We reuse the robust Huffman tree implementation from the `compress` package (used for PNG/DEFLATE).

```go
tree := compress.BuildTree(frequencies)
codesMap := compress.GenerateCodes(tree)
```

## Performance Trade-off
Optimizing Huffman tables requires an extra pass over the pixel data, making encoding roughly **2x slower**. In most web applications, the smaller file size (faster download) is worth the extra encoding time.
