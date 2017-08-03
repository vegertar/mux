// Package http implements a http mux.
package http

import (
	"net"
	"net/http"

	"github.com/vegertar/mux/x"
)

// Router is a wrapper of HTTP mux.
type Router struct {
	*x.Router
}

// NewRouter creates an HTTP router.
func NewRouter() *Router {
	return &Router{
		Router: &x.Router{
			Breed: func(up *x.Label) x.Node {
				return x.NewRadixNode(up),
			},
		},
	}
}

// Routes returns registered route sequences.
func (p *Router) Routes() []Route {
	var out []Route

	for _, route := range p.Router.Routes() {
		var r Route
		r.Scheme = route[0][0].String()
		if len(route) > 1 {
			r.Method = route[1][0].String()
		}
		if len(route) > 2 {
			r.Host = route[2].StringWith(".")
		}
		if len(route) > 3 {
			r.Path = route[3].StringWith("/")
		}
		out = append(out, r)
	}

	return out
}

// Match returns an associated `http.Handle` by given route.
func (p *Router) Match(c Route) http.Handler {
	r, err := newRoute(c)
	if err != nil {
		return func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, err, 500)
		}
	}
	return newHandlerFromLabels(p.Router.Match(r))
}

// Use associates a route with middleware.
func (p *Router) Use(c Route, m ...Middleware) (x.CloseFunc, error) {
	r, err := newRoute(c)
	if err != nil {
		return nil, err
	}

	m2 := make([]interface{}, 0, len(m))
	for _, v := range m {
		m2 = append(m2, v)
	}

	return p.Router.Use(r, m2...)
}

// UseFunc associates a route with middleware functions.
func (p *Router) UseFunc(r Route, m ...MiddlewareFunc) (x.CloseFunc, error) {
	m2 := make([]Middleware, 0, len(m))
	for _, v := range m {
		m2 = append(m2, v)
	}
	return p.Use(r, m2...)
}

// Handle associates a route with a `http.Handler`.
func (p *Router) Handle(c Route, h http.Handler) (x.CloseFunc, error) {
	r, err := newRoute(c)
	if err != nil {
		return nil, err
	}

	return p.Router.Handle(r, h)
}

// HandleFunc associates a route with an `http.HandlerFunc`.
func (p *Router) HandleFunc(c Route, h http.HandlerFunc) (x.CloseFunc, error) {
	return p.Handle(c, h)
}

// ServeHTTP implements the `http.Handler` interface.
func (p *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var r Route
	r.UseLiteral = true

	r.Scheme = req.URL.Scheme
	if r.Scheme == "" {
		r.Scheme = "http"
	}

	r.Method = req.Method
	r.Host, _, _ = net.SplitHostPort(req.Host)
	if r.Host == "" {
		r.Host = req.Host
	}
	r.Path = req.URL.Path

	p.Match(r).ServeHTTP(w, req)
}
