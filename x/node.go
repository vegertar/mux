package x

import (
	"sync"

	"github.com/vegertar/mux/x/radix"
)

// Label is used to represent a value.
type Label struct {
	Value
	Key radix.Key
}

// Node defines a interface to add, delete, match and iterate a router.
// A node should implement concurrent safety.
type Node interface {
	// Get returns a label from this Node.
	Get(key radix.Key, createIfMissing bool) *Label

	// Delete deletes a label.
	Delete(label *Label)

	// Up returns a parent label.
	Up() *Label

	// Empty returns if it is empty.
	Empty() bool

	// Leaves returns all labels has nil `Down` node.
	Leaves() []*Label

	// Match returns all labels matched given route.
	Match(route Route) []Label
}

// RadixNode uses radix tree to store and search route components.
type RadixNode struct {
	tree *radix.Tree
	up   *Label
	mu   sync.RWMutex
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
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tree.Len() == 0
}

// Delete implements the `Node` interface.
func (p *RadixNode) Delete(label *Label) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tree.Delete(label.Key)
}

// Match implements the `Node` interface.
func (p *RadixNode) Match(route Route) (leaves []Label) {
	if len(route) > 0 {
		k := route[0]
		p.mu.RLock()
		match := p.tree.Match(k)
		p.mu.RUnlock()

		for _, v := range match {
			label := v.Value.(*Label)
			if len(route) > 1 && label.Down != nil {
				leaves = append(leaves, label.Down.Match(route[1:])...)
				continue
			}

			// check if label is a leaf
			if len(route) == 1 || label.Down == nil && label.Key.Wildcard() {
				leaves = append(leaves, *label)
				continue
			}

			// remains as a middleware
			if len(label.Middleware) > 0 {
				x := *label
				// clears unnecessary handlers
				x.Handler = nil
				leaves = append(leaves, x)
			}
		}
	}

	return
}

// Leaves implements the `Node` interface.
func (p *RadixNode) Leaves() []*Label {
	p.mu.RLock()
	defer p.mu.RUnlock()

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
	p.mu.RLock()
	value, ok := p.tree.Get(k)
	p.mu.RUnlock()
	if ok && value != nil {
		return value.(*Label)
	}

	if createIfMissing {
		label := new(Label)
		label.Key = k
		label.Node = p
		p.mu.Lock()
		p.tree.Insert(k, label)
		p.mu.Unlock()
		return label
	}

	return nil
}
