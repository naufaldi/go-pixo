package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/mac/go-pixo/src/png"
)

func main() {
	var (
		inputFile  = flag.String("input", "", "Input image file (PNG or JPEG)")
		outputFile = flag.String("output", "", "Output PNG file (default: input with .png extension)")
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

	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Decoded %s image: %dx%d\n", format, img.Bounds().Dx(), img.Bounds().Dy())

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var colorType png.ColorType
	var pixels []byte

	switch img.(type) {
	case *image.RGBA:
		colorType = png.ColorRGBA
		rgba := img.(*image.RGBA)
		pixels = rgba.Pix
	case *image.NRGBA:
		colorType = png.ColorRGBA
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
		colorType = png.ColorRGBA
		pixels = rgba.Pix
	}

	encoder, err := png.NewEncoder(width, height, colorType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating encoder: %v\n", err)
		os.Exit(1)
	}

	pngData, err := encoder.Encode(pixels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding PNG: %v\n", err)
		os.Exit(1)
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
