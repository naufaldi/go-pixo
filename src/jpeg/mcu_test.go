package jpeg

import (
	"testing"
)

func TestExtractMCU420(t *testing.T) {
	// Create a 2x2 RGB image where each pixel is a different color
	// (0,0)=Red, (1,0)=Green, (0,1)=Blue, (1,1)=White
	data := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 255,
	}
	width, height := 2, 2

	y, cb, cr := ExtractMCU420(data, width, height, 0, 0)

	// Y blocks: y[0][0] should be Red, others should be padding (White)
	// Red (255,0,0) -> Y=77 -> Y-128 = -51
	if y[0][0] != -51.0 {
		t.Errorf("Y[0][0]: got %f, want -51.0", y[0][0])
	}

	// Cb/Cr at (0,0) should be average of the 2x2 image
	// Red (77, 85, 255)
	// Green (150, 43, 21) -> Wait, YCbCr(0,255,0) = (149, 43, 21)
	// Blue (0,0,255) -> YCbCr(0,0,255) = (29, 255, 107)
	// White (255,255,255) -> YCbCr(255,128,128)
	//
	// Red: Cb=85, Cr=255
	// Green: Cb=43, Cr=21
	// Blue: Cb=255, Cr=107
	// White: Cb=128, Cr=128
	//
	// Cb Average = (85 + 43 + 255 + 128) / 4 = 511 / 4 = 127.75
	// Cr Average = (255 + 21 + 107 + 128) / 4 = 511 / 4 = 127.75
	//
	// Level shifted: 127.75 - 128 = -0.25

	if cb[0] != -0.25 || cr[0] != -0.25 {
		t.Errorf("CbCr[0]: got Cb=%f, Cr=%f, want -0.25", cb[0], cr[0])
	}
}
