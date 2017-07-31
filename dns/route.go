package dns

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

var (
	aType   = []string{"A"}
	inClass = []string{"IN"}
)

type Route struct {
	Name       string `json:"name,omitempty"`
	Type       string `json:"type,omitempty"`
	Class      string `json:"class,omitempty"`
	UseLiteral bool   `json:"useLiteral,omitempty"`
}

func (r Route) String() string {
	name, typ, class := r.Name, r.Type, r.Class
	if len(r.Name) {
		name = strings.ToLower(r.Name)
	}
	if len(r.Type) > 0 {
		typ = strings.ToUpper(r.Type)
	}
	if len(r.Class) > 0 {
		class = strings.ToUpper(r.Class)
	}
	return fmt.Sprintf("%s %s %s", name, typ, class)
}

func newRoute(r Route) (x.Route, error) {
	v := make([]radix.Key, 0, 3)
	f := x.NewGlobSliceKey
	if r.UseLiteral {
		f = x.NewStringSliceKey
	}

	var (
		key radix.Key
		err error
	)

	if len(r.Name) > 0 {
		r.Name = strings.ToLower(r.Name)
	} else {
		r.Name = "."
	}
	name := dns.SplitDomainName(r.Name)
	// Since dns.SplitDomainName returns nil for root label,
	// so we append an empty label here for intercepting the root label.
	name = append(name, "")
	reverse(name)

	for _, s := range name {
		var (
			label radix.Label
			err   error
		)

		// Since wildcard records are different from HTTP hostname,
		// so we treat it as a string literal.
		// For matching any characters, please use double asterisks (**) instead.
		if s == "*" {
			label, err = x.NewLiteralLabel(s)
		} else {
			label, err = f(s)
		}
		if err != nil {
			return nil, err
		}
		key = append(key, label)
	}
	v = append(v, key)

	if len(r.Type) > 0 {
		key, err = f([]string{strings.ToUpper(r.Type)})
	} else {
		key, err = f(aType)
	}
	v = append(v, key)

	if len(r.Class) > 0 {
		key, err = f([]string{strings.ToUpper(r.Class)})
	} else {
		key, err = f(inClass)
	}
	v = append(v, key)

	return v, nil
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
