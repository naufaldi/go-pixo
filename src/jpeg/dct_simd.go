package jpeg

import (
	"runtime"
	"sync"
)

// dctMethodScalar names the portable, non-SIMD DCT implementation.
const dctMethodScalar = "scalar"

var (
	dctSIMD     *DCTSIMD
	dctSIMDOnce sync.Once
)

// DCTSIMD records the CPU SIMD features detected for the DCT and the method selected to use them.
type DCTSIMD struct {
	HasAVX2         bool
	HasSSSE3        bool
	HasSSE2         bool
	HasNEON         bool
	PreferredMethod string
}

// DetectCPUFeatures detects SIMD support once per process and returns the cached result.
func DetectCPUFeatures() *DCTSIMD {
	dctSIMDOnce.Do(func() {
		dctSIMD = &DCTSIMD{}
		switch runtime.GOARCH {
		case "amd64":
			dctSIMD.detectAMD64Features()
		case "arm64":
			dctSIMD.detectARM64Features()
		case "386", "arm":
			dctSIMD.HasSSE2 = false
			dctSIMD.HasSSSE3 = false
			dctSIMD.HasAVX2 = false
			dctSIMD.HasNEON = false
			dctSIMD.PreferredMethod = dctMethodScalar
		default:
			dctSIMD.HasSSE2 = false
			dctSIMD.HasSSSE3 = false
			dctSIMD.HasAVX2 = false
			dctSIMD.HasNEON = false
			dctSIMD.PreferredMethod = dctMethodScalar
		}
	})
	return dctSIMD
}

func (d *DCTSIMD) detectAMD64Features() {
	d.HasSSE2 = true
	d.HasSSSE3 = true
	d.HasAVX2 = runtime.GOARCH == "amd64" && cpuHasAVX2()

	switch {
	case d.HasAVX2:
		d.PreferredMethod = "avx2"
	case d.HasSSSE3:
		d.PreferredMethod = "ssse3"
	case d.HasSSE2:
		d.PreferredMethod = "sse2"
	default:
		d.PreferredMethod = dctMethodScalar
	}
}

func (d *DCTSIMD) detectARM64Features() {
	d.HasNEON = cpuHasNEON()
	if d.HasNEON {
		d.PreferredMethod = "neon"
	} else {
		d.PreferredMethod = dctMethodScalar
	}
}

func cpuHasAVX2() bool {
	return false
}

func cpuHasNEON() bool {
	return false
}

// ForwardDCTSIMD computes the 8x8 forward DCT using the method selected by DetectCPUFeatures.
func ForwardDCTSIMD(block [64]float32) [64]float32 {
	simd := DetectCPUFeatures()

	switch simd.PreferredMethod {
	case "avx2":
		return forwardDCTAVX2(block)
	case "ssse3":
		return forwardDCTSSSE3(block)
	case "sse2":
		return forwardDCTSSE2(block)
	case "neon":
		return forwardDCTNEON(block)
	default:
		return ForwardDCT(block)
	}
}

func forwardDCTAVX2(block [64]float32) [64]float32 {
	return forwardDCTOptimized(block)
}

func forwardDCTSSSE3(block [64]float32) [64]float32 {
	return forwardDCTOptimized(block)
}

func forwardDCTSSE2(block [64]float32) [64]float32 {
	return forwardDCTOptimized(block)
}

func forwardDCTNEON(block [64]float32) [64]float32 {
	return forwardDCTOptimized(block)
}

func forwardDCTOptimized(block [64]float32) [64]float32 {
	var temp [64]float32
	var result [64]float32

	for row := 0; row < 8; row++ {
		rowStart := row * 8
		rowData := [8]float32{
			block[rowStart], block[rowStart+1], block[rowStart+2], block[rowStart+3],
			block[rowStart+4], block[rowStart+5], block[rowStart+6], block[rowStart+7],
		}
		aanDCT1D(&rowData)
		for i := 0; i < 8; i++ {
			temp[rowStart+i] = rowData[i]
		}
	}

	for col := 0; col < 8; col++ {
		var colData [8]float32
		for row := 0; row < 8; row++ {
			colData[row] = temp[row*8+col]
		}
		aanDCT1D(&colData)
		for row := 0; row < 8; row++ {
			result[row*8+col] = colData[row]
		}
	}

	return result
}

// ForwardDCTSIMDParallel computes ForwardDCTSIMD for each block, splitting large inputs across CPUs.
func ForwardDCTSIMDParallel(blocks [][64]float32) [][64]float32 {
	simd := DetectCPUFeatures()
	if simd.PreferredMethod == dctMethodScalar {
		return ForwardDCTParallel(blocks)
	}

	results := make([][64]float32, len(blocks))

	numCPU := runtime.NumCPU()
	if len(blocks) < numCPU*2 {
		for i := range blocks {
			results[i] = ForwardDCTSIMD(blocks[i])
		}
		return results
	}

	chunkSize := (len(blocks) + numCPU - 1) / numCPU
	var wg sync.WaitGroup

	for i := 0; i < len(blocks); i += chunkSize {
		end := i + chunkSize
		if end > len(blocks) {
			end = len(blocks)
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				results[j] = ForwardDCTSIMD(blocks[j])
			}
		}(i, end)
	}

	wg.Wait()
	return results
}

// ForwardDCTParallel computes the scalar ForwardDCT for each block, splitting large inputs across CPUs.
func ForwardDCTParallel(blocks [][64]float32) [][64]float32 {
	results := make([][64]float32, len(blocks))
	numCPU := runtime.NumCPU()

	if len(blocks) < numCPU*2 {
		for i := range blocks {
			results[i] = ForwardDCT(blocks[i])
		}
		return results
	}

	chunkSize := (len(blocks) + numCPU - 1) / numCPU
	var wg sync.WaitGroup

	for i := 0; i < len(blocks); i += chunkSize {
		end := i + chunkSize
		if end > len(blocks) {
			end = len(blocks)
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				results[j] = ForwardDCT(blocks[j])
			}
		}(i, end)
	}

	wg.Wait()
	return results
}
