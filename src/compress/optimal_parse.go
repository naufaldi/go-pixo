package compress

import (
	"bytes"
	"math"
)

const (
	defaultMaxIterations        = 20
	defaultConvergenceThreshold = 0.001
	defaultBlockSize            = 65535
)

type OptimalConfig struct {
	MaxIterations        int
	ConvergenceThreshold float64
	BlockSize            int
	BlockSplitting       bool
	MinMatchLen          int
	MaxChainLen          int
	ProgressCallback     func(iteration, improvement float64)
}

func DefaultOptimalConfig() OptimalConfig {
	return OptimalConfig{
		MaxIterations:        defaultMaxIterations,
		ConvergenceThreshold: defaultConvergenceThreshold,
		BlockSize:            defaultBlockSize,
		BlockSplitting:       true,
		MinMatchLen:          3,
		MaxChainLen:          256,
	}
}

func OptimalConfigForLevel(level int) OptimalConfig {
	config := DefaultOptimalConfig()
	switch level {
	case 1, 2:
		config.MaxIterations = 5
		config.BlockSplitting = false
		config.MaxChainLen = 32
	case 3, 4:
		config.MaxIterations = 10
		config.BlockSplitting = true
		config.MaxChainLen = 128
	case 5, 6:
		config.MaxIterations = 15
		config.BlockSplitting = true
		config.MaxChainLen = 256
	case 7, 8:
		config.MaxIterations = 20
		config.BlockSplitting = true
		config.MaxChainLen = 512
	case 9:
		config.MaxIterations = 30
		config.BlockSplitting = true
		config.MaxChainLen = 1024
	}
	return config
}

type parseResult struct {
	tokens []Token
	cost   float64
	size   int
}

func estimateBlockCost(tokens []Token, useDynamic bool) float64 {
	if len(tokens) == 0 {
		return 0
	}

	litFreq := make([]int, 286)
	distFreq := make([]int, 30)

	literalCount := 0
	matchCount := 0
	extraBits := 0

	for _, token := range tokens {
		if token.IsLiteral {
			if token.Literal < 144 {
				litFreq[token.Literal]++
			} else {
				litFreq[144+int(token.Literal)-144]++
			}
			literalCount++
		} else {
			length := int(token.Match.Length)
			lengthCode := findLengthCode(length)
			if lengthCode >= 257 && lengthCode <= 285 {
				litFreq[lengthCode]++
			}

			distance := int(token.Match.Distance)
			distanceCode := findDistanceCode(distance)
			if distanceCode >= 0 && distanceCode < 30 {
				distFreq[distanceCode]++
			}

			extraBits += int(LengthExtraBits[lengthCode-257])
			extraBits += int(DistanceExtraBits[distanceCode])
			matchCount++
		}
	}

	litFreq[EndOfBlockSymbol] = 1

	totalSymbols := literalCount + matchCount + 1
	if totalSymbols == 0 {
		return 0
	}

	cost := 0.0

	if useDynamic {
		litCost := calculateHuffmanCost(litFreq, totalSymbols)
		distCost := calculateHuffmanCost(distFreq, matchCount+1)
		cost = litCost + distCost
	} else {
		for i := 0; i < 286; i++ {
			if litFreq[i] > 0 {
				lengthCodeBits := 0
				if i >= 257 && i <= 279 {
					lengthCodeBits = 7
				} else if i >= 280 && i <= 285 {
					lengthCodeBits = 8
				}
				p := float64(litFreq[i]) / float64(totalSymbols)
				cost += p * float64(lengthCodeBits)
			}
		}

		for i := 0; i < 30; i++ {
			if distFreq[i] > 0 {
				distCodeBits := 0
				if i < 4 {
					distCodeBits = 5
				} else if i < 8 {
					distCodeBits = 5
				} else if i < 16 {
					distCodeBits = 7
				} else {
					distCodeBits = 8
				}
				p := float64(distFreq[i]) / float64(totalSymbols)
				cost += p * float64(distCodeBits)
			}
		}
	}

	cost += float64(extraBits)

	headerBits := 3.0
	if useDynamic {
		litHLIT := 0
		for i := 285; i >= 257; i-- {
			if litFreq[i] > 0 {
				litHLIT = i - 257
				break
			}
		}
		if litHLIT < 0 {
			litHLIT = 0
		}

		distHDIST := 0
		for i := 29; i >= 0; i-- {
			if distFreq[i] > 0 {
				distHDIST = i + 1
				break
			}
		}
		if distHDIST < 0 {
			distHDIST = 0
		}

		headerBits += 14
		headerBits += 5 + float64(litHLIT)
		headerBits += 5 + float64(distHDIST)

		litCodeCount := litHLIT + 257
		distCodeCount := distHDIST

		huffmanLengths := make([]int, litCodeCount)
		for i := 0; i < litCodeCount; i++ {
			huffmanLengths[i] = 8
		}

		headerBits += calculateHLITCost(litFreq, litCodeCount)
		headerBits += calculateHLITCost(distFreq, distCodeCount)
	}

	cost += headerBits

	alignmentBits := float64((8 - int(cost)%8) % 8)
	cost += alignmentBits

	return cost
}

