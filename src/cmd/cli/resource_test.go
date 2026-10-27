package main

import (
	"bytes"
	"image"
	stdpng "image/png"
	"strings"
	"testing"

	"github.com/mac/go-pixo/src/png"
)

func TestDecodeInputChecksHeaderBeforeFullDecode(t *testing.T) {
	var b bytes.Buffer
	b.Write(png.Signature())
	ihdr, err := png.NewIHDRData(5000, 5000, 8, 6)
	if err != nil {
		t.Fatal(err)
	}
	if err = png.WriteIHDR(&b, ihdr); err != nil {
		t.Fatal(err)
	}
	if _, _, err = decodeInput(b.Bytes()); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("want limit before missing pixels, got %v", err)
	}
	b.Reset()
	if err = stdpng.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	img, format, err := decodeInput(b.Bytes())
	if err != nil || format != "png" || img.Bounds().Dx() != 2 || img.Bounds().Dy() != 3 {
		t.Fatalf("ordinary decode: %v %s %v", img, format, err)
	}
}

func TestResizeChecksBeforeAllocation(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	for _, size := range [][2]int{{-1, 2}, {5000, 5000}, {16385, 0}, {int(^uint(0) >> 1), 1}} {
		if _, err := resizeImage(img, size[0], size[1]); err == nil {
			t.Errorf("accepted resize %v", size)
		}
	}
}
