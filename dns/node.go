package dns

import (
	"strings"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

type Node struct {
	*x.RadixNode
}

// Match implements the `x.Node` interface.
func (p *Node) Match(route x.Route) (leaves []x.Label) {
	if len(route) > 2 {
		// first matches qname only
		nameLeaves := p.RadixNode.Match(route[:1])
		for _, nameLeaf := range nameLeaves {
			// then matches qtype and qclass
			v := nameLeaf.Node.Match(route[1:])
			if len(v) == 0 {
				nameLeaf.Handler = append(nameLeaf.Handler, NoErrorHandler)
				leaves = append(leaves, nameLeaf)
			} else {
				for _, leaf := range v {
					up := leaf.Node.Up()
					qtype := up.Key[0].String()
					if len(leaf.Handler) > 0 {
						switch qtype {
						case "CNAME":
							// no needs to follow name
						case "NS":
							labels.addGlue(false)
						case "SOA":
							labels.addSOA(false)
						default:
							labels.addCNAME(qtype)
						}
					} else {
						// domain is existed, but no exactly matched data
						switch qtype {
						case "CNAME", "NS":
							// no needs to match again
						case "SOA":
							labels = m[1:].matchSOA(qclass, false)
						default:
							labels = m.matchCNAME(qtype, qclass)
							if !labels.available() {
								labels = m.matchNS(qclass, true)
							}
						}

						if !labels.available() && qtype != "SOA" {
							labels = m.matchSOA(qclass, false)
						}
					}
				}
			}
		}

		return
	}

	return p.RadixNode.Match(route)
}

func (p *Node) cnameMiddleware(qtype string) Middleware {
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

				r := Name(ns.Target)
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

						r := Name(req.Msg.Question[0].Name)
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

				r := Name(ns.Ns)
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

type Match []x.Label

func (m Match) add(q Match) Match {
	if len(m) == 0 {
		return q
	}
	if len(q) == 0 {
		return q
	}

	v := append(m, q...)
	leaf := q[0]
	for i := range m {
		label := m[i]
		label.Middleware = append(leaf.Middleware, label.Middleware...)
		v[i] = label
	}

	return v
}

func (m Match) available() bool {
	return len(m) != 0 && len(m[0].Handler) != 0
}

func (m Match) match(qtype, qclass radix.Key) (labels Match) {
	if nameLeaf := m[0]; nameLeaf.Down != nil {
		labels = nameLeaf.Down.Match(x.Route{qtype, qclass}, false).add(m)
		if labels.available() {
			switch qtype {
			case "CNAME":
			// no needs to follow name
			case "NS":
				labels.addGlue(false)
			case "SOA":
				labels.addSOA(false)
			default:
				labels.addCNAME(qtype)
			}
		} else {
			// domain is existed, but no exactly matched data
			switch qtype {
			case "CNAME", "NS":
			// no needs to match again
			case "SOA":
				labels = m[1:].matchSOA(qclass, false)
			default:
				labels = m.matchCNAME(qtype, qclass)
				if !labels.available() {
					labels = m.matchNS(qclass, true)
				}
			}

			if !labels.available() && qtype != "SOA" {
				labels = m.matchSOA(qclass, false)
			}
		}
	} else {
		labels = m.matchSOA(qclass, true)
	}

	return
}

func (m Match) matchSOA(qclass string, nameError bool) Match {
	for i, label := range m {
		if down := label.Down; down != nil {
			q := down.(*Node).match(x.Route{"SOA", qclass}, false)
			if q.available() {
				q = q.add(m[i:])
				q.addSOA(nameError)
				return q
			}
		}
	}

	return m
}

func (m Match) matchNS(qclass string, delegated bool) Match {
	for i, label := range m {
		if down := label.Down; down != nil {
			q := down.(*Node).match(x.Route{"NS", qclass}, false)
			if q.available() {
				q = q.add(m[i:])
				q.addGlue(delegated)
				return q
			}
		}
	}

	return m
}

func (m Match) matchCNAME(qtype, qclass string) Match {
	if len(m) == 0 || m[0].Down == nil {
		return m
	}

	q := m[0].Down.(*Node).match(x.Route{"CNAME", qclass}, false).add(m)
	if q.available() && qtype != "CNAME" {
		q.addCNAME(qtype)
	}

	return q
}

func (m Match) addSOA(nameError bool) {
	root := m[len(m) - 1].Node.(*Node)
	soaMiddleware := root.soaMiddleware(nameError)
	leaf := &m[0]
	leaf.Middleware = append([]interface{}{soaMiddleware}, leaf.Middleware...)
}

func (m Match) addGlue(delegated bool) {
	root := m[len(m) - 1].Node.(*Node)
	glueMiddleware := root.glueMiddleware(delegated)
	leaf := &m[0]
	leaf.Middleware = append([]interface{}{glueMiddleware}, leaf.Middleware...)
}

func (m Match) addCNAME(qtype string) {
	root := m[len(m) - 1].Node.(*Node)
	cnameMiddleware := root.cnameMiddleware(qtype)
	leaf := &m[0]
	leaf.Middleware = append([]interface{}{cnameMiddleware}, leaf.Middleware...)
}
