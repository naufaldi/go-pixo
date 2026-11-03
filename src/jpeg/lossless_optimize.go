package jpeg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/mac/go-pixo/src/internal/imagelimit"
)

// OptimizeLossless rewrites a JPEG without changing its decoded pixels, in
// the manner of `jpegtran -optimize -progressive -copy none`.
//
// The input's quantized DCT coefficients, quantization tables and frame
// parameters are kept exactly; only the entropy coding changes. Two
// candidates are encoded, sequential with per-image optimal Huffman tables and
// progressive with libjpeg's default successive-approximation script and
// per-scan optimal tables, and the smaller one is returned. The result is not
// compared with the input; callers keep the input when it is smaller.
//
// Metadata follows the strip-metadata policy: ICC profiles (APP2) and the
// Adobe APP14 color-transform segment are copied verbatim, a JFIF APP0 header
// is kept without its thumbnail, EXIF is reduced to its Orientation tag (only
// when it is not 1), and all other APPn and COM segments are dropped.
//
// Baseline, extended-sequential Huffman and progressive inputs with one or
// three components, any sampling factors and restart intervals are supported.
// Unsupported inputs return an error matching ErrUnsupportedJPEG (see the
// ErrUnsupportedJPEG* sentinels); malformed or truncated inputs return an
// error matching ErrInvalidJPEG. Size limits from imagelimit apply before any
// coefficient storage is allocated.
func OptimizeLossless(input []byte) ([]byte, error) {
	if err := imagelimit.CheckEncodedSize(len(input)); err != nil {
		return nil, err
	}
	return optimizeLossless(input, anyLayout)
}

// losslessLayout restricts which output layouts optimizeLossless may pick.
type losslessLayout int

const (
	anyLayout losslessLayout = iota
	sequentialLayouts
	progressiveLayouts
)

// optimizeLossless is OptimizeLossless without the encoded-size limit, for
// input this package produced itself; dimensions are still bounded.
func optimizeLossless(input []byte, layout losslessLayout) ([]byte, error) {
	d, err := readCoefficients(input)
	if err != nil {
		return nil, err
	}
	orientation, err := jpegOrientation(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidJPEG, err)
	}
	return optimizeCoefficients(d, orientation, layout)
}

// optimizeCoefficients writes the smallest allowed layout of d's coefficients.
func optimizeCoefficients(d *coeffDecoder, orientation uint16, layout losslessLayout) ([]byte, error) {
	header, err := losslessHeader(d, orientation)
	if err != nil {
		return nil, err
	}

	var candidates []losslessCandidate
	for _, candidate := range losslessCandidates(d) {
		if layout == anyLayout || candidate.progressive == (layout == progressiveLayouts) {
			candidates = append(candidates, candidate)
		}
	}
	ranked, stats, err := rankCandidates(d, candidates)
	if err != nil {
		return nil, err
	}
	// Estimates ignore byte stuffing, so fully encode the two best layouts
	// and keep the smaller real output.
	var best []byte
	for _, index := range ranked[:min(2, len(ranked))] {
		out, err := encodeLossless(d, header, candidates[index], 0, stats)
		if err != nil {
			return nil, err
		}
		if best == nil || len(out) < len(best) {
			best = out
		}
	}
	return best, nil
}

// losslessCandidate is one complete output layout.
type losslessCandidate struct {
	progressive bool
	scans       []losslessScan
}

// losslessCandidates lists the layouts OptimizeLossless chooses from:
// sequential, libjpeg's progressive script, that script with DC sent in one
// scan (often smaller, as mozjpeg found), and spectral selection without
// successive approximation (best for noisy, coefficient-dense images).
func losslessCandidates(d *coeffDecoder) []losslessCandidate {
	return []losslessCandidate{
		{progressive: false, scans: sequentialScript(d)},
		{progressive: true, scans: progressiveScript(d, true)},
		{progressive: true, scans: progressiveScript(d, false)},
		{progressive: true, scans: spectralScript(d)},
	}
}

