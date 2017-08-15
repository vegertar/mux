// Package dns implements a DNS mux.
package dns

import (
	"context"
	"strings"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
)

// Router is a wrapper of DNS mux.
type Router struct {
	*x.Router
}

// NewRouter creates a DNS router.
func NewRouter() *Router {
	return &Router{
		Router: &x.Router{
			Breed: func(up *x.Label) x.Node {
				return &Node{
					RadixNode: x.NewRadixNode(up),
				}
			},
		},
	}
}

// Routes returns registered route sequences.
func (p *Router) Routes() []Route {
	var out []Route

	for _, route := range p.Router.Routes() {
		var r Route
		names := route[0].Strings()
		reverse(names)
		r.Name = strings.Join(names, ".")

		if len(route) > 1 {
			r.Class = route[1][0].String()
		}
		if len(route) > 2 {
			r.Type = route[2][0].String()
		}

		out = append(out, r)
	}

	return out
}

// Match returns an associated `http.Handle` by given route.
func (p *Router) Match(c Route) Handler {
	r, err := newRoute(c)
	if err != nil {
		return FailureErrorHandler
	}
	return newHandlerFromLabels(r, p.Router.Match(r))
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

// Handle associates a route with a `Handler`.
func (p *Router) Handle(c Route, h Handler) (x.CloseFunc, error) {
	r, err := newRoute(c)
	if err != nil {
		return nil, err
	}
	return p.Router.Handle(r, h)
}

// HandleFunc associates a route with an `HandlerFunc`.
func (p *Router) HandleFunc(r Route, h HandlerFunc) (x.CloseFunc, error) {
	return p.Handle(r, h)
}

// ServeDNS implements `Handler` interface.
func (p *Router) ServeDNS(w ResponseWriter, req *Request) {
	var r Route
	r.Class = dns.ClassToString[req.Question[0].Qclass]
	r.Type = dns.TypeToString[req.Question[0].Qtype]
	r.Name = req.Question[0].Name
	r.UseLiteral = true

	var h Handler
	if r.Class == "ANY" || r.Class == "" || r.Type == "ANY" || r.Type == "" {
		h = FormatErrorHandler
	} else {
		h = p.Match(r)
	}

	h.ServeDNS(w, req)
}

// ServeFunc returns a `dns.HandlerFunc`.
func (p *Router) ServeFunc(ctx context.Context) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		localCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		p.ServeDNS(&responseWriter{ResponseWriter: w}, &Request{Msg: r, ctx: localCtx})
	}
}
