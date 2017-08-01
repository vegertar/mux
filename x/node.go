package x

import (
	"github.com/vegertar/mux/x/radix"
)

// Label is used to represent a value.
type Label struct {
	Value
	Key radix.Key
}

// Node defines a interface to add, delete, match and iterate a router.
type Node interface {
	// Make should make all necessary nodes and labels while matching through the given route.
	// If successful, it returns a label matched the last component of route.
	Make(route Route, breed BreedFunc) (*Label, error)

	// Delete deletes a label.
	Delete(label *Label)

	// Leaves returns all labels has nil `Down` node.
	Leaves() []*Label

	// Get returns a label from this Node.
	Get(key radix.Key, createIfMissing bool) *Label

	// Up returns a parent label.
	Up() *Label

	// Empty returns if it is empty.
	Empty() bool

	// Match returns a label matched by given route.
	// The returned label should contain all associated handlers and middleware
	// while matching by BFS.
	Match(route Route) Label
}

// RadixNode uses radix tree to store and search route components.
type RadixNode struct {
	tree *radix.Tree
	up   *Label
}

// NewRadixNode creates a Node instance.
func NewRadixNode(up *Label) *RadixNode {
	return &RadixNode{
		tree: radix.New(),
		up:   up,
	}
}

// Up implements the `Node` interface.
func (p *RadixNode) Up() *Label {
	return p.up
}

// Empty implements the `Node` interface.
func (p *RadixNode) Empty() bool {
	return p.tree.Len() == 0
}

// Delete implements the `Node` interface.
func (p *RadixNode) Delete(label *Label) {
	p.tree.Delete(label.Key)
}

// Make implements the `Node` interface.
func (p *RadixNode) Make(route Route, breed BreedFunc) (*Label, error) {
	var (
		leaf *Label
		node Node = p
	)

	for _, k := range route {
		if leaf != nil {
			if leaf.Down == nil {
				leaf.Down = breed(leaf)
			}
			node = leaf.Down
		}

		leaf = node.Get(k, true)
	}

	return leaf, nil
}

// Match implements the `Node` interface.
func (p *RadixNode) Match(route Route) Label {
	var leaf Label
	leaf.Node = p

	if len(route) > 0 {
		k := route[0]
		if len(route) == 1 {
			leaf.Key = k
		}
		for _, v := range p.tree.Match(k) {
			label := v.Value.(*Label)
			if label.Down != nil && len(route) > 1 {
				value := label.Down.Match(route[1:])
				leaf.Key = value.Key
				leaf.Handler = append(leaf.Handler, value.Handler...)
				leaf.Middleware = append(leaf.Middleware, value.Middleware...)
			} else if len(route) == 1 {
				leaf.Handler = append(leaf.Handler, label.Handler...)
				leaf.Middleware = append(leaf.Middleware, label.Middleware...)
			} else {
				leaf.Middleware = append(leaf.Middleware, label.Middleware...)
			}
		}
	}

	return leaf
}

// Leaves implements the `Node` interface.
func (p *RadixNode) Leaves() []*Label {
	var out []*Label
	p.tree.Walk(func(leaf radix.Leaf) bool {
		label := leaf.Value.(*Label)
		if label.Down != nil {
			out = append(out, label.Down.Leaves()...)
		} else {
			out = append(out, label)
		}
		return false
	})

	return out
}

// Get implements the `Node` interface.
func (p *RadixNode) Get(k radix.Key, createIfMissing bool) *Label {
	value, ok := p.tree.Get(k)
	if ok && value != nil {
		return value.(*Label)
	}

	if createIfMissing {
		label := new(Label)
		label.Key = k
		label.Node = p
		p.tree.Insert(k, label)
		return label
	}

	return nil
}
