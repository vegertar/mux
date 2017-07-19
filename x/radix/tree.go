package radix

// WalkFn is used when walking the tree. Takes a
// key and value, returning if iteration should
// be terminated.
type WalkFn func(s []Label, v interface{}) bool

// Tree implements a radix tree. This can be treated as a
// Dictionary abstract data type. The main advantage over
// a standard hash map is prefix-based lookups and
// ordered iteration,
type Tree struct {
	root *node
	size int
}

// New returns an empty Tree
func New() *Tree {
	return &Tree{
		root: &node{},
	}
}

// Len is used to return the number of elements in the tree
func (p *Tree) Len() int {
	return p.size
}

// longestPrefix finds the length of the shared prefix
// of two label sequences.
func longestPrefix(k1, k2 []Label) int {
	max := len(k1)
	if l := len(k2); l < max {
		max = l
	}
	var i int
	for i = 0; i < max; i++ {
		if k1[i].Compare(k2[i].Bytes()) != 0 {
			break
		}
	}
	return i
}

// Insert is used to add a new entry or update
// an existing entry. Returns if updated.
func (p *Tree) Insert(s []Label, v interface{}) (interface{}, bool) {
	var parent *node
	n := p.root
	search := s
	for {
		// Handle key exhaution
		if len(search) == 0 {
			if n.isLeaf() {
				old := n.leaf.val
				n.leaf.val = v
				return old, true
			}

			n.leaf = &leaf{
				key: s,
				val: v,
			}
			p.size++
			return nil, false
		}

		// Look for the edge
		parent = n
		n = n.getEdge(search[0])

		// No edge, create one
		if n == nil {
			e := edge{
				label: search[0],
				node: &node{
					leaf: &leaf{
						key: s,
						val: v,
					},
					prefix: search,
				},
			}
			parent.addEdge(e)
			p.size++
			return nil, false
		}

		// Determine longest prefix of the search key on match
		commonPrefix := longestPrefix(search, n.prefix)
		if commonPrefix == len(n.prefix) {
			search = search[commonPrefix:]
			continue
		}

		// Split the node
		p.size++
		child := &node{
			prefix: search[:commonPrefix],
		}
		parent.replaceEdge(edge{
			label: search[0],
			node:  child,
		})

		// Restore the existing node
		child.addEdge(edge{
			label: n.prefix[commonPrefix],
			node:  n,
		})
		n.prefix = n.prefix[commonPrefix:]

		// Create a new leaf node
		leaf := &leaf{
			key: s,
			val: v,
		}

		// If the new key is a subset, add to to this node
		search = search[commonPrefix:]
		if len(search) == 0 {
			child.leaf = leaf
			return nil, false
		}

		// Create a new edge for the node
		child.addEdge(edge{
			label: search[0],
			node: &node{
				leaf:   leaf,
				prefix: search,
			},
		})
		return nil, false
	}
}

// Delete is used to delete a key, returning the previous
// value and if it was deleted
func (p *Tree) Delete(s []Label) (interface{}, bool) {
	var parent *node
	var label Label
	n := p.root
	search := s
	for {
		// Check for key exhaution
		if len(search) == 0 {
			if !n.isLeaf() {
				break
			}
			goto DELETE
		}

		// Look for an edge
		parent = n
		label = search[0]
		n = n.getEdge(label)
		if n == nil {
			break
		}

		// Consume the search prefix
		if commonPrefix := longestPrefix(search, n.prefix); commonPrefix == len(n.prefix) {
			search = search[commonPrefix:]
		} else {
			break
		}
	}
	return nil, false

DELETE:
	// Delete the leaf
	leaf := n.leaf
	n.leaf = nil
	p.size--

	// Check if we should delete this node from the parent
	if parent != nil && len(n.edges) == 0 {
		parent.delEdge(label)
	}

	// Check if we should merge this node
	if n != p.root && len(n.edges) == 1 {
		n.mergeChild()
	}

	// Check if we should merge the parent's other child
	if parent != nil && parent != p.root && len(parent.edges) == 1 && !parent.isLeaf() {
		parent.mergeChild()
	}

	return leaf.val, true
}

