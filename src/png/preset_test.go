package png

import (
	"bytes"
	"image"
	_ "image/png"
	"testing"
)

func createTestImage(width, height int) []byte {
	pixels := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 4
			// Create a simple pattern that can be compressed
			if (x+y)%2 == 0 {
				pixels[idx] = 255   // R
				pixels[idx+1] = 0   // G
				pixels[idx+2] = 0   // B
				pixels[idx+3] = 255 // A
			} else {
				pixels[idx] = 0     // R
				pixels[idx+1] = 255 // G
				pixels[idx+2] = 0   // B
				pixels[idx+3] = 128 // A (semi-transparent)
			}
		}
	}
	return pixels
}

func TestPresets(t *testing.T) {
	width, height := 100, 100
	pixels := createTestImage(width, height)

	t.Run("FastPreset", func(t *testing.T) {
		opts := FastOptions(width, height)
		data, err := EncodeWithOptions(pixels, opts)
		if err != nil {
			t.Fatalf("Fast preset encoding failed: %v", err)
		}
		verifyPNG(t, data, width, height)
		t.Logf("Fast preset size: %d bytes", len(data))
	})

	t.Run("BalancedPreset", func(t *testing.T) {
		opts := BalancedOptions(width, height)
		data, err := EncodeWithOptions(pixels, opts)
		if err != nil {
			t.Fatalf("Balanced preset encoding failed: %v", err)
		}
		verifyPNG(t, data, width, height)
		t.Logf("Balanced preset size: %d bytes", len(data))
	})

	t.Run("MaxPreset", func(t *testing.T) {
		opts := MaxOptions(width, height)
		data, err := EncodeWithOptions(pixels, opts)
		if err != nil {
			t.Fatalf("Max preset encoding failed: %v", err)
		}
		verifyPNG(t, data, width, height)
		t.Logf("Max preset size: %d bytes", len(data))
	})

	t.Run("Comparison", func(t *testing.T) {
		fastData, _ := EncodeWithOptions(pixels, FastOptions(width, height))
		balancedData, _ := EncodeWithOptions(pixels, BalancedOptions(width, height))
		maxData, _ := EncodeWithOptions(pixels, MaxOptions(width, height))

		t.Logf("Fast: %d, Balanced: %d, Max: %d", len(fastData), len(balancedData), len(maxData))

		if len(balancedData) >= len(fastData) {
			t.Errorf("Balanced preset (%d) should be smaller than Fast preset (%d)", len(balancedData), len(fastData))
		}
		if len(maxData) >= len(balancedData) {
			t.Errorf("Max preset (%d) should be smaller than Balanced preset (%d)", len(maxData), len(balancedData))
		}
	})
}

func verifyPNG(t *testing.T, data []byte, width, height int) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to decode generated PNG: %v", err)
	}
	if format != "png" {
		t.Errorf("Expected format 'png', got '%s'", format)
	}
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("Expected dimensions %dx%d, got %dx%d", width, height, img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func EncodeWithOptions(pixels []byte, opts Options) ([]byte, error) {
	encoder, err := NewEncoderWithOptions(opts)
	if err != nil {
		return nil, err
	}
	return encoder.Encode(pixels)
}

func TestAlphaOptimizationEffect(t *testing.T) {
	width, height := 10, 10
	pixels := make([]byte, width*height*4)
	for i := 0; i < len(pixels); i += 4 {
		pixels[i] = 100   // R
		pixels[i+1] = 150 // G
		pixels[i+2] = 200 // B
		pixels[i+3] = 0   // Fully transparent
	}

	// With Alpha optimization, all R,G,B should become 0, leading to better compression
	optsNoOpt := FastOptions(width, height)
	optsNoOpt.OptimizeAlpha = false
	dataNoOpt, _ := EncodeWithOptions(pixels, optsNoOpt)

	optsOpt := FastOptions(width, height)
	optsOpt.OptimizeAlpha = true
	dataOpt, _ := EncodeWithOptions(pixels, optsOpt)

	t.Logf("No Alpha Opt: %d bytes, With Alpha Opt: %d bytes", len(dataNoOpt), len(dataOpt))
	if len(dataOpt) >= len(dataNoOpt) {
		t.Errorf("Alpha optimization should result in smaller or equal size. Got %d vs %d", len(dataOpt), len(dataNoOpt))
	}
}

func TestColorReductionEffect(t *testing.T) {
	width, height := 10, 10
	pixels := make([]byte, width*height*4)
	for i := 0; i < len(pixels); i += 4 {
		pixels[i] = 100   // R
		pixels[i+1] = 100 // G
		pixels[i+2] = 100 // B
		pixels[i+3] = 255 // Opaque
	}
	// This image is grayscale and opaque. Can be reduced to Grayscale (1 byte per pixel).

	optsNoRed := FastOptions(width, height)
	optsNoRed.ReduceColorType = false
	dataNoRed, _ := EncodeWithOptions(pixels, optsNoRed)

	optsRed := FastOptions(width, height)
	optsRed.ReduceColorType = true
	dataRed, _ := EncodeWithOptions(pixels, optsRed)

	t.Logf("No Color Red: %d bytes, With Color Red: %d bytes", len(dataNoRed), len(dataRed))
	if len(dataRed) >= len(dataNoRed) {
		t.Errorf("Color reduction should result in smaller size. Got %d vs %d", len(dataRed), len(dataNoRed))
	}
}
