package wasm

import (
	"bytes"
	"image"
	"image/color"
	stdjpeg "image/jpeg"
	stdpng "image/png"
	"testing"
)

func TestValidateImageBytes(t *testing.T) {
	img := image.NewNRGBA64(image.Rect(0, 0, 2, 1))
	img.SetNRGBA64(0, 0, color.NRGBA64{R: 12345, A: 65535})
	var png, jpeg bytes.Buffer
	if err := stdpng.Encode(&png, img); err != nil {
		t.Fatal(err)
	}
	if err := stdjpeg.Encode(&jpeg, img, nil); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		data  []byte
		valid bool
	}{
		{"16-bit PNG", png.Bytes(), true},
		{"JPEG", jpeg.Bytes(), true},
		{"truncated PNG", png.Bytes()[:33], false},
		{"truncated JPEG", []byte{255, 216, 255, 192, 0, 7, 8, 0, 1, 0, 1}, false},
		{"empty", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := bytes.Clone(tc.data)
			if err := ValidateImageBytes(tc.data); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
			if !bytes.Equal(before, tc.data) {
				t.Fatal("validation changed original bytes")
			}
		})
	}
}
