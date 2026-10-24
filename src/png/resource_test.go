package png

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	stdpng "image/png"
	"io"
	"strconv"
	"strings"
	"testing"
)

func TestRecompressRejectsOversizedHeaderBeforeDecode(t *testing.T) {
	for _, size := range [][2]int{{5000, 5000}, {16385, 1}} {
		var input bytes.Buffer
		input.Write(Signature())
		data := make([]byte, 13)
		binary.BigEndian.PutUint32(data, uint32(size[0]))
		binary.BigEndian.PutUint32(data[4:], uint32(size[1]))
		data[8], data[9] = 8, 6
		binary.Write(&input, binary.BigEndian, uint32(len(data)))
		input.WriteString("IHDR")
		input.Write(data)
		binary.Write(&input, binary.BigEndian, crc32.ChecksumIEEE(append([]byte("IHDR"), data...)))
		if err := WriteIEND(&input); err != nil {
			t.Fatal(err)
		}
		_, err := RecompressPNGBytesLossless(input.Bytes(), BalancedOptions(1, 1))
		if err == nil || !strings.Contains(err.Error(), "limit") {
			t.Fatalf("%v: want resource limit before missing pixel data, got %v", size, err)
		}
	}
}

func TestPNGAcceptsDimensionBoundaryAndRejectsTruncatedPixels(t *testing.T) {
	enc, err := NewEncoder(16384, 1, ColorGrayscale)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enc.Encode(make([]byte, 16383)); err == nil {
		t.Fatal("accepted truncated pixels")
	}
	output, err := enc.Encode(make([]byte, 16384))
	if err != nil {
		t.Fatal(err)
	}
	img, err := stdpng.Decode(bytes.NewReader(output))
	if err != nil || img.Bounds().Dx() != 16384 || img.Bounds().Dy() != 1 {
		t.Fatalf("boundary image: %v, %v", img, err)
	}
}

func TestPNGRejectsUnsafeDimensionsBeforePixelWork(t *testing.T) {
	sizes := [][2]int{{0, 8}, {-1, 8}, {5000, 5000}, {16385, 1}}
	if strconv.IntSize == 64 {
		large := int64(1<<61) + 1
		sizes = append(sizes, [2]int{int(large), 8})
		if _, err := NewIHDRData(int(large), 8, 8, 0); err == nil {
			t.Error("IHDR narrowed overflowing width")
		}
	}
	for _, size := range sizes {
		t.Run(strconv.Itoa(size[0]), func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("dimension validation panicked: %v", p)
				}
			}()
			w, h := size[0], size[1]
			if _, err := NewEncoder(w, h, ColorGrayscale); err == nil {
				t.Error("constructor accepted unsafe dimensions")
			}
			opts := FasterOptions(w, h)
			opts.ColorType = ColorGrayscale
			enc, err := NewEncoder(1, 1, ColorGrayscale)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := enc.EncodeWithOptions(make([]byte, 8), opts); err == nil {
				t.Error("override accepted unsafe dimensions")
			}
			if err := WriteIDATWithOptions(io.Discard, make([]byte, 8), w, h, ColorGrayscale, opts); err == nil {
				t.Error("IDAT accepted unsafe dimensions")
			}
			if _, err := IDATDataBytesWithOptions(make([]byte, 8), w, h, ColorGrayscale, opts); err == nil {
				t.Error("IDAT bytes accepted unsafe dimensions")
			}
		})
	}
}
