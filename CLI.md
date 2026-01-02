# Go-Pixo CLI

Minimal command-line tool for compressing images locally using the go-pixo PNG encoder.

## Usage

```bash
go run ./src/cmd/cli -input <image-file> [-output <output-file>]
```

### Examples

```bash
# Compress a JPEG to PNG
go run ./src/cmd/cli -input photo.jpg -output compressed.png

# Compress a PNG (output defaults to input with .png extension)
go run ./src/cmd/cli -input image.png
# Creates image.png (overwrites original)

# Build standalone binary
go build -o go-pixo ./src/cmd/cli
./go-pixo -input photo.jpg -output compressed.png
```

## Supported Input Formats

- PNG
- JPEG

## Output

Always produces PNG format (lossless compression with filter selection).

## Use Cases

- Verify PNG encoder works correctly
- Debug compression issues
- Batch processing scripts
- Testing compression ratios
