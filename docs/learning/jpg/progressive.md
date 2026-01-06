# Progressive JPEG

Standard (baseline) JPEGs load from top to bottom. Progressive JPEGs contain multiple "scans" of the image, each adding more detail. This provides two main benefits:
1. **User Experience**: A low-quality "preview" appears almost instantly, which then sharpens as more data arrives.
2. **Compression**: Surprisingly, progressive encoding often results in **smaller file sizes** (3-5% saving) because it can group similar frequency components across the entire image.

## How it Works

Progressive JPEG uses two techniques to split the image data:

### 1. Spectral Selection
Instead of encoding all 64 DCT coefficients of a block at once, we encode ranges of frequencies.
- **Scan 1**: Only the DC coefficients (average brightness/color).
- **Scan 2**: Low-frequency AC coefficients (rough shapes).
- **Scan 3+**: High-frequency AC coefficients (fine details).

### 2. Successive Approximation
Each coefficient can be sent bit-by-bit.
- **First Scan**: Send the most significant bits.
- **Refinement Scan**: Send the remaining lower bits.

## Implementation in go-pixo

The progressive implementation is in `src/jpeg/progressive.go` and integrated into the main encoder loop in `src/jpeg/encoder.go`.

### Scan Script
The encoder follows a "scan script" that defines what data goes into each pass.

```go
func SimpleProgressiveScript() []ScanSpec {
    return []ScanSpec{
        {Components: []uint8{1, 2, 3}, SS: 0, SE: 0, AH: 0, AL: 0}, // DC
        {Components: []uint8{1}, SS: 1, SE: 63, AH: 0, AL: 0},       // Y AC
        {Components: []uint8{2}, SS: 1, SE: 63, AH: 0, AL: 0},       // Cb AC
        {Components: []uint8{3}, SS: 1, SE: 63, AH: 0, AL: 0},       // Cr AC
    }
}
```

### Memory Requirement
Unlike baseline encoding, which only needs to keep a single MCU in memory, progressive encoding requires storing **all quantized coefficients** for the entire image before writing the first scan.

## Markers
Progressive JPEG uses the **SOF2** marker instead of SOF0. Each scan is introduced by an **SOS** (Start of Scan) marker specifying the frequency range and bit position.
