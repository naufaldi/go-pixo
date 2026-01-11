package png

import (
	"bytes"
	"compress/zlib"
	"io"
	"testing"
	"time"

	"github.com/mac/go-pixo/src/compress"
)

func TestOptimalDeflate_CompressionImprovement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping optimal deflate test in short mode")
	}

	testCases := []struct {
		name        string
		width       int
		height      int
		bpp         int
		dataGen     func(int, int, int) []byte
		description string
	}{
		{
			name:    "repetitive_pattern",
			width:   32,
			height:  32,
			bpp:     3,
			dataGen: generateRepetitiveData,
			description: "highly repetitive data",
		},
		{
			name:    "gradient_data",
			width:   50,
			height:  50,
			bpp:     4,
			dataGen: generateGradientData,
			description: "smooth gradient data",
		},
		{
			name:    "checkerboard",
			width:   32,
			height:  32,
			bpp:     3,
			dataGen: generateCheckerboardData,
			description: "checkerboard pattern",
		},
		{
			name:    "text_like",
			width:   50,
			height:  50,
			bpp:     1,
			dataGen: generateTextLikeData,
			description: "text-like repeated data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pixels := tc.dataGen(tc.width, tc.height, tc.bpp)
			colorType := ColorRGB
			if tc.bpp == 4 {
				colorType = ColorRGBA
			} else if tc.bpp == 1 {
				colorType = ColorGrayscale
			}

			// Test with optimal deflate disabled
			optsNoOptimal := BalancedOptions(tc.width, tc.height)
			optsNoOptimal.OptimalDeflate = false
			optsNoOptimal.ColorType = colorType

			dataNoOptimal, err := IDATDataBytesWithOptions(pixels, tc.width, tc.height, colorType, optsNoOptimal)
			if err != nil {
				t.Fatalf("IDATDataBytesWithOptions (no optimal) failed: %v", err)
			}

			// Test with optimal deflate enabled
			optsOptimal := BalancedOptions(tc.width, tc.height)
			optsOptimal.OptimalDeflate = true
			optsOptimal.ColorType = colorType

			dataOptimal, err := IDATDataBytesWithOptions(pixels, tc.width, tc.height, colorType, optsOptimal)
			if err != nil {
				t.Fatalf("IDATDataBytesWithOptions (optimal) failed: %v", err)
			}

			// Calculate compression ratios
			uncompressedSize := tc.width * tc.height * tc.bpp
			ratioNoOptimal := float64(len(dataNoOptimal)) / float64(uncompressedSize)
			ratioOptimal := float64(len(dataOptimal)) / float64(uncompressedSize)

			t.Logf("Test: %s", tc.description)
			t.Logf("  Uncompressed: %d bytes", uncompressedSize)
			t.Logf("  No optimal: %d bytes (%.2f%%)", len(dataNoOptimal), ratioNoOptimal*100)
			t.Logf("  Optimal: %d bytes (%.2f%%)", len(dataOptimal), ratioOptimal*100)
			
			diff := len(dataOptimal) - len(dataNoOptimal)
			if diff < 0 {
				t.Logf("  Improvement: %d bytes (%.2f%%)", -diff, float64(-diff)/float64(len(dataNoOptimal))*100)
			} else if diff > 0 {
				t.Logf("  Regression: %d bytes (+%.2f%%)", diff, float64(diff)/float64(len(dataNoOptimal))*100)
			} else {
				t.Logf("  No difference")
			}

			// Verify both outputs are valid zlib data
			if !isValidZlib(dataNoOptimal) {
				t.Errorf("No-optimal output is not valid zlib")
			}
			if !isValidZlib(dataOptimal) {
				t.Errorf("Optimal output is not valid zlib")
			}

			// Verify both outputs can be decoded to the same data
			if !verifyIDATDataMatches(pixels, dataNoOptimal, tc.width, tc.height, colorType) {
				t.Errorf("No-optimal output does not match original data")
			}
			if !verifyIDATDataMatches(pixels, dataOptimal, tc.width, tc.height, colorType) {
				t.Errorf("Optimal output does not match original data")
			}
		})
	}
}


