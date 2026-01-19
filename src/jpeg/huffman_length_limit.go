package jpeg

import "sort"

type lengthLimitNode struct {
	symbol int
	freq   int
}

type lengthLimitLevel struct {
	level        int
	lastFreq     int
	nextCharFreq int
	nextPairFreq int
	needed       int
}

func buildLengthLimitedHuffmanTable(freq []int, maxLen int) (bits [16]uint8, vals []uint8) {
	nodes := make([]lengthLimitNode, 0, len(freq))
	for sym, f := range freq {
		if f > 0 {
			nodes = append(nodes, lengthLimitNode{symbol: sym, freq: f})
		}
	}

	if len(nodes) == 0 {
		bits[0] = 1
		return bits, []uint8{0}
	}
	if len(nodes) == 1 {
		bits[0] = 1
		return bits, []uint8{uint8(nodes[0].symbol)}
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].freq != nodes[j].freq {
			return nodes[i].freq < nodes[j].freq
		}
		return nodes[i].symbol < nodes[j].symbol
	})

	if maxLen > 16 {
		maxLen = 16
	}
	if maxLen > len(nodes)-1 {
		maxLen = len(nodes) - 1
	}
	if maxLen < 1 {
		maxLen = 1
	}

	bitCount := lengthLimitedBitCounts(nodes, maxLen)
	for i := 1; i <= maxLen; i++ {
		bits[i-1] = uint8(bitCount[i])
	}

	lengths := make(map[int]int, len(nodes))
	remaining := make([]lengthLimitNode, len(nodes))
	copy(remaining, nodes)
	for length := 1; length <= maxLen; length++ {
		count := bitCount[length]
		if count == 0 {
			continue
		}
		start := len(remaining) - count
		chunk := remaining[start:]
		sort.Slice(chunk, func(i, j int) bool {
			return chunk[i].symbol < chunk[j].symbol
		})
		for _, node := range chunk {
			lengths[node.symbol] = length
		}
		remaining = remaining[:start]
	}

	for length := 1; length <= maxLen; length++ {
		symbols := make([]int, 0, bitCount[length])
		for _, node := range nodes {
			if lengths[node.symbol] == length {
				symbols = append(symbols, node.symbol)
			}
		}
		sort.Ints(symbols)
		for _, sym := range symbols {
			vals = append(vals, uint8(sym))
		}
	}

	return bits, vals
}

func lengthLimitedBitCounts(nodes []lengthLimitNode, maxLen int) []int {
	n := len(nodes)
	if maxLen > n-1 {
		maxLen = n - 1
	}

	list := make([]lengthLimitNode, n+1)
	copy(list, nodes)
	list[n] = lengthLimitNode{symbol: 0, freq: int(^uint(0) >> 1)}

	levels := make([]lengthLimitLevel, maxLen+2)
	leafCounts := make([][]int, maxLen+2)
	for i := range leafCounts {
		leafCounts[i] = make([]int, maxLen+2)
	}

	for level := 1; level <= maxLen; level++ {
		levels[level] = lengthLimitLevel{
			level:        level,
			lastFreq:     list[1].freq,
			nextCharFreq: list[2].freq,
			nextPairFreq: list[0].freq + list[1].freq,
		}
		leafCounts[level][level] = 2
		if level == 1 {
			levels[level].nextPairFreq = list[n].freq
		}
	}

	levels[maxLen].needed = 2*n - 4
	level := maxLen
	for {
		l := &levels[level]
		if l.nextPairFreq == list[n].freq && l.nextCharFreq == list[n].freq {
			l.needed = 0
			if level < maxLen {
				levels[level+1].nextPairFreq = list[n].freq
			}
			level++
			continue
		}

		prevFreq := l.lastFreq
		if l.nextCharFreq < l.nextPairFreq {
			next := leafCounts[level][level] + 1
			l.lastFreq = l.nextCharFreq
			leafCounts[level][level] = next
			l.nextCharFreq = list[next].freq
		} else {
			l.lastFreq = l.nextPairFreq
			copy(leafCounts[level][:level], leafCounts[level-1][:level])
			levels[l.level-1].needed = 2
		}

		l.needed--
		if l.needed == 0 {
			if l.level == maxLen {
				break
			}
			levels[l.level+1].nextPairFreq = prevFreq + l.lastFreq
			level++
			continue
		}

		for levels[level-1].needed > 0 {
			level--
		}
	}

	if leafCounts[maxLen][maxLen] != n {
		panic("invalid length-limited counts")
	}

	bitCount := make([]int, maxLen+1)
	bits := 1
	counts := leafCounts[maxLen]
	for level := maxLen; level > 0; level-- {
		bitCount[bits] = counts[level] - counts[level-1]
		bits++
	}
	return bitCount
}