// scanKey identifies a scan whose cost does not depend on other scans: each
// scan restarts DC prediction and EOB runs and gets its own tables.
type scanKey struct {
	comps          [coeffMaxComponents]int8
	ss, se, ah, al int
	progressive    bool
}

func newScanKey(scan losslessScan, progressive bool) scanKey {
	key := scanKey{ss: scan.ss, se: scan.se, ah: scan.ah, al: scan.al, progressive: progressive}
	for i := range key.comps {
		key.comps[i] = -1
	}
	for i, ci := range scan.comps {
		key.comps[i] = int8(ci)
	}
	return key
}

// rankCandidates returns candidate indexes ordered by their estimated size,
// computed from symbol counts and optimal code lengths without writing data,
// plus the per-scan counts so that encoding does not count them again.
func rankCandidates(d *coeffDecoder, candidates []losslessCandidate) ([]int, map[scanKey]*scanStats, error) {
	stats := make(map[scanKey]*scanStats)
	e := &entropyEncoder{d: d, stats: stats}
	estimates := make([]int64, len(candidates))
	for i, candidate := range candidates {
		for _, scan := range candidate.scans {
			cost, err := e.scanCost(scan, scan.kind(candidate.progressive))
			if err != nil {
				return nil, nil, err
			}
			estimates[i] += cost
		}
	}
	ranked := make([]int, len(candidates))
	for i := range ranked {
		ranked[i] = i
	}
	sort.SliceStable(ranked, func(a, b int) bool { return estimates[ranked[a]] < estimates[ranked[b]] })
	return ranked, stats, nil
}

// canInterleave reports whether all components fit one interleaved MCU
// (section B.2.3 limits the sum of H*V to 10).
func canInterleave(d *coeffDecoder) bool {
	if d.nComp == 1 {
		return true
	}
	total := 0
	for i := 0; i < d.nComp; i++ {
		total += d.comp[i].h * d.comp[i].v
	}
	return total <= 10
}

func allComponents(d *coeffDecoder) []int {
	comps := make([]int, d.nComp)
	for i := range comps {
		comps[i] = i
	}
	return comps
}

func sequentialScript(d *coeffDecoder) []losslessScan {
	if canInterleave(d) {
		return []losslessScan{{comps: allComponents(d), ss: 0, se: 63}}
	}
	scans := make([]losslessScan, d.nComp)
	for i := range scans {
		scans[i] = losslessScan{comps: []int{i}, ss: 0, se: 63}
	}
	return scans
}

// dcScans returns the DC scans for one successive-approximation step,
// interleaved when the sampling factors allow it.
func dcScans(d *coeffDecoder, ah, al int) []losslessScan {
	if canInterleave(d) {
		return []losslessScan{{comps: allComponents(d), ah: ah, al: al}}
	}
	scans := make([]losslessScan, d.nComp)
	for i := range scans {
		scans[i] = losslessScan{comps: []int{i}, ah: ah, al: al}
	}
	return scans
}

func acScan(ci, ss, se, ah, al int) losslessScan {
	return losslessScan{comps: []int{ci}, ss: ss, se: se, ah: ah, al: al}
}

// progressiveScript returns libjpeg's jpeg_simple_progression script. With
// dcRefine false, DC is sent at full precision in the first scan instead.
func progressiveScript(d *coeffDecoder, dcRefine bool) []losslessScan {
	dcAl := 0
	if dcRefine {
		dcAl = 1
	}
	scans := dcScans(d, 0, dcAl)
	if d.nComp == 3 {
		scans = append(scans,
			acScan(0, 1, 5, 0, 2), // Get some luma data out in a hurry.
			acScan(2, 1, 63, 0, 1),
			acScan(1, 1, 63, 0, 1),
			acScan(0, 6, 63, 0, 2),
			acScan(0, 1, 63, 2, 1),
		)
	} else {
		scans = append(scans,
			acScan(0, 1, 5, 0, 2),
			acScan(0, 6, 63, 0, 2),
			acScan(0, 1, 63, 2, 1),
		)
	}
	if dcRefine {
		scans = append(scans, dcScans(d, 1, 0)...)
	}
	if d.nComp == 3 {
		return append(scans,
			acScan(2, 1, 63, 1, 0),
			acScan(1, 1, 63, 1, 0),
			acScan(0, 1, 63, 1, 0), // Luma bottom bit comes last; it is usually largest.
		)
	}
	return append(scans, acScan(0, 1, 63, 1, 0))
}