func calculateHuffmanCost(freq []int, total int) float64 {
	cost := 0.0
	if total == 0 {
		return cost
	}

	nonZero := 0
	for _, f := range freq {
		if f > 0 {
			nonZero++
		}
	}

	if nonZero <= 1 {
		cost = float64(total) * 8.0
		return cost
	}

	for i := 0; i < len(freq); i++ {
		if freq[i] > 0 {
			p := float64(freq[i]) / float64(total)
			entropy := -p * math.Log2(p)
			cost += p * (entropy + 1.0)
		}
	}

	return cost
}

func calculateHLITCost(freq []int, maxCode int) float64 {
	nonZeroCount := 0
	for i := 0; i < maxCode && i < len(freq); i++ {
		if freq[i] > 0 {
			nonZeroCount++
		}
	}

	cost := 0.0
	for i := 0; i < maxCode && i < len(freq); i++ {
		if freq[i] > 0 {
			codeLen := 1
			if nonZeroCount > 1 {
				avgFreq := float64(totalFreq(freq[:maxCode])) / float64(nonZeroCount)
				if float64(freq[i]) > avgFreq {
					codeLen = 1
				} else if float64(freq[i]) > avgFreq/2.0 {
					codeLen = 2
				} else {
					codeLen = 3
				}
			}
			cost += 3.0 + float64(codeLen)
		}
	}

	return cost
}

func totalFreq(freq []int) int {
	total := 0
	for _, f := range freq {
		total += f
	}
	return total
}

type optimalParser struct {
	config     OptimalConfig
	data       []byte
	bestTokens []Token
	bestCost   float64
	bestSize   int
	history    []float64
}

func newOptimalParser(data []byte, config OptimalConfig) *optimalParser {
	if config.MaxIterations <= 0 {
		config.MaxIterations = defaultMaxIterations
	}
	if config.ConvergenceThreshold <= 0 {
		config.ConvergenceThreshold = defaultConvergenceThreshold
	}
	if config.BlockSize <= 0 {
		config.BlockSize = defaultBlockSize
	}
	if config.MinMatchLen <= 0 {
		config.MinMatchLen = MinMatchLength
	}
	if config.MaxChainLen <= 0 {
		config.MaxChainLen = 256
	}

	return &optimalParser{
		config:  config,
		data:    data,
		history: make([]float64, 0, config.MaxIterations),
	}
}

