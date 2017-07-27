package radix

import (
	"sort"
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

type sortEdgeByLiteral []edge

func (e sortEdgeByLiteral) Len() int { return len(e) }

func (e sortEdgeByLiteral) Less(i, j int) bool { return e[i].label.String() < e[j].label.String() }

func (e sortEdgeByLiteral) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

type sortEdgeByPattern []edge

func (e sortEdgeByPattern) Len() int { return len(e) }

func (e sortEdgeByPattern) Less(i, j int) bool { return lessLabelByPattern(e[i].label, e[j].label) }

func (e sortEdgeByPattern) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

func lessLabelByPattern(x, y Label) bool {
	a, b := x.String(), y.String()
	if x.Literal() && y.Literal() {
		return a < b
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
		for j := i; j > a && data.Less(j, j-1); j-- {
			data.Swap(j, j-1)
		}
	}
}

type node struct {
	// leaf is used to store possible leaf
	leaf *leaf

	// prefix is the common prefix we ignore
	prefix Key

	// Edges should be stored in-order for iteration.
	// We avoid a fully materialized slice to save memory,
	// since in most cases we expect to be sparse
	edges struct {
		literalEdges   []edge
		patternedEdges []edge
	}
}

func (p *node) isLeaf() bool {
	return p.leaf != nil
}

func (p *node) addEdge(e edge) {
	if e.label.Literal() {
		p.edges.literalEdges = append(p.edges.literalEdges, e)
		sort.Sort(sortEdgeByLiteral(p.edges.literalEdges))
	} else {
		p.edges.patternedEdges = append(p.edges.patternedEdges, e)
		sort.Sort(sortEdgeByLiteral(p.edges.patternedEdges))
	}
}

func (p *node) delEdge(l Label) {
	s := l.String()
	if l.Literal() {
		x := sortEdgeByLiteral(p.edges.literalEdges)
		i := sort.Search(len(x), func(i int) bool {
			return x[i].label.String() >= s
		})
		if i < len(x) && x[i].label.String() == s {
			p.edges.literalEdges = append(p.edges.literalEdges[:i], p.edges.literalEdges[i+1:]...)
		}
	} else {
		x := sortEdgeByLiteral(p.edges.patternedEdges)
		i := sort.Search(len(x), func(i int) bool {
			return x[i].label.String() >= s
		})
		if i < len(x) && x[i].label.String() == s {
			p.edges.patternedEdges = append(p.edges.patternedEdges[:i], p.edges.patternedEdges[i+1:]...)
		}
	}
}

func (p *node) search(s string) []edge {
	var found []edge
	if len(p.edges.literalEdges) > 0 {
		x := sortEdgeByLiteral(p.edges.literalEdges)
		i := sort.Search(len(x), func(i int) bool {
			return x[i].label.String() >= s
		})
		if i < len(x) && x[i].label.String() == s {
			found = append(found, x[i])
		}
	}
	for _, e := range p.edges.patternedEdges {
		if e.label.Match(s) {
			found = append(found, e)
		}
	}
	return found
}

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
//func (p *node) mergeChild() {
//	child := p.edges[0].node
//	p.prefix = append(p.prefix, child.prefix...)
//	p.leaf = child.leaf
//	p.edges = child.edges
//}
