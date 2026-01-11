package compress

import (
	"bytes"
	"sort"
)

const (
	DefaultZopfliIterations = 10
	DefaultBlockSize        = 65535
	DefaultSplitThreshold   = 0.001
	MaxConvergentIterations = 3
	MinImprovementForSplit  = 0.001
)

type ZopfliIterationConfig struct {
	Iterations       int
	BlockSize        int
	SplitThreshold   float64
	BlockSplitting   bool
	Verbose          bool
	ProgressCallback func(iteration, improvement float64, size int)
}

func NewZopfliIterationConfig() ZopfliIterationConfig {
	return ZopfliIterationConfig{
		Iterations:     DefaultZopfliIterations,
		BlockSize:      DefaultBlockSize,
		SplitThreshold: DefaultSplitThreshold,
		BlockSplitting: true,
		Verbose:        false,
	}
}

type blockSplit struct {
	splitPoints []int
	cost        float64
	size        int
}

type zopfliIterationState struct {
	data            []byte
	config          ZopfliIterationConfig
	bestResult      []byte
	bestSize        int
	bestCost        float64
	iteration       int
	lastImprovement float64
	stagnantCount   int
	history         []float64
}

func newZopfliIterationState(data []byte, config ZopfliIterationConfig) *zopfliIterationState {
	if config.Iterations <= 0 {
		config.Iterations = DefaultZopfliIterations
	}
	if config.BlockSize <= 0 {
		config.BlockSize = DefaultBlockSize
	}
	if config.SplitThreshold <= 0 {
		config.SplitThreshold = DefaultSplitThreshold
	}

	state := &zopfliIterationState{
		data:     data,
		config:   config,
		bestSize: len(data),
		bestCost: float64(len(data)) * 8.0,
		history:  make([]float64, 0, config.Iterations),
	}

	encoder := NewDeflateEncoder()
	encoder.SetCompressionLevel(9)
	initialResult, err := encoder.EncodeAuto(data)
	if err == nil {
		state.bestResult = initialResult
		state.bestSize = len(initialResult)
		state.bestCost = float64(len(initialResult)) * 8.0
	} else {
		state.bestResult = data
	}

	return state
}

func ZopfliIterate(data []byte, config ZopfliIterationConfig) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	state := newZopfliIterationState(data, config)

	for iteration := 0; iteration < state.config.Iterations; iteration++ {
		state.iteration = iteration

		iterationCost := state.runIteration()

		state.history = append(state.history, iterationCost)

		if state.config.ProgressCallback != nil {
			var improvement float64
			if len(state.history) > 1 {
				improvement = (state.history[0] - state.bestCost) / state.history[0]
			}
			state.config.ProgressCallback(float64(iteration), improvement, state.bestSize)
		}

		if state.hasConverged() {
			break
		}
	}

	return state.bestResult, nil
}

func ZopfliEncodeIterative(data []byte, iterations int) ([]byte, error) {
	config := NewZopfliIterationConfig()
	if iterations <= 0 {
		iterations = 12
	}
	config.Iterations = iterations
	return ZopfliIterate(data, config)
}

func (s *zopfliIterationState) runIteration() float64 {
	encoder := NewDeflateEncoder()
	encoder.SetCompressionLevel(9)

	var bestCost float64 = s.bestCost

	trialConfigs := s.generateTrialConfigs()

	for _, trial := range trialConfigs {
		result, err := s.encodeWithConfig(encoder, trial)
		if err != nil {
			continue
		}

		cost := float64(len(result)) * 8.0

		if cost < bestCost {
			bestCost = cost
			s.bestResult = result
			s.bestSize = len(result)
			s.bestCost = cost
		}
	}

	if len(s.history) > 0 {
		improvement := s.history[len(s.history)-1] - bestCost
		s.lastImprovement = improvement
		if improvement < float64(s.bestSize)*s.config.SplitThreshold {
			s.stagnantCount++
		} else {
			s.stagnantCount = 0
		}
	}

	return bestCost
}

func (s *zopfliIterationState) generateTrialConfigs() []encodingTrial {
	trials := make([]encodingTrial, 0, 20)

	modes := []bool{true, false}
	for _, useDynamic := range modes {
		trials = append(trials, encodingTrial{
			useDynamic:    useDynamic,
			blockSplit:    false,
			matchStrategy: 0,
		})
	}

	matchStrategies := []int{0, 1, 2, 3, 4}
	for _, strategy := range matchStrategies {
		for _, useDynamic := range modes {
			trials = append(trials, encodingTrial{
				useDynamic:    useDynamic,
				blockSplit:    false,
				matchStrategy: strategy,
			})
		}
	}

	if s.shouldTrySplit() {
		splits := s.findOptimalSplits()
		for _, split := range splits {
			for _, useDynamic := range modes {
				trials = append(trials, encodingTrial{
					useDynamic:    useDynamic,
					blockSplit:    true,
					splitPoints:   split,
					matchStrategy: 0,
				})
			}
		}
	}

	return trials
}

type encodingTrial struct {
	useDynamic    bool
	blockSplit    bool
	splitPoints   []int
	matchStrategy int
}

func (s *zopfliIterationState) encodeWithConfig(encoder *DeflateEncoder, trial encodingTrial) ([]byte, error) {
	if trial.blockSplit && len(trial.splitPoints) > 0 {
		return s.encodeWithSplit(encoder, trial)
	}

	return encoder.Encode(s.data, trial.useDynamic)
}

