package x

import (
	"container/list"
	"sync"
	"sync/atomic"

	"github.com/vegertar/mux/x/radix"
)

// Label is used to represent a value.
type Label struct {
	Value
	Key radix.Key

	h  list.List
	m  list.List
	mu sync.RWMutex
}

// Clone returns a shadow copy
func (p *Label) Clone() *Label {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var v Label
	v.Key = p.Key
	v.Value = p.Value

	return &v
}

func (p *Label) getDown(breed BreedFunc) Node {
	p.mu.RLock()
	down := p.Down
	p.mu.RUnlock()
	if down == nil {
		down = breed(p)
		p.mu.Lock()
		if p.Down == nil {
			p.Down = down
		} else {
			down = p.Down
		}
		p.mu.Unlock()
	}
	return down
}

// free delete all trivial labels down to up
func (p *Label) free() {
	for v := p; v != nil &&
		len(v.Handler) == 0 &&
		len(v.Middleware) == 0 &&
		(v.Down == nil || v.Down.Empty()); v = v.Node.Up() {
		v.Node.Delete(v)
	}
}

func (p *Label) setupHandler(h []interface{}) CloseFunc {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Handler = append(p.Handler, h...)
	elem := p.h.PushBack(h)

	var closed int32
	return func() {
		if atomic.CompareAndSwapInt32(&closed, 0, 1) {
			p.mu.Lock()
			defer p.mu.Unlock()

			p.h.Remove(elem)
			p.Handler = p.Handler[:0]
			for e := p.h.Front(); e != nil; e = e.Next() {
				p.Handler = append(p.Handler, e.Value.([]interface{})...)
			}

			p.free()
		}
	}
}

func (p *Label) setupMiddleware(m []interface{}) CloseFunc {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Middleware = append(p.Middleware, m...)
	elem := p.m.PushBack(m)

	var closed int32
	return func() {
		if atomic.CompareAndSwapInt32(&closed, 0, 1) {
			p.mu.Lock()
			defer p.mu.Unlock()

			p.m.Remove(elem)
			p.Middleware = p.Middleware[:0]
			for e := p.m.Front(); e != nil; e = e.Next() {
				p.Middleware = append(p.Middleware, e.Value.([]interface{})...)
			}

			p.free()
		}
	}
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
	Match(route Route) []*Label
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
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tree.Len() == 0
}

// Delete implements the `Node` interface.
func (p *RadixNode) Delete(label *Label) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tree.Delete(label.Key)
}

// Match implements the `Node` interface.
func (p *RadixNode) Match(route Route) (leaves []*Label) {
	if len(route) > 0 {
		p.mu.RLock()
		match := p.tree.Match(route[0])
		p.mu.RUnlock()

		for _, v := range match {
			label := v.Value.(*Label).Clone()
			if len(route) > 1 && label.Down != nil {
				leaves = append(leaves, label.Down.Match(route[1:])...)
				continue
			}

			// check if label is a leaf
			if len(route) == 1 || label.Down == nil && label.Key.Wildcards() {
				leaves = append(leaves, label)
				continue
			}

			// remains as a middleware
			if len(label.Middleware) > 0 {
				// clears unnecessary handlers
				label.Handler = nil
				leaves = append(leaves, label)
			}
		}
	}

	return
}

// Leaves implements the `Node` interface.
func (p *RadixNode) Leaves() (leaves []*Label) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.tree.Walk(func(leaf radix.Leaf) bool {
		label := leaf.Value.(*Label).Clone()
		if label.Down != nil {
			leaves = append(leaves, label.Down.Leaves()...)
		} else {
			leaves = append(leaves, label)
		}
		return false
	})

	return
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
