package jpeg

import (
	"testing"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrInvalidQuality, "jpeg: quality must be between 1 and 100"},
		{ErrInvalidDimensions, "jpeg: invalid image dimensions"},
		{ErrUnsupportedColorType, "jpeg: unsupported color type"},
		{ErrInvalidDataLength, "jpeg: invalid data length"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("got %q, want %q", tt.err.Error(), tt.expected)
			}
		})
	}
}
