package compress

import (
	"bytes"
	"compress/flate"
	"io"
	"math/rand"
	"strconv"
	"testing"
)

func compressionWorkCorpus() map[string][]byte {
	corpus := map[string][]byte{}
	for _, name := range []string{"gradient", "screenshot", "photo", "incompressible"} {
		data := make([]byte, 64*64*3)
		rng := rand.New(rand.NewSource(47))
		for i := range data {
			x, y := (i/3)%64, i/(64*3)
			switch name {
			case "gradient":
				data[i] = byte(x + y)
			case "screenshot":
				data[i] = byte((x/12 + y/8) % 4 * 60)
			case "photo":
				data[i] = byte(x + y + rng.Intn(24))
			default:
				data[i] = byte(rng.Intn(256))
			}
		}
		corpus[name] = data
	}
	return corpus
}

func TestCompressionWorkRoundTripAndQuality(t *testing.T) {
	for name, data := range compressionWorkCorpus() {
		t.Run(name, func(t *testing.T) {
			// Retain both historically useful parses, including the level-6 baseline.
			bestSize := len(data) * 2
			for _, level := range []int{6, 9} {
				enc := NewDeflateEncoder()
				enc.SetCompressionLevel(level)
				for _, dynamic := range []bool{false, true} {
					candidate, err := enc.Encode(data, dynamic)
					if err != nil {
						if !dynamic {
							t.Fatal(err)
						}
						continue // The existing dynamic coder may reject a frequency table.
					}
					if validateDeflateRoundTrip(candidate, data) && len(candidate) < bestSize {
						bestSize = len(candidate)
					}
				}
			}
			for _, iterations := range []int{1, 3, 20} {
				out, err := ZopfliEncode(data, ZopfliConfig{Iterations: iterations})
				if err != nil {
					t.Fatal(err)
				}
				reader := flate.NewReader(bytes.NewReader(out))
				decoded, readErr := io.ReadAll(reader)
				closeErr := reader.Close()
				if readErr != nil || closeErr != nil || !bytes.Equal(decoded, data) {
					t.Fatalf("iterations=%d round trip: read=%v close=%v equal=%v", iterations, readErr, closeErr, bytes.Equal(decoded, data))
				}
				if len(out) > bestSize {
					t.Errorf("iterations=%d size=%d exceeds preserved candidate=%d", iterations, len(out), bestSize)
				}
				t.Logf("iterations=%d size=%d baseline=%d", iterations, len(out), bestSize)
			}
		})
	}
}

func TestCompressionWorkReportsOnlyDistinctTrials(t *testing.T) {
	for _, requested := range []int{1, 2, 3, 15, 20} {
		t.Run(strconv.Itoa(requested), func(t *testing.T) {
			wantTrials := min(requested, 3)
			var completed []int
			_, err := ZopfliEncode([]byte("abcabcabcabc"), ZopfliConfig{
				Iterations: requested,
				ProgressCallback: func(done, total int) {
					completed = append(completed, done)
					if total != wantTrials || done != len(completed) {
						t.Errorf("progress=(%d,%d), want (%d,%d) actual trials", done, total, len(completed), wantTrials)
					}
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(completed) != wantTrials {
				t.Errorf("reported %d trials, want %d", len(completed), wantTrials)
			}
		})
	}
}

func TestCompressionWorkRetainsUsefulParses(t *testing.T) {
	for _, tc := range []struct {
		name       string
		seed       int64
		iterations int
		maxSize    int
	}{
		// Level 6 beats level 9 here (1962 vs 2001 bytes).
		{"baseline", 0, 1, 1962},
		// Adding level 7 finds a smaller parse (1962 vs 1971 bytes).
		{"first trial", 2, 1, 1971},
		{"additional effort", 2, 2, 1962},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(tc.seed))
			data := make([]byte, 12000)
			for i := range data {
				data[i] = byte(rng.Intn(2))
			}
			out, err := ZopfliEncode(data, ZopfliConfig{Iterations: tc.iterations})
			if err != nil {
				t.Fatal(err)
			}
			if len(out) > tc.maxSize {
				t.Errorf("size=%d, want <=%d", len(out), tc.maxSize)
			}
			reader := flate.NewReader(bytes.NewReader(out))
			decoded, readErr := io.ReadAll(reader)
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil || !bytes.Equal(decoded, data) {
				t.Fatalf("round trip: read=%v close=%v equal=%v", readErr, closeErr, bytes.Equal(decoded, data))
			}
		})
	}
}

func BenchmarkCompressionWork(b *testing.B) {
	for name, data := range compressionWorkCorpus() {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := ZopfliEncodeSimple(data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