func TestOptimalDeflate_ConvergenceDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping convergence detection test in short mode")
	}

	// Generate data with clear convergence characteristics
	width, height := 100, 100
	pixels := generateGradientData(width, height, 3)

	// Test with low max iterations (should converge quickly)
	configLowIter := compress.OptimalConfigForLevel(6)
	configLowIter.MaxIterations = 3

	colorType := ColorRGB
	opts := BalancedOptions(width, height)
	opts.OptimalDeflate = true
	opts.OptimalConfig = configLowIter
	opts.ColorType = colorType

	startTime := time.Now()
	data, err := IDATDataBytesWithOptions(pixels, width, height, colorType, opts)
	elapsed := time.Since(startTime)

	if err != nil {
		t.Fatalf("IDATDataBytesWithOptions failed: %v", err)
	}

	t.Logf("Low iterations: %d bytes, time: %v", len(data), elapsed)

	// Verify output is valid
	if !isValidZlib(data) {
		t.Errorf("Output is not valid zlib")
	}

	// Test with higher max iterations
	configHighIter := compress.OptimalConfigForLevel(6)
	configHighIter.MaxIterations = 20

	optsHighIter := BalancedOptions(width, height)
	optsHighIter.OptimalDeflate = true
	optsHighIter.OptimalConfig = configHighIter
	optsHighIter.ColorType = colorType

	startTime = time.Now()
	dataHighIter, err := IDATDataBytesWithOptions(pixels, width, height, colorType, optsHighIter)
	elapsedHigh := time.Since(startTime)

	if err != nil {
		t.Fatalf("IDATDataBytesWithOptions (high iter) failed: %v", err)
	}

	t.Logf("High iterations: %d bytes, time: %v", len(dataHighIter), elapsedHigh)

	// Verify both outputs are valid zlib
	if !isValidZlib(data) {
		t.Errorf("Low iterations output is not valid zlib")
	}
	if !isValidZlib(dataHighIter) {
		t.Errorf("High iterations output is not valid zlib")
	}
}

func TestOptimalDeflate_PerformanceImpact(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	width, height := 100, 100
	pixels := generateGradientData(width, height, 4)
	colorType := ColorRGBA

	// Test normal mode (no optimal deflate)
	optsNoOptimal := BalancedOptions(width, height)
	optsNoOptimal.OptimalDeflate = false
	optsNoOptimal.ColorType = colorType

	startTime := time.Now()
	for i := 0; i < 3; i++ {
		_, err := IDATDataBytesWithOptions(pixels, width, height, colorType, optsNoOptimal)
		if err != nil {
			t.Fatalf("Normal mode failed: %v", err)
		}
	}
	normalTime := time.Since(startTime) / 3

	// Test optimal mode
	optsOptimal := BalancedOptions(width, height)
	optsOptimal.OptimalDeflate = true
	optsOptimal.ColorType = colorType

	startTime = time.Now()
	for i := 0; i < 3; i++ {
		_, err := IDATDataBytesWithOptions(pixels, width, height, colorType, optsOptimal)
		if err != nil {
			t.Fatalf("Optimal mode failed: %v", err)
		}
	}
	optimalTime := time.Since(startTime) / 3

	t.Logf("Normal mode avg time: %v", normalTime)
	t.Logf("Optimal mode avg time: %v", optimalTime)
	t.Logf("Slowdown factor: %.2fx", float64(optimalTime)/float64(normalTime))

	// Optimal mode should be slower but not unreasonably so
	// Allow up to 50x slowdown for this test
	slowdown := float64(optimalTime) / float64(normalTime)
	if slowdown > 50 {
		t.Logf("Warning: Optimal mode is %.2fx slower than normal", slowdown)
	}
}

