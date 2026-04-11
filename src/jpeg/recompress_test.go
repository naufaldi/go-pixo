package jpeg

import (
	"bytes"
	stdjpeg "image/jpeg"
	"testing"
)

// TestRecompressJPEGBytes_ValidOutput encodes a synthetic JPEG, then recompresses it.
// The result must be a valid JPEG that decodes to the same dimensions.
func TestRecompressJPEGBytes_ValidOutput(t *testing.T) {
	// 1. Generate 16x16 RGB pixels (solid color, all 128)
	width, height := 16, 16
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = 128
	}

	// 2. Encode as JPEG with BalancedOptions
	opts := BalancedOptions(width, height, 85)
	enc, err := NewEncoder(opts)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	original, err := enc.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// 3. Recompress with MaxOptions
	recompressOpts := MaxOptions(width, height, 85)
	result, err := RecompressJPEGBytes(original, recompressOpts)
	if err != nil {
		t.Fatalf("RecompressJPEGBytes: %v", err)
	}

	// 4. Result must be valid JPEG
	img, err := stdjpeg.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("output not a valid JPEG: %v", err)
	}
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("dimensions changed: got %dx%d want %dx%d",
			img.Bounds().Dx(), img.Bounds().Dy(), width, height)
	}
}

// TestRecompressJPEGBytes_NeverLarger ensures the function never returns output
// larger than the input (it should return the original bytes in that case).
func TestRecompressJPEGBytes_NeverLarger(t *testing.T) {
	width, height := 8, 8
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := MaxOptions(width, height, 95) // Already well-compressed
	enc, err := NewEncoder(opts)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	original, err := enc.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	result, err := RecompressJPEGBytes(original, opts)
	if err != nil {
		t.Fatalf("RecompressJPEGBytes: %v", err)
	}

	if len(result) > len(original) {
		t.Errorf("result (%d bytes) larger than original (%d bytes)", len(result), len(original))
	}
}

// TestRecompressJPEGBytes_InvalidInput ensures garbage input returns an error.
func TestRecompressJPEGBytes_InvalidInput(t *testing.T) {
	garbage := []byte("not a jpeg file at all")
	opts := BalancedOptions(1, 1, 85)
	_, err := RecompressJPEGBytes(garbage, opts)
	if err == nil {
		t.Fatal("expected error for invalid JPEG input, got nil")
	}
}
