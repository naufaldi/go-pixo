package png

import (
	"bytes"
	"image"
	stdpng "image/png"
	"testing"
)

func TestEncodeLargeImageStoredBlocks(t *testing.T) {
	// Create a large image that will exceed the 65535 byte limit for a single stored block.
	// 500x500 RGBA = 1,000,000 bytes of pixel data.
	width, height := 500, 500
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with some data
	for i := 0; i < len(img.Pix); i++ {
		img.Pix[i] = byte(i % 256)
	}

	// Use options that force stored blocks (level 0) or where compression doesn't help much
	opts := FastOptions(width, height)
	opts.CompressionLevel = 0
	opts.ColorType = ColorRGBA

	encoder, err := NewEncoderWithOptions(opts)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	data, err := encoder.Encode(img.Pix)
	if err != nil {
		t.Fatalf("Failed to encode large image: %v", err)
	}

	// Verify the output can be decoded by the standard library
	_, err = stdpng.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to decode output PNG: %v", err)
	}
}

func TestEncodeLargeImageCompressed(t *testing.T) {
	// 500x500 RGBA = 1,000,000 bytes
	width, height := 500, 500
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with compressible data
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, image.White)
		}
	}

	opts := BalancedOptions(width, height)
	opts.ColorType = ColorRGBA

	encoder, err := NewEncoderWithOptions(opts)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	data, err := encoder.Encode(img.Pix)
	if err != nil {
		t.Fatalf("Failed to encode large image: %v", err)
	}

	// Verify the output can be decoded by the standard library
	decoded, err := stdpng.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to decode output PNG: %v", err)
	}

	if decoded.Bounds().Dx() != width || decoded.Bounds().Dy() != height {
		t.Errorf("Decoded image dimensions mismatch: got %dx%d, want %dx%d", 
			decoded.Bounds().Dx(), decoded.Bounds().Dy(), width, height)
	}
}
