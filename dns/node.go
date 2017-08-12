package dns

import (
	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
)

// Node derives the `x.RadixNode` with specialized dns matching.
type Node struct {
	*x.RadixNode
}

// Match implements the `x.Node` interface.
func (p *Node) Match(route x.Route) (leaves []*x.Label) {
	if len(route) > 2 {
		// first matches qname only
		nameLeaves := p.RadixNode.Match(route[:1])
		typeAndClass := route[1:]
		qtype := route[1][0].String()

		for index, nameLeaf := range nameLeaves {
			// then matches qtype and qclass
			down := nameLeaf.Down
			v := down.Match(typeAndClass)
			noData := true

			for i, leaf := range v {
				// remains middleware only for non-first matches
				if index > 0 {
					leaf.Handler = nil
				}
				if len(leaf.Handler) > 0 {
					noData = false
					var middleware Middleware
					switch qtype {
					case "CNAME":
						// no needs to follow name
					case "NS":
						middleware = p.glueMiddleware(false)
					case "SOA":
						middleware = p.soaMiddleware(false)
					default:
						middleware = p.cnameMiddleware(qtype)
					}
					if middleware != nil {
						h := middleware.GenerateHandler(newMultiHandler(leaf.Handler...))
						leaf.Handler = []interface{}{h}
						v[i] = leaf
					}
				}
			}

			if index == 0 && noData {
				// domain is existed, but no exactly matched data
				switch qtype {
				case "CNAME", "NS":
					// no needs to match again
				case "SOA":
					// delays handling
				default:
					// checking CNAME
					cnameKey, err := x.NewStringSliceKey(cnameType)
					if err != nil {
						panic(err)
					}
					labels := down.Match(x.Route{cnameKey, route[2]})
					for i, leaf := range labels {
						if len(leaf.Handler) > 0 {
							noData = false
							h := p.cnameMiddleware(qtype).GenerateHandler(newMultiHandler(leaf.Handler...))
							leaf.Handler = []interface{}{h}
							labels[i] = leaf
						}
					}
					if noData {
						// if no data then checks NS
						nsKey, err := x.NewStringSliceKey(nsType)
						if err != nil {
							panic(err)
						}
						labels = down.Match(x.Route{nsKey, route[2]})
						for i, leaf := range labels {
							if len(leaf.Handler) > 0 {
								noData = false
								h := p.glueMiddleware(true).GenerateHandler(newMultiHandler(leaf.Handler...))
								leaf.Handler = []interface{}{h}
								labels[i] = leaf
							}
						}
						if !noData {
							v = append(v, labels...)
						}
					}
				}

				if noData {
					// finally checking SOA
					soaKey, err := x.NewStringSliceKey(soaType)
					if err != nil {
						panic(err)
					}
					labels := down.Match(x.Route{soaKey, route[2]})
					for i, leaf := range labels {
						if len(leaf.Handler) > 0 {
							noData = false
							h := p.soaMiddleware(false).GenerateHandler(newMultiHandler(leaf.Handler...))
							leaf.Handler = []interface{}{h}
							labels[i] = leaf
						}
					}
					if qtype == "SOA" || !noData {
						v = append(v, labels...)
					}
				}

				if noData {
					noError := new(x.Label)
					noError.Key = nameLeaf.Key
					noError.Handler = []interface{}{NoErrorHandler}
					v = append(v, noError)
				}
			}

			leaves = append(leaves, v...)
		}

		return
	}

	return p.RadixNode.Match(route)
}

