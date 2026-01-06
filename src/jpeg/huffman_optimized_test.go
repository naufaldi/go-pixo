package jpeg

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"testing"
)

func TestOptimizedHuffman(t *testing.T) {
	width, height := 64, 64
	// Create an image with repetitive patterns to benefit from custom Huffman
	pixels := make([]byte, width*height*3)
	for i := 0; i < len(pixels); i++ {
		pixels[i] = uint8((i / 3) % 10) // Very few unique values
	}

	// 1. Encode with standard tables
	optsStd := BalancedOptions(width, height, 75)
	optsStd.OptimizeHuffman = false
	eStd, _ := NewEncoder(optsStd)
	bytesStd, err := eStd.Encode(pixels)
	if err != nil {
		t.Fatalf("Standard encode failed: %v", err)
	}

	// 2. Encode with optimized tables
	optsOpt := BalancedOptions(width, height, 75)
	optsOpt.OptimizeHuffman = true
	eOpt, _ := NewEncoder(optsOpt)
	bytesOpt, err := eOpt.Encode(pixels)
	if err != nil {
		t.Fatalf("Optimized encode failed: %v", err)
	}

	if len(bytesOpt) >= len(bytesStd) {
		t.Errorf("Expected optimized JPEG (%d) to be smaller than standard (%d)", len(bytesOpt), len(bytesStd))
	}

	// 3. Verify optimized JPEG decodes correctly
	_, format, err := image.Decode(bytes.NewReader(bytesOpt))
	if err != nil {
		t.Fatalf("Failed to decode optimized JPEG: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("Expected format jpeg, got %s", format)
	}
}
