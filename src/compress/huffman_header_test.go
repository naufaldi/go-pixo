package compress

import (
	"bytes"
	"testing"
)

func TestWriteHLIT_Validation(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{"Valid minimum", 257, false},
		{"Valid maximum", 286, false},
		{"Valid middle", 270, false},
		{"Too small", 256, true},
		{"Too large", 287, true},
		{"Zero", 0, true},
		{"Negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			bw := NewBitWriter(&buf)

			err := WriteHLIT(bw, tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteHLIT(%d) error = %v, wantErr %v", tt.n, err, tt.wantErr)
			}

			if !tt.wantErr {
				if err := bw.Flush(); err != nil {
					t.Fatalf("Flush failed: %v", err)
				}

				if buf.Len() == 0 {
					t.Error("Expected bytes written, got empty buffer")
				}
			}
		})
	}
}

func TestWriteHDIST_Validation(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{"Valid minimum", 1, false},
		{"Valid maximum", 30, false},
		{"Valid middle", 15, false},
		{"Too large", 31, true},
		{"Zero", 0, true},
		{"Negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			bw := NewBitWriter(&buf)

			err := WriteHDIST(bw, tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteHDIST(%d) error = %v, wantErr %v", tt.n, err, tt.wantErr)
			}

			if !tt.wantErr {
				if err := bw.Flush(); err != nil {
					t.Fatalf("Flush failed: %v", err)
				}

				if buf.Len() == 0 {
					t.Error("Expected bytes written, got empty buffer")
				}
			}
		})
	}
}

func TestWriteHCLEN_Validation(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{"Valid minimum", 4, false},
		{"Valid maximum", 19, false},
		{"Valid middle", 10, false},
		{"Too small", 3, true},
		{"Too large", 20, true},
		{"Zero", 0, true},
		{"Negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			bw := NewBitWriter(&buf)

			err := WriteHCLEN(bw, tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteHCLEN(%d) error = %v, wantErr %v", tt.n, err, tt.wantErr)
			}

			if !tt.wantErr {
				if err := bw.Flush(); err != nil {
					t.Fatalf("Flush failed: %v", err)
				}

				if buf.Len() == 0 {
					t.Error("Expected bytes written, got empty buffer")
				}
			}
		})
	}
}

func TestWriteDynamicHeader_Output(t *testing.T) {
	litLengths := make([]int, 288)
	distLengths := make([]int, 30)

	litLengths[65] = 8
	litLengths[66] = 8
	litLengths[67] = 9
	litLengths[256] = 7

	distLengths[1] = 5
	distLengths[2] = 5

	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := WriteDynamicHeader(bw, litLengths, distLengths); err != nil {
		t.Fatalf("WriteDynamicHeader failed: %v", err)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("Expected header bytes written, got empty buffer")
	}
}

func TestWriteDynamicHeader_EmptyLengths(t *testing.T) {
	litLengths := make([]int, 288)
	distLengths := make([]int, 30)

	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	err := WriteDynamicHeader(bw, litLengths, distLengths)
	if err == nil {
		t.Log("WriteDynamicHeader with empty lengths succeeded (may be valid)")
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
}

func TestWriteDynamicHeader_RLEEncoding(t *testing.T) {
	litLengths := make([]int, 288)
	distLengths := make([]int, 30)

	for i := 0; i < 10; i++ {
		litLengths[i+65] = 8
	}

	for i := 0; i < 5; i++ {
		distLengths[i] = 5
	}

	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := WriteDynamicHeader(bw, litLengths, distLengths); err != nil {
		t.Fatalf("WriteDynamicHeader failed: %v", err)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("Expected header bytes written, got empty buffer")
	}
}

func TestWriteDynamicHeader_ZeroRuns(t *testing.T) {
	litLengths := make([]int, 288)
	distLengths := make([]int, 30)

	litLengths[65] = 8
	litLengths[100] = 8
	litLengths[200] = 9

	distLengths[1] = 5
	distLengths[10] = 5

	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := WriteDynamicHeader(bw, litLengths, distLengths); err != nil {
		t.Fatalf("WriteDynamicHeader failed: %v", err)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("Expected header bytes written, got empty buffer")
	}
}

func TestWriteDynamicHeader_MaxValues(t *testing.T) {
	litLengths := make([]int, 288)
	distLengths := make([]int, 30)

	for i := 0; i < 286; i++ {
		litLengths[i] = 8
	}

	for i := 0; i < 30; i++ {
		distLengths[i] = 5
	}

	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := WriteDynamicHeader(bw, litLengths, distLengths); err != nil {
		t.Fatalf("WriteDynamicHeader failed: %v", err)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("Expected header bytes written, got empty buffer")
	}
}

func TestCodeLengthOrder(t *testing.T) {
	if len(CodeLengthOrder) != 19 {
		t.Errorf("Expected CodeLengthOrder length 19, got %d", len(CodeLengthOrder))
	}

	expectedOrder := []int{16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15}
	for i, val := range CodeLengthOrder {
		if val != expectedOrder[i] {
			t.Errorf("CodeLengthOrder[%d] = %d, want %d", i, val, expectedOrder[i])
		}
	}
}
