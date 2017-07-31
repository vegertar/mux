package http

import (
	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

// Node uses trie tree to store and search route components.
type Node struct {
	tree *radix.Tree
	up   *x.Value
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
func (p *Node) Delete(key radix.Key) {
	p.tree.Delete(key)
}

// Make implements the `x.Node` interface.
func (p *Node) Make(route x.Route) (leaf *x.Value, err error) {
	node := p
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
			leaf = new(x.Value)
			leaf.Node = node
			node.tree.Insert(k, leaf)
		}
	}

	return leaf, nil
}

// Get implements the `x.Node` interface.
func (p *Node) Get(route x.Route) *x.Value {
	var leaf *x.Value

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
func (p *Node) Match(route x.Route) x.Value {
	var leaf x.Value
	leaf.Node = p

	if len(route) > 0 {
		for _, v := range p.tree.Match(route[0]) {
			value := v.Value.(*x.Value)
			if value.Down != nil && len(route) > 1 {
				x := value.Down.Match(route[1:])
				if len(x.Handler) > 0 {
					leaf.Handler = append(leaf.Handler, x.Handler...)
				}
				leaf.Middleware = append(middleware, x.Middleware...)
			} else if len(route) == 1 {
				if len(value.Handler) > 0 {
					leaf.Handler = append(leaf.Handler, value.Handler...)
				}
				leaf.Middleware = append(middleware, value.Middleware...)
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
	for _, label := range p.labels {
		if label.Down != nil {
			out = append(out, label.Down.Leaves()...)
		} else {
			out = append(out, label)
		}
	}

	return out
}

func (p *Node) find(k radix.Key) *x.Label {
	value, ok := p.tree.Get(k)
	if ok && value != nil {
		return value.(*x.Label)
	}
	return nil
}
