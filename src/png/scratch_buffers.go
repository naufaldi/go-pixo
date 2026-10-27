package png

// AdaptiveScratch holds reusable per-row buffers for filter selection, sized for one scanline.
type AdaptiveScratch struct {
	rowBuffer   []byte
	filteredRow []byte
	scoreBuffer []int
	bpp         int
	rowLen      int
	Config      FilterSelectionConfig
}

// NewAdaptiveScratch allocates scratch buffers for rows of rowLen bytes with a zero FilterSelectionConfig.
func NewAdaptiveScratch(rowLen, bpp int) *AdaptiveScratch {
	return NewAdaptiveScratchWithConfig(rowLen, bpp, FilterSelectionConfig{})
}

// NewAdaptiveScratchWithConfig allocates scratch buffers for rows of rowLen bytes with the given selection config.
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

// GetRowBuffer returns the reusable rowLen-byte row buffer.
func (as *AdaptiveScratch) GetRowBuffer() []byte {
	as.rowBuffer = as.rowBuffer[:as.rowLen]
	return as.rowBuffer
}

// GetFilteredRow returns the reusable buffer for a filter-type byte plus a filtered row.
func (as *AdaptiveScratch) GetFilteredRow() []byte {
	as.filteredRow = as.filteredRow[:as.rowLen+1]
	return as.filteredRow
}

// GetScoreBuffer returns the reusable rowLen+1 integer scoring buffer.
func (as *AdaptiveScratch) GetScoreBuffer() []int {
	as.scoreBuffer = as.scoreBuffer[:as.rowLen+1]
	return as.scoreBuffer
}

// RowLen returns the scanline length in bytes the scratch was sized for.
func (as *AdaptiveScratch) RowLen() int {
	return as.rowLen
}

// BPP returns the bytes per pixel the scratch was created with.
func (as *AdaptiveScratch) BPP() int {
	return as.bpp
}

// Release is retained for compatibility. Scratch is owned by the caller and
// reclaimed by Go normally; releasing it must not force a process-wide GC.
func (as *AdaptiveScratch) Release() {}
