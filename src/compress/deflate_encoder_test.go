package compress

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func TestDeflateEncoder_EncodeFixed(t *testing.T) {
	enc := NewDeflateEncoder()
	data := []byte("Hello, World!")

	compressed, err := enc.Encode(data, false)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("expected compressed output")
	}

	reader := flate.NewReader(bytes.NewReader(compressed))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestDeflateEncoder_EncodeDynamic(t *testing.T) {
	enc := NewDeflateEncoder()
	data := []byte("Hello, World!")

	compressed, err := enc.Encode(data, true)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("expected compressed output")
	}

	reader := flate.NewReader(bytes.NewReader(compressed))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestDeflateEncoder_EncodeAuto(t *testing.T) {
	enc := NewDeflateEncoder()
	data := []byte("Hello, World!")

	compressed, err := enc.EncodeAuto(data)
	if err != nil {
		t.Fatalf("EncodeAuto failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("expected compressed output")
	}

	reader := flate.NewReader(bytes.NewReader(compressed))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestDeflateEncoder_EncodeEmpty(t *testing.T) {
	enc := NewDeflateEncoder()
	data := []byte{}

	compressed, err := enc.Encode(data, false)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("expected compressed output for empty data")
	}

	reader := flate.NewReader(bytes.NewReader(compressed))
	decompressed := make([]byte, 10)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if n != 0 {
		t.Errorf("got %d bytes, want 0", n)
	}
}

func TestDeflateEncoder_EncodeRepetitive(t *testing.T) {
	enc := NewDeflateEncoder()
	repetitive := bytes.Repeat([]byte("ABC"), 100)
	data := repetitive

	fixed, err := enc.Encode(data, false)
	if err != nil {
		t.Fatalf("Encode fixed failed: %v", err)
	}

	dynamic, err := enc.Encode(data, true)
	if err != nil {
		t.Fatalf("Encode dynamic failed: %v", err)
	}

	if len(fixed) >= len(data) {
		t.Errorf("fixed compression didn't reduce size: %d >= %d", len(fixed), len(data))
	}

	if len(dynamic) >= len(data) {
		t.Errorf("dynamic compression didn't reduce size: %d >= %d", len(dynamic), len(data))
	}

	reader := flate.NewReader(bytes.NewReader(fixed))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("fixed decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("fixed: got %q, want %q", decompressed[:n], data)
	}

	reader = flate.NewReader(bytes.NewReader(dynamic))
	decompressed = make([]byte, len(data)*2)
	n, err = reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("dynamic decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("dynamic: got %q, want %q", decompressed[:n], data)
	}
}

func TestDeflateEncoder_EncodeAuto_ChoosesSmaller(t *testing.T) {
	enc := NewDeflateEncoder()
	data := bytes.Repeat([]byte("Hello, World! "), 50)

	fixed, err := enc.Encode(data, false)
	if err != nil {
		t.Fatalf("Encode fixed failed: %v", err)
	}

	dynamic, err := enc.Encode(data, true)
	if err != nil {
		t.Fatalf("Encode dynamic failed: %v", err)
	}

	auto, err := enc.EncodeAuto(data)
	if err != nil {
		t.Fatalf("EncodeAuto failed: %v", err)
	}

	if len(auto) > len(fixed) && len(auto) > len(dynamic) {
		t.Errorf("EncodeAuto should choose smaller: auto=%d, fixed=%d, dynamic=%d",
			len(auto), len(fixed), len(dynamic))
	}

	if len(auto) != len(fixed) && len(auto) != len(dynamic) {
		t.Errorf("EncodeAuto should match one of fixed/dynamic: auto=%d, fixed=%d, dynamic=%d",
			len(auto), len(fixed), len(dynamic))
	}
}

func TestDeflateEncoder_EncodeTo(t *testing.T) {
	enc := NewDeflateEncoder()
	data := []byte("Test data")

	var buf bytes.Buffer
	if err := enc.EncodeTo(&buf, data, false); err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected bytes written")
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}
