package http

import (
	"fmt"
	"strings"

	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

const (
	glob      = "*"
	wildcards = "**"
)

var (
	globSlice      = []string{glob}
	wildcardsSlice = []string{wildcards}
)

// Route is the HTTP route component configure.
type Route struct {
	Scheme     string `json:"scheme,omitempty" default:"*"`
	Method     string `json:"method,omitempty" default:"*"`
	Host       string `json:"host,omitempty" default:"**"`
	Path       string `json:"path,omitempty" default:"**"`
	UseLiteral bool   `json:"useLiteral,omitempty"`
}

// String returns the string representation.
func (r Route) String() string {
	scheme, method, host, path := glob, glob, wildcards, wildcards
	if len(r.Scheme) > 0 {
		scheme = strings.ToLower(r.Scheme)
	}
	if len(r.Method) > 0 {
		method = strings.ToUpper(r.Method)
	}
	if len(r.Host) > 0 {
		host = strings.ToLower(r.Host)
	}
	if len(r.Path) > 0 {
		path = strings.TrimPrefix(strings.ToLower(r.Path), "/")
	}
	return fmt.Sprintf("%s %s://%s/%s", method, scheme, host, path)
}

func newRoute(r Route) (x.Route, error) {
	v := make([]radix.Key, 0, 4)
	f := x.NewGlobSliceKey
	if r.UseLiteral {
		f = x.NewStringSliceKey
	}

	var (
		key radix.Key
		err error
	)

	if len(r.Scheme) > 0 {
		key, err = f([]string{strings.ToLower(r.Scheme)})
	} else {
		key, err = f(globSlice)
	}
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	if len(r.Method) > 0 {
		key, err = f([]string{strings.ToUpper(r.Method)})
	} else {
		key, err = f(globSlice)
	}
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	if len(r.Host) > 0 {
		key, err = f(strings.Split(strings.ToLower(r.Host), "."))
	} else {
		key, err = f(wildcardsSlice)
	}
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	if len(r.Path) > 0 {
		key, err = f(strings.Split(strings.ToLower(r.Path), "/"))
	} else {
		key, err = f(wildcardsSlice)
	}
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	return v, nil
}
