package dns

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

const wildcards = "**"

var (
	nsType    = []string{"NS"}
	soaType   = []string{"SOA"}
	cnameType = []string{"CNAME"}
	aType     = []string{"A"}
	inClass   = []string{"IN"}
)

// Route is the DNS route component configure.
type Route struct {
	Name       string
	Type       string
	Class      string
	UseLiteral bool
}

// String returns the string representation.
func (r Route) String() string {
	name, typ, class := wildcards, wildcards, wildcards
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
		r.Name = wildcards
	}
	name := dns.SplitDomainName(r.Name)
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

// RR creates a route from a RR record.
func RR(r dns.RR) Route {
	return Route{
		Name:  r.Header().Name,
		Type:  dns.TypeToString[r.Header().Rrtype],
		Class: dns.ClassToString[r.Header().Class],
	}
}

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
