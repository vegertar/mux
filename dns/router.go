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
			Node: NewNode(),
		},
	}
}

func (p *Router) Routes() []Route {
	var out []Route

	for _, route := range p.Router.Routes() {
		var r Route
		r.Class = route[len(route)-1]
		if len(route) > 1 {
			r.Type = route[len(route)-2]
		}
		if len(route) > 2 {
			names := route[:len(route)-2]
			reverse(names)
			r.Name = strings.Join(names, ".")
		}
		out = append(out, r)
	}

	return out
}

func (p *Router) Match(r Route) Handler {
	return newHandlerFromLabel(p.Router.Match(r.Strings()))
}

func (p *Router) Use(r Route, m ...Middleware) (x.CloseFunc, error) {
	m2 := make([]interface{}, 0, len(m))
	for _, v := range m {
		m2 = append(m2, v)
	}

	s := r.Strings()
	if r.Type == "" {
		s = s[:len(s)-2]
	} else if r.Class == "" {
		s = s[:len(s)-1]
	}

	return p.Router.Use(s, m2...)
}

func (p *Router) UseFunc(r Route, m ...MiddlewareFunc) (x.CloseFunc, error) {
	m2 := make([]Middleware, 0, len(m))
	for _, v := range m {
		m2 = append(m2, v)
	}
	return p.Use(r, m2...)
}

func (p *Router) Handle(r Route, h Handler) (x.CloseFunc, error) {
	return p.Router.Handle(r.Strings(), h)
}

func (p *Router) HandleFunc(r Route, h HandlerFunc) (x.CloseFunc, error) {
	return p.Handle(r, h)
}

func (p *Router) ServeDNS(w ResponseWriter, req *Request) {
	var r Route
	r.Class = dns.ClassToString[req.Question[0].Qclass]
	r.Type = dns.TypeToString[req.Question[0].Qtype]
	r.Name = req.Question[0].Name

	var h Handler
	if r.Class == "ANY" || r.Class == "" || r.Type == "ANY" || r.Type == "" {
		h = FormatErrorHandler
	} else {
		h = p.Match(r)
	}

	h.ServeDNS(w, req)
	if rw, ok := w.(*responseWriter); ok && !rw.written {
		if err := rw.WriteMsg(req.Msg); err != nil {
			panic(err)
		}
	}
}