func (p *optimalParser) parseWithStrategy(strategy int) []Token {
	tokens := make([]Token, 0, len(p.data))
	pos := 0

	for pos < len(p.data) {
		remaining := len(p.data) - pos
		if remaining < p.config.MinMatchLen {
			tokens = append(tokens, TokenLiteral(p.data[pos]))
			pos++
			continue
		}

		matchLen := 0
		matchDist := 0

		maxMatchLen := MaxMatchLength
		if remaining < maxMatchLen {
			maxMatchLen = remaining
		}

		windowStart := 0
		if pos > MaxDistance {
			windowStart = pos - MaxDistance
		}

		switch strategy % 5 {
		case 0:
			matchLen, matchDist = p.greedyMatch(pos, windowStart, maxMatchLen)
		case 1:
			matchLen, matchDist = p.lazyMatch(pos, windowStart, maxMatchLen)
		case 2:
			matchLen, matchDist = p.qualityMatch(pos, windowStart, maxMatchLen)
		case 3:
			matchLen, matchDist = p.longestMatch(pos, windowStart, maxMatchLen)
		case 4:
			matchLen, matchDist = p.optimalMatch(pos, windowStart, maxMatchLen)
		}

		if matchLen >= p.config.MinMatchLen && matchDist > 0 {
			tokens = append(tokens, TokenMatch(uint16(matchDist), uint16(matchLen)))
			pos += matchLen
		} else {
			tokens = append(tokens, TokenLiteral(p.data[pos]))
			pos++
		}
	}

	return tokens
}

func (p *optimalParser) greedyMatch(pos, windowStart, maxLen int) (int, int) {
	if pos >= len(p.data) {
		return 0, 0
	}
	bestLen := 0
	bestDist := 0

	searchPos := pos - 1
	for searchPos >= windowStart && bestLen < maxLen {
		if p.data[searchPos] == p.data[pos] {
			matchLen := 0
			for matchLen < maxLen && pos+matchLen < len(p.data) &&
				p.data[searchPos+matchLen] == p.data[pos+matchLen] {
				matchLen++
			}
			if matchLen > bestLen {
				bestLen = matchLen
				bestDist = pos - searchPos
			}
		}
		searchPos--
	}

	return bestLen, bestDist
}

func (p *optimalParser) lazyMatch(pos, windowStart, maxLen int) (int, int) {
	firstLen, firstDist := p.greedyMatch(pos, windowStart, maxLen)

	if firstLen >= p.config.MinMatchLen {
		if firstLen < maxLen && pos+1 < len(p.data) {
			secondLen, _ := p.greedyMatch(pos+1, windowStart, maxLen-1)
			if secondLen > firstLen+1 {
				return 0, 0
			}
		}
	}

	return firstLen, firstDist
}

func (p *optimalParser) qualityMatch(pos, windowStart, maxLen int) (int, int) {
	if pos >= len(p.data) {
		return 0, 0
	}
	bestLen := 0
	bestDist := 0
	bestScore := 0.0

	searchPos := pos - 1
	searchCount := 0
	maxSearch := p.config.MaxChainLen

	for searchPos >= windowStart && searchCount < maxSearch && bestLen < maxLen {
		if p.data[searchPos] == p.data[pos] {
			matchLen := 0
			for matchLen < maxLen && pos+matchLen < len(p.data) &&
				p.data[searchPos+matchLen] == p.data[pos+matchLen] {
				matchLen++
			}

			if matchLen >= p.config.MinMatchLen {
				dist := pos - searchPos
				savings := float64(matchLen) - (3.0 + math.Log2(float64(dist)))
				if savings > bestScore {
					bestScore = savings
					bestLen = matchLen
					bestDist = dist
				}
			}
		}
		searchPos--
		searchCount++
	}

	return bestLen, bestDist
}

