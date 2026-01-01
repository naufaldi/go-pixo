package compress

import "container/heap"

// Node represents a node in the Huffman tree.
type Node struct {
	Left   *Node
	Right  *Node
	Symbol int
	Weight int
}

// nodeHeap implements heap.Interface for building Huffman tree.
type nodeHeap []*Node

func (h nodeHeap) Len() int           { return len(h) }
func (h nodeHeap) Less(i, j int) bool { return h[i].Weight < h[j].Weight }
func (h nodeHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nodeHeap) Push(x interface{}) {
	*h = append(*h, x.(*Node))
}

func (h *nodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// BuildTree constructs a Huffman tree from symbol frequencies.
// Frequencies should be indexed by symbol value.
// Returns nil if no symbols have non-zero frequency.
func BuildTree(frequencies []int) *Node {
	var nodes nodeHeap
	for symbol, freq := range frequencies {
		if freq > 0 {
			nodes = append(nodes, &Node{
				Symbol: symbol,
				Weight: freq,
			})
		}
	}

	if len(nodes) == 0 {
		return nil
	}

	if len(nodes) == 1 {
		return nodes[0]
	}

	heap.Init(&nodes)

	for nodes.Len() > 1 {
		left := heap.Pop(&nodes).(*Node)
		right := heap.Pop(&nodes).(*Node)

		parent := &Node{
			Left:   left,
			Right:  right,
			Weight: left.Weight + right.Weight,
			Symbol: -1,
		}

		heap.Push(&nodes, parent)
	}

	return heap.Pop(&nodes).(*Node)
}
