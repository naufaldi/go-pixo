package png

import (
	"testing"
)

func TestIsGrayscale(t *testing.T) {
	t.Run("Grayscale color type", func(t *testing.T) {
		pixels := []byte{128, 64}
		if !IsGrayscale(pixels, ColorGrayscale) {
			t.Error("expected Grayscale color type to return true")
		}
	})

	t.Run("RGB grayscale pixels", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 200, 200, 200}
		if !IsGrayscale(pixels, ColorRGB) {
			t.Error("expected RGB grayscale pixels to return true")
		}
	})

	t.Run("RGB non-grayscale pixels", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 200, 100, 200}
		if IsGrayscale(pixels, ColorRGB) {
			t.Error("expected RGB non-grayscale pixels to return false")
		}
	})

	t.Run("RGBA grayscale opaque pixels", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 255, 200, 200, 200, 255}
		if !IsGrayscale(pixels, ColorRGBA) {
			t.Error("expected RGBA grayscale opaque pixels to return true")
		}
	})

	t.Run("RGBA non-grayscale pixels", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 255, 200, 100, 200, 255}
		if IsGrayscale(pixels, ColorRGBA) {
			t.Error("expected RGBA non-grayscale pixels to return false")
		}
	})

	t.Run("empty pixels", func(t *testing.T) {
		if !IsGrayscale([]byte{}, ColorRGB) {
			t.Error("expected empty pixels to return true")
		}
	})
}

func TestCanReduceToGrayscale(t *testing.T) {
	t.Run("RGB grayscale", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 200, 200, 200}
		if !CanReduceToGrayscale(pixels, 2, 1, ColorRGB) {
			t.Error("expected RGB grayscale to be reducible")
		}
	})

	t.Run("RGB non-grayscale", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 200, 100, 200}
		if CanReduceToGrayscale(pixels, 2, 1, ColorRGB) {
			t.Error("expected RGB non-grayscale to not be reducible")
		}
	})

	t.Run("RGBA grayscale opaque", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 255, 200, 200, 200, 255}
		if !CanReduceToGrayscale(pixels, 2, 1, ColorRGBA) {
			t.Error("expected RGBA grayscale opaque to be reducible")
		}
	})

	t.Run("wrong size", func(t *testing.T) {
		pixels := []byte{100, 100, 100, 200, 200, 200}
		if CanReduceToGrayscale(pixels, 2, 2, ColorRGB) {
			t.Error("expected wrong size to not be reducible")
		}
	})
}

func TestCanReduceToRGB(t *testing.T) {
	t.Run("RGBA all opaque", func(t *testing.T) {
		pixels := []byte{100, 150, 200, 255, 50, 100, 150, 255}
		if !CanReduceToRGB(pixels, 2, 1) {
			t.Error("expected RGBA all opaque to be reducible to RGB")
		}
	})

	t.Run("RGBA with transparency", func(t *testing.T) {
		pixels := []byte{100, 150, 200, 255, 50, 100, 150, 128}
		if CanReduceToRGB(pixels, 2, 1) {
			t.Error("expected RGBA with transparency to not be reducible to RGB")
		}
	})

	t.Run("RGBA all zero alpha", func(t *testing.T) {
		pixels := []byte{100, 150, 200, 0, 50, 100, 150, 0}
		if CanReduceToRGB(pixels, 2, 1) {
			t.Error("expected RGBA with zero alpha to not be reducible to RGB")
		}
	})

	t.Run("wrong size", func(t *testing.T) {
		pixels := []byte{100, 150, 200, 255}
		if CanReduceToRGB(pixels, 2, 1) {
			t.Error("expected wrong size to not be reducible")
		}
	})
}

func TestColorAnalysisLargeImages(t *testing.T) {
	width, height := 100, 100

	t.Run("large grayscale image", func(t *testing.T) {
		pixels := make([]byte, width*height*3)
		for i := 0; i < len(pixels); i += 3 {
			pixels[i] = 128
			pixels[i+1] = 128
			pixels[i+2] = 128
		}
		if !CanReduceToGrayscale(pixels, width, height, ColorRGB) {
			t.Error("expected large grayscale image to be reducible")
		}
	})

	t.Run("large non-grayscale image", func(t *testing.T) {
		pixels := make([]byte, width*height*3)
		for i := 0; i < len(pixels); i += 3 {
			pixels[i] = byte(i % 256)
			pixels[i+1] = byte((i + 1) % 256)
			pixels[i+2] = byte((i + 2) % 256)
		}
		if CanReduceToGrayscale(pixels, width, height, ColorRGB) {
			t.Error("expected large non-grayscale image to not be reducible")
		}
	})
}
