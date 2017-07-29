package http

import (
	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

// Node uses trie tree to store and search labeled route components.
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
	if label.Glob == nil {
		p.tree.Delete(label.String())
	} else {
		index := -1
		for i, v := range p.labels {
			if v.String() == label.String() {
				index = i
				break
			}
		}
		if index != -1 {
			p.labels = append(p.labels[:index], p.labels[index+1:]...)
		}
	}
}

// Make implements the `x.Node` interface.
func (p *Node) Make(route x.Route) (leaf *x.Label, err error) {
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
			leaf, err = x.NewLabel(s)
			if err != nil {
				return nil, err
			}

			leaf.Node = node
			if leaf.Glob != nil {
			} else {
				node.tree.Insert(leaf.String(), leaf)
			}
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
		s := route[0]

		var (
			labels     []*x.Label
			middleware []interface{}
		)

		if v, ok := p.tree.Get(s); ok {
			label := v.(*x.Label)
			middleware = append(middleware, label.Middleware...)
			labels = append(labels, label)
		}

		for _, label := range p.labels {
			if label.Match(s) {
				middleware = append(middleware, label.Middleware...)
				labels = append(labels, label)
			}
		}

		var available *x.Label
		for _, label := range labels {
			if len(label.Handler) > 0 {
				available = label
			}
			leaf = *label

			if label.Down != nil && len(route) > 1 {
				leaf = label.Down.Match(route[1:])
			}

			if len(leaf.Handler) > 0 {
				break
			}
		}

		for i, j := 0, len(middleware)-1; i < j; i, j = i+1, j-1 {
			middleware[i], middleware[j] = middleware[j], middleware[i]
		}
		middleware = append(middleware, leaf.Middleware...)

		if len(leaf.Handler) == 0 && available != nil {
			leaf = *available
		}
		leaf.Middleware = middleware
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
