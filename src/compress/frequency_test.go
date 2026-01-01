package compress

import "testing"

func TestDistanceCode(t *testing.T) {
	tests := []struct {
		name     string
		distance uint16
		want     int
	}{
		{"distance 0 (invalid)", 0, -1},
		{"distance 1", 1, 0},
		{"distance 2", 2, 1},
		{"distance 3", 3, 2},
		{"distance 4", 4, 3},
		{"distance 5", 5, 4},
		{"distance 6", 6, 4},
		{"distance 7", 7, 4},
		{"distance 8", 8, 4},
		{"distance 9", 9, 5},
		{"distance 16", 16, 5},
		{"distance 17", 17, 6},
		{"distance 32", 32, 6},
		{"distance 32768", 32768, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := distanceCode(tt.distance)
			if got != tt.want {
				t.Errorf("distanceCode(%d) = %d, want %d", tt.distance, got, tt.want)
			}
		})
	}
}

func TestDistanceCode_InvalidZero(t *testing.T) {
	code := distanceCode(0)
	if code != -1 {
		t.Errorf("distanceCode(0) = %d, want -1 (invalid)", code)
	}
}

func TestDistanceCode_DistanceOneMapsToCodeZero(t *testing.T) {
	code := distanceCode(1)
	if code != 0 {
		t.Errorf("distanceCode(1) = %d, want 0", code)
	}
}
