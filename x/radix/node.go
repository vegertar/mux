package radix

// leaf is used to represent a value
type leaf struct {
	key []Label
	val interface{}
}

type node struct {
	// leaf is used to store possible leaf
	leaf *leaf

	// prefix is the common prefix we ignore
	prefix []Label

	// Edges should be stored in-order for iteration.
	// We avoid a fully materialized slice to save memory,
	// since in most cases we expect to be sparse
	edges edges
}

func (p *node) isLeaf() bool {
	return p.leaf != nil
}

func (p *node) addEdge(e edge) {
	p.edges = append(p.edges, e)
	p.edges.Sort()
}

func (p *node) replaceEdge(i int, n *node) {
	p.edges[i].node = n
}

func (p *node) getEdge(label Label) (int, *node) {
	if i := p.edges.Search(label); i != -1 {
		return i, p.edges[i].node
	}
	return -1, nil
}

func (p *node) delEdge(i int) {
	p.edges = append(p.edges[:i], p.edges[i+1:]...)
}

func (p *node) mergeChild() {
	child := p.edges[0].node
	p.prefix = append(p.prefix, child.prefix...)
	p.leaf = child.leaf
	p.edges = child.edges
}
