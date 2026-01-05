package wasm

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"testing"
)

func TestEncodeJpeg(t *testing.T) {
	width, height := 8, 8
	// 8x8 RGB image (all red)
	pixels := make([]byte, width*height*3)
	for i := 0; i < len(pixels); i += 3 {
		pixels[i] = 255   // R
		pixels[i+1] = 0   // G
		pixels[i+2] = 0   // B
	}

	quality := uint8(75)
	colorType := 3 // RGB

	output, err := EncodeJpeg(pixels, width, height, colorType, quality)
	if err != nil {
		t.Fatalf("EncodeJpeg failed: %v", err)
	}

	if len(output) == 0 {
		t.Fatal("Output is empty")
	}

	// Verify it's a valid JPEG
	img, format, err := image.Decode(bytes.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to decode output: %v", err)
	}

	if format != "jpeg" {
		t.Errorf("Expected format jpeg, got %s", format)
	}

	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("Expected dimensions %dx%d, got %dx%d", width, height, img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEncodeJpeg_Grayscale(t *testing.T) {
	width, height := 8, 8
	// 8x8 Grayscale image
	pixels := make([]byte, width*height)
	for i := 0; i < len(pixels); i++ {
		pixels[i] = uint8(i * 4)
	}

	quality := uint8(75)
	colorType := 1 // Grayscale

	output, err := EncodeJpeg(pixels, width, height, colorType, quality)
	if err != nil {
		t.Fatalf("EncodeJpeg failed: %v", err)
	}

	// Verify it's a valid JPEG
	img, format, err := image.Decode(bytes.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to decode output: %v", err)
	}

	if format != "jpeg" {
		t.Errorf("Expected format jpeg, got %s", format)
	}

	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("Expected dimensions %dx%d, got %dx%d", width, height, img.Bounds().Dx(), img.Bounds().Dy())
	}
}
