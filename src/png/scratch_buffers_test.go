package png

import (
	"runtime"
	"testing"
)

func TestAdaptiveScratch_BufferReuse(t *testing.T) {
	scratch := NewAdaptiveScratch(100, 4)

	row := scratch.GetRowBuffer()
	if len(row) != 100 {
		t.Errorf("expected row buffer length 100, got %d", len(row))
	}

	filtered := scratch.GetFilteredRow()
	if len(filtered) != 101 {
		t.Errorf("expected filtered row length 101, got %d", len(filtered))
	}

	scores := scratch.GetScoreBuffer()
	if len(scores) != 101 {
		t.Errorf("expected score buffer length 101, got %d", len(scores))
	}

	for i := 0; i < 1000; i++ {
		row := scratch.GetRowBuffer()
		filtered := scratch.GetFilteredRow()
		scores := scratch.GetScoreBuffer()

		for j := range row {
			row[j] = byte(i + j)
		}

		filtered[0] = 0
		for j := 1; j < len(filtered); j++ {
			filtered[j] = row[j-1]
		}

		for j := range scores {
			scores[j] = i + j
		}

		if len(row) != 100 || len(filtered) != 101 || len(scores) != 101 {
			t.Errorf("buffer lengths changed unexpectedly at iteration %d", i)
		}
	}
}

func TestAdaptiveScratch_NoAllocation(t *testing.T) {
	scratch := NewAdaptiveScratch(1000, 3)

	allocs := testing.AllocsPerRun(100, func() {
		_ = scratch.GetRowBuffer()
		_ = scratch.GetFilteredRow()
		_ = scratch.GetScoreBuffer()
	})

	if allocs > 0 {
		t.Errorf("expected zero allocations per call, got %f", allocs)
	}
}

func TestAdaptiveScratch_GCReduction(t *testing.T) {
	scratch := NewAdaptiveScratch(1000, 4)
	runtime.GC()
	var memStatsBefore, memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	for i := 0; i < 10000; i++ {
		row := scratch.GetRowBuffer()
		filtered := scratch.GetFilteredRow()
		_ = row
		_ = filtered
		_ = scratch.GetScoreBuffer()
	}

	runtime.GC()
	runtime.ReadMemStats(&memStatsAfter)

	allocsCount := memStatsAfter.Mallocs - memStatsBefore.Mallocs
	if allocsCount > 100 {
		t.Errorf("too many allocations: %d (expected < 100 with buffer reuse)", allocsCount)
	}
}

func TestAdaptiveScratch_ResetBehavior(t *testing.T) {
	scratch := NewAdaptiveScratch(50, 2)

	row := scratch.GetRowBuffer()
	for i := range row {
		row[i] = 0xFF
	}

	filtered := scratch.GetFilteredRow()
	for i := range filtered {
		filtered[i] = 0xFF
	}

	scores := scratch.GetScoreBuffer()
	for i := range scores {
		scores[i] = 9999
	}

	_ = scratch.GetRowBuffer()
	_ = scratch.GetFilteredRow()
	_ = scratch.GetScoreBuffer()

	if row[0] != 0xFF || filtered[0] != 0xFF || scores[0] != 9999 {
		t.Errorf("buffers should retain their data after reset")
	}
}

func TestAdaptiveScratch_MultipleInstances(t *testing.T) {
	scratch1 := NewAdaptiveScratch(100, 3)
	scratch2 := NewAdaptiveScratch(200, 4)

	row1 := scratch1.GetRowBuffer()
	row2 := scratch2.GetRowBuffer()

	if cap(row1) != 100 || cap(row2) != 200 {
		t.Errorf("unexpected buffer capacities: got %d, %d", cap(row1), cap(row2))
	}

	filtered1 := scratch1.GetFilteredRow()
	filtered2 := scratch2.GetFilteredRow()

	if cap(filtered1) != 101 || cap(filtered2) != 201 {
		t.Errorf("unexpected filtered row capacities: got %d, %d", cap(filtered1), cap(filtered2))
	}
}

func BenchmarkAdaptiveScratch_NoReuse(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		row := make([]byte, 1000)
		filtered := make([]byte, 1001)
		scores := make([]int, 1001)
		_ = row
		_ = filtered
		_ = scores
	}
}

func BenchmarkAdaptiveScratch_WithReuse(b *testing.B) {
	scratch := NewAdaptiveScratch(1000, 4)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = scratch.GetRowBuffer()
		_ = scratch.GetFilteredRow()
		_ = scratch.GetScoreBuffer()
	}
}
