package compress

import (
	"testing"
)

func TestAdler32(t *testing.T) {
	if got := Adler32(nil); got != 1 {
		t.Errorf("Adler32(nil) = %d, want 1", got)
	}
	if got := Adler32([]byte{}); got != 1 {
		t.Errorf("Adler32([]) = %d, want 1", got)
	}

	if got := Adler32([]byte{0x41}); got != 0x00420042 {
		t.Errorf("Adler32([0x41]) = 0x%08X, want 0x00420042", got)
	}

	expected := uint32(0x018D00C7)
	if got := Adler32([]byte("ABC")); got != expected {
		t.Errorf("Adler32('ABC') = 0x%08X, want 0x%08X", got, expected)
	}

	data := []byte("Hello, World! This is a longer test string for validation.")
	result := Adler32(data)
	if result == 0 {
		t.Error("Adler32 should not return 0 for valid data")
	}
}

func TestNewAdler32(t *testing.T) {
	h := NewAdler32()
	if h == nil {
		t.Fatal("NewAdler32 returned nil")
	}

	h.Write([]byte("test"))
	sum := h.Sum(nil)
	if len(sum) != 4 {
		t.Errorf("Sum() returned %d bytes, want 4", len(sum))
	}

	h.Reset()
	sum2 := h.Sum(nil)
	if sum2[0] != 0 || sum2[1] != 0 || sum2[2] != 0 || sum2[3] != 1 {
		t.Errorf("After reset, Sum() = %v, want [0 0 0 1]", sum2)
	}
}

func TestAdler32Streaming(t *testing.T) {
	data := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit.")

	oneShot := Adler32(data)

	h := NewAdler32()
	h.Write(data[:10])
	h.Write(data[10:20])
	h.Write(data[20:])
	streaming := h.Sum32()

	if oneShot != streaming {
		t.Errorf("Streaming Adler32 (0x%08X) != one-shot (0x%08X)", streaming, oneShot)
	}
}