// spectralScript splits bands without successive approximation.
func spectralScript(d *coeffDecoder) []losslessScan {
	scans := dcScans(d, 0, 0)
	if d.nComp == 3 {
		return append(scans,
			acScan(0, 1, 5, 0, 0),
			acScan(2, 1, 63, 0, 0),
			acScan(1, 1, 63, 0, 0),
			acScan(0, 6, 63, 0, 0),
		)
	}
	return append(scans, acScan(0, 1, 5, 0, 0), acScan(0, 6, 63, 0, 0))
}

func writeSegment(out *bytes.Buffer, marker byte, payload []byte) error {
	if len(payload) > 0xffff-2 {
		return coeffFormatError("metadata segment too large")
	}
	out.Write([]byte{0xff, marker})
	var length [2]byte
	binary.BigEndian.PutUint16(length[:], uint16(len(payload)+2))
	out.Write(length[:])
	out.Write(payload)
	return nil
}

// exifOrientationPayload builds a minimal big-endian EXIF APP1 payload whose
// IFD0 holds only the Orientation tag.
func exifOrientationPayload(orientation uint16) []byte {
	payload := []byte{
		'E', 'x', 'i', 'f', 0, 0,
		'M', 'M', 0, 42, 0, 0, 0, 8, // TIFF header, IFD0 at offset 8.
		0, 1, // One entry.
		0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 0, 0, 0, // Orientation, SHORT, count 1.
		0, 0, 0, 0, // No next IFD.
	}
	binary.BigEndian.PutUint16(payload[24:], orientation)
	return payload
}