func (s *zopfliIterationState) encodeWithSplit(encoder *DeflateEncoder, trial encodingTrial) ([]byte, error) {
	var result bytes.Buffer

	splits := append(append([]int{}, trial.splitPoints...), len(s.data))
	sort.Ints(splits)

	prevPos := 0
	isFinal := false

	for i, splitPos := range splits {
		if splitPos > len(s.data) {
			splitPos = len(s.data)
		}

		if i == len(splits)-1 || splitPos >= len(s.data) {
			isFinal = true
		}

		segment := s.data[prevPos:splitPos]
		tokens := encoder.lz77.Encode(segment)

		var err error
		if trial.useDynamic {
			err = WriteDynamicBlock(&result, isFinal, tokens)
		} else {
			err = WriteFixedBlock(&result, isFinal, tokens)
		}
		if err != nil {
			return nil, err
		}

		prevPos = splitPos
		if isFinal {
			break
		}
	}

	return result.Bytes(), nil
}

func (s *zopfliIterationState) shouldTrySplit() bool {
	if !s.config.BlockSplitting {
		return false
	}

	if s.stagnantCount >= 2 {
		return true
	}

	if s.iteration > 0 && s.iteration%3 == 0 {
		return true
	}

	return false
}

func (s *zopfliIterationState) findOptimalSplits() [][]int {
	splits := s.findGoodSplitPoints()

	optimalSplits := s.evaluateSplitCombinations(splits)

	return optimalSplits
}

func (s *zopfliIterationState) findGoodSplitPoints() []int {
	potentialSplits := make([]int, 0)

	blockSize := s.config.BlockSize
	dataLen := len(s.data)

	for pos := blockSize; pos < dataLen; pos += blockSize {
		potentialSplits = append(potentialSplits, pos)
	}

	if len(potentialSplits) == 0 {
		return nil
	}

	scoredSplits := make([]splitScore, 0, len(potentialSplits))
	encoder := NewDeflateEncoder()
	encoder.SetCompressionLevel(9)

	for _, splitPos := range potentialSplits {
		leftCost := s.estimateBlockCost(0, splitPos, encoder)
		rightCost := s.estimateBlockCost(splitPos, dataLen, encoder)
		combinedCost := leftCost + rightCost
		singleCost := s.estimateBlockCost(0, dataLen, encoder)

		improvement := (singleCost - combinedCost) / singleCost

		if improvement > MinImprovementForSplit {
			scoredSplits = append(scoredSplits, splitScore{
				pos:         splitPos,
				improvement: improvement,
			})
		}
	}

	sort.Slice(scoredSplits, func(i, j int) bool {
		return scoredSplits[i].improvement > scoredSplits[j].improvement
	})

	maxSplits := 5
	if len(scoredSplits) > maxSplits {
		scoredSplits = scoredSplits[:maxSplits]
	}

	result := make([]int, len(scoredSplits))
	for i, ss := range scoredSplits {
		result[i] = ss.pos
	}

	return result
}

type splitScore struct {
	pos         int
	improvement float64
}

func (s *zopfliIterationState) evaluateSplitCombinations(splits []int) [][]int {
	if len(splits) == 0 {
		return nil
	}

	encoder := NewDeflateEncoder()
	encoder.SetCompressionLevel(9)

	bestSize := len(s.bestResult)
	bestSplit := []int{}

	for i := 1; i <= len(splits) && i <= 3; i++ {
		combinations := s.generateCombinations(splits, i)
		for _, combo := range combinations {
			result, err := s.encodeWithSplit(encoder, encodingTrial{
				useDynamic:    true,
				blockSplit:    true,
				splitPoints:   combo,
				matchStrategy: 0,
			})
			if err != nil {
				continue
			}

			if len(result) < bestSize {
				bestSize = len(result)
				bestSplit = combo
			}
		}
	}

	if len(bestSplit) > 0 {
		return [][]int{bestSplit}
	}

	return nil
}

func (s *zopfliIterationState) generateCombinations(splits []int, k int) [][]int {
	result := make([][]int, 0)

	var generate func([]int, int, int)
	generate = func(current []int, start, depth int) {
		if depth == k {
			combo := make([]int, len(current))
			copy(combo, current)
			result = append(result, combo)
			return
		}

		for i := start; i <= len(splits)-k+depth; i++ {
			generate(append(current, splits[i]), i+1, depth+1)
		}
	}

	generate([]int{}, 0, 0)
	return result
}

func (s *zopfliIterationState) estimateBlockCost(start, end int, encoder *DeflateEncoder) float64 {
	if start >= end {
		return 0
	}

	segment := s.data[start:end]
	tokens := encoder.lz77.Encode(segment)

	return estimateBlockCost(tokens, true)
}

func (s *zopfliIterationState) hasConverged() bool {
	if len(s.history) < 3 {
		return false
	}

	recentHistory := s.history[len(s.history)-3:]
	improvement := (recentHistory[0] - recentHistory[2]) / recentHistory[0]

	if improvement < s.config.SplitThreshold {
		return true
	}

	if len(s.history) >= 5 {
		window := s.history[len(s.history)-5:]
		totalChange := window[0] - window[4]
		avgChange := totalChange / 4.0
		if avgChange < s.config.SplitThreshold/10.0 {
			return true
		}
	}

	if s.stagnantCount >= MaxConvergentIterations {
		return true
	}

	return false
}

func CalculateZopfliImprovement(original, optimized []byte) float64 {
	if len(original) == 0 {
		return 0
	}
	return (float64(len(original)) - float64(len(optimized))) / float64(len(original)) * 100
}

func (s *zopfliIterationState) calculateCompressionRatio(original, compressed []byte) float64 {
	if len(original) == 0 {
		return 0
	}
	return float64(len(compressed)) / float64(len(original))
}