func (p *Node) cnameMiddleware(qtype string) Middleware {
	if p.Up() != nil {
		panic("required root")
	}

	return MiddlewareFunc(func(h Handler) Handler {
		if qtype == "CNAME" {
			return h
		}

		return HandlerFunc(func(w ResponseWriter, req *Request) {
			cnameWriter := &responseWriter{}
			h.ServeDNS(cnameWriter, req)

			recursiveWriter := &responseWriter{}
			recursiveQuestion := &Request{Msg: new(dns.Msg)}

			for _, rr := range cnameWriter.msg.Answer {
				ns, ok := rr.(*dns.CNAME)
				if !ok {
					continue
				}

				r := Route{Name: ns.Target}
				r.UseLiteral = true
				r.Type = qtype

				route, err := newRoute(r)
				if err != nil {
					panic(err)
				}

				recursiveQuestion.SetQuestion(ns.Target, req.Question[0].Qtype)
				newHandlerFromLabels(p.Match(route)).ServeDNS(recursiveWriter, recursiveQuestion)
			}

			cnameWriter.WriteMsg(&recursiveWriter.msg)
			cnameWriter.WriteMsg(req.Msg)
			w.WriteMsg(&cnameWriter.msg)
		})
	})
}

func (p *Node) soaMiddleware(nameError bool) Middleware {
	if p.Up() != nil {
		panic("required root")
	}

	return MiddlewareFunc(func(h Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, req *Request) {
			soaWriter := &responseWriter{}
			h.ServeDNS(soaWriter, req)

			if len(soaWriter.msg.Ns) == 0 && len(soaWriter.msg.Answer) > 0 {
				if soa, ok := soaWriter.msg.Answer[0].(*dns.SOA); ok {
					if soa.Hdr.Name == req.Msg.Question[0].Name {
						// adding NS records for an original name
						nsWriter := &responseWriter{}
						nsQuestion := &Request{Msg: new(dns.Msg)}
						nsQuestion.SetQuestion(req.Msg.Question[0].Name, dns.TypeNS)

						r := Route{Name: req.Msg.Question[0].Name}
						r.UseLiteral = true
						r.Type = "NS"

						route, err := newRoute(r)
						if err != nil {
							panic(err)
						}

						newHandlerFromLabels(p.Match(route)).ServeDNS(nsWriter, nsQuestion)

						soaWriter.Ns(nsWriter.msg.Answer...)
						soaWriter.Extra(nsWriter.msg.Extra...)
					} else {
						soaWriter.msg.Ns, soaWriter.msg.Answer = soaWriter.msg.Answer, soaWriter.msg.Ns
					}
				}
			}

			soaWriter.WriteMsg(req.Msg)
			if nameError {
				w.Header().Rcode = dns.RcodeNameError
			}
			w.WriteMsg(&soaWriter.msg)
		})
	})
}

func (p *Node) glueMiddleware(delegated bool) Middleware {
	if p.Up() != nil {
		panic("required root")
	}

	return MiddlewareFunc(func(h Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, req *Request) {
			nsWriter := &responseWriter{}
			h.ServeDNS(nsWriter, req)

			glueWriter := &responseWriter{}
			glueQuestion := &Request{Msg: new(dns.Msg)}

			for _, rr := range append(nsWriter.msg.Answer, nsWriter.msg.Ns...) {
				ns, ok := rr.(*dns.NS)
				if !ok {
					continue
				}

				r := Route{Name: ns.Ns}
				r.UseLiteral = true

				glueQuestion.SetQuestion(ns.Ns, dns.TypeA)
				r.Type = "A"
				route, err := newRoute(r)
				if err != nil {
					panic(err)
				}
				newHandlerFromLabels(p.Match(route)).ServeDNS(glueWriter, glueQuestion)

				glueQuestion.SetQuestion(ns.Ns, dns.TypeAAAA)
				r.Type = "AAAA"
				route, err = newRoute(r)
				if err != nil {
					panic(err)
				}
				newHandlerFromLabels(p.Match(route)).ServeDNS(glueWriter, glueQuestion)
			}

			if delegated {
				nsWriter.msg.Answer, nsWriter.msg.Ns = nsWriter.msg.Ns, nsWriter.msg.Answer
				w.Header().Authoritative = false
			}

			nsWriter.Extra(glueWriter.msg.Answer...)
			nsWriter.WriteMsg(req.Msg)
			w.WriteMsg(&nsWriter.msg)
		})
	})
}
