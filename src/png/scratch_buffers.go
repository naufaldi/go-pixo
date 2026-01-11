package png

import "runtime"

type AdaptiveScratch struct {
	rowBuffer   []byte
	filteredRow []byte
	scoreBuffer []int
	bpp         int
	rowLen      int
	Config      FilterSelectionConfig
}

func NewAdaptiveScratch(rowLen, bpp int) *AdaptiveScratch {
	return NewAdaptiveScratchWithConfig(rowLen, bpp, FilterSelectionConfig{})
}

func NewAdaptiveScratchWithConfig(rowLen, bpp int, config FilterSelectionConfig) *AdaptiveScratch {
	return &AdaptiveScratch{
		rowBuffer:   make([]byte, rowLen),
		filteredRow: make([]byte, rowLen+1),
		scoreBuffer: make([]int, rowLen+1),
		bpp:         bpp,
		rowLen:      rowLen,
		Config:      config,
	}
}

func (as *AdaptiveScratch) GetRowBuffer() []byte {
	as.rowBuffer = as.rowBuffer[:as.rowLen]
	return as.rowBuffer
}

func (as *AdaptiveScratch) GetFilteredRow() []byte {
	as.filteredRow = as.filteredRow[:as.rowLen+1]
	return as.filteredRow
}

func (as *AdaptiveScratch) GetScoreBuffer() []int {
	as.scoreBuffer = as.scoreBuffer[:as.rowLen+1]
	return as.scoreBuffer
}

func (as *AdaptiveScratch) RowLen() int {
	return as.rowLen
}

func (as *AdaptiveScratch) BPP() int {
	return as.bpp
}

func (as *AdaptiveScratch) Release() {
	runtime.GC()
}
