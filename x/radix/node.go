package radix

import (
	"sort"
	"fmt"
)

// leaf is used to represent a value
type leaf struct {
	key Key
	val interface{}
}

// edge is used to represent an edge node
type edge struct {
	label Label
	node  *node
}

type edges []edge

func (self edges) Len() int {
	return len(self)
}

func (self edges) Less(i, j int) bool {
	return lessLabel(self[i].label, self[j].label)
}

func (self edges) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self edges) Search(l Label) []int {
	n := len(self)

	start := sort.Search(n, func(i int) bool {
		return !lessLabel(self[i].label, l)
	})
	end := n
	s := l.String()

	if start < n {
		i := sort.Search(n - start, func(i int) bool {
			return lessLabel(l, self[i].label)
		})
		if start + i < n {
			end = start + i

			for ; end < n; end++ {
				if !self[end].label.Match(s) {
					break
				}
			}
		}
	}
	fmt.Println(">>>", start, end)

	var found []int
	for i := start; i < end; i++ {
		if self[i].label.Match(s) {
			found = append(found, i)
		}
	}
	//if len(found) > 0 && end < n {
	//	rest := self[end:].Search(l)
	//	for _, i := range rest {
	//		found = append(found, end + i)
	//	}
	//}

	return found
}

func (self edges) Sort() {
	insertionSort(self, 0, self.Len())
}

func lessLabel(x, y Label) bool {
	a, b := x.String(), y.String()
	if a == b {
		return false
	}

	if y.Match(a) {
		// keep the relative order
		if x.Match(b) {
			return false
		}
		return true
	}
	if x.Match(b) {
		return false
	}
	return a < b
}

func insertionSort(data sort.Interface, a, b int) {
	for i := a + 1; i < b; i++ {
		for j := i; j > a && data.Less(j, j - 1); j-- {
			data.Swap(j, j - 1)
		}
	}
}

type node struct {
	// leaf is used to store possible leaf
	leaf   *leaf

	// prefix is the common prefix we ignore
	prefix Key

	// Edges should be stored in-order for iteration.
	// We avoid a fully materialized slice to save memory,
	// since in most cases we expect to be sparse
	edges  edges
}

func (p *node) isLeaf() bool {
	return p.leaf != nil
}
//
//func (p *node) addEdge(e edge) {
//	// since sort immediately after every adding, we use insertion sort here
//
//}
//
//func (p *node) replaceEdge(i int, n *node) {
//	p.edges[i].node = n
//}
//
//func (p *node) getEdge(label Label) (int, *node) {
//	if i := p.edges.Search(label); i != -1 {
//		return i, p.edges[i].node
//	}
//	return -1, nil
//}
//
//func (p *node) delEdge(i int) {
//	p.edges = append(p.edges[:i], p.edges[i+1:]...)
//}
//
//func (p *node) mergeChild() {
//	child := p.edges[0].node
//	p.prefix = append(p.prefix, child.prefix...)
//	p.leaf = child.leaf
//	p.edges = child.edges
//}
