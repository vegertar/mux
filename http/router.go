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
			Node: NewNode(),
		},
	}
}

// Routes returns registered route sequences.
func (p *Router) Routes() []Route {
	var out []Route

	for _, route := range p.Router.Routes() {
		var r Route
		r.Scheme = route[len(route)-1]
		if len(route) > 1 {
			r.Method = route[len(route)-2]
		}
		if len(route) > 2 {
			r.Host = route[len(route)-3]
		}
		if len(route) > 3 {
			r.Path = route[len(route)-4]
		}
		out = append(out, r)
	}

	return out
}

// Match returns an associated `http.Handle` by given route.
func (p *Router) Match(r Route) http.Handler {
	return newHandlerFromLabel(p.Router.Match(r.Strings()))
}

// Use associates a route with middlewares.
func (p *Router) Use(r Route, m ...Middleware) (x.CloseFunc, error) {
	m2 := make([]interface{}, 0, len(m))
	for _, v := range m {
		m2 = append(m2, v)
	}

	return p.Router.Use(r.Strings(), m2...)
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
func (p *Router) Handle(r Route, h http.Handler) (x.CloseFunc, error) {
	return p.Router.Handle(r.Strings(), h)
}

// HandleFunc associates a route with an `http.HandlerFunc`.
func (p *Router) HandleFunc(r Route, h http.HandlerFunc) (x.CloseFunc, error) {
	return p.Handle(r, h)
}

// ServeHTTP implements the `http.Handler` interface.
func (p *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var r Route
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
