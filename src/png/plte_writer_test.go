package png

import (
	"bytes"
	"testing"
)

func TestWritePLTE(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{255, 0, 0}) // red
	palette.AddColor(Color{0, 255, 0}) // green
	palette.AddColor(Color{0, 0, 255}) // blue

	var buf bytes.Buffer
	err := WritePLTE(&buf, *palette)

	if err != nil {
		t.Errorf("WritePLTE() error = %v", err)
	}

	data := buf.Bytes()

	// Should have: 4-byte length + 4-byte type + 9-byte data (3*3) + 4-byte CRC = 21 bytes
	if len(data) != 21 {
		t.Errorf("WritePLTE() length = %v, want 21", len(data))
	}

	// Check length field (big-endian)
	length := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	if length != 9 {
		t.Errorf("WritePLTE() length field = %v, want 9", length)
	}

	// Check type field
	if string(data[4:8]) != "PLTE" {
		t.Errorf("WritePLTE() type = %v, want 'PLTE'", string(data[4:8]))
	}

	// Check RGB values
	if data[8] != 255 || data[9] != 0 || data[10] != 0 {
		t.Errorf("WritePLTE() first color = (%v, %v, %v), want (255, 0, 0)", data[8], data[9], data[10])
	}
}

func TestWritePLTESingleColor(t *testing.T) {
	palette := NewPalette(1)
	palette.AddColor(Color{128, 128, 128})

	var buf bytes.Buffer
	err := WritePLTE(&buf, *palette)

	if err != nil {
		t.Errorf("WritePLTE() single color error = %v", err)
	}

	data := buf.Bytes()

	// Should have: 4 + 4 + 3 + 4 = 15 bytes
	if len(data) != 15 {
		t.Errorf("WritePLTE() single color length = %v, want 15", len(data))
	}
}

func TestWritePLTEMaxColors(t *testing.T) {
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{uint8(i), uint8(i), uint8(i)})
	}

	var buf bytes.Buffer
	err := WritePLTE(&buf, *palette)

	if err != nil {
		t.Errorf("WritePLTE() max colors error = %v", err)
	}

	data := buf.Bytes()

	// 256 colors * 3 = 768 bytes data
	// Plus 4 (length) + 4 (type) + 4 (CRC) = 780 bytes
	expectedLen := 4 + 4 + 768 + 4
	if len(data) != expectedLen {
		t.Errorf("WritePLTE() max colors length = %v, want %v", len(data), expectedLen)
	}
}

func TestWritePLEEmpty(t *testing.T) {
	palette := NewPalette(4)

	var buf bytes.Buffer
	err := WritePLTE(&buf, *palette)

	// Should error because palette is empty
	if err == nil {
		t.Error("WritePLTE() empty palette should return error")
	}
}

func TestPLTEChunkData(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{255, 0, 0})
	palette.AddColor(Color{0, 255, 0})
	palette.AddColor(Color{0, 0, 255})

	data := PLTEChunkData(*palette)

	if len(data) != 9 {
		t.Errorf("PLTEChunkData() length = %v, want 9", len(data))
	}

	if data[0] != 255 || data[1] != 0 || data[2] != 0 {
		t.Errorf("PLTEChunkData() first = (%v, %v, %v)", data[0], data[1], data[2])
	}
}

func TestPLTEChunkDataEmpty(t *testing.T) {
	palette := NewPalette(4)

	data := PLTEChunkData(*palette)

	if data != nil {
		t.Errorf("PLTEChunkData() empty = %v, want nil", data)
	}
}

func TestValidatePalette(t *testing.T) {
	tests := []struct {
		name      string
		numColors int
		wantErr   bool
	}{
		{"valid 1", 1, false},
		{"valid 2", 2, false},
		{"valid 256", 256, false},
		{"empty", 0, true},
		{"too many", 257, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := NewPalette(tt.numColors)
			for i := 0; i < tt.numColors && tt.numColors > 0; i++ {
				palette.AddColor(Color{uint8(i), uint8(i), uint8(i)})
			}

			err := ValidatePalette(*palette)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePalette() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWritePLETwoColors(t *testing.T) {
	palette := NewPalette(2)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{255, 255, 255})

	var buf bytes.Buffer
	err := WritePLTE(&buf, *palette)

	if err != nil {
		t.Errorf("WritePLTE() two colors error = %v", err)
	}

	data := buf.Bytes()

	// 4 + 4 + 6 + 4 = 18 bytes
	if len(data) != 18 {
		t.Errorf("WritePLTE() two colors length = %v, want 18", len(data))
	}

	// Check length
	length := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	if length != 6 {
		t.Errorf("WritePLTE() two colors length field = %v, want 6", length)
	}
}
