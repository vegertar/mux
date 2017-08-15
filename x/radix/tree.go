package radix

import (
	"sort"
)

// WalkFn is used when walking the tree. Takes a
// leaf, returning if iteration should
// be terminated.
type WalkFn func(leaf Leaf) bool

// Tree implements a radix tree. This can be treated as a
// Dictionary abstract data type. The main advantage over
// a standard hash map is prefix-based lookups and
// ordered iteration
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

// Insert is used to add a new entry or update
// an existing entry. Returns if updated.
func (p *Tree) Insert(k Key, v interface{}) (interface{}, bool) {
	n := p.root
	search := k
	for {
		// Handle key exhaustion
		if len(search) == 0 {
			if n.isLeaf() {
				old := n.leaf.Value
				n.leaf.Value = v
				return old, true
			}

			n.leaf = &Leaf{
				Key:   k,
				Value: v,
			}
			p.size++
			return nil, false
		}

		// Look for the edge
		parent := n
		e := n.getEdge(search[0])

		// No edge, create one
		if e == nil {
			parent.addEdge(edge{
				label: search[0],
				node: &node{
					leaf: &Leaf{
						Key:   k,
						Value: v,
					},
					prefix: search,
				},
			})
			p.size++
			return nil, false
		}
		n = e.node

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
		e.node = child

		// Restore the existing node
		child.addEdge(edge{
			label: n.prefix[commonPrefix],
			node:  n,
		})
		n.prefix = n.prefix[commonPrefix:]

		// Create a new leaf node
		leaf := &Leaf{
			Key:   k,
			Value: v,
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
func (p *Tree) Delete(k Key) (interface{}, bool) {
	var (
		parent *node
		label  Label

		n      = p.root
		search = k
	)

	for {
		// Check for key exhaustion
		if len(search) == 0 {
			if !n.isLeaf() {
				break
			}
			// Delete the leaf
			leaf := n.leaf
			n.leaf = nil
			p.size--

			// Check if we should delete this node from the parent
			if parent != nil && n.size() == 0 {
				parent.delEdge(label)
			}

			// Check if we should merge this node
			if n != p.root && n.size() == 1 {
				n.mergeChild()
			}

			// Check if we should merge the parent's other child
			if parent != nil && parent != p.root && parent.size() == 1 && !parent.isLeaf() {
				parent.mergeChild()
			}

			return leaf.Value, true
		}

		// Look for an edge
		parent = n
		label = search[0]
		if e := n.getEdge(label); e == nil {
			break
		} else {
			n = e.node
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

// DeletePrefix is used to delete the subtree under a prefix
// Returns how many nodes were deleted
// Use this to delete large subtrees efficiently
func (p *Tree) DeletePrefix(prefix Key) int {
	return p.deletePrefix(nil, p.root, prefix)
}

// delete does a recursive deletion
func (p *Tree) deletePrefix(parent, n *node, prefix Key) int {
	// Check for key exhaustion
	if len(prefix) == 0 {
		// Remove the leaf node
		subTreeSize := 0
		//recursively walk from all edges of the node to be deleted
		recursiveWalk(n, func(leaf Leaf) bool {
			subTreeSize++
			return false
		})
		if n.isLeaf() {
			n.leaf = nil
		}
		// deletes the entire subtree
		n.edges.literalEdges = nil
		n.edges.patternedEdges = nil

		// Check if we should merge the parent's other child
		if parent != nil && parent != p.root && parent.size() == 1 && !parent.isLeaf() {
			parent.mergeChild()
		}
		p.size -= subTreeSize
		return subTreeSize
	}

	// Look for an edge
	e := n.getEdge(prefix[0])
	if e == nil {
		return 0
	}
	child := e.node
	commonPrefix := longestPrefix(child.prefix, prefix)
	if commonPrefix != len(child.prefix) && commonPrefix != len(prefix) {
		return 0
	}

	// Consume the search prefix
	if len(child.prefix) > len(prefix) {
		prefix = prefix[len(prefix):]
	} else {
		prefix = prefix[len(child.prefix):]
	}
	return p.deletePrefix(n, child, prefix)
}

// Get is used to lookup a specific key, returning
// the value and if it was found
func (p *Tree) Get(k Key) (interface{}, bool) {
	if len(k) > 0 {
		// Look for an edge
		e := p.root.getEdge(k[0])
		if e == nil {
			return nil, false
		}

		n := e.node
		// Consume the search prefix
		if i := len(n.prefix); i <= len(k) && n.prefix.Equal(k[:i]) {
			t := &Tree{root: n}
			return t.Get(k[i:])
		}
	} else if p.root.isLeaf() {
		return p.root.leaf.Value, true
	}

	return nil, false
}

// Match is used to lookup a specific key, returning all matched leaves.
func (p *Tree) Match(k Key) []Leaf {
	v := p.match(k)
	if len(v) > 1 {
		sort.Sort(sortLeafByPattern(v))
	}
	return v
}

func (p *Tree) match(k Key) (leaves []Leaf) {
	if len(k) > 0 {
		// Look for edges
		for _, e := range p.root.search(k[0]) {
			n := e.node
			// Consume the search prefix
			if i := isPrefixOfLiteralKey(n.prefix, k); i > 0 {
				t := &Tree{root: n}
				leaves = append(leaves, t.match(k[i:])...)
			}
		}
	} else if p.root.isLeaf() {
		leaves = append(leaves, *p.root.leaf)
	}

	return
}

// LongestPrefix is like Match, but instead of an
// exact match, it will return the longest prefix match.
func (p *Tree) LongestPrefix(k Key) (leaves []Leaf) {
	v := p.longestPrefix(k)
	sort.Sort(sortLeafByPattern(v))
	return v
}

func (p *Tree) longestPrefix(k Key) (leaves []Leaf) {
	if p.root.isLeaf() {
		leaves = append(leaves, *p.root.leaf)
	}
	if len(k) > 0 {
		// Look for an edge
		for _, e := range p.root.search(k[0]) {
			n := e.node
			// Consume the search prefix
			if i := isPrefixOfLiteralKey(n.prefix, k); i > 0 {
				t := &Tree{root: n}
				leaves = append(leaves, t.longestPrefix(k[i:])...)
			}
		}

		var (
			longestSize   int
			longestLeaves []Leaf
		)
		for _, l := range leaves {
			n := len(l.Key)
			if n > longestSize {
				longestSize = n
				longestLeaves = longestLeaves[:0]
			}
			if n == longestSize {
				longestLeaves = append(longestLeaves, l)
			}
		}
		leaves = longestLeaves
	}

	return
}

// Walk is used to walk the tree
func (p *Tree) Walk(fn WalkFn) {
	recursiveWalk(p.root, fn)
}

// WalkPrefix is used to walk the tree under a prefix
func (p *Tree) WalkPrefix(prefix Key, fn WalkFn) {
	if len(prefix) == 0 {
		recursiveWalk(p.root, fn)
		return
	}

	for _, e := range p.root.search(prefix[0]) {
		n := e.node
		if i := isPrefixOfLiteralKey(n.prefix, prefix); i > 0 {
			t := &Tree{root: n}
			t.WalkPrefix(prefix[i:], fn)
		} else if longestPrefix(n.prefix, prefix) == len(prefix) {
			recursiveWalk(n, fn)
		}
	}
}

// WalkPath is used to walk the tree, but only visiting nodes
// from the root down to a given leaf. Where WalkPrefix walks
// all the entries *under* the given prefix, this walks the
// entries *above* the given prefix.
func (p *Tree) WalkPath(path Key, fn WalkFn) {
	if p.root.isLeaf() && fn(*p.root.leaf) {
		return
	}

	if len(path) == 0 {
		return
	}

	for _, e := range p.root.search(path[0]) {
		n := e.node
		if i := isPrefixOfLiteralKey(n.prefix, path); i > 0 {
			t := &Tree{root: n}
			t.WalkPath(path[i:], fn)
		}
	}
}

// recursiveWalk is used to do a pre-order walk of a node
// recursively. Returns true if the walk should be aborted
func recursiveWalk(n *node, fn WalkFn) bool {
	// Visit the leaf values if any
	if n.isLeaf() && fn(*n.leaf) {
		return true
	}

	// Recurse on the children
	x := n.edges.literalEdges
	if len(n.edges.patternedEdges) > 0 {
		x = append(x, n.edges.patternedEdges...)
		sort.Sort(sortEdgeByPattern(x))
	}

	for _, e := range x {
		if recursiveWalk(e.node, fn) {
			return true
		}
	}
	return false
}