// Get is used to lookup a specific key, returning
// the value and if it was found
func (p *Tree) Get(s []Label) (interface{}, bool) {
	n := p.root
	search := s
	for {
		// Check for key exhaution
		if len(search) == 0 {
			if n.isLeaf() {
				return n.leaf.val, true
			}
			break
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if commonPrefix := longestPrefix(search, n.prefix); commonPrefix == len(n.prefix) {
			search = search[commonPrefix:]
		} else {
			break
		}
	}
	return nil, false
}

// LongestPrefix is like Get, but instead of an
// exact match, it will return the longest prefix match.
func (p *Tree) LongestPrefix(s []Label) ([]Label, interface{}, bool) {
	var last *leaf
	n := p.root
	search := s
	for {
		// Look for a leaf node
		if n.isLeaf() {
			last = n.leaf
		}

		// Check for key exhaution
		if len(search) == 0 {
			break
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if commonPrefix := longestPrefix(search, n.prefix); commonPrefix == len(n.prefix) {
			search = search[commonPrefix:]
		} else {
			break
		}
	}
	if last != nil {
		return last.key, last.val, true
	}
	return nil, nil, false
}

// Minimum is used to return the minimum value in the tree
func (p *Tree) Minimum() ([]Label, interface{}, bool) {
	n := p.root
	for {
		if n.isLeaf() {
			return n.leaf.key, n.leaf.val, true
		}
		if len(n.edges) > 0 {
			n = n.edges[0].node
		} else {
			break
		}
	}
	return nil, nil, false
}

// Maximum is used to return the maximum value in the tree
func (p *Tree) Maximum() ([]Label, interface{}, bool) {
	n := p.root
	for {
		if num := len(n.edges); num > 0 {
			n = n.edges[num-1].node
			continue
		}
		if n.isLeaf() {
			return n.leaf.key, n.leaf.val, true
		}
		break
	}
	return nil, nil, false
}

// Walk is used to walk the tree
func (p *Tree) Walk(fn WalkFn) {
	recursiveWalk(p.root, fn)
}

// WalkPrefix is used to walk the tree under a prefix
func (p *Tree) WalkPrefix(prefix []Label, fn WalkFn) {
	n := p.root
	search := prefix
	for {
		// Check for key exhaution
		if len(search) == 0 {
			recursiveWalk(n, fn)
			return
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if commonPrefix := longestPrefix(search, n.prefix); commonPrefix == len(n.prefix) {
			search = search[len(n.prefix):]
		} else if commonPrefix == len(search) {
			// Child may be under our search prefix
			recursiveWalk(n, fn)
			return
		} else {
			break
		}
	}
}

// WalkPath is used to walk the tree, but only visiting nodes
// from the root down to a given leaf. Where WalkPrefix walks
// all the entries *under* the given prefix, this walks the
// entries *above* the given prefix.
func (p *Tree) WalkPath(path []Label, fn WalkFn) {
	n := p.root
	search := path
	for {
		// Visit the leaf values if any
		if n.leaf != nil && fn(n.leaf.key, n.leaf.val) {
			return
		}

		// Check for key exhaution
		if len(search) == 0 {
			return
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			return
		}

		// Consume the search prefix
		if commonPrefix := longestPrefix(search, n.prefix); commonPrefix == len(n.prefix) {
			search = search[commonPrefix:]
		} else {
			break
		}
	}
}

// recursiveWalk is used to do a pre-order walk of a node
// recursively. Returns true if the walk should be aborted
func recursiveWalk(n *node, fn WalkFn) bool {
	// Visit the leaf values if any
	if n.leaf != nil && fn(n.leaf.key, n.leaf.val) {
		return true
	}

	// Recurse on the children
	for _, e := range n.edges {
		if recursiveWalk(e.node, fn) {
			return true
		}
	}
	return false
}
