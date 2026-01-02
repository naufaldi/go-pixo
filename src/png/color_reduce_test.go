package png

import (
	"testing"
)

func TestReduceToGrayscale(t *testing.T) {
	t.Run("RGB to grayscale", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 200, 200, 200}
		result, newColorType, err := ReduceToGrayscale(pixels, 2, 1, ColorRGB)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if newColorType != ColorGrayscale {
			t.Errorf("expected color type Grayscale, got %v", newColorType)
		}
		if len(result) != 2 {
			t.Errorf("expected length 2, got %d", len(result))
		}
		if result[0] != 100 || result[1] != 200 {
			t.Errorf("expected [100, 200], got %v", result)
		}
	})

	t.Run("RGBA to grayscale", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 255, 200, 200, 200, 255}
		result, newColorType, err := ReduceToGrayscale(pixels, 2, 1, ColorRGBA)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if newColorType != ColorGrayscale {
			t.Errorf("expected color type Grayscale, got %v", newColorType)
		}
		if len(result) != 2 {
			t.Errorf("expected length 2, got %d", len(result))
		}
		if result[0] != 100 || result[1] != 200 {
			t.Errorf("expected [100, 200], got %v", result)
		}
	})

	t.Run("already grayscale", func(t *testing.T) {
		pixels := []byte{100, 200}
		result, newColorType, err := ReduceToGrayscale(pixels, 2, 1, ColorGrayscale)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if newColorType != ColorGrayscale {
			t.Errorf("expected color type Grayscale, got %v", newColorType)
		}
		if len(result) != 2 {
			t.Errorf("expected length 2, got %d", len(result))
		}
	})

	t.Run("cannot reduce non-grayscale", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 200, 100, 200}
		_, _, err := ReduceToGrayscale(pixels, 2, 1, ColorRGB)

		if err == nil {
			t.Error("expected error for non-grayscale pixels")
		}
		if err != ErrCannotReduceColorType {
			t.Errorf("expected ErrCannotReduceColorType, got %v", err)
		}
	})

	t.Run("wrong size", func(t *testing.T) {
		pixels := []byte{100, 100, 100}
		_, _, err := ReduceToGrayscale(pixels, 2, 1, ColorRGB)

		if err == nil {
			t.Error("expected error for wrong size")
		}
	})
}

func TestReduceToRGB(t *testing.T) {
	t.Run("RGBA all opaque to RGB", func(t *testing.T) {
		pixels := []byte{100, 150, 200, 255, 50, 100, 150, 255}
		result, newColorType, err := ReduceToRGB(pixels, 2, 1)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if newColorType != ColorRGB {
			t.Errorf("expected color type RGB, got %v", newColorType)
		}
		if len(result) != 6 {
			t.Errorf("expected length 6, got %d", len(result))
		}
		expected := []byte{100, 150, 200, 50, 100, 150}
		for i := range expected {
			if result[i] != expected[i] {
				t.Errorf("expected %v, got %v", expected, result)
			}
		}
	})

	t.Run("RGBA with transparency", func(t *testing.T) {
		pixels := []byte{100, 150, 200, 128, 50, 100, 150, 255}
		_, _, err := ReduceToRGB(pixels, 2, 1)

		if err == nil {
			t.Error("expected error for pixels with transparency")
		}
		if err != ErrCannotReduceColorType {
			t.Errorf("expected ErrCannotReduceColorType, got %v", err)
		}
	})

	t.Run("wrong size", func(t *testing.T) {
		pixels := []byte{100, 150, 200, 255}
		_, _, err := ReduceToRGB(pixels, 2, 1)

		if err == nil {
			t.Error("expected error for wrong size")
		}
	})
}

func TestColorReduceLargeImages(t *testing.T) {
	width, height := 100, 100

	t.Run("large RGB to grayscale", func(t *testing.T) {
		pixels := make([]byte, width*height*3)
		for i := 0; i < width*height; i++ {
			offset := i * 3
			pixels[offset] = byte(i % 256)
			pixels[offset+1] = pixels[offset]
			pixels[offset+2] = pixels[offset]
		}

		result, newColorType, err := ReduceToGrayscale(pixels, width, height, ColorRGB)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if newColorType != ColorGrayscale {
			t.Errorf("expected color type Grayscale, got %v", newColorType)
		}
		if len(result) != width*height {
			t.Errorf("expected length %d, got %d", width*height, len(result))
		}
	})

	t.Run("large RGBA to RGB", func(t *testing.T) {
		pixels := make([]byte, width*height*4)
		for i := 0; i < width*height; i++ {
			offset := i * 4
			pixels[offset] = byte(i % 256)
			pixels[offset+1] = byte((i + 1) % 256)
			pixels[offset+2] = byte((i + 2) % 256)
			pixels[offset+3] = 255
		}

		result, newColorType, err := ReduceToRGB(pixels, width, height)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if newColorType != ColorRGB {
			t.Errorf("expected color type RGB, got %v", newColorType)
		}
		if len(result) != width*height*3 {
			t.Errorf("expected length %d, got %d", width*height*3, len(result))
		}
	})
}
