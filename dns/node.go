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
func (p *Node) Match(route x.Route) x.Label {
	var labels Match
	if len(route) > 2 {
		qname, qtype, qclass := route[0], route[1], route[2]
		if match := p.match(qname, true); len(match) != 0 {
			labels = match.match(qtype, qclass)
		}
	} else {
		labels = p.match(route, false)
	}

	if len(labels) != 0 {
		return labels[0]
	}

	return x.Label{}
}

func (p *Node) cnameMiddleware(qtype string) Middleware {
	return MiddlewareFunc(func(h Handler) Handler {
		if qtype == "CNAME" {
			return h
		}

		return HandlerFunc(func(w ResponseWriter, r *Request) {
			cnameWriter := &responseWriter{}
			h.ServeDNS(cnameWriter, r)

			recursiveWriter := &responseWriter{}
			recursiveQuestion := &Request{Msg: new(dns.Msg)}

			for _, rr := range cnameWriter.msg.Answer {
				ns, ok := rr.(*dns.CNAME)
				if !ok {
					continue
				}

				route := Name(ns.Target)
				route.Type = qtype
				recursiveQuestion.SetQuestion(ns.Target, r.Question[0].Qtype)
				newHandlerFromLabel(p.Match(x.Route(route.Strings()))).ServeDNS(recursiveWriter, recursiveQuestion)
			}

			cnameWriter.WriteMsg(&recursiveWriter.msg)
			cnameWriter.WriteMsg(r.Msg)
			w.WriteMsg(&cnameWriter.msg)
		})
	})
}

func (p *Node) soaMiddleware(nameError bool) Middleware {
	return MiddlewareFunc(func(h Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			soaWriter := &responseWriter{}
			h.ServeDNS(soaWriter, r)

			if len(soaWriter.msg.Ns) == 0 && len(soaWriter.msg.Answer) > 0 {
				if soa, ok := soaWriter.msg.Answer[0].(*dns.SOA); ok {
					if soa.Hdr.Name == r.Msg.Question[0].Name {
						// adding NS records for an original name
						nsWriter := &responseWriter{}
						nsQuestion := &Request{Msg: new(dns.Msg)}
						nsQuestion.SetQuestion(r.Msg.Question[0].Name, dns.TypeNS)

						route := Name(r.Msg.Question[0].Name)
						route.Type = "NS"
						newHandlerFromLabel(p.Match(x.Route(route.Strings()))).ServeDNS(nsWriter, nsQuestion)

						soaWriter.Ns(nsWriter.msg.Answer...)
						soaWriter.Extra(nsWriter.msg.Extra...)
					} else {
						soaWriter.msg.Ns, soaWriter.msg.Answer = soaWriter.msg.Answer, soaWriter.msg.Ns
					}
				}
			}

			soaWriter.WriteMsg(r.Msg)
			if nameError {
				w.Header().Rcode = dns.RcodeNameError
			}
			w.WriteMsg(&soaWriter.msg)
		})
	})
}

func (p *Node) glueMiddleware(delegated bool) Middleware {
	return MiddlewareFunc(func(h Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			nsWriter := &responseWriter{}
			h.ServeDNS(nsWriter, r)

			glueWriter := &responseWriter{}
			glueQuestion := &Request{Msg: new(dns.Msg)}

			for _, r := range append(nsWriter.msg.Answer, nsWriter.msg.Ns...) {
				ns, ok := r.(*dns.NS)
				if !ok {
					continue
				}

				route := Name(ns.Ns)

				glueQuestion.SetQuestion(ns.Ns, dns.TypeA)
				route.Type = "A"
				newHandlerFromLabel(p.Match(x.Route(route.Strings()))).ServeDNS(glueWriter, glueQuestion)

				glueQuestion.SetQuestion(ns.Ns, dns.TypeAAAA)
				route.Type = "AAAA"
				newHandlerFromLabel(p.Match(x.Route(route.Strings()))).ServeDNS(glueWriter, glueQuestion)
			}

			if delegated {
				nsWriter.msg.Answer, nsWriter.msg.Ns = nsWriter.msg.Ns, nsWriter.msg.Answer
				w.Header().Authoritative = false
			}

			nsWriter.Extra(glueWriter.msg.Answer...)
			nsWriter.WriteMsg(r.Msg)
			w.WriteMsg(&nsWriter.msg)
		})
	})
}

