package jpeg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	stdjpeg "image/jpeg"
	"testing"
)

func exifOrientationSegment(orientation uint16, order binary.ByteOrder) []byte {
	segment := make([]byte, 36)
	copy(segment, []byte{0xff, 0xe1, 0, 34, 'E', 'x', 'i', 'f', 0, 0})
	tiff := segment[10:]
	if order == binary.LittleEndian {
		copy(tiff, "II")
	} else {
		copy(tiff, "MM")
	}
	order.PutUint16(tiff[2:], 42)
	order.PutUint32(tiff[4:], 8)
	order.PutUint16(tiff[8:], 1)
	order.PutUint16(tiff[10:], 0x0112)
	order.PutUint16(tiff[12:], 3)
	order.PutUint32(tiff[14:], 1)
	order.PutUint16(tiff[18:], orientation)
	return segment
}

func TestRecompressJPEGBytes_Orientation(t *testing.T) {
	const width, height = 96, 64
	colors := []color.NRGBA{
		{R: 240, G: 16, B: 16, A: 255}, {R: 16, G: 240, B: 16, A: 255},
		{R: 16, G: 16, B: 240, A: 255}, {R: 240, G: 240, B: 16, A: 255},
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, colors[(y/(height/2))*2+x/(width/2)])
		}
	}
	var encoded bytes.Buffer
	if err := stdjpeg.Encode(&encoded, img, &stdjpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	// A legal COM segment forces the recompressed candidate to win, even for
	// simple fixtures that otherwise return the original bytes unchanged.
	padding := make([]byte, 32_004)
	copy(padding, []byte{0xff, 0xfe, 0x7d, 0x02})
	cases := []struct {
		orientation uint16
		corners     [4]int
	}{
		{1, [4]int{0, 1, 2, 3}},
		{2, [4]int{1, 0, 3, 2}},
		{3, [4]int{3, 2, 1, 0}},
		{4, [4]int{2, 3, 0, 1}},
		{5, [4]int{0, 2, 1, 3}},
		{6, [4]int{2, 0, 3, 1}},
		{7, [4]int{3, 1, 2, 0}},
		{8, [4]int{1, 3, 0, 2}},
	}
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%s/%d", order, tc.orientation), func(t *testing.T) {
				// EXIF may follow SOF; DecodeConfig alone does not read this far.
				jpeg := encoded.Bytes()
				scan := bytes.Index(jpeg, []byte{0xff, 0xda})
				if scan < 0 {
					t.Fatal("fixture has no scan")
				}
				input := append([]byte{}, jpeg[:scan]...)
				input = append(input, exifOrientationSegment(tc.orientation, order)...)
				input = append(input, padding...)
				input = append(input, jpeg[scan:]...)
				out, err := RecompressJPEGBytes(input, BalancedOptions(1, 1, 85))
				if err != nil {
					t.Fatal(err)
				}
				if len(out) >= len(input) || bytes.Equal(out, input) {
					t.Fatal("fixture did not exercise the smaller recompressed candidate")
				}
				orientation, err := jpegOrientation(out)
				if err != nil || orientation != 1 {
					t.Fatalf("output must be upright without EXIF transforms: %d, %v", orientation, err)
				}
				decoded, err := stdjpeg.Decode(bytes.NewReader(out))
				if err != nil {
					t.Fatal(err)
				}
				w, h := width, height
				if tc.orientation >= 5 {
					w, h = h, w
				}
				if decoded.Bounds() != image.Rect(0, 0, w, h) {
					t.Fatalf("display bounds: got %v, want %dx%d", decoded.Bounds(), w, h)
				}
				for corner, want := range tc.corners {
					x := w/4 + (corner%2)*(w/2)
					y := h/4 + (corner/2)*(h/2)
					got := color.NRGBAModel.Convert(decoded.At(x, y)).(color.NRGBA)
					for channel, pair := range [][2]uint8{{got.R, colors[want].R}, {got.G, colors[want].G}, {got.B, colors[want].B}} {
						if difference := int(pair[0]) - int(pair[1]); difference < -30 || difference > 30 {
							t.Errorf("corner %d channel %d: got %v, want near %v", corner, channel, got, colors[want])
						}
					}
				}
			})
		}
	}
}

