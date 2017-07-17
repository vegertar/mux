package dns

import (
	"fmt"
	"github.com/miekg/dns"
	"strings"
)

type Route struct {
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
	Class string `json:"class,omitempty"`
}

func (r Route) String() string {
	if len(r.Name) > 0 {
		r.Name = strings.ToLower(r.Name)
	} else {
		r.Name = "."
	}

	if len(r.Type) > 0 {
		r.Type = strings.ToUpper(r.Type)
	} else {
		r.Type = "A"
	}

	if len(r.Class) > 0 {
		r.Class = strings.ToUpper(r.Class)
	} else {
		r.Class = "IN"
	}

	return fmt.Sprintf("%s %s %s", r.Name, r.Type, r.Class)
}

func (r Route) Strings() []string {
	if len(r.Name) > 0 {
		r.Name = strings.ToLower(r.Name)
	} else {
		r.Name = "."
	}

	out := dns.SplitDomainName(r.Name)
	// Since dns.SplitDomainName returns nil for root label,
	// so we append an empty label here for intercepting the root label.
	out = append(out, "")
	reverse(out)

	if len(r.Type) > 0 {
		out = append(out, strings.ToUpper(r.Type))
	} else {
		out = append(out, "A")
	}

	if len(r.Class) > 0 {
		out = append(out, strings.ToUpper(r.Class))
	} else {
		out = append(out, "IN")
	}

	return out
}

func RR(r dns.RR) Route {
	return Route{
		Name:  strings.ToLower(r.Header().Name),
		Type:  dns.TypeToString[r.Header().Rrtype],
		Class: dns.ClassToString[r.Header().Class],
	}
}

func Name(s string) Route {
	return Route{
		Name: strings.ToLower(s),
	}
}

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
