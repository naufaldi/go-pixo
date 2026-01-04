# Go-Pixo CLI

Minimal command-line tool for compressing images locally using the go-pixo PNG encoder.

## Usage

```bash
go run ./src/cmd/cli -input <image-file> [-output <output-file>]
```

## Quickstart (repo example)

```bash
mkdir -p compress

# Lossless (may not shrink if the PNG is already optimized)
go run ./src/cmd/cli \
  -input images/cursor-meetup.png \
  -output compress/cursor-meetup.png \
  -preset balanced \
  -compare -verbose

# Lossy (palette quantization; usually much smaller)
go run ./src/cmd/cli \
  -input images/cursor-meetup.png \
  -output compress/cursor-meetup-lossy-128q75.png \
  -preset balanced \
  -lossy -max-colors 128 -quality 75 -dither 0.5 \
  -compare -verbose
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

## Flags

- `-input` (required): input image file (`.png`, `.jpg`, `.jpeg`)
- `-output`: output PNG file path (defaults to `<input-without-ext>.png`; note: `image.png` overwrites `image.png`)
- `-preset`: `fast`, `balanced`, `max`, `extreme` (default: `balanced`)
- `-compare`: print original file size vs output size
- `-verbose`: print detailed encoder settings and timings

Lossy (palette quantization):
- `-lossy`: enable palette quantization output (Indexed PNG)
- `-max-colors`: `2`-`256` (default: `256`)
- `-quality`: `0`-`100` (default: `75`)
- `-dither`: `0.0`-`1.0` (default: `0.5`)

Advanced:
- `-iterations`: Zopfli iterations `0`-`100` (default: `0`)
- `-benchmark`: run multiple encodes and report statistics
- `-benchmark-runs`: number of benchmark runs (default: `3`)

## Supported Input Formats

- PNG
- JPEG

## Output

Always produces PNG format (lossless compression with filter selection).

## Notes for already-optimized PNGs

Some PNGs are already compressed very close to their optimum. In those cases, a lossless re-encode may produce a file that is the same size or slightly larger. Use `-compare` to see the size delta, and use `-lossy` if you want size reduction via palette quantization.

## Use Cases

- Verify PNG encoder works correctly
- Debug compression issues
- Batch processing scripts
- Testing compression ratios