func (p *optimalParser) longestMatch(pos, windowStart, maxLen int) (int, int) {
	if pos >= len(p.data) {
		return 0, 0
	}
	bestLen := 0
	bestDist := 0

	searchPos := pos - 1
	searchCount := 0
	maxSearch := p.config.MaxChainLen

	for searchPos >= windowStart && searchCount < maxSearch && bestLen < maxLen {
		if p.data[searchPos] == p.data[pos] {
			matchLen := 0
			for matchLen < maxLen && pos+matchLen < len(p.data) &&
				p.data[searchPos+matchLen] == p.data[pos+matchLen] {
				matchLen++
			}

			if matchLen > bestLen {
				bestLen = matchLen
				bestDist = pos - searchPos
			}
		}
		searchPos--
		searchCount++
	}

	return bestLen, bestDist
}

func (p *optimalParser) optimalMatch(pos, windowStart, maxLen int) (int, int) {
	if pos >= len(p.data) {
		return 0, 0
	}
	bestLen := 0
	bestDist := 0
	bestCost := float64(len(p.data)-pos) * 8.0

	searchPos := pos - 1
	searchCount := 0
	maxSearch := p.config.MaxChainLen

	for searchPos >= windowStart && searchCount < maxSearch && bestLen < maxLen {
		if p.data[searchPos] == p.data[pos] {
			matchLen := 0
			for matchLen < maxLen && pos+matchLen < len(p.data) &&
				p.data[searchPos+matchLen] == p.data[pos+matchLen] {
				matchLen++
			}

			if matchLen >= p.config.MinMatchLen {
				dist := pos - searchPos
				literalCost := float64(matchLen) * 5.0
				matchCost := 3.0 + math.Log2(float64(dist))
				totalCost := literalCost - matchCost

				if totalCost < bestCost {
					bestCost = totalCost
					bestLen = matchLen
					bestDist = dist
				}
			}
		}
		searchPos--
		searchCount++
	}

	return bestLen, bestDist
}

func (p *optimalParser) refineTokens(tokens []Token) []Token {
	refined := make([]Token, len(tokens))
	copy(refined, tokens)

	changed := true
	iterations := 0
	maxRefineIterations := 10

	for changed && iterations < maxRefineIterations {
		changed = false
		iterations++

		for i := 0; i < len(refined); i++ {
			if refined[i].IsLiteral && i+2 < len(refined) {
				if candidate := p.tryCombineMatch(refined, i); candidate != nil {
					refined = candidate
					changed = true
				}
			}

			if !refined[i].IsLiteral {
				if candidate := p.trySplitMatch(refined, i); candidate != nil {
					refined = candidate
					changed = true
				}
			}
		}
	}

	return refined
}

func (p *optimalParser) tryCombineMatch(tokens []Token, pos int) []Token {
	if pos+2 >= len(tokens) {
		return nil
	}

	if !tokens[pos].IsLiteral || !tokens[pos+1].IsLiteral {
		return nil
	}

	lit1 := tokens[pos].Literal
	lit2 := tokens[pos+1].Literal

		searchPos := p.findBackwardMatch(pos, lit1, lit2)
	if searchPos < 0 {
		return nil
	}

	matchLen := 2
	testPos := pos + 2

	for testPos < len(tokens) && searchPos+matchLen < len(p.data) &&
		tokens[testPos].IsLiteral &&
		tokens[testPos].Literal == p.data[searchPos+matchLen] {
		matchLen++
		testPos++
	}

	if matchLen >= 3 {
		dist := pos - searchPos
		if dist <= 0 {
			return nil
		}
		result := make([]Token, 0, len(tokens)-2)
		result = append(result, tokens[:pos]...)
		result = append(result, TokenMatch(uint16(dist), uint16(matchLen)))
		result = append(result, tokens[testPos:]...)
		return result
	}

	return nil
}

func (p *optimalParser) findBackwardMatch(pos int, b1, b2 byte) int {
	if pos <= 0 {
		return -1
	}
	start := pos - 1
	if start >= len(p.data) {
		start = len(p.data) - 1
	}
	limit := pos - 32768
	if limit < 0 {
		limit = 0
	}
	for i := start; i >= limit; i-- {
		if p.data[i] == b1 && i+1 < len(p.data) && p.data[i+1] == b2 {
			return i
		}
	}
	return -1
}