func TestJPEGOrientation_BoundedMetadata(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		t.Run(order.String(), func(t *testing.T) {
			segment := exifOrientationSegment(6, order)
			wrap := func(metadata []byte) []byte {
				input := append([]byte{0xff, 0xd8}, metadata...)
				return append(input, 0xff, 0xda)
			}
			for length := 0; length < len(segment); length++ {
				// A lone 0xff merges with SOS's prefix into legal marker fill,
				// so lengths zero and one do not contain an APP1 marker.
				if _, err := jpegOrientation(wrap(segment[:length])); err == nil && length > 1 {
					t.Errorf("accepted truncated EXIF prefix of length %d", length)
				}
			}
			cases := []struct {
				name   string
				mutate func([]byte)
			}{
				{"byte order", func(b []byte) { copy(b[10:], "XX") }},
				{"TIFF version", func(b []byte) { order.PutUint16(b[12:], 0) }},
				{"overflowing IFD", func(b []byte) { order.PutUint32(b[14:], 0xfffffff0) }},
				{"oversized count", func(b []byte) { order.PutUint16(b[18:], 65535) }},
				{"orientation type", func(b []byte) { order.PutUint16(b[22:], 4) }},
				{"orientation count", func(b []byte) { order.PutUint32(b[24:], 2) }},
				{"orientation zero", func(b []byte) { order.PutUint16(b[28:], 0) }},
				{"orientation nine", func(b []byte) { order.PutUint16(b[28:], 9) }},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					metadata := bytes.Clone(segment)
					tc.mutate(metadata)
					if _, err := jpegOrientation(wrap(metadata)); err == nil {
						t.Fatal("accepted malformed metadata")
					}
				})
			}
			duplicate := append(bytes.Clone(segment), segment...)
			if _, err := jpegOrientation(wrap(duplicate)); err == nil {
				t.Fatal("accepted ambiguous duplicate EXIF blocks")
			}
			duplicateTag := make([]byte, len(segment)+12)
			copy(duplicateTag, segment[:32])
			copy(duplicateTag[32:], segment[20:32])
			binary.BigEndian.PutUint16(duplicateTag[2:], uint16(len(duplicateTag)-2))
			order.PutUint16(duplicateTag[18:], 2)
			if _, err := jpegOrientation(wrap(duplicateTag)); err == nil {
				t.Fatal("accepted ambiguous duplicate orientation tags")
			}
			// Unrelated pointers are not followed, even when cyclic or out of bounds.
			metadata := bytes.Clone(segment)
			order.PutUint32(metadata[32:], 0xffffffff)
			if got, err := jpegOrientation(wrap(metadata)); err != nil || got != 6 {
				t.Fatalf("next-IFD pointer affected orientation: %d, %v", got, err)
			}
			order.PutUint16(metadata[20:], 0x010f)
			if got, err := jpegOrientation(wrap(metadata)); err != nil || got != 1 {
				t.Fatalf("EXIF without orientation must keep stored axes: %d, %v", got, err)
			}
		})
	}
}

func TestRecompressJPEGBytes_OrientationFallbackAndInvalidMetadata(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	seed := uint32(1)
	for index := range img.Pix {
		seed = seed*1664525 + 1013904223
		img.Pix[index] = byte(seed >> 24)
		if index%4 == 3 {
			img.Pix[index] = 255
		}
	}
	var encoded bytes.Buffer
	if err := stdjpeg.Encode(&encoded, img, &stdjpeg.Options{Quality: 1}); err != nil {
		t.Fatal(err)
	}
	metadata := exifOrientationSegment(6, binary.LittleEndian)
	input := append([]byte{0xff, 0xd8}, metadata...)
	input = append(input, encoded.Bytes()[2:]...)
	out, err := RecompressJPEGBytes(input, BalancedOptions(1, 1, 100))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, input) {
		t.Fatal("larger candidate must retain original bytes and orientation metadata")
	}
	binary.LittleEndian.PutUint32(input[16:], 0xfffffff0)
	if _, err := RecompressJPEGBytes(input, BalancedOptions(1, 1, 85)); err == nil {
		t.Fatal("public entry point accepted an out-of-bounds EXIF pointer")
	}
}
