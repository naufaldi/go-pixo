package jpeg

import (
	"bytes"
	"testing"
)

func TestWriteSOI(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := WriteSOI(buf); err != nil {
		t.Fatal(err)
	}
	expected := []byte{0xFF, 0xD8}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %X, want %X", buf.Bytes(), expected)
	}
}

func TestWriteAPP0(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := WriteAPP0(buf); err != nil {
		t.Fatal(err)
	}
	// FF E0 + length 00 10 + JFIF\0 ...
	if buf.Len() != 18 { // 2 + 2 + 5 + 9 = 18
		t.Errorf("expected length 18, got %d", buf.Len())
	}
}

func TestWriteDQT(t *testing.T) {
	buf := new(bytes.Buffer)
	var table [64]uint8
	for i := range table {
		table[i] = uint8(i)
	}
	if err := WriteDQT(buf, 0, table); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 69 { // 2 + 2 + 1 + 64 = 69
		t.Errorf("expected length 69, got %d", buf.Len())
	}
}

func TestWriteSOF0(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := WriteSOF0(buf, 100, 200, ColorRGB, Subsampling444); err != nil {
		t.Fatal(err)
	}
	// 2 (marker) + 2 (length) + 1 (precision) + 2 (height) + 2 (width) + 1 (numComponents) + 3*3 = 19
	if buf.Len() != 19 {
		t.Errorf("expected length 19, got %d", buf.Len())
	}
}

func TestWriteDHT(t *testing.T) {
	buf := new(bytes.Buffer)
	var bits [16]uint8
	bits[0] = 1
	vals := []uint8{5}
	if err := WriteDHT(buf, 0, bits, vals); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 2+2+1+16+1 {
		t.Errorf("expected length 21, got %d", buf.Len())
	}
}

func TestWriteSOS(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := WriteSOS(buf, ColorRGB); err != nil {
		t.Fatal(err)
	}
	// 2 + 2 + 1 + 3*2 + 3 = 14
	if buf.Len() != 14 {
		t.Errorf("expected length 14, got %d", buf.Len())
	}
}
