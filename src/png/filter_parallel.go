package png

import (
	"runtime"
	"sync"
)

type ParallelConfig struct {
	NumWorkers int
	Threshold  int
	Strategy   FilterStrategy
}

func DefaultParallelConfig() ParallelConfig {
	return ParallelConfig{
		NumWorkers: runtime.NumCPU(),
		Threshold:  32,
		Strategy:   FilterStrategyAdaptive,
	}
}

func SelectAllParallel(pixels []byte, width, height, bpp int) []FilterType {
	return SelectAllParallelWithConfig(pixels, width, height, bpp, DefaultParallelConfig())
}

func SelectAllParallelWithConfig(pixels []byte, width, height, bpp int, config ParallelConfig) []FilterType {
	if height <= config.Threshold {
		return SelectAllWithStrategy(pixels, width, height, bpp, config.Strategy)
	}

	if config.NumWorkers <= 0 {
		config.NumWorkers = runtime.NumCPU()
	}
	if config.Threshold <= 0 {
		config.Threshold = 32
	}
	if config.Strategy == 0 {
		config.Strategy = FilterStrategyAdaptive
	}

	rowsPerWorker := (height + config.NumWorkers - 1) / config.NumWorkers
	if rowsPerWorker < config.Threshold {
		rowsPerWorker = config.Threshold
	}

	numWorkers := (height + rowsPerWorker - 1) / rowsPerWorker
	if numWorkers > config.NumWorkers {
		numWorkers = config.NumWorkers
	}

	var wg sync.WaitGroup
	results := make([][]FilterType, numWorkers)
	rowOffsets := make([]int, numWorkers+1)

	for i := 0; i <= numWorkers; i++ {
		rowOffsets[i] = i * rowsPerWorker
	}
	if rowOffsets[numWorkers] > height {
		rowOffsets[numWorkers] = height
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startRow := rowOffsets[workerID]
			endRow := rowOffsets[workerID+1]
			chunkHeight := endRow - startRow

			workerResults := make([]FilterType, chunkHeight)

			var prevRow []byte
			if startRow > 0 {
				prevRow = pixels[(startRow-1)*width*bpp : startRow*width*bpp]
			}

			for y := 0; y < chunkHeight; y++ {
				globalRow := startRow + y
				offset := globalRow * width * bpp
				row := pixels[offset : offset+width*bpp]
				workerResults[y] = selectRowFilter(row, prevRow, bpp, config.Strategy)
				prevRow = row
			}

			results[workerID] = workerResults
		}(i)
	}

	wg.Wait()

	flatResults := make([]FilterType, height)
	for i := 0; i < numWorkers; i++ {
		startRow := rowOffsets[i]
		copy(flatResults[startRow:], results[i])
	}

	return flatResults
}

func selectRowFilter(row, prevRow []byte, bpp int, strategy FilterStrategy) FilterType {
	filterType, _ := SelectFilterWithStrategy(row, prevRow, bpp, strategy)
	return filterType
}

func selectRowFilterBigrams(row, prevRow []byte, bpp int) FilterType {
	filterType, _ := SelectFilterBigrams(row, prevRow, bpp)
	return filterType
}

func SelectAllParallelBigrams(pixels []byte, width, height, bpp int) []FilterType {
	return SelectAllParallelBigramsWithConfig(pixels, width, height, bpp, DefaultParallelConfig())
}

func SelectAllParallelBigramsWithConfig(pixels []byte, width, height, bpp int, config ParallelConfig) []FilterType {
	if height <= config.Threshold {
		return SelectAllBigrams(pixels, width, height, bpp)
	}

	if config.NumWorkers <= 0 {
		config.NumWorkers = runtime.NumCPU()
	}
	if config.Threshold <= 0 {
		config.Threshold = 32
	}

	rowsPerWorker := (height + config.NumWorkers - 1) / config.NumWorkers
	if rowsPerWorker < config.Threshold {
		rowsPerWorker = config.Threshold
	}

	numWorkers := (height + rowsPerWorker - 1) / rowsPerWorker
	if numWorkers > config.NumWorkers {
		numWorkers = config.NumWorkers
	}

	var wg sync.WaitGroup
	results := make([][]FilterType, numWorkers)
	rowOffsets := make([]int, numWorkers+1)

	for i := 0; i <= numWorkers; i++ {
		rowOffsets[i] = i * rowsPerWorker
	}
	if rowOffsets[numWorkers] > height {
		rowOffsets[numWorkers] = height
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startRow := rowOffsets[workerID]
			endRow := rowOffsets[workerID+1]
			chunkHeight := endRow - startRow

			workerResults := make([]FilterType, chunkHeight)

			var prevRow []byte
			if startRow > 0 {
				prevRow = pixels[(startRow-1)*width*bpp : startRow*width*bpp]
			}

			for y := 0; y < chunkHeight; y++ {
				globalRow := startRow + y
				offset := globalRow * width * bpp
				row := pixels[offset : offset+width*bpp]
				workerResults[y] = selectRowFilterBigrams(row, prevRow, bpp)
				prevRow = row
			}

			results[workerID] = workerResults
		}(i)
	}

	wg.Wait()

	flatResults := make([]FilterType, height)
	for i := 0; i < numWorkers; i++ {
		startRow := rowOffsets[i]
		copy(flatResults[startRow:], results[i])
	}

	return flatResults
}
