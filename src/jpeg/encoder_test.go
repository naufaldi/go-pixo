package jpeg

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"testing"
)

func TestEncoder_1x1(t *testing.T) {
	width, height := 1, 1
	pixels := []byte{255, 0, 0} // Single Red pixel
	encoder, _ := NewEncoder(width, height, ColorRGB, 75)
	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify it can be decoded by standard library
	img, format, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("Failed to decode produced JPEG: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format jpeg, got %s", format)
	}
	if img.Bounds().Dx() != 1 || img.Bounds().Dy() != 1 {
		t.Errorf("expected 1x1, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEncoder_Grayscale(t *testing.T) {
	width, height := 8, 8
	pixels := make([]byte, width*height)
	for i := range pixels {
		pixels[i] = uint8(i * 4)
	}
	encoder, _ := NewEncoder(width, height, ColorGrayscale, 75)
	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	_, format, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("Failed to decode produced JPEG: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format jpeg, got %s", format)
	}
}

func TestEncoder_NonMultipleOf8(t *testing.T) {
	width, height := 10, 10
	pixels := make([]byte, width*height*3)
	encoder, _ := NewEncoder(width, height, ColorRGB, 75)
	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 10 {
		t.Errorf("expected 10x10, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEncoder_QualityLevels(t *testing.T) {
	width, height := 32, 32
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	eLow, _ := NewEncoder(width, height, ColorRGB, 10)
	jpegLow, _ := eLow.Encode(pixels)

	eHigh, _ := NewEncoder(width, height, ColorRGB, 90)
	jpegHigh, _ := eHigh.Encode(pixels)

	if len(jpegLow) >= len(jpegHigh) {
		t.Errorf("expected low quality JPEG (%d) to be smaller than high quality (%d)", len(jpegLow), len(jpegHigh))
	}
}
