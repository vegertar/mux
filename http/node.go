package http

import (
	radix "github.com/armon/go-radix"
	"github.com/gobwas/glob"
	"github.com/vegertar/mux/x"
)

type Node struct {
	labels []*x.Label
	tree   *radix.Tree
	up     *x.Label
}

func NewNode() *Node {
	return &Node{
		tree: radix.New(),
	}
}

func (p *Node) Up() *x.Label {
	return p.up
}

func (p *Node) Empty() bool {
	return len(p.labels) == 0 && p.tree.Len() == 0
}

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

func (p *Node) Make(route x.Route) (leaf *x.Label, err error) {
	node := p
	for _, s := range route {
		if leaf != nil {
			if leaf.Down == nil {
				down := NewNode()
				down.up = leaf

				leaf.Down = down
			}
			node = leaf.Down.(*Node)
		}

		leaf = node.find(s)

		if leaf == nil {
			leaf, err = x.NewLabel(s)
			if err != nil {
				return nil, err
			}

			leaf.Node = node
			if leaf.Glob != nil {
				// TODO: adjusts the insertion algorithm

				var group1, group2, group3 []*x.Label
				for _, l := range node.labels {
					if leaf.Match(l.String()) {
						group1 = append(group1, l)
					} else if l.Match(leaf.String()) {
						group2 = append(group2, l)
					} else {
						group3 = append(group3, l)
					}
				}

				node.labels = append(append(append(group1, leaf), group2...), group3...)
			} else {
				node.tree.Insert(leaf.String(), leaf)
			}
		}
	}

	return leaf, nil
}

func (p *Node) Get(route x.Route) *x.Label {
	var leaf *x.Label

	node := p
	for _, s := range route {
		if leaf != nil {
			node = leaf.Down.(*Node)
			leaf = nil
		}

		if node != nil {
			leaf = node.find(s)
		}

		if leaf == nil {
			break
		}
	}

	return leaf
}

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

func (p *Node) walk(depth int, f func(int, *x.Label)) {
	var next []*Node
	p.tree.Walk(func(s string, v interface{}) bool {
		label := v.(*x.Label)
		f(depth, label)
		if label.Down != nil {
			next = append(next, label.Down.(*Node))
		}
		return false
	})
	for _, label := range p.labels {
		f(depth, label)
		if label.Down != nil {
			next = append(next, label.Down.(*Node))
		}
	}

	for _, node := range next {
		node.walk(depth+1, f)
	}
}

func (p *Node) find(s string) *x.Label {
	if p.tree.Len() > 0 || len(p.labels) > 0 {
		if glob.QuoteMeta(s) == s {
			if v, ok := p.tree.Get(s); ok {
				return v.(*x.Label)
			}
		} else {
			for _, l := range p.labels {
				if l.String() == s {
					return l
				}
			}
		}
	}

	return nil
}
