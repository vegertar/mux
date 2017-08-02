// Package dns implements a dns mux.
package dns

import (
	"strings"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
)

type Router struct {
	*x.Router
}

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

func (p *Router) Match(c Route) Handler {
	r, err := newRoute(c)
	if err != nil {
		return FailureErrorHandler
	}
	return newHandlerFromLabels(p.Router.Match(r))
}

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

func (p *Router) UseFunc(r Route, m ...MiddlewareFunc) (x.CloseFunc, error) {
	m2 := make([]Middleware, 0, len(m))
	for _, v := range m {
		m2 = append(m2, v)
	}
	return p.Use(r, m2...)
}

func (p *Router) Handle(c Route, h Handler) (x.CloseFunc, error) {
	r, err := newRoute(c)
	if err != nil {
		return nil, err
	}
	return p.Router.Handle(r, h)
}

func (p *Router) HandleFunc(r Route, h HandlerFunc) (x.CloseFunc, error) {
	return p.Handle(r, h)
}

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
	if rw, ok := w.(*responseWriter); ok && !rw.written {
		if err := rw.WriteMsg(req.Msg); err != nil {
			// TODO: no panic
			panic(err)
		}
	}
}
