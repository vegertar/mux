package dns

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

var (
	nsType    = []string{"NS"}
	soaType   = []string{"SOA"}
	cnameType = []string{"CNAME"}
	aType     = []string{"A"}
	inClass   = []string{"IN"}
)

// Route is the DNS route component configure.
type Route struct {
	Name       string `json:"name,omitempty" default:"."`
	Type       string `json:"type,omitempty" default:"A"`
	Class      string `json:"class,omitempty" default:"IN"`
	UseLiteral bool   `json:"useLiteral,omitempty"`
}

// String returns the string representation.
func (r Route) String() string {
	name, typ, class := ".", "A", "IN"
	if len(r.Name) > 0 {
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

	key, err = f(name)
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	if len(r.Type) > 0 {
		key, err = f([]string{strings.ToUpper(r.Type)})
	} else {
		key, err = f(aType)
	}
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	if len(r.Class) > 0 {
		key, err = f([]string{strings.ToUpper(r.Class)})
	} else {
		key, err = f(inClass)
	}
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	return v, nil
}

// RR creates a route from RR records.
func RR(r dns.RR) Route {
	return Route{
		Name:  strings.ToLower(r.Header().Name),
		Type:  dns.TypeToString[r.Header().Rrtype],
		Class: dns.ClassToString[r.Header().Class],
	}
}

// Name creates a route from a domain name with default type A and class IN.
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
