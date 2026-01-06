package jpeg

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"testing"
)

func TestProgressiveJPEG(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.Progressive = true
	
	encoder, _ := NewEncoder(opts)
	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Progressive encode failed: %v", err)
	}

	// Verify it can be decoded
	_, format, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("Failed to decode progressive JPEG: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("Expected format jpeg, got %s", format)
	}
}

func TestProgressiveJPEG_Grayscale(t *testing.T) {
	width, height := 32, 32
	pixels := make([]byte, width*height)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.Progressive = true
	opts.ColorType = ColorGrayscale
	
	encoder, _ := NewEncoder(opts)
	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Progressive grayscale encode failed: %v", err)
	}

	_, format, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("Failed to decode progressive grayscale JPEG: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("Expected format jpeg, got %s", format)
	}
}
