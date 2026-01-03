package png

import (
	"bytes"
	"testing"
)

func TestWriteTRNS(t *testing.T) {
	alphaValues := []uint8{255, 128, 0}

	var buf bytes.Buffer
	err := WriteTRNS(&buf, alphaValues)

	if err != nil {
		t.Errorf("WriteTRNS() error = %v", err)
	}

	data := buf.Bytes()

	// 4-byte length + 4-byte type + 3-byte data + 4-byte CRC = 15 bytes
	if len(data) != 15 {
		t.Errorf("WriteTRNS() length = %v, want 15", len(data))
	}

	// Check length field
	length := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	if length != 3 {
		t.Errorf("WriteTRNS() length field = %v, want 3", length)
	}

	// Check type field
	if string(data[4:8]) != "tRNS" {
		t.Errorf("WriteTRNS() type = %v, want 'tRNS'", string(data[4:8]))
	}

	// Check alpha values
	if data[8] != 255 || data[9] != 128 || data[10] != 0 {
		t.Errorf("WriteTRNS() values = (%v, %v, %v), want (255, 128, 0)", data[8], data[9], data[10])
	}
}

func TestWriteTRNSEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteTRNS(&buf, []uint8{})

	if err != nil {
		t.Errorf("WriteTRNS() empty error = %v", err)
	}

	// Empty should return 0 bytes (no chunk written)
	// Actually, let's check what happens
	data := buf.Bytes()
	if len(data) != 0 {
		t.Errorf("WriteTRNS() empty length = %v, want 0", len(data))
	}
}

func TestWriteTRNSSingleValue(t *testing.T) {
	alphaValues := []uint8{255}

	var buf bytes.Buffer
	err := WriteTRNS(&buf, alphaValues)

	if err != nil {
		t.Errorf("WriteTRNS() single value error = %v", err)
	}

	data := buf.Bytes()

	// 4 + 4 + 1 + 4 = 13 bytes
	if len(data) != 13 {
		t.Errorf("WriteTRNS() single value length = %v, want 13", len(data))
	}
}

func TestWriteTRNSMax(t *testing.T) {
	alphaValues := make([]uint8, 256)
	for i := range alphaValues {
		alphaValues[i] = uint8(255 - i%256)
	}

	var buf bytes.Buffer
	err := WriteTRNS(&buf, alphaValues)

	if err != nil {
		t.Errorf("WriteTRNS() max error = %v", err)
	}

	data := buf.Bytes()

	// 4 + 4 + 256 + 4 = 268 bytes
	if len(data) != 268 {
		t.Errorf("WriteTRNS() max length = %v, want 268", len(data))
	}
}

func TestWriteTRNSTooMany(t *testing.T) {
	alphaValues := make([]uint8, 257)

	var buf bytes.Buffer
	err := WriteTRNS(&buf, alphaValues)

	// Should error because 257 > 256
	if err == nil {
		t.Error("WriteTRNS() too many should return error")
	}
}

func TestTRNSChunkData(t *testing.T) {
	alphaValues := []uint8{255, 128, 0}

	data := TRNSChunkData(alphaValues)

	if len(data) != 3 {
		t.Errorf("TRNSChunkData() length = %v, want 3", len(data))
	}

	if data[0] != 255 || data[1] != 128 || data[2] != 0 {
		t.Errorf("TRNSChunkData() = (%v, %v, %v), want (255, 128, 0)", data[0], data[1], data[2])
	}
}

func TestTRNSChunkDataEmpty(t *testing.T) {
	data := TRNSChunkData([]uint8{})

	if data != nil {
		t.Errorf("TRNSChunkData() empty = %v, want nil", data)
	}
}

func TestTRNSChunkDataTooMany(t *testing.T) {
	alphaValues := make([]uint8, 257)
	data := TRNSChunkData(alphaValues)

	if data != nil {
		t.Errorf("TRNSChunkData() too many = %v, want nil", data)
	}
}

func TestExtractAlphaFromPixels(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{255, 0, 0})
	palette.AddColor(Color{0, 255, 0})
	palette.AddColor(Color{0, 0, 255})

	alphaValues, hasTransparency := ExtractAlphaFromPixels([]byte{}, *palette)

	if hasTransparency {
		t.Errorf("ExtractAlphaFromPixels() no pixels should have no transparency")
	}

	if len(alphaValues) != 3 {
		t.Errorf("ExtractAlphaFromPixels() alpha length = %v, want 3", len(alphaValues))
	}

	// All should be 255 (opaque)
	for i, alpha := range alphaValues {
		if alpha != 255 {
			t.Errorf("ExtractAlphaFromPixels()[%v] = %v, want 255", i, alpha)
		}
	}
}

func TestValidateTRNS(t *testing.T) {
	tests := []struct {
		name        string
		alphaLen    int
		paletteSize int
		wantErr     bool
	}{
		{"valid", 3, 3, false},
		{"valid partial", 2, 3, false},
		{"too many", 4, 3, true},
		{"empty", 0, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alphaValues := make([]uint8, tt.alphaLen)
			err := ValidateTRNS(alphaValues, tt.paletteSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTRNS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteTRNSAllOpaque(t *testing.T) {
	alphaValues := []uint8{255, 255, 255}

	var buf bytes.Buffer
	err := WriteTRNS(&buf, alphaValues)

	if err != nil {
		t.Errorf("WriteTRNS() all opaque error = %v", err)
	}

	data := buf.Bytes()

	if len(data) != 15 {
		t.Errorf("WriteTRNS() all opaque length = %v, want 15", len(data))
	}
}

func TestWriteTRNSAllTransparent(t *testing.T) {
	alphaValues := []uint8{0, 0, 0}

	var buf bytes.Buffer
	err := WriteTRNS(&buf, alphaValues)

	if err != nil {
		t.Errorf("WriteTRNS() all transparent error = %v", err)
	}

	data := buf.Bytes()

	if len(data) != 15 {
		t.Errorf("WriteTRNS() all transparent length = %v, want 15", len(data))
	}
}

func TestWriteTRNSMixedAlpha(t *testing.T) {
	alphaValues := []uint8{255, 128, 64, 32, 0}

	var buf bytes.Buffer
	err := WriteTRNS(&buf, alphaValues)

	if err != nil {
		t.Errorf("WriteTRNS() mixed alpha error = %v", err)
	}

	data := buf.Bytes()

	// 4 + 4 + 5 + 4 = 17 bytes
	if len(data) != 17 {
		t.Errorf("WriteTRNS() mixed alpha length = %v, want 17", len(data))
	}

	// Verify alpha values
	if data[8] != 255 || data[9] != 128 || data[10] != 64 || data[11] != 32 || data[12] != 0 {
		t.Errorf("WriteTRNS() mixed alpha values incorrect")
	}
}
