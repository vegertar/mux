package http

import (
	"strings"

	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

var globSlice = []string{"*"}

// RouteConfig is the HTTP route component configure.
type RouteConfig struct {
	Scheme     string `json:"scheme,omitempty"`
	Method     string `json:"method,omitempty"`
	Host       string `json:"host,omitempty"`
	Path       string `json:"path,omitempty"`
	UseLiteral bool   `json:"useLiteral,omitempty"`
}

func newRoute(c RouteConfig) (x.Route, error) {
	v := make([]radix.Key, 0, 4)
	f := x.NewGlobSliceKey
	if c.UseLiteral {
		f = x.NewStringSliceKey
	}

	var (
		key radix.Key
		err error
	)

	if len(r.Scheme) > 0 {
		key, err = f([]string{strings.ToUpper(r.Scheme)})
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
		key, err = f(globSlice)
	}
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	if len(r.Path) > 0 {
		key, err = f(strings.Split(strings.ToLower(r.Path), "/"))
	} else {
		key, err = f(globSlice)
	}
	if err != nil {
		return nil, err
	}
	v = append(v, key)

	return v, nil
}
