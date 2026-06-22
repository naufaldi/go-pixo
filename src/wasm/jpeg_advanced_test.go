package wasm

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"testing"
)

func TestEncodeJpegAdvancedPresetFourDecodes(t *testing.T) {
	width, height := 17, 19
	pixels := makeGradientRGB(width, height)

	output, err := EncodeJpegAdvanced(pixels, width, height, 3, 72, 0, true, false, true, 4)
	if err != nil {
		t.Fatalf("EncodeJpegAdvanced failed: %v", err)
	}

	assertDecodedJpegDimensions(t, output, width, height)
}

func TestEncodeJpegAdvancedCombinationsDecode(t *testing.T) {
	tests := []struct {
		name            string
		width           int
		height          int
		progressive     bool
		trellis         bool
		optimizeHuffman bool
		preset          int
	}{
		{
			name:            "baseline odd dimensions",
			width:           17,
			height:          19,
			progressive:     false,
			trellis:         false,
			optimizeHuffman: true,
			preset:          2,
		},
		{
			name:            "progressive without trellis",
			width:           23,
			height:          21,
			progressive:     true,
			trellis:         false,
			optimizeHuffman: true,
			preset:          4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixels := makeGradientRGB(tt.width, tt.height)

			output, err := EncodeJpegAdvanced(
				pixels,
				tt.width,
				tt.height,
				3,
				75,
				0,
				tt.progressive,
				tt.trellis,
				tt.optimizeHuffman,
				tt.preset,
			)
			if err != nil {
				t.Fatalf("EncodeJpegAdvanced failed: %v", err)
			}

			assertDecodedJpegDimensions(t, output, tt.width, tt.height)
		})
	}
}

func makeGradientRGB(width, height int) []byte {
	pixels := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * 3
			pixels[offset] = uint8((x * 255) / width)
			pixels[offset+1] = uint8((y * 255) / height)
			pixels[offset+2] = uint8(((x + y) * 255) / (width + height))
		}
	}
	return pixels
}

func assertDecodedJpegDimensions(t *testing.T, output []byte, width, height int) {
	t.Helper()

	if len(output) < 4 {
		t.Fatalf("output too short: %d bytes", len(output))
	}
	if output[0] != 0xff || output[1] != 0xd8 {
		t.Fatalf("missing JPEG SOI marker: %x", output[:2])
	}
	if output[len(output)-2] != 0xff || output[len(output)-1] != 0xd9 {
		t.Fatalf("missing JPEG EOI marker: %x", output[len(output)-2:])
	}

	img, format, err := image.Decode(bytes.NewReader(output))
	if err != nil {
		t.Fatalf("failed to decode output: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("expected jpeg format, got %s", format)
	}
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Fatalf("expected dimensions %dx%d, got %dx%d", width, height, img.Bounds().Dx(), img.Bounds().Dy())
	}
}