func (p *Node) match(route x.Route, forName bool) (labels Match) {
	if len(route) > 0 {
		k := route[0]
		v := p.Get(k, false)
		if v == nil && k != "*" && len(route) == 1 && forName {
			v, ok = p.tree.Get("*")
		}

		if ok {
			label = v.(*x.Label)

			if label.Down != nil && len(route) > 1 {
				labels = label.Down.(*Node).match(route[1:], forName)
				if len(labels) != 0 {
					last := &labels[len(labels)-1]
					last.Middleware = append(label.Middleware, last.Middleware...)
				}
			}
		}

		labels = append(labels, *label)
	}

	return
}

type Match []x.Label

func (p Match) String() string {
	names := make([]string, 0, len(p))
	for _, label := range p {
		names = append(names, label.String())
	}
	return dns.Fqdn(strings.Join(names, "."))
}

func (p Match) add(q Match) Match {
	if len(p) == 0 {
		return q
	}
	if len(q) == 0 {
		return q
	}

	v := append(p, q...)
	leaf := q[0]
	for i := range p {
		label := p[i]
		label.Middleware = append(leaf.Middleware, label.Middleware...)
		v[i] = label
	}

	return v
}

func (p Match) available() bool {
	return len(p) != 0 && len(p[0].Handler) != 0
}

func (p Match) match(qtype, qclass radix.Key) (labels Match) {
	if nameLeaf := p[0]; nameLeaf.Down != nil {
		labels = nameLeaf.Down.(*Node).match(x.Route{qtype, qclass}, false).add(p)
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
				labels = p[1:].matchSOA(qclass, false)
			default:
				labels = p.matchCNAME(qtype, qclass)
				if !labels.available() {
					labels = p.matchNS(qclass, true)
				}
			}

			if !labels.available() && qtype != "SOA" {
				labels = p.matchSOA(qclass, false)
			}
		}
	} else {
		labels = p.matchSOA(qclass, true)
	}

	return
}

func (p Match) matchSOA(qclass string, nameError bool) Match {
	for i, label := range p {
		if down := label.Down; down != nil {
			q := down.(*Node).match(x.Route{"SOA", qclass}, false)
			if q.available() {
				q = q.add(p[i:])
				q.addSOA(nameError)
				return q
			}
		}
	}

	return p
}

func (p Match) matchNS(qclass string, delegated bool) Match {
	for i, label := range p {
		if down := label.Down; down != nil {
			q := down.(*Node).match(x.Route{"NS", qclass}, false)
			if q.available() {
				q = q.add(p[i:])
				q.addGlue(delegated)
				return q
			}
		}
	}

	return p
}

func (p Match) matchCNAME(qtype, qclass string) Match {
	if len(p) == 0 || p[0].Down == nil {
		return p
	}

	q := p[0].Down.(*Node).match(x.Route{"CNAME", qclass}, false).add(p)
	if q.available() && qtype != "CNAME" {
		q.addCNAME(qtype)
	}

	return q
}

func (p Match) addSOA(nameError bool) {
	root := p[len(p)-1].Node.(*Node)
	soaMiddleware := root.soaMiddleware(nameError)
	leaf := &p[0]
	leaf.Middleware = append([]interface{}{soaMiddleware}, leaf.Middleware...)
}

func (p Match) addGlue(delegated bool) {
	root := p[len(p)-1].Node.(*Node)
	glueMiddleware := root.glueMiddleware(delegated)
	leaf := &p[0]
	leaf.Middleware = append([]interface{}{glueMiddleware}, leaf.Middleware...)
}

func (p Match) addCNAME(qtype string) {
	root := p[len(p)-1].Node.(*Node)
	cnameMiddleware := root.cnameMiddleware(qtype)
	leaf := &p[0]
	leaf.Middleware = append([]interface{}{cnameMiddleware}, leaf.Middleware...)
}
