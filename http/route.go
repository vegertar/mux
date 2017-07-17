package http

import (
	"fmt"
	"path"
	"strings"
)

// Route is the HTTP route components.
type Route struct {
	Scheme string `json:"scheme,omitempty"`
	Method string `json:"method,omitempty"`
	Host   string `json:"host,omitempty"`
	Path   string `json:"path,omitempty"`
}

// String returns the string representation.
// Pay attention that the empty component is replaced by asterisk (*).
func (r Route) String() string {
	if len(r.Scheme) == 0 {
		r.Scheme = "*"
	}
	if len(r.Method) == 0 {
		r.Method = "*"
	}
	if len(r.Host) == 0 {
		r.Host = "*"
	}
	if len(r.Path) == 0 {
		r.Path = "*"
	}
	return fmt.Sprintf("%s %s://%s%s", r.Method, r.Scheme, r.Host, r.Path)
}

// Strings returns a route sequence with all empty components are replaced by asterisks.
// Furthermore, the last continuous asterisks are merged as one component.
func (r Route) Strings() []string {
	var out []string

	if len(r.Scheme) > 0 {
		out = append(out, strings.ToLower(r.Scheme))
	} else {
		out = append(out, "*")
	}

	if len(r.Method) > 0 {
		out = append(out, strings.ToUpper(r.Method))
	} else {
		out = append(out, "*")
	}

	if len(r.Host) > 0 {
		out = append(out, strings.ToLower(r.Host))
	} else {
		out = append(out, "*")
	}

	if len(r.Path) > 0 {
		out = append(out, r.Path)
	} else {
		out = append(out, "*")
	}

	index := len(out)
	for index > 0 && out[index-1] == "*" {
		index--
	}

	if index < len(out) {
		out = out[:index+1]
	}

	return out
}

// Path is a helper function creating a route from a path.
func Path(s string) Route {
	return Route{
		Path: s,
	}
}

// PathPrefix is a helper function creating a route from a path prefix.
func PathPrefix(s string) Route {
	return Route{
		Path: path.Join(s, "*"),
	}
}
