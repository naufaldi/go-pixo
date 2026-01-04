package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/mac/go-pixo/src/png"
)

func main() {
	var (
		inputFile       = flag.String("input", "", "Input image file (PNG or JPEG)")
		outputFile      = flag.String("output", "", "Output PNG file (default: input with .png extension)")
		preset          = flag.String("preset", "balanced", "Compression preset: fast, balanced, max, extreme")
		lossy           = flag.Bool("lossy", false, "Enable lossy compression with palette quantization")
		quality         = flag.Int("quality", 75, "Quality level for lossy compression (0-100)")
		compare         = flag.Bool("compare", false, "Show original vs compressed size comparison")
		verbose         = flag.Bool("verbose", false, "Enable detailed output")
		iterations      = flag.Int("iterations", 0, "Number of Zopfli iterations (0-100)")
		ditherStrength  = flag.Float64("dither", 0.5, "Dithering strength 0.0-1.0 (default: 0.5)")
		maxColors       = flag.Int("max-colors", 256, "Maximum number of colors for lossy compression (2-256)")
		benchmark       = flag.Bool("benchmark", false, "Run compression multiple times and report statistics")
		benchmarkRuns   = flag.Int("benchmark-runs", 3, "Number of benchmark runs")
	)
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -input is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *outputFile == "" {
		*outputFile = (*inputFile)[:len(*inputFile)-len(getExt(*inputFile))] + ".png"
	}

	file, err := os.Open(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Get input file size for comparison
	fileInfo, err := file.Stat()
	inputFileSize := int64(0)
	if err == nil {
		inputFileSize = fileInfo.Size()
	}

	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Decoded %s image: %dx%d\n", format, img.Bounds().Dx(), img.Bounds().Dy())
	if inputFileSize > 0 {
		fmt.Printf("Input file size: %d bytes\n", inputFileSize)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var pixels []byte

	switch img.(type) {
	case *image.RGBA:
		rgba := img.(*image.RGBA)
		pixels = rgba.Pix
	case *image.NRGBA:
		nrgba := img.(*image.NRGBA)
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

		encoder, err = png.NewEncoderWithOptions(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating encoder: %v\n", err)
			os.Exit(1)
		}

		startTime := time.Now()
		pngData, err = encoder.Encode(pixels)
		elapsed := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding PNG: %v\n", err)
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
	defer outFile.Close()

	_, err = outFile.Write(pngData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
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
