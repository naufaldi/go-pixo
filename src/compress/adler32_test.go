package compress

import (
	"hash/adler32"
	"testing"
)

func TestAdler32_RFC1950Vectors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint32
	}{
		{
			name: "nil",
			data: nil,
			want: 1,
		},
		{
			name: "empty",
			data: []byte{},
			want: 1,
		},
		{
			name: "single byte A",
			data: []byte{0x41},
			want: 0x00420042,
		},
		{
			name: "ABC",
			data: []byte("ABC"),
			want: 0x018D00C7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Adler32(tt.data); got != tt.want {
				t.Fatalf("Adler32(%q) = 0x%08X, want 0x%08X", tt.data, got, tt.want)
			}
		})
	}
}

func TestAdler32_MatchesStdlib(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "short",
			data: []byte("hello"),
		},
		{
			name: "long",
			data: []byte("Hello, World! This is a longer test string for validation."),
		},
		{
			name: "binary",
			data: []byte{0x00, 0xFF, 0x10, 0x20, 0x00, 0x01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := adler32.Checksum(tt.data)
			if got := Adler32(tt.data); got != want {
				t.Fatalf("Adler32(%q) = 0x%08X, want 0x%08X", tt.data, got, want)
			}
		})
	}
}

func TestNewAdler32_Hash32Interface(t *testing.T) {
	h := NewAdler32()
	if h == nil {
		t.Fatal("NewAdler32 returned nil")
	}

	if got, want := h.Size(), 4; got != want {
		t.Fatalf("Size() = %d, want %d", got, want)
	}
	if got, want := h.BlockSize(), 1; got != want {
		t.Fatalf("BlockSize() = %d, want %d", got, want)
	}

	prefix := []byte{0xAA}
	sum := h.Sum(append([]byte{}, prefix...))
	if len(sum) != len(prefix)+4 {
		t.Fatalf("Sum(prefix) len = %d, want %d", len(sum), len(prefix)+4)
	}
	if sum[0] != 0xAA {
		t.Fatalf("Sum(prefix) did not preserve prefix")
	}
	if got := h.Sum32(); got != 1 {
		t.Fatalf("Sum32() at initial state = 0x%08X, want 0x00000001", got)
	}

	_, _ = h.Write([]byte("test"))
	if got, want := h.Sum32(), Adler32([]byte("test")); got != want {
		t.Fatalf("Sum32() after Write = 0x%08X, want 0x%08X", got, want)
	}

	h.Reset()
	if got := h.Sum32(); got != 1 {
		t.Fatalf("Sum32() after Reset = 0x%08X, want 0x00000001", got)
	}
}

func TestAdler32_StreamingConsistency(t *testing.T) {
	data := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit.")
	oneShot := Adler32(data)

	h := NewAdler32()
	_, _ = h.Write(data[:10])
	_, _ = h.Write(data[10:20])
	_, _ = h.Write(data[20:])

	if got := h.Sum32(); got != oneShot {
		t.Fatalf("streaming Sum32() = 0x%08X, want 0x%08X", got, oneShot)
	}
}
