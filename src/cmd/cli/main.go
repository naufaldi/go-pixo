// Command pixo compresses images into optimized PNGs.
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"

	"github.com/mac/go-pixo/src/jpeg"
	"github.com/mac/go-pixo/src/png"
	"golang.org/x/image/draw"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		inputFile      = flag.String("input", "", "Input image file (PNG or JPEG)")
		outputFile     = flag.String("output", "", "Output PNG file (default: input with .png extension)")
		preset         = flag.String("preset", "balanced", "Compression preset: fast, balanced, max, extreme")
		lossy          = flag.Bool("lossy", false, "Enable lossy compression with palette quantization")
		quality        = flag.Int("quality", 75, "Quality level for lossy compression (0-100)")
		compare        = flag.Bool("compare", false, "Show original vs compressed size comparison")
		verbose        = flag.Bool("verbose", false, "Enable detailed output")
		iterations     = flag.Int("iterations", 0, "Number of Zopfli iterations (0-100)")
		ditherStrength = flag.Float64("dither", 0.5, "Dithering strength 0.0-1.0 (default: 0.5)")
		maxColors      = flag.Int("max-colors", 256, "Maximum number of colors for lossy compression (2-256)")
		benchmark      = flag.Bool("benchmark", false, "Run compression multiple times and report statistics")
		benchmarkRuns  = flag.Int("benchmark-runs", 3, "Number of benchmark runs")
		outputFormat   = flag.String("output-format", "png", "Output format: png or jpeg")
		jpegQuality    = flag.Int("jpeg-quality", 85, "JPEG quality 1-100 (used with -output-format jpeg)")
		progressive    = flag.Bool("progressive", false, "Progressive JPEG encoding")
		trellis        = flag.Bool("trellis", false, "Trellis quantization (JPEG only)")
		optimHuffman   = flag.Bool("optimize-huffman", true, "Optimized Huffman tables (JPEG only)")
		resizeWidth    = flag.Int("width", 0, "Resize output width in pixels (0 = no resize, preserves aspect ratio)")
		resizeHeight   = flag.Int("height", 0, "Resize output height in pixels (0 = no resize, preserves aspect ratio)")
	)
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -input is required\n")
		flag.Usage()
		return 2
	}

	if *outputFormat != "png" && *outputFormat != "jpeg" {
		fmt.Fprintf(os.Stderr, "Error: -output-format must be png or jpeg\n")
		return 2
	}
	if *jpegQuality < 1 || *jpegQuality > 100 {
		fmt.Fprintf(os.Stderr, "Error: -jpeg-quality must be between 1 and 100\n")
		return 2
	}

	if *outputFile == "" {
		ext := getExt(*inputFile)
		base := ""
		if ext == "" {
			base = *inputFile
		} else {
			base = (*inputFile)[:len(*inputFile)-len(ext)]
		}
		if *outputFormat == "jpeg" {
			*outputFile = base + ".jpg"
		} else {
			*outputFile = base + ".png"
		}
	}

	file, err := os.Open(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		return 1
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Error closing input file: %v\n", closeErr)
		}
	}()

	// Get input file size for comparison
	fileInfo, err := file.Stat()
	inputFileSize := int64(0)
	if err == nil {
		inputFileSize = fileInfo.Size()
	}

	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding image: %v\n", err)
		return 1
	}

	fmt.Printf("Decoded %s image: %dx%d\n", format, img.Bounds().Dx(), img.Bounds().Dy())
	if inputFileSize > 0 {
		fmt.Printf("Input file size: %d bytes\n", inputFileSize)
	}

	if *resizeWidth > 0 || *resizeHeight > 0 {
		img = resizeImage(img, *resizeWidth, *resizeHeight)
		if *verbose {
			b := img.Bounds()
			fmt.Printf("  Resized to:   %dx%d\n", b.Dx(), b.Dy())
		}
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var pixels []byte

	switch typed := img.(type) {
	case *image.RGBA:
		pixels = typed.Pix
	case *image.NRGBA:
		pixels = make([]byte, width*height*4)
		for i := 0; i < len(typed.Pix); i += 4 {
			pixels[i] = typed.Pix[i]
			pixels[i+1] = typed.Pix[i+1]
			pixels[i+2] = typed.Pix[i+2]
			pixels[i+3] = typed.Pix[i+3]
		}
	default:
		rgba := image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
		pixels = rgba.Pix
	}

	originalSize := len(pixels)

	var encoder *png.Encoder
	// outputBytes holds the final encoded data written to disk.
	// The benchmark path uses pngData (PNG-only) and assigns it here before the write section.
	var outputBytes []byte

	if *verbose {
		fmt.Printf("Input pixel data size: %d bytes\n", originalSize)
		fmt.Printf("Preset: %s\n", *preset)
		fmt.Printf("Lossy: %v\n", *lossy)
		if *lossy {
			fmt.Printf("Quality: %d\n", *quality)
			fmt.Printf("Max Colors: %d\n", *maxColors)
			fmt.Printf("Dither Strength: %.2f\n", *ditherStrength)
		}
		fmt.Printf("Zopfli Iterations: %d\n", *iterations)
		if *outputFormat == "jpeg" {
			fmt.Printf("  JPEG quality:   %d\n", *jpegQuality)
			fmt.Printf("  Progressive:    %v\n", *progressive)
			fmt.Printf("  Trellis:        %v\n", *trellis)
		}
	}

	if *benchmark {
		fmt.Printf("\n=== Benchmark Mode (%d runs) ===\n", *benchmarkRuns)

		var sizes []int
		var totalTime int64

		for i := 0; i < *benchmarkRuns; i++ {
			startTime := time.Now()

			var opts png.Options
			switch *preset {
			case "fast":
				opts = png.FastOptions(width, height)
			case "max":
				opts = png.MaxOptions(width, height)
			case "extreme":
				opts = png.ExtremeOptions(width, height)
			default:
				opts = png.BalancedOptions(width, height)
			}

			opts.ZopfliIterations = *iterations

			if *lossy {
				if *maxColors < 2 {
					*maxColors = 2
				}
				if *maxColors > 256 {
					*maxColors = 256
				}
				opts.ApplyLossy(*maxColors, *quality, *ditherStrength)
			}

			encoder, err = png.NewEncoderWithOptions(opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating encoder: %v\n", err)
				return 1
			}

			var pngData []byte
			pngData, err = encoder.Encode(pixels)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding PNG: %v\n", err)
				return 1
			}

			elapsed := time.Since(startTime).Milliseconds()
			sizes = append(sizes, len(pngData))
			totalTime += elapsed

			if *verbose || *benchmarkRuns <= 3 {
				fmt.Printf("  Run %d: %d bytes, %d ms\n", i+1, len(pngData), elapsed)
			}

			// Keep last run's output for writing to disk
			outputBytes = pngData
		}

		minSize := sizes[0]
		maxSize := sizes[0]
		sizeSum := 0
		for _, s := range sizes {
			if s < minSize {
				minSize = s
			}
			if s > maxSize {
				maxSize = s
			}
			sizeSum += s
		}
		avgSize := sizeSum / len(sizes)

		fmt.Printf("\n=== Benchmark Results ===\n")
		fmt.Printf("Min Size:  %d bytes\n", minSize)
		fmt.Printf("Max Size:  %d bytes\n", maxSize)
		fmt.Printf("Avg Size:  %d bytes\n", avgSize)
		fmt.Printf("Avg Time:  %d ms\n", totalTime/int64(*benchmarkRuns))

		if *compare {
			ratio := float64(avgSize) / float64(inputFileSize) * 100
			savings := float64(inputFileSize-int64(avgSize)) / float64(inputFileSize) * 100
			fmt.Printf("Original File: %d bytes\n", inputFileSize)
			fmt.Printf("Raw Pixels:    %d bytes\n", originalSize)
			fmt.Printf("Avg Output:    %d bytes\n", avgSize)
			fmt.Printf("Ratio (vs File): %.2f%%\n", ratio)
			fmt.Printf("Savings (vs File): %.2f%%\n", savings)
		}
	} else if *outputFormat == "jpeg" {
		jpegQualityByte := uint8(*jpegQuality)

		// If input is JPEG and no lossy changes — use RecompressJPEGBytes for best quality.
		inputExt := strings.ToLower(getExt(*inputFile))
		isJpegInput := inputExt == ".jpg" || inputExt == ".jpeg"

		var encodeErr error
		if isJpegInput && !*lossy {
			// Read original file bytes for lossless re-encode path.
			inputBytes, readErr := os.ReadFile(*inputFile)
			if readErr != nil {
				fmt.Fprintf(os.Stderr, "Error reading input: %v\n", readErr)
				return 1
			}
			jpegOpts := jpeg.MaxOptions(0, 0, jpegQualityByte)
			jpegOpts.Progressive = *progressive
			jpegOpts.TrellisQuant = *trellis
			jpegOpts.OptimizeHuffman = *optimHuffman

			startTime := time.Now()
			outputBytes, encodeErr = jpeg.RecompressJPEGBytes(inputBytes, jpegOpts)
			elapsed := time.Since(startTime).Milliseconds()
			if encodeErr != nil {
				fmt.Fprintf(os.Stderr, "Error recompressing JPEG: %v\n", encodeErr)
				return 1
			}
			fmt.Printf("Encoding time: %d ms\n", elapsed)
		} else {
			// Pixel-based path (PNG input or lossy mode).
			rgbPixels := imageToRGB(img)
			jpegOpts := jpeg.BalancedOptions(width, height, jpegQualityByte)
			jpegOpts.Progressive = *progressive
			jpegOpts.TrellisQuant = *trellis
			jpegOpts.OptimizeHuffman = *optimHuffman

			enc, newEncErr := jpeg.NewEncoder(jpegOpts)
			if newEncErr != nil {
				fmt.Fprintf(os.Stderr, "Error creating JPEG encoder: %v\n", newEncErr)
				return 1
			}

			startTime := time.Now()
			outputBytes, encodeErr = enc.Encode(rgbPixels)
			elapsed := time.Since(startTime).Milliseconds()
			if encodeErr != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JPEG: %v\n", encodeErr)
				return 1
			}
			fmt.Printf("Encoding time: %d ms\n", elapsed)
		}

		if *compare {
			ratio := float64(len(outputBytes)) / float64(inputFileSize) * 100
			savings := float64(inputFileSize-int64(len(outputBytes))) / float64(inputFileSize) * 100
			fmt.Printf("Original File: %d bytes\n", inputFileSize)
			fmt.Printf("Raw Pixels:    %d bytes\n", originalSize)
			fmt.Printf("Compressed:    %d bytes\n", len(outputBytes))
			fmt.Printf("Ratio (vs File): %.2f%%\n", ratio)
			fmt.Printf("Savings (vs File): %.2f%%\n", savings)
		}
	} else {
		// PNG encoding path.
		var opts png.Options
		switch *preset {
		case "fast":
			opts = png.FastOptions(width, height)
		case "max":
			opts = png.MaxOptions(width, height)
		case "extreme":
			opts = png.ExtremeOptions(width, height)
		default:
			opts = png.BalancedOptions(width, height)
		}

		opts.ZopfliIterations = *iterations

		if *lossy {
			if *maxColors < 2 {
				*maxColors = 2
			}
			if *maxColors > 256 {
				*maxColors = 256
			}
			opts.ApplyLossy(*maxColors, *quality, *ditherStrength)
		}

		encoder, err = png.NewEncoderWithOptions(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating encoder: %v\n", err)
			return 1
		}

		startTime := time.Now()
		outputBytes, err = encoder.Encode(pixels)
		elapsed := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding PNG: %v\n", err)
			return 1
		}

		fmt.Printf("Encoding time: %d ms\n", elapsed)

		if *compare {
			ratio := float64(len(outputBytes)) / float64(inputFileSize) * 100
			savings := float64(inputFileSize-int64(len(outputBytes))) / float64(inputFileSize) * 100
			fmt.Printf("Original File: %d bytes\n", inputFileSize)
			fmt.Printf("Raw Pixels:    %d bytes\n", originalSize)
			fmt.Printf("Compressed:    %d bytes\n", len(outputBytes))
			fmt.Printf("Ratio (vs File): %.2f%%\n", ratio)
			fmt.Printf("Savings (vs File): %.2f%%\n", savings)
		}
	}

	outFile, err := os.Create(*outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		return 1
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", closeErr)
		}
	}()

	_, err = outFile.Write(outputBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully compressed to %s (%d bytes)\n", *outputFile, len(outputBytes))
	return 0
}

func getExt(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

// resizeImage scales img to fit within the given width and height,
// preserving aspect ratio. If only one dimension is given, the other
// is computed automatically.
func resizeImage(src image.Image, targetW, targetH int) image.Image {
	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()

	if targetW == 0 && targetH == 0 {
		return src
	}

	aspect := float64(srcW) / float64(srcH)
	if targetW == 0 {
		targetW = int(float64(targetH) * aspect)
	}
	if targetH == 0 {
		targetH = int(float64(targetW) / aspect)
	}
	if targetW == srcW && targetH == srcH {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// imageToRGB extracts packed RGB bytes from any image.Image, dropping the alpha channel.
func imageToRGB(img image.Image) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pixels := make([]byte, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			i := (y*w + x) * 3
			pixels[i] = uint8(r >> 8)
			pixels[i+1] = uint8(g >> 8)
			pixels[i+2] = uint8(bl >> 8)
		}
	}
	return pixels
}