// losslessHeader writes SOI, the preserved metadata and the DQT segments.
func losslessHeader(d *coeffDecoder, orientation uint16) ([]byte, error) {
	out := new(bytes.Buffer)
	out.Write([]byte{0xff, mkSOI})
	if d.jfif != nil {
		// Keep JFIF's version and density, drop any thumbnail. The marker's
		// presence also selects YCbCr in decoders, so it must survive.
		jfif := []byte{'J', 'F', 'I', 'F', 0, 1, 1, 0, 0, 1, 0, 1, 0, 0}
		if len(d.jfif) >= 12 {
			copy(jfif, d.jfif[:12])
		}
		if err := writeSegment(out, mkAPP0, jfif); err != nil {
			return nil, err
		}
	}
	if orientation != 1 {
		if err := writeSegment(out, mkAPP0+1, exifOrientationPayload(orientation)); err != nil {
			return nil, err
		}
	}
	for _, icc := range d.iccSegments {
		if err := writeSegment(out, mkAPP2, icc); err != nil {
			return nil, err
		}
	}
	if d.adobe != nil {
		if err := writeSegment(out, mkAPP14, d.adobe); err != nil {
			return nil, err
		}
	}

	var written [coeffMaxTq + 1]bool
	for i := 0; i < d.nComp; i++ {
		tq := d.comp[i].tq
		if written[tq] {
			continue
		}
		written[tq] = true
		table := &d.quant[tq]
		wide := quantTableIsWide(table)
		payload := []byte{tq}
		if wide {
			payload[0] |= 0x10
		}
		for _, q := range table {
			if wide {
				payload = append(payload, byte(q>>8), byte(q))
			} else {
				payload = append(payload, byte(q))
			}
		}
		if err := writeSegment(out, mkDQT, payload); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

func quantTableIsWide(table *[coeffBlockSize]uint16) bool {
	for _, q := range table {
		if q > 0xff {
			return true
		}
	}
	return false
}

func usesWideQuant(d *coeffDecoder) bool {
	for i := 0; i < d.nComp; i++ {
		if quantTableIsWide(&d.quant[d.comp[i].tq]) {
			return true
		}
	}
	return false
}

// encodeLossless writes one complete candidate. restartInterval is only
// non-zero when tests build restart-marker fixtures. stats may hold counts
// gathered by rankCandidates; nil counts every scan again.
func encodeLossless(d *coeffDecoder, header []byte, candidate losslessCandidate, restartInterval int, stats map[scanKey]*scanStats) ([]byte, error) {
	out := new(bytes.Buffer)
	out.Grow(len(header) + len(d.coeffs)/8)
	out.Write(header)

	marker := byte(mkSOF0)
	switch {
	case candidate.progressive:
		marker = mkSOF2
	case usesWideQuant(d):
		marker = mkSOF1 // Baseline forbids 16-bit quantization tables.
	}
	sof := []byte{8, byte(d.height >> 8), byte(d.height), byte(d.width >> 8), byte(d.width), byte(d.nComp)}
	for i := 0; i < d.nComp; i++ {
		sof = append(sof, d.comp[i].c, d.comp[i].hv, d.comp[i].tq)
	}
	if err := writeSegment(out, marker, sof); err != nil {
		return nil, err
	}
	if restartInterval > 0 {
		if err := writeSegment(out, mkDRI, []byte{byte(restartInterval >> 8), byte(restartInterval)}); err != nil {
			return nil, err
		}
	}

	if restartInterval > 0 {
		stats = nil // Restarts change EOB runs, so cached counts do not apply.
	}
	e := &entropyEncoder{d: d, out: out, restartInterval: restartInterval, stats: stats}
	for _, scan := range candidate.scans {
		if err := e.writeScan(scan, scan.kind(candidate.progressive)); err != nil {
			return nil, err
		}
	}
	out.Write([]byte{0xff, mkEOI})
	return out.Bytes(), nil
}

// scanTables reports which table classes and destinations a scan uses.
func scanTables(scan losslessScan, kind losslessScanKind) (usesDC, usesAC bool, slots [2]bool) {
	usesDC = kind == scanSequential || kind == scanDCFirst
	usesAC = kind == scanSequential || kind == scanACFirst || kind == scanACRefine
	for _, ci := range scan.comps {
		slots[huffmanSlot(ci)] = true
	}
	return usesDC, usesAC, slots
}

// scanStats is the result of a scan's counting pass.
type scanStats struct {
	dcFreq, acFreq [2][256]int
	extraBits      int64
}

// gatherScan runs the counting pass of a scan, reusing cached counts.
func (e *entropyEncoder) gatherScan(scan losslessScan, kind losslessScanKind) error {
	key := newScanKey(scan, kind != scanSequential)
	if cached, ok := e.stats[key]; ok {
		e.dcFreq, e.acFreq, e.extraBits = cached.dcFreq, cached.acFreq, cached.extraBits
		return nil
	}
	e.dcFreq = [2][256]int{}
	e.acFreq = [2][256]int{}
	e.extraBits = 0
	e.gather = true
	if err := e.encodeScan(scan, kind); err != nil {
		return err
	}
	if e.stats != nil {
		e.stats[key] = &scanStats{dcFreq: e.dcFreq, acFreq: e.acFreq, extraBits: e.extraBits}
	}
	return nil
}

// scanCost returns the size in bits of a scan's DHT, SOS and entropy-coded
// segments, excluding byte stuffing.
func (e *entropyEncoder) scanCost(scan losslessScan, kind losslessScanKind) (int64, error) {
	if err := e.gatherScan(scan, kind); err != nil {
		return 0, err
	}
	usesDC, usesAC, slots := scanTables(scan, kind)
	total := e.extraBits + 7 + 8*int64(4+1+2*len(scan.comps)+3)
	tableCost := func(freq *[256]int) error {
		bitCounts, vals, err := optimalHuffmanTable(freq)
		if err != nil {
			return err
		}
		total += 8 * int64(4+1+16+len(vals))
		index := 0
		for length, count := range bitCounts {
			for ; count > 0; count-- {
				total += int64(freq[vals[index]]) * int64(length+1)
				index++
			}
		}
		return nil
	}
	for slot, used := range slots {
		if !used {
			continue
		}
		if usesDC {
			if err := tableCost(&e.dcFreq[slot]); err != nil {
				return 0, err
			}
		}
		if usesAC {
			if err := tableCost(&e.acFreq[slot]); err != nil {
				return 0, err
			}
		}
	}
	return total, nil
}

// writeScan counts symbols, writes optimal tables and SOS, then the data.
func (e *entropyEncoder) writeScan(scan losslessScan, kind losslessScanKind) error {
	if err := e.gatherScan(scan, kind); err != nil {
		return err
	}
	usesDC, usesAC, slots := scanTables(scan, kind)
	for slot, used := range slots {
		if !used {
			continue
		}
		if usesDC {
			if err := e.defineTable(0, slot, &e.dcFreq[slot], &e.dcCodes[slot]); err != nil {
				return err
			}
		}
		if usesAC {
			if err := e.defineTable(1, slot, &e.acFreq[slot], &e.acCodes[slot]); err != nil {
				return err
			}
		}
	}

	sos := []byte{byte(len(scan.comps))}
	for _, ci := range scan.comps {
		slot := byte(huffmanSlot(ci))
		var selectors byte
		if usesDC {
			selectors |= slot << 4
		}
		if usesAC {
			selectors |= slot
		}
		sos = append(sos, e.d.comp[ci].c, selectors)
	}
	sos = append(sos, byte(scan.ss), byte(scan.se), byte(scan.ah<<4|scan.al))
	if err := writeSegment(e.out, mkSOS, sos); err != nil {
		return err
	}

	e.gather = false
	e.bw = NewBitWriter(e.out)
	if err := e.encodeScan(scan, kind); err != nil {
		return err
	}
	return e.bw.Flush()
}

// defineTable builds an optimal length-limited table from freq, writes its
// DHT segment and fills codes.
func (e *entropyEncoder) defineTable(class, slot int, freq *[256]int, codes *[256]HuffCode) error {
	bitCounts, vals, err := optimalHuffmanTable(freq)
	if err != nil {
		return err
	}
	if err := WriteDHT(e.out, byte(class<<4|slot), bitCounts, vals); err != nil {
		return err
	}
	*codes = buildCodeTable256(bitCounts, vals)
	return nil
}

// reservedHuffmanSymbol is never a valid DC or AC symbol (size 15), so it can
// stand in for libjpeg's reserved code point.
const reservedHuffmanSymbol = 0xff

// optimalHuffmanTable returns a JPEG table with code lengths of at most 16
// bits in which no real symbol receives the all-ones code (section C).
func optimalHuffmanTable(freq *[256]int) (bitCounts [16]uint8, vals []uint8, err error) {
	if freq[reservedHuffmanSymbol] != 0 {
		return bitCounts, nil, coeffFormatError("internal: reserved Huffman symbol used")
	}
	// Doubling every count and giving the reserved symbol a count of 1 makes
	// it the unique rarest symbol, so it takes the last, all-ones code.
	weights := make([]int, 256)
	symbols := 0
	for i, f := range freq {
		if f > 0 {
			weights[i] = 2 * f
			symbols++
		}
	}
	if symbols == 0 {
		// Scans that never code a symbol still need a defined table.
		bitCounts[0] = 1
		return bitCounts, []uint8{0}, nil
	}
	weights[reservedHuffmanSymbol] = 1
	bitCounts, vals = buildLengthLimitedHuffmanTable(weights, 16)
	if len(vals) != symbols+1 || vals[len(vals)-1] != reservedHuffmanSymbol {
		return bitCounts, nil, coeffFormatError("internal: reserved Huffman code misplaced")
	}
	for length := 15; length >= 0; length-- {
		if bitCounts[length] > 0 {
			bitCounts[length]--
			break
		}
	}
	return bitCounts, vals[:len(vals)-1], nil
}
