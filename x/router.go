// Package x implements a common router entry to concurrently add and delete mux handlers and middleware.
package x

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/vegertar/mux/x/radix"
)

var (
	// ErrExistedRoute resulted from adding a handler with an existed route if configured `DisableDupRoute`.
	ErrExistedRoute = errors.New("existed route")
)

type (
	// CloseFunc tells to unload a handler or a middleware.
	// After the first call, subsequent calls to a CloseFunc do nothing.
	CloseFunc func()

	// BreedFunc creates a node under the given label.
	BreedFunc func(up *Label) Node

	// Route is a matching sequence for muxing request, e.g. an array of `scheme`, `method`, `path`, etc.
	Route []radix.Key

	// Router is a root node of mux and carries a few options.
	Router struct {
		Breed           BreedFunc
		DisableDupRoute bool

		mu   sync.RWMutex
		tree Node
	}
)

// Routes returns all routes which has associated handlers or middleware.
func (p *Router) Routes() []Route {
	var out []Route

	for _, leaf := range p.root().Leaves() {
		var layers []radix.Key
		for leaf != nil {
			if len(leaf.Handler) > 0 || len(leaf.Middleware) > 0 {
				layers = append(layers, leaf.Key)
			}
			leaf = leaf.Node.Up()
		}

		if len(layers) > 0 {
			out = append(out, layers)
		}
	}

	return out
}

// Match matches a route and returns all associated labels.
func (p *Router) Match(r Route) []Label {
	return p.root().Match(r)
}

// Use associates a route with middleware.
func (p *Router) Use(r Route, m ...interface{}) (CloseFunc, error) {
	leaf := p.leaf(r)

	// TODO: locking
	for _, v := range m {
		leaf.Middleware = append(leaf.Middleware, v)
	}

	var closed int32
	return func() {
		if atomic.CompareAndSwapInt32(&closed, 0, 1) {
			for _, t := range m {
				index := -1
				for i, v := range leaf.Middleware {
					if v == t {
						index = i
						break
					}
				}

				if index != -1 {
					leaf.Middleware = append(leaf.Middleware[:index], leaf.Middleware[index+1:]...)
				}
			}

			p.free(leaf)
		}
	}, nil
}

// Handle associates a route with handlers.
// If configured `DisableDupRoute`, only one handle can be added or `ErrExistedRoute` will be returned.
func (p *Router) Handle(r Route, h ...interface{}) (CloseFunc, error) {
	leaf := p.leaf(r)
	if p.DisableDupRoute && len(leaf.Handler) > 0 {
		return nil, ErrExistedRoute
	}

	for _, v := range h {
		leaf.Handler = append(leaf.Handler, v)
	}

	var closed int32
	return func() {
		if atomic.CompareAndSwapInt32(&closed, 0, 1) {
			p.mu.Lock()
			defer p.mu.Unlock()

			for _, t := range h {
				index := -1
				for i, v := range leaf.Handler {
					if v == t {
						index = i
						break
					}
				}

				if index != -1 {
					leaf.Handler = append(leaf.Handler[:index], leaf.Handler[index+1:]...)
				}
			}

			p.free(leaf)
		}
	}, nil
}

// free deletes a trivial route from down to up recursively.
func (p *Router) free(v *Label) {
	for v != nil && len(v.Handler) == 0 && len(v.Middleware) == 0 && (v.Down == nil || v.Down.Empty()) {
		node := v.Node
		node.Delete(v)
		v = node.Up()
	}
}

func (p *Router) root() Node {
	p.mu.RLock()
	root := p.tree
	p.mu.RUnlock()

	if root == nil {
		root = p.Breed(nil)
		p.mu.Lock()
		if p.tree != nil {
			root = p.tree
		} else {
			p.tree = root
		}
		p.mu.Unlock()
	}

	return root
}

func (p *Router) leaf(r Route) *Label {
	var (
		leaf *Label
		node = p.root()
	)

	for _, k := range r {
		if leaf != nil {
			// TODO: locking leaf
			if leaf.Down == nil {
				leaf.Down = p.Breed(leaf)
			}
			node = leaf.Down
		}

		leaf = node.Get(k, true)
	}

	return leaf
}
