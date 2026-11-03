package jpeg

import (
	"bytes"
	"errors"
	"runtime"
	"testing"
)

// rgbPixels drops alpha from photoLike's image.
func rgbPixels(width, height int) []byte {
	img := photoLike(width, height, 128)
	pixels := make([]byte, 0, width*height*3)
	for i := 0; i < len(img.Pix); i += 4 {
		pixels = append(pixels, img.Pix[i:i+3]...)
	}
	return pixels
}

// Parallel workers each encode a band of MCU rows; the assembled file must be
// byte-identical to the serial encode, and the serial encode must still match
// the previous baseline-then-recode pipeline.
func TestEncodeBandsMatchesSerialEncode(t *testing.T) {
	// 203x157 has partial MCUs on the right and bottom edges, so the last band
	// exercises edge replication; the uneven split puts a single-row band
	// between larger ones.
	const width, height = 203, 157
	rgb := rgbPixels(width, height)
	gray := grayLike(width, height, 128).Pix
	tests := []struct {
		name   string
		opts   Options
		pixels []byte
	}{
		{"smaller/balanced 4:2:0 trellis progressive", MaxOptions(width, height, 85), rgb},
		{"optimized sequential 4:2:0", Options{Width: width, Height: height, ColorType: ColorRGB, Quality: 80, Subsampling: Subsampling420, OptimizeHuffman: true, UseSIMD: true}, rgb},
		{"faster baseline standard tables", BalancedOptions(width, height, 88), rgb},
		{"4:4:4 progressive", Options{Width: width, Height: height, ColorType: ColorRGB, Quality: 90, Subsampling: Subsampling444, Progressive: true, OptimizeHuffman: true}, rgb},
		{"grayscale trellis progressive", Options{Width: width, Height: height, ColorType: ColorGrayscale, Quality: 75, Subsampling: Subsampling444, Progressive: true, OptimizeHuffman: true, TrellisQuant: true, TrellisLambda: 1}, gray},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewEncoder(tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			serial, err := enc.Encode(tt.pixels)
			if err != nil {
				t.Fatal(err)
			}
			if tt.opts.Progressive || tt.opts.OptimizeHuffman {
				baseline, baseErr := enc.encodeSequential(tt.pixels)
				if baseErr != nil {
					t.Fatal(baseErr)
				}
				layout := sequentialLayouts
				if tt.opts.Progressive {
					layout = progressiveLayouts
				}
				legacy, recodeErr := optimizeLossless(baseline, layout)
				if recodeErr != nil {
					t.Fatal(recodeErr)
				}
				if !bytes.Equal(serial, legacy) {
					t.Fatalf("serial encode differs from baseline-then-recode: %d vs %d bytes", len(serial), len(legacy))
				}
			}

			rows := enc.MCURows()
			bpp := len(tt.pixels) / (width * height)
			var bands [][]int16
			for _, span := range [][2]int{{0, 4}, {4, 1}, {5, rows - 5}} {
				y, h := enc.BandPixelRows(span[0], span[1])
				band, bandErr := enc.EncodeBand(tt.pixels[y*width*bpp:(y+h)*width*bpp], span[0], span[1])
				if bandErr != nil {
					t.Fatal(bandErr)
				}
				bands = append(bands, band)
			}
			parallel, err := enc.EncodeBands(bands)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(parallel, serial) {
				t.Fatalf("banded encode differs from serial: %d vs %d bytes", len(parallel), len(serial))
			}

			if _, err := enc.EncodeBands(bands[:2]); !errors.Is(err, ErrInvalidBand) {
				t.Fatalf("missing rows: got %v, want ErrInvalidBand", err)
			}
		})
	}
}

// A band worker holds only its band's coefficients, not the whole image's
// (#128 review): one MCU row of a 2048x2048 4:4:4 image allocates about
// 2048*8*3 coefficients, while the whole frame would be 2048*2048*3.
func TestEncodeBandAllocatesOnlyItsBand(t *testing.T) {
	const width, height = 2048, 2048
	enc, err := NewEncoder(Options{Width: width, Height: height, ColorType: ColorRGB, Quality: 85, Subsampling: Subsampling444})
	if err != nil {
		t.Fatal(err)
	}
	_, rows := enc.BandPixelRows(0, 1)
	pixels := make([]byte, width*rows*3)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	if _, err := enc.EncodeBand(pixels, 0, 1); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	frameBytes := uint64(width * height * 3 * 2) // int16 coefficients
	if got := after.TotalAlloc - before.TotalAlloc; got > frameBytes/8 {
		t.Fatalf("EncodeBand allocated %d bytes for one MCU row; whole frame is %d", got, frameBytes)
	}
}