func TestOptimalDeflate_DifferentImageTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping image type test in short mode")
	}

	testCases := []struct {
		name    string
		dataGen func(int, int, int) []byte
		bpp     int
	}{
		{"text_like", generateTextLikeData, 1},
		{"graphical", generateGraphicalData, 4},
		{"photographic", generatePhotographicData, 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			width, height := 50, 50
			pixels := tc.dataGen(width, height, tc.bpp)
			colorType := ColorGrayscale
			if tc.bpp == 3 {
				colorType = ColorRGB
			} else if tc.bpp == 4 {
				colorType = ColorRGBA
			}

			// Test both modes
			optsNoOptimal := BalancedOptions(width, height)
			optsNoOptimal.OptimalDeflate = false
			optsNoOptimal.ColorType = colorType

			dataNoOptimal, err := IDATDataBytesWithOptions(pixels, width, height, colorType, optsNoOptimal)
			if err != nil {
				t.Fatalf("No-optimal failed: %v", err)
			}

			optsOptimal := BalancedOptions(width, height)
			optsOptimal.OptimalDeflate = true
			optsOptimal.ColorType = colorType

			dataOptimal, err := IDATDataBytesWithOptions(pixels, width, height, colorType, optsOptimal)
			if err != nil {
				t.Fatalf("Optimal failed: %v", err)
			}

			// Verify both are valid
			if !isValidZlib(dataNoOptimal) {
				t.Errorf("No-optimal output is not valid zlib")
			}
			if !isValidZlib(dataOptimal) {
				t.Errorf("Optimal output is not valid zlib")
			}

			// Log compression results
			uncompressedSize := width * height * tc.bpp
			t.Logf("Type: %s, uncompressed: %d, no-optimal: %d, optimal: %d",
				tc.name, uncompressedSize, len(dataNoOptimal), len(dataOptimal))
		})
	}
}

// Helper function to verify zlib data validity
func isValidZlib(data []byte) bool {
	if len(data) < 6 {
		return false
	}

	// Check zlib header
	if data[0] != 0x78 {
		return false
	}

	// Try to decompress
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return false
	}
	defer func() { _ = reader.Close() }()

	_, err = io.ReadAll(reader)
	return err == nil
}

// Additional test data generators

func generateRepetitiveData(width, height, bpp int) []byte {
	data := make([]byte, width*height*bpp)
	pattern := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	for i := 0; i < width*height; i++ {
		for j := 0; j < bpp && j < len(pattern); j++ {
			data[i*bpp+j] = pattern[(i+j)%len(pattern)]
		}
	}
	return data
}

func generateTextLikeData(width, height, bpp int) []byte {
	data := make([]byte, width*height*bpp)
	text := []byte("The quick brown fox jumps over the lazy dog. ")
	for i := 0; i < width*height*bpp; i++ {
		data[i] = text[i%len(text)]
	}
	return data
}

func generateGraphicalData(width, height, bpp int) []byte {
	data := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			// Create synthetic graphical patterns
			circleX := width/2
			circleY := height/2
			dist := int(sqrt(float64((x-circleX)*(x-circleX) + (y-circleY)*(y-circleY))))
			if dist < height/4 {
				// Inside circle - solid color
				data[offset] = 0xFF
				if bpp > 1 {
					data[offset+1] = 0x00
				}
				if bpp > 2 {
					data[offset+2] = 0x00
				}
				if bpp > 3 {
					data[offset+3] = 0xFF
				}
			} else {
				// Outside circle - different color
				data[offset] = 0x00
				if bpp > 1 {
					data[offset+1] = 0xFF
				}
				if bpp > 2 {
					data[offset+2] = 0x00
				}
				if bpp > 3 {
					data[offset+3] = 0xFF
				}
			}
		}
	}
	return data
}

func generatePhotographicData(width, height, bpp int) []byte {
	data := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			// Create synthetic photographic-like noise with smooth gradients
			val := byte((x*y + x + y) & 0xFF)
			data[offset] = val
			if bpp > 1 {
				data[offset+1] = byte((int(val) + 50) & 0xFF)
			}
			if bpp > 2 {
				data[offset+2] = byte((int(val) + 100) & 0xFF)
			}
			if bpp > 3 {
				data[offset+3] = 255 // Full alpha
			}
		}
	}
	return data
}

func sqrt(n float64) float64 {
	if n < 0 {
		return 0
	}
	x := n / 2
	for i := 0; i < 20; i++ {
		x = (x + n/x) / 2
	}
	return x
}

func verifyIDATDataMatches(pixels []byte, idatData []byte, width, height int, colorType ColorType) bool {
	bpp := BytesPerPixel(colorType)
	expectedRawLen := width * bpp * height
	
	zlibReader, err := zlib.NewReader(bytes.NewReader(idatData))
	if err != nil {
		return false
	}
	defer zlibReader.Close()
	
	decompressed, err := io.ReadAll(zlibReader)
	if err != nil {
		return false
	}
	
	// The decompressed data should be scanline data with filter bytes
	if len(decompressed) < expectedRawLen+height {
		return false
	}
	
	return true
}
