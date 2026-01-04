package compress

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func TestZopfliEncode_Empty(t *testing.T) {
	data := []byte{}
	result, err := ZopfliEncodeSimple(data)
	if err != nil {
		t.Fatalf("ZopfliEncodeSimple failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d bytes", len(result))
	}
}

func TestZopfliEncode_Simple(t *testing.T) {
	data := []byte("Hello, World!")
	result, err := ZopfliEncodeSimple(data)
	if err != nil {
		t.Fatalf("ZopfliEncodeSimple failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}

	// Verify decompressibility
	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestZopfliEncode_Repetitive(t *testing.T) {
	data := bytes.Repeat([]byte("ABC"), 100)
	result, err := ZopfliEncodeSimple(data)
	if err != nil {
		t.Fatalf("ZopfliEncodeSimple failed: %v", err)
	}
	if len(result) >= len(data) {
		t.Errorf("compression didn't reduce size: %d >= %d", len(result), len(data))
	}

	// Verify decompressibility
	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestZopfliEncode_VersusStandard(t *testing.T) {
	enc := NewDeflateEncoder()
	data := bytes.Repeat([]byte("Hello, World! This is a test string. "), 50)

	// Standard encoding
	standard, err := enc.EncodeAuto(data)
	if err != nil {
		t.Fatalf("EncodeAuto failed: %v", err)
	}

	// Zopfli encoding
	zopfli, err := ZopfliEncodeSimple(data)
	if err != nil {
		t.Fatalf("ZopfliEncodeSimple failed: %v", err)
	}

	// Zopfli should be equal or smaller
	if len(zopfli) > len(standard) {
		t.Logf("Zopfli (%d) > Standard (%d) - unusual but possible", len(zopfli), len(standard))
	}

	t.Logf("Original: %d bytes", len(data))
	t.Logf("Standard: %d bytes (%.1f%%)", len(standard), float64(len(standard))/float64(len(data))*100)
	t.Logf("Zopfli:   %d bytes (%.1f%%)", len(zopfli), float64(len(zopfli))/float64(len(data))*100)
}

func TestZopfliEncode_WithConfig(t *testing.T) {
	data := bytes.Repeat([]byte("Test data for compression. "), 20)

	config := ZopfliConfig{
		Iterations:     5,
		BlockSplitting: true,
		MaxBlockSize:   65535,
	}

	result, err := ZopfliEncode(data, config)
	if err != nil {
		t.Fatalf("ZopfliEncode failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}

	// Verify decompressibility
	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestZopfliEncode_IterationsImpact(t *testing.T) {
	data := bytes.Repeat([]byte("Pattern: AAAAABBBBBCCCCCDDDDD "), 30)

	config1 := ZopfliConfig{Iterations: 1}
	config5 := ZopfliConfig{Iterations: 5}
	config15 := ZopfliConfig{Iterations: 15}

	result1, err := ZopfliEncode(data, config1)
	if err != nil {
		t.Fatalf("ZopfliEncode (1 iter) failed: %v", err)
	}

	result5, err := ZopfliEncode(data, config5)
	if err != nil {
		t.Fatalf("ZopfliEncode (5 iters) failed: %v", err)
	}

	result15, err := ZopfliEncode(data, config15)
	if err != nil {
		t.Fatalf("ZopfliEncode (15 iters) failed: %v", err)
	}

	t.Logf("1 iteration:  %d bytes", len(result1))
	t.Logf("5 iterations: %d bytes", len(result5))
	t.Logf("15 iterations: %d bytes", len(result15))

	// More iterations should not produce larger output
	if len(result5) > len(result1) {
		t.Logf("5 iters > 1 iter: unusual")
	}
	if len(result15) > len(result5) {
		t.Logf("15 iters > 5 iters: unusual")
	}
}

func TestZopfliEncode_LargeData(t *testing.T) {
	// Create larger data to test compression
	data := make([]byte, 100000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	result, err := ZopfliEncodeSimple(data)
	if err != nil {
		t.Fatalf("ZopfliEncodeSimple failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}
	if len(result) >= len(data) {
		t.Errorf("compression didn't reduce size: %d >= %d", len(result), len(data))
	}

	// Verify decompressibility with io.ReadAll
	reader := flate.NewReader(bytes.NewReader(result))
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("decompression failed: %v", err)
	}
	if len(decompressed) != len(data) {
		t.Errorf("decompressed %d bytes, want %d", len(decompressed), len(data))
	}
	if !bytes.Equal(decompressed, data) {
		t.Errorf("decompressed data doesn't match original")
	}
}

func TestCalculateDeflateCost(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single", []byte{42}},
		{"repetitive", bytes.Repeat([]byte{1, 2, 3}, 100)},
		{"random", []byte{0xAB, 0xCD, 0xEF, 0x12, 0x34, 0x56, 0x78, 0x9A}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateDeflateCost(tt.data)
			if cost < 0 {
				t.Errorf("cost should be non-negative, got %f", cost)
			}
			if len(tt.data) == 0 && cost != 0 {
				t.Errorf("empty data should have zero cost, got %f", cost)
			}
		})
	}
}

func TestDefaultZopfliConfig(t *testing.T) {
	config := DefaultZopfliConfig()

	if config.Iterations != 15 {
		t.Errorf("expected 15 iterations, got %d", config.Iterations)
	}
	if !config.BlockSplitting {
		t.Error("expected block splitting to be enabled")
	}
	if config.MaxBlockSize != 65535 {
		t.Errorf("expected max block size 65535, got %d", config.MaxBlockSize)
	}
}

func TestDeflateEncoder_EncodeOptimal(t *testing.T) {
	enc := NewDeflateEncoder()
	data := bytes.Repeat([]byte("Optimal compression test. "), 50)

	result, err := enc.EncodeOptimal(data)
	if err != nil {
		t.Fatalf("EncodeOptimal failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}
	if len(result) >= len(data) {
		t.Errorf("compression didn't reduce size: %d >= %d", len(result), len(data))
	}

	// Verify decompressibility
	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestDeflateEncoder_EncodeOptimalWithConfig(t *testing.T) {
	enc := NewDeflateEncoder()
	data := bytes.Repeat([]byte("Configurable optimal compression. "), 40)

	result, err := enc.EncodeOptimalWithConfig(data, 5, true)
	if err != nil {
		t.Fatalf("EncodeOptimalWithConfig failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}

	// Verify decompressibility
	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}
