package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/mac/go-pixo/src/jpeg"
	"github.com/mac/go-pixo/src/png"
)

func main() {
	var (
		inputFile      = flag.String("input", "", "Input image file (PNG or JPEG)")
		outputFile     = flag.String("output", "", "Output file (default: input with format extension)")
		format         = flag.String("format", "png", "Output format: png or jpeg")
		preset         = flag.String("preset", "balanced", "Compression preset: fast, balanced, max")
		lossy          = flag.Bool("lossy", false, "Enable lossy compression with palette quantization (PNG only)")
		quality        = flag.Int("quality", 75, "Quality level (PNG: 0-100 for lossy, JPEG: 1-100)")
		compare        = flag.Bool("compare", false, "Show original vs compressed size comparison")
		verbose        = flag.Bool("verbose", false, "Enable detailed output")
		iterations     = flag.Int("iterations", 0, "Number of Zopfli iterations (PNG only, 0-100)")
		ditherStrength = flag.Float64("dither", 0.5, "Dithering strength 0.0-1.0 (default: 0.5)")
		maxColors      = flag.Int("max-colors", 256, "Maximum number of colors for lossy compression (2-256)")
		benchmark      = flag.Bool("benchmark", false, "Run compression multiple times and report statistics")
		benchmarkRuns  = flag.Int("benchmark-runs", 3, "Number of benchmark runs")

		// JPEG-specific flags
		progressive     = flag.Bool("progressive", false, "Enable progressive JPEG encoding")
		subsampling     = flag.String("subsampling", "420", "Chroma subsampling: 420 or 444 (JPEG only)")
		trellis         = flag.Bool("trellis", false, "Enable trellis quantization (JPEG only)")
		optimizeHuffman = flag.Bool("optimize-huffman", false, "Enable optimized Huffman tables (JPEG only)")
	)
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -input is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *format != "png" && *format != "jpeg" {
		fmt.Fprintf(os.Stderr, "Error: -format must be 'png' or 'jpeg'\n")
		flag.Usage()
		os.Exit(1)
	}

	if *format == "jpeg" && (*lossy || *iterations > 0) {
		fmt.Fprintf(os.Stderr, "Error: -lossy and -iterations are only valid for PNG format\n")
		flag.Usage()
		os.Exit(1)
	}

	if *outputFile == "" {
		ext := ".png"
		if *format == "jpeg" {
			ext = ".jpeg"
		}
		*outputFile = (*inputFile)[:len(*inputFile)-len(getExt(*inputFile))] + ext
	}

	inputBytes, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	inputFileSize := int64(len(inputBytes))

	img, formatName, err := image.Decode(bytes.NewReader(inputBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Decoded %s image: %dx%d\n", formatName, img.Bounds().Dx(), img.Bounds().Dy())
	if inputFileSize > 0 {
		fmt.Printf("Input file size: %d bytes\n", inputFileSize)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var pixels []byte

	switch m := img.(type) {
	case *image.RGBA:
		pixels = m.Pix
	case *image.NRGBA:
		nrgba := m
		pixels = make([]byte, width*height*4)
		for i := 0; i < len(nrgba.Pix); i += 4 {
			pixels[i] = nrgba.Pix[i]
			pixels[i+1] = nrgba.Pix[i+1]
			pixels[i+2] = nrgba.Pix[i+2]
			pixels[i+3] = nrgba.Pix[i+3]
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

	_ = png.ColorRGBA // Suppress unused import warning
	originalSize := len(pixels)

	var encoder *png.Encoder
	var pngData []byte

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

			if *format == "png" && !*lossy {
				pngData, err = png.RecompressPNGBytesLossless(inputBytes, opts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error recompressing PNG: %v\n", err)
					os.Exit(1)
				}
			} else {
				encoder, err = png.NewEncoderWithOptions(opts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating encoder: %v\n", err)
					os.Exit(1)
				}

				pngData, err = encoder.Encode(pixels)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error encoding PNG: %v\n", err)
					os.Exit(1)
				}
			}

			elapsed := time.Since(startTime).Milliseconds()
			sizes = append(sizes, len(pngData))
			totalTime += elapsed

			if *verbose || *benchmarkRuns <= 3 {
				fmt.Printf("  Run %d: %d bytes, %d ms\n", i+1, len(pngData), elapsed)
			}
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
	} else {
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

		startTime := time.Now()

		if *format == "png" {
			if !*lossy && formatName == "png" {
				pngData, err = png.RecompressPNGBytesLossless(inputBytes, opts)
			} else {
				encoder, err = png.NewEncoderWithOptions(opts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating encoder: %v\n", err)
					os.Exit(1)
				}
				pngData, err = encoder.Encode(pixels)
			}
		} else {
			// JPEG encoding
			// Convert RGBA to RGB if needed
			var jpegPixels []byte
			var jpegColorType jpeg.ColorType

			if len(pixels) == width*height*4 {
				// RGBA -> RGB conversion
				jpegPixels = make([]byte, width*height*3)
				for i := 0; i < width*height; i++ {
					jpegPixels[i*3] = pixels[i*4]
					jpegPixels[i*3+1] = pixels[i*4+1]
					jpegPixels[i*3+2] = pixels[i*4+2]
				}
				jpegColorType = jpeg.ColorRGB
			} else if len(pixels) == width*height*3 {
				// Already RGB
				jpegPixels = pixels
				jpegColorType = jpeg.ColorRGB
			} else if len(pixels) == width*height {
				// Grayscale
				jpegPixels = pixels
				jpegColorType = jpeg.ColorGrayscale
			} else {
				fmt.Fprintf(os.Stderr, "Error: unsupported pixel format for JPEG (length: %d, expected: %d or %d or %d)\n",
					len(pixels), width*height, width*height*3, width*height*4)
				os.Exit(1)
			}

			var jpegOpts jpeg.Options
			switch *preset {
			case "fast":
				jpegOpts = jpeg.FastOptions(width, height, uint8(*quality))
			case "max":
				jpegOpts = jpeg.MaxOptions(width, height, uint8(*quality))
			default:
				jpegOpts = jpeg.BalancedOptions(width, height, uint8(*quality))
			}

			// Apply JPEG-specific options
			jpegOpts.ColorType = jpegColorType
			if *progressive {
				jpegOpts.Progressive = true
			}
			if *trellis {
				jpegOpts.TrellisQuant = true
			}
			if *optimizeHuffman {
				jpegOpts.OptimizeHuffman = true
			}
			if *subsampling == "444" {
				jpegOpts.Subsampling = jpeg.Subsampling444
			}

			if *verbose {
				fmt.Printf("JPEG encoding with color type: %v\n", jpegColorType)
				fmt.Printf("JPEG pixel data size: %d bytes\n", len(jpegPixels))
			}

			jpegEncoder, err := jpeg.NewEncoder(jpegOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating JPEG encoder: %v\n", err)
				os.Exit(1)
			}
			pngData, err = jpegEncoder.Encode(jpegPixels)
		}
		elapsed := time.Since(startTime).Milliseconds()

		if err != nil {
			if *format == "png" {
				fmt.Fprintf(os.Stderr, "Error encoding PNG: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Error encoding JPEG: %v\n", err)
			}
			os.Exit(1)
		}

		fmt.Printf("Encoding time: %d ms\n", elapsed)

		if *compare {
			ratio := float64(len(pngData)) / float64(inputFileSize) * 100
			savings := float64(inputFileSize-int64(len(pngData))) / float64(inputFileSize) * 100
			fmt.Printf("Original File: %d bytes\n", inputFileSize)
			fmt.Printf("Raw Pixels:    %d bytes\n", originalSize)
			fmt.Printf("Compressed:    %d bytes\n", len(pngData))
			fmt.Printf("Ratio (vs File): %.2f%%\n", ratio)
			fmt.Printf("Savings (vs File): %.2f%%\n", savings)
		}
	}

	outFile, err := os.Create(*outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}

	if _, err := outFile.Write(pngData); err != nil {
		_ = outFile.Close()
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	if err := outFile.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully compressed to %s (%d bytes)\n", *outputFile, len(pngData))
}

func getExt(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}
