package jpeg

import (
	"testing"
)

func TestExtractBlock(t *testing.T) {
	// 2x2 RGB image
	data := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 255,
	}
	width, height := 2, 2

	t.Run("ExtractRGB", func(t *testing.T) {
		y, cb, cr := ExtractBlock(data, width, height, 0, 0, ColorRGB)

		// Check first pixel (0,0) - Red (255,0,0)
		// Y = 77, Cb = 85, Cr = 255 (from previous color tests)
		// Level shifted: Y = -51, Cb = -43, Cr = 127
		if y[0] != -51.0 || cb[0] != -43.0 || cr[0] != 127.0 {
			t.Errorf("Pixel(0,0): got Y=%f, Cb=%f, Cr=%f, want Y=-51, Cb=-43, Cr=127", y[0], cb[0], cr[0])
		}

		// Check padding at (7,7) - should be same as (1,1) - White (255,255,255)
		// Y = 255, Cb = 128, Cr = 128
		// Level shifted: Y = 127, Cb = 0, Cr = 0
		if y[63] != 127.0 || cb[63] != 0.0 || cr[63] != 0.0 {
			t.Errorf("Pixel(7,7) padding: got Y=%f, Cb=%f, Cr=%f, want Y=127, Cb=0, Cr=0", y[63], cb[63], cr[63])
		}
	})

	t.Run("ExtractGrayscale", func(t *testing.T) {
		gData := []byte{100, 200, 50, 150}
		y, cb, cr := ExtractBlock(gData, 2, 2, 0, 0, ColorGrayscale)

		// Check first pixel (0,0) - 100
		// Level shifted: 100 - 128 = -28
		if y[0] != -28.0 || cb[0] != 0.0 || cr[0] != 0.0 {
			t.Errorf("Pixel(0,0): got Y=%f, Cb=%f, Cr=%f, want Y=-28, Cb=0, Cr=0", y[0], cb[0], cr[0])
		}
	})
}