func (p *optimalParser) trySplitMatch(tokens []Token, pos int) []Token {
	if !tokens[pos].IsLiteral {
		return nil
	}

	matchLen := int(tokens[pos].Match.Length)
	if matchLen <= 3 {
		return nil
	}

	firstHalf := matchLen / 2
	_ = firstHalf

	if firstHalf < 3 {
		return nil
	}

	result := make([]Token, 0, len(tokens)+1)
	result = append(result, tokens[:pos]...)
	result = append(result, TokenMatch(tokens[pos].Match.Distance, uint16(firstHalf)))
	result = append(result, tokens[pos+1:]...)
	return result
}

func (p *optimalParser) encodeTokens(tokens []Token, useDynamic bool) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	if useDynamic {
		err = WriteDynamicBlock(&buf, true, tokens)
	} else {
		err = WriteFixedBlock(&buf, true, tokens)
	}
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *optimalParser) calculateTrueCost(tokens []Token, useDynamic bool) (float64, int, error) {
	encoded, err := p.encodeTokens(tokens, useDynamic)
	if err != nil {
		return 0, 0, err
	}
	return float64(len(encoded)) * 8.0, len(encoded), nil
}

func (p *optimalParser) runIteration(iteration int) (float64, int, []Token) {
	var bestLocalCost float64
	var bestLocalSize int
	var bestLocalTokens []Token

	strategies := []bool{true, false}

	for _, useDynamic := range strategies {
		tokens := p.parseWithStrategy(iteration*len(strategies) + boolToInt(useDynamic))
		tokens = p.refineTokens(tokens)

		cost, size, err := p.calculateTrueCost(tokens, useDynamic)
		if err != nil {
			continue
		}

		if bestLocalTokens == nil || cost < bestLocalCost {
			bestLocalCost = cost
			bestLocalSize = size
			bestLocalTokens = tokens
		}
	}

	return bestLocalCost, bestLocalSize, bestLocalTokens
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (p *optimalParser) hasConverged() bool {
	if len(p.history) < 3 {
		return false
	}

	recent := p.history[len(p.history)-3:]
	improvement := (recent[0] - recent[2]) / recent[0]

	if improvement < p.config.ConvergenceThreshold {
		return true
	}

	if len(p.history) >= 5 {
		window := p.history[len(p.history)-5:]
		totalChange := window[0] - window[4]
		avgChange := totalChange / 4.0
		if avgChange < p.config.ConvergenceThreshold/10.0 {
			return true
		}
	}

	return false
}

func OptimalParse(data []byte, config OptimalConfig) ([]Token, error) {
	if len(data) == 0 {
		return nil, nil
	}

	parser := newOptimalParser(data, config)

	parser.bestCost = float64(len(data)) * 8.0
	parser.bestSize = len(data)
	parser.bestTokens = NewLZ77Encoder().Encode(data)

	for iteration := 0; iteration < config.MaxIterations; iteration++ {
		cost, size, tokens := parser.runIteration(iteration)

		if tokens != nil && cost < parser.bestCost {
			parser.bestCost = cost
			parser.bestSize = size
			parser.bestTokens = tokens
		}

		parser.history = append(parser.history, parser.bestCost)

		if parser.hasConverged() {
			break
		}

		if config.ProgressCallback != nil {
			var improvement float64
			if len(parser.history) >= 2 {
				improvement = (parser.history[0] - parser.bestCost) / parser.history[0]
			}
			config.ProgressCallback(float64(iteration), improvement)
		}
	}

	return parser.bestTokens, nil
}

func EncodeOptimalLZ77(data []byte, config OptimalConfig) ([]byte, error) {
	tokens, err := OptimalParse(data, config)
	if err != nil {
		return nil, err
	}
	if tokens == nil {
		tokens = NewLZ77Encoder().Encode(data)
	}

	var buf bytes.Buffer
	if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
