package compress

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEncodeLiteral(t *testing.T) {
	table := LiteralLengthTable()
	
	tests := []struct {
		name   string
		symbol int
		wantErr bool
	}{
		{"literal 0", 0, false},
		{"literal 65", 65, false},
		{"literal 255", 255, false},
		{"end of block", 256, false},
		{"invalid negative", -1, true},
		{"invalid too large", 300, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			bw := NewBitWriter(&buf)
			
			err := EncodeLiteral(bw, tt.symbol, table)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeLiteral() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if err := bw.Flush(); err != nil {
					t.Fatalf("Flush failed: %v", err)
				}
				if buf.Len() == 0 {
					t.Error("expected bytes written")
				}
			}
		})
	}
}

func TestEncodeLength(t *testing.T) {
	table := LiteralLengthTable()
	
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"min length", 3, false},
		{"length 4", 4, false},
		{"length 10", 10, false},
		{"length 258", 258, false},
		{"max length", 258, false},
		{"too short", 2, true},
		{"too long", 259, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			bw := NewBitWriter(&buf)
			
			err := EncodeLength(bw, tt.length, table)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeLength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if err := bw.Flush(); err != nil {
					t.Fatalf("Flush failed: %v", err)
				}
				if buf.Len() == 0 {
					t.Error("expected bytes written")
				}
			}
		})
	}
}

func TestEncodeLength_BoundaryValues(t *testing.T) {
	table := LiteralLengthTable()
	
	boundaries := []int{
		3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		23, 24, 27, 28, 31, 32, 35, 36, 43, 44, 51, 52, 59, 60, 67, 68,
		83, 84, 99, 100, 115, 116, 131, 132, 163, 164, 195, 196, 227, 228, 258,
	}
	
	for _, length := range boundaries {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			var buf bytes.Buffer
			bw := NewBitWriter(&buf)
			
			if err := EncodeLength(bw, length, table); err != nil {
				t.Errorf("EncodeLength(%d) failed: %v", length, err)
			}
			if err := bw.Flush(); err != nil {
				t.Fatalf("Flush failed: %v", err)
			}
		})
	}
}

func TestEncodeDistance(t *testing.T) {
	table := DistanceTable()
	
	tests := []struct {
		name     string
		distance int
		wantErr  bool
	}{
		{"min distance", 1, false},
		{"distance 2", 2, false},
		{"distance 10", 10, false},
		{"distance 32768", 32768, false},
		{"max distance", 32768, false},
		{"too short", 0, true},
		{"too long", 32769, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			bw := NewBitWriter(&buf)
			
			err := EncodeDistance(bw, tt.distance, table)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeDistance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if err := bw.Flush(); err != nil {
					t.Fatalf("Flush failed: %v", err)
				}
				if buf.Len() == 0 {
					t.Error("expected bytes written")
				}
			}
		})
	}
}

func TestEncodeDistance_BoundaryValues(t *testing.T) {
	table := DistanceTable()
	
	boundaries := []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 13, 14, 17, 18, 25, 26, 33, 34,
		49, 50, 65, 66, 97, 98, 129, 130, 193, 194, 257, 258, 385, 386,
		513, 514, 769, 770, 1025, 1026, 1537, 1538, 2049, 2050, 3073, 3074,
		4097, 4098, 6145, 6146, 8193, 8194, 12289, 12290, 16385, 16386, 24577, 24578, 32768,
	}
	
	for _, distance := range boundaries {
		t.Run(fmt.Sprintf("distance_%d", distance), func(t *testing.T) {
			var buf bytes.Buffer
			bw := NewBitWriter(&buf)
			
			if err := EncodeDistance(bw, distance, table); err != nil {
				t.Errorf("EncodeDistance(%d) failed: %v", distance, err)
			}
			if err := bw.Flush(); err != nil {
				t.Fatalf("Flush failed: %v", err)
			}
		})
	}
}
