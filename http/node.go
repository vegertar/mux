package http

import (
	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

// Node uses trie tree to store and search route components.
type Node struct {
	tree *radix.Tree
	up   *x.Label
}

// NewNode creates a node instance.
func NewNode() *Node {
	return &Node{
		tree: radix.New(),
	}
}

// Up implements the `x.Node` interface.
func (p *Node) Up() *x.Label {
	return p.up
}

// Empty implements the `x.Node` interface.
func (p *Node) Empty() bool {
	return p.tree.Len() == 0
}

// Delete implements the `x.Node` interface.
func (p *Node) Delete(label *x.Label) {
	p.tree.Delete(label.Key)
}

// Make implements the `x.Node` interface.
func (p *Node) Make(route x.Route) (*x.Label, error) {
	node := p
	var leaf *x.Label

	for _, k := range route {
		if leaf != nil {
			if leaf.Down == nil {
				down := NewNode()
				down.up = leaf

				leaf.Down = down
			}
			node = leaf.Down.(*Node)
		}

		leaf = node.find(k)
		if leaf == nil {
			leaf = new(x.Label)
			leaf.Key = k
			leaf.Node = node
			node.tree.Insert(k, leaf)
		}
	}

	return leaf, nil
}

// Get implements the `x.Node` interface.
func (p *Node) Get(route x.Route) *x.Label {
	var leaf *x.Label

	node := p
	for _, k := range route {
		if leaf != nil {
			node = leaf.Down.(*Node)
			leaf = nil
		}

		if node != nil {
			leaf = node.find(k)
		}

		if leaf == nil {
			break
		}
	}

	return leaf
}

// Match implements the `x.Node` interface.
func (p *Node) Match(route x.Route) x.Label {
	var leaf x.Label
	leaf.Node = p

	if len(route) > 0 {
		k := route[0]
		if len(route) == 1 {
			leaf.Key = k
		}
		for _, v := range p.tree.Match(k) {
			label := v.Value.(*x.Label)
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

// Leaves implements the `x.Node` interface.
func (p *Node) Leaves() []*x.Label {
	var out []*x.Label
	p.tree.Walk(func(s string, v interface{}) bool {
		label := v.(*x.Label)
		if label.Down != nil {
			out = append(out, label.Down.Leaves()...)
		} else {
			out = append(out, label)
		}
		return false
	})

	return out
}

func (p *Node) find(k radix.Key) *x.Label {
	value, ok := p.tree.Get(k)
	if ok && value != nil {
		return value.(*x.Label)
	}
	return nil
}
