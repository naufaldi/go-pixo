package jpeg

import (
	"bytes"
	"encoding/binary"
	"image"
	stdjpeg "image/jpeg"
	"strings"
	"testing"
)

func TestRecompressRejectsOversizedHeaderBeforeDecode(t *testing.T) {
	var b bytes.Buffer
	if err := stdjpeg.Encode(&b, image.NewGray(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	input := b.Bytes()
	// Mark as JFIF so DecodeConfig can stop at SOF, without needing SOS.
	input = append([]byte{0xff, 0xd8, 0xff, 0xe0, 0, 16, 'J', 'F', 'I', 'F', 0, 1, 1, 0, 0, 1, 0, 1, 0, 0}, input[2:]...)
	sof := bytes.Index(input, []byte{0xff, 0xc0})
	if sof < 0 {
		t.Fatal("missing SOF")
	}
	for _, size := range [][2]uint16{{5000, 5000}, {16385, 1}} {
		binary.BigEndian.PutUint16(input[sof+5:], size[1])
		binary.BigEndian.PutUint16(input[sof+7:], size[0])
		// Stop before SOS: DecodeConfig succeeds, Decode fails without a pixel allocation.
		header := input[:sof+2+int(binary.BigEndian.Uint16(input[sof+2:]))]
		if _, err := stdjpeg.DecodeConfig(bytes.NewReader(header)); err != nil {
			t.Fatal(err)
		}
		_, err := RecompressJPEGBytes(header, BalancedOptions(1, 1, 75))
		if err == nil || !strings.Contains(err.Error(), "limit") {
			t.Fatalf("%v: want limit before missing scan, got %v", size, err)
		}
	}
}

func TestEncoderRejectsResourceLimitAndMutatedDimensions(t *testing.T) {
	if _, err := NewEncoder(BalancedOptions(5000, 5000, 75)); err == nil {
		t.Error("accepted oversized raster")
	}
	enc, err := NewEncoder(BalancedOptions(1, 1, 75))
	if err != nil {
		t.Fatal(err)
	}
	enc.Options.Width, enc.Options.Height = 0, 0
	if _, err := enc.Encode(nil); err == nil {
		t.Error("accepted invalid mutated dimensions")
	}
}
