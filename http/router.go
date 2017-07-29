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
func (p *Router) Match(c RouteConfig) http.Handler {
	r, err := newRoute(c)
	if err != nil {
		return func(w Response, req *Request) {
			http.Error(w, err, 500)
		}
	}
	return newHandlerFromLabel(p.Router.Match(r))
}

// Use associates a route with middlewares.
func (p *Router) Use(c RouteConfig, m ...Middleware) (x.CloseFunc, error) {
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
func (p *Router) UseFunc(r RouteConfig, m ...MiddlewareFunc) (x.CloseFunc, error) {
	m2 := make([]Middleware, 0, len(m))
	for _, v := range m {
		m2 = append(m2, v)
	}
	return p.Use(r, m2...)
}

// Handle associates a route with a `http.Handler`.
func (p *Router) Handle(c RouteConfig, h http.Handler) (x.CloseFunc, error) {
	r, err := newRoute(c)
	if err != nil {
		return nil, err
	}

	return p.Router.Handle(r, h)
}

// HandleFunc associates a route with an `http.HandlerFunc`.
func (p *Router) HandleFunc(c RouteConfig, h http.HandlerFunc) (x.CloseFunc, error) {
	return p.Handle(r, h)
}

// ServeHTTP implements the `http.Handler` interface.
func (p *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var c RouteConfig
	c.UseLiteral = true

	c.Scheme = req.URL.Scheme
	if c.Scheme == "" {
		c.Scheme = "http"
	}

	c.Method = req.Method
	c.Host, _, _ = net.SplitHostPort(req.Host)
	if c.Host == "" {
		c.Host = req.Host
	}
	c.Path = req.URL.Path

	p.Match(c).ServeHTTP(w, req)
}
