package png

import (
	"bytes"
	"hash/crc32"
	"image"
	"image/color"
	stdpng "image/png"
	"os"
	"testing"
)

func TestRecompressPNGBytesLossless_CursorMeetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fixture-based test in short mode")
	}

	inputPath := "../../images/cursor-meetup.png"
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skipf("fixture not found: %s", inputPath)
	}
	in, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read %s: %v", inputPath, err)
	}

	out, err := RecompressPNGBytesLossless(in, BalancedOptions(1, 1))
	if err != nil {
		t.Fatalf("RecompressPNGBytesLossless: %v", err)
	}

	if len(out) >= len(in) {
		t.Fatalf("expected output smaller than input: in=%d out=%d", len(in), len(out))
	}

	inImg, err := stdpng.Decode(bytes.NewReader(in))
	if err != nil {
		t.Fatalf("decode input png: %v", err)
	}
	outImg, err := stdpng.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode output png: %v", err)
	}

	if !inImg.Bounds().Eq(outImg.Bounds()) {
		t.Fatalf("bounds mismatch: in=%v out=%v", inImg.Bounds(), outImg.Bounds())
	}

	inSum := checksumNRGBA(inImg)
	outSum := checksumNRGBA(outImg)
	if inSum != outSum {
		t.Fatalf("pixel checksum mismatch: in=%d out=%d", inSum, outSum)
	}
}

func TestRecompressPNGBytesLossless_OutputDecodes(t *testing.T) {
	in := mustEncodeTinyPNG(t)

	out, err := RecompressPNGBytesLossless(in, BalancedOptions(1, 1))
	if err != nil {
		t.Fatalf("RecompressPNGBytesLossless: %v", err)
	}

	if _, err := stdpng.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("decode output: %v", err)
	}
}

func mustEncodeTinyPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
	img.SetNRGBA(0, 1, color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	var buf bytes.Buffer
	if err := stdpng.Encode(&buf, img); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	return buf.Bytes()
}

func checksumNRGBA(img image.Image) uint32 {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	pix := make([]byte, 0, w*h*4)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			pix = append(pix, c.R, c.G, c.B, c.A)
		}
	}
	return crc32.ChecksumIEEE(pix)
}
