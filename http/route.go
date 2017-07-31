package http

import (
	"fmt"
	"strings"

	"github.com/vegertar/mux/x"
	"github.com/vegertar/mux/x/radix"
)

var globSlice = []string{"*"}

// Route is the HTTP route component configure.
type Route struct {
	Scheme     string `json:"scheme,omitempty"`
	Method     string `json:"method,omitempty"`
	Host       string `json:"host,omitempty"`
	Path       string `json:"path,omitempty"`
	UseLiteral bool   `json:"useLiteral,omitempty"`
}

// String returns the string representation.
func (r Route) String() string {
	scheme, method, host, path := "*", "*", "*", "*"
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
		path = strings.ToLower(r.Path)
	}
	return fmt.Sprintf("%s %s://%s%s", method, scheme, host, path)
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
