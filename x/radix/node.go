package radix

import (
	"sort"
)

// Leaf is used to represent a value
type Leaf struct {
	Key   Key
	Value interface{}
}

type sortLeafByPattern []Leaf

func (l sortLeafByPattern) Len() int { return len(l) }

func (l sortLeafByPattern) Less(i, j int) bool { return lessKey(l[i].Key, l[j].Key) }

func (l sortLeafByPattern) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

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

func (e sortEdgeByPattern) Less(i, j int) bool { return lessLabel(e[i].label, e[j].label) }

func (e sortEdgeByPattern) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

func lessLabel(x, y Label) bool {
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

func lessKey(x, y Key) bool {
	commonPrefix := longestPrefix(x, y)
	if commonPrefix < len(x) && commonPrefix < len(y) {
		return lessLabel(x[commonPrefix], y[commonPrefix])
	}
	if commonPrefix == len(x) && commonPrefix < len(y) {
		return true
	}
	return false
}

func longestPrefix(x, y Key) int {
	max := len(x)
	if l := len(y); l < max {
		max = l
	}
	var i int
	for i = 0; i < max; i++ {
		if x[i].String() != y[i].String() {
			break
		}
	}
	return i
}

func isPrefixOfLiteralKey(x, y Key) bool {
	if len(x) <= len(y) {
		return x.Match(y[:len(x)])
	}
	return false
}

type node struct {
	// leaf is used to store possible leaf
	leaf *Leaf

	// prefix is the common prefix we ignore
	prefix Key

	// edges should be stored in-order for iteration and searching.
	edges struct {
		literalEdges   []edge
		patternedEdges []edge
	}
}

func (p *node) size() int {
	return len(p.edges.literalEdges) + len(p.edges.patternedEdges)
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
		x := p.edges.literalEdges
		i := sort.Search(len(x), func(i int) bool {
			return x[i].label.String() >= s
		})
		if i < len(x) && x[i].label.String() == s {
			p.edges.literalEdges = append(p.edges.literalEdges[:i], p.edges.literalEdges[i+1:]...)
		}
	} else {
		x := p.edges.patternedEdges
		i := sort.Search(len(x), func(i int) bool {
			return x[i].label.String() >= s
		})
		if i < len(x) && x[i].label.String() == s {
			p.edges.patternedEdges = append(p.edges.patternedEdges[:i], p.edges.patternedEdges[i+1:]...)
		}
	}
}

func (p *node) getEdge(l Label) *edge {
	s := l.String()
	if len(p.edges.literalEdges) > 0 {
		x := p.edges.literalEdges
		i := sort.Search(len(x), func(i int) bool {
			return x[i].label.String() >= s
		})
		if i < len(x) && x[i].label.String() == s {
			return &x[i]
		}
	}

	if len(p.edges.patternedEdges) > 0 {
		x := p.edges.patternedEdges
		i := sort.Search(len(x), func(i int) bool {
			return x[i].label.String() >= s
		})
		if i < len(x) && x[i].label.String() == s {
			return &x[i]
		}
	}

	return nil
}

func (p *node) search(l Label) []edge {
	s := l.String()

	var found []edge
	if len(p.edges.literalEdges) > 0 {
		x := p.edges.literalEdges
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

func (p *node) mergeChild() {
	if p.size() != 1 {
		panic("required total size 1")
	}

	var child *node
	if len(p.edges.literalEdges) == 1 {
		child = p.edges.literalEdges[0].node
	} else {
		child = p.edges.patternedEdges[0].node
	}

	p.prefix = append(p.prefix, child.prefix...)
	p.leaf = child.leaf
	p.edges = child.edges
}
