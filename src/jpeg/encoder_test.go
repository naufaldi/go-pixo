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
	opts := BalancedOptions(width, height, 75)
	encoder, _ := NewEncoder(opts)
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
	opts := BalancedOptions(width, height, 75)
	opts.ColorType = ColorGrayscale
	encoder, _ := NewEncoder(opts)
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
	opts := BalancedOptions(width, height, 75)
	encoder, _ := NewEncoder(opts)
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

	optsLow := BalancedOptions(width, height, 10)
	eLow, _ := NewEncoder(optsLow)
	jpegLow, _ := eLow.Encode(pixels)

	optsHigh := BalancedOptions(width, height, 90)
	eHigh, _ := NewEncoder(optsHigh)
	jpegHigh, _ := eHigh.Encode(pixels)

	if len(jpegLow) >= len(jpegHigh) {
		t.Errorf("expected low quality JPEG (%d) to be smaller than high quality (%d)", len(jpegLow), len(jpegHigh))
	}
}

func TestOptionsBuilder(t *testing.T) {
	width, height := 64, 64
	builder := NewOptionsBuilder(width, height)
	opts := builder.Quality(85).
		Subsampling(Subsampling420).
		OptimizeHuffman(true).
		Progressive(true).
		Build()

	if opts.Quality != 85 {
		t.Errorf("expected quality 85, got %d", opts.Quality)
	}
	if opts.Subsampling != Subsampling420 {
		t.Errorf("expected subsampling 420")
	}
	if !opts.OptimizeHuffman {
		t.Errorf("expected OptimizeHuffman true")
	}
	if !opts.Progressive {
		t.Errorf("expected Progressive true")
	}
}

func TestPresets(t *testing.T) {
	width, height := 64, 64
	quality := uint8(75)

	fast := FastOptions(width, height, quality)
	if fast.Subsampling != Subsampling444 || fast.OptimizeHuffman || fast.Progressive {
		t.Errorf("Fast preset incorrect: %+v", fast)
	}

	balanced := BalancedOptions(width, height, quality)
	if balanced.Subsampling != Subsampling420 || balanced.OptimizeHuffman || balanced.Progressive {
		t.Errorf("Balanced preset incorrect: %+v", balanced)
	}

	max := MaxOptions(width, height, quality)
	if max.Subsampling != Subsampling420 || !max.OptimizeHuffman || !max.Progressive {
		t.Errorf("Max preset incorrect: %+v", max)
	}
}
