// Package x implements a common router entry to concurrently add and delete mux handlers and middleware.
package x

import (
	"errors"
	"sync"

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

	// Router is the mux in which carries a few options and a root node.
	Router struct {
		// Breed is a factory function to create a new node.
		Breed BreedFunc
		// DisableDupRoute disallowed to register duplicated routes.
		DisableDupRoute bool

		mu   sync.RWMutex
		tree Node
	}
)

// Routes returns all routes which has associated handlers or middleware.
func (p *Router) Routes() []Route {
	var out []Route

	for _, leaf := range p.root().Leaves() {
		for {
			if len(leaf.Handler) > 0 || len(leaf.Middleware) > 0 {
				if route := p.route(leaf); len(route) > 0 {
					out = append(out, route)
				}
			}
			up := leaf.Node.Up()
			if up == nil {
				break
			}
			leaf = up.Clone()
		}
	}

	return out
}

// Match matches a route and returns all associated labels.
func (p *Router) Match(r Route) []*Label {
	return p.root().Match(r)
}

// Use associates a route with middleware.
func (p *Router) Use(r Route, m ...interface{}) (CloseFunc, error) {
	return p.leaf(r).setupMiddleware(m), nil
}

// Handle associates a route with handlers.
// If set `DisableDupRoute`, only one handle can be added or `ErrExistedRoute` is returned.
func (p *Router) Handle(r Route, h ...interface{}) (CloseFunc, error) {
	leaf := p.leaf(r)
	if p.DisableDupRoute && len(leaf.Handler) > 0 {
		return nil, ErrExistedRoute
	}
	return leaf.setupHandler(h), nil
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
			node = leaf.getDown(p.Breed)
		}

		leaf = node.Get(k, true)
	}

	return leaf
}

func (p *Router) route(leaf *Label) Route {
	var route []radix.Key

	for {
		route = append(route, leaf.Key)
		up := leaf.Node.Up()
		if up == nil {
			break
		}
		leaf = up.Clone()
	}
	for i, j := 0, len(route) - 1; i < j; i, j = i+1, j-1 {
		route[i], route[j] = route[j], route[i]
	}
	return route
}
