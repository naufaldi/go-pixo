package png

import (
	"testing"
)

func TestHasAlpha(t *testing.T) {
	t.Run("RGBA with transparent pixels", func(t *testing.T) {
		pixels := []byte{255, 0, 0, 0, 0, 255, 0, 128}
		if !HasAlpha(pixels, ColorRGBA) {
			t.Error("expected HasAlpha to return true for pixels with alpha != 255")
		}
	})

	t.Run("RGBA with all opaque pixels", func(t *testing.T) {
		pixels := []byte{255, 0, 0, 255, 0, 255, 0, 255}
		if HasAlpha(pixels, ColorRGBA) {
			t.Error("expected HasAlpha to return false for all opaque pixels")
		}
	})

	t.Run("RGB color type", func(t *testing.T) {
		pixels := []byte{255, 0, 0, 0, 255, 0}
		if HasAlpha(pixels, ColorRGB) {
			t.Error("expected HasAlpha to return false for RGB color type")
		}
	})

	t.Run("Grayscale color type", func(t *testing.T) {
		pixels := []byte{128, 64}
		if HasAlpha(pixels, ColorGrayscale) {
			t.Error("expected HasAlpha to return false for Grayscale color type")
		}
	})

	t.Run("empty pixels", func(t *testing.T) {
		if HasAlpha([]byte{}, ColorRGBA) {
			t.Error("expected HasAlpha to return false for empty pixels")
		}
	})
}

func TestOptimizeAlpha(t *testing.T) {
	t.Run("RGBA with transparent pixels", func(t *testing.T) {
		pixels := []byte{255, 128, 64, 0, 100, 150, 200, 255}
		result := OptimizeAlpha(pixels, ColorRGBA)

		if result[0] != 0 || result[1] != 0 || result[2] != 0 {
			t.Errorf("expected RGB to be 0,0,0 for transparent pixel, got %d,%d,%d", result[0], result[1], result[2])
		}
		if result[4] != 100 || result[5] != 150 || result[6] != 200 {
			t.Errorf("expected opaque pixel to remain unchanged")
		}
		if result[7] != 255 {
			t.Errorf("expected alpha to remain 255 for opaque pixel")
		}
	})

	t.Run("RGBA with all opaque pixels", func(t *testing.T) {
		pixels := []byte{255, 128, 64, 255, 100, 150, 200, 255}
		result := OptimizeAlpha(pixels, ColorRGBA)

		if result[0] != 255 || result[1] != 128 || result[2] != 64 {
			t.Error("expected all opaque pixels to remain unchanged")
		}
	})

	t.Run("RGB color type", func(t *testing.T) {
		pixels := []byte{255, 128, 64, 100, 150, 200}
		result := OptimizeAlpha(pixels, ColorRGB)

		if len(result) != len(pixels) {
			t.Error("expected same length for RGB")
		}
		for i := range result {
			if result[i] != pixels[i] {
				t.Error("expected RGB pixels to remain unchanged")
			}
		}
	})

	t.Run("Grayscale color type", func(t *testing.T) {
		pixels := []byte{128, 64}
		result := OptimizeAlpha(pixels, ColorGrayscale)

		if len(result) != len(pixels) {
			t.Error("expected same length for Grayscale")
		}
		for i := range result {
			if result[i] != pixels[i] {
				t.Error("expected Grayscale pixels to remain unchanged")
			}
		}
	})

	t.Run("empty pixels", func(t *testing.T) {
		result := OptimizeAlpha([]byte{}, ColorRGBA)
		if len(result) != 0 {
			t.Error("expected empty result for empty input")
		}
	})

	t.Run("single pixel", func(t *testing.T) {
		pixels := []byte{255, 0, 0, 0}
		result := OptimizeAlpha(pixels, ColorRGBA)

		if result[0] != 0 || result[1] != 0 || result[2] != 0 {
			t.Errorf("expected RGB to be 0,0,0 for single transparent pixel")
		}
	})
}
