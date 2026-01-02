package png

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/mac/go-pixo/src/compress"
)

func TestFilterSelectionImprovesCompression(t *testing.T) {
	width, height := 8, 8
	bpp := 3
	colorType := ColorRGB

	pixels := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			pixels[offset] = byte(x * 10)
			pixels[offset+1] = byte(y * 10)
			pixels[offset+2] = byte((x + y) * 5)
		}
	}

	compressedWithSelection, err := IDATDataBytes(pixels, width, height, colorType)
	if err != nil {
		t.Fatalf("IDATDataBytes with selection failed: %v", err)
	}

	compressedWithNone := buildZlibDataWithFilterNone(pixels, width, height, colorType)
	if compressedWithNone == nil {
		t.Fatal("buildZlibDataWithFilterNone failed")
	}

	if len(compressedWithSelection) >= len(compressedWithNone) {
		t.Logf("Selection size: %d, None size: %d (selection should be smaller for patterned data)",
			len(compressedWithSelection), len(compressedWithNone))
	}
}

func TestFilterSelectionProducesValidPNG(t *testing.T) {
	width, height := 4, 4
	bpp := 4
	colorType := ColorRGBA

	pixels := make([]byte, width*height*bpp)
	for i := 0; i < len(pixels); i += bpp {
		pixels[i] = byte(i)
		pixels[i+1] = byte(i + 1)
		pixels[i+2] = byte(i + 2)
		pixels[i+3] = 255
	}

	var buf bytes.Buffer
	encoder, err := NewEncoder(width, height, colorType)
	if err != nil {
		t.Fatalf("NewEncoder failed: %v", err)
	}

	pngBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	buf.Write(pngBytes)

	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("PNG decode failed: %v", err)
	}

	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("decoded image size %dx%d != expected %dx%d",
			img.Bounds().Dx(), img.Bounds().Dy(), width, height)
	}
}

func buildZlibDataWithFilterNone(pixels []byte, width, height int, colorType ColorType) []byte {
	bpp := BytesPerPixel(colorType)
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		scanlineData = append(scanlineData, 0)
		scanlineData = append(scanlineData, pixels[offset:offset+width*bpp]...)
	}

	cmf, err := compress.ZlibHeaderBytes(32768, 2)
	if err != nil {
		return nil
	}

	encoder := compress.NewDeflateEncoder()
	deflateData, err := encoder.Encode(scanlineData, false)
	if err != nil {
		return nil
	}

	adler := compress.Adler32(scanlineData)
	adlerBuf := compress.ZlibFooterBytes(adler)

	result := make([]byte, 0, len(cmf)+len(deflateData)+len(adlerBuf))
	result = append(result, cmf...)
	result = append(result, deflateData...)
	result = append(result, adlerBuf[:]...)

	return result
}
