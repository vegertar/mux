package x

import (
	"strings"

	"github.com/gobwas/glob"
	"github.com/vegertar/mux/x/radix"
)

// NewStringKey creates a string literal key with a separator.
func NewStringKey(s, separator string) (radix.Key, error) {
	return NewStringSliceKey(strings.Split(s, separator))
}

// NewStringSliceKey creates a key from a string literal slice.
func NewStringSliceKey(v []string) (radix.Key, error) {
	var key radix.Key
	for _, s := range v {
		label, err := NewLiteralLabel(s)
		if err != nil {
			return nil, err
		}
		key = append(key, label)
	}
	return key, nil
}

// NewGlobKey creates a glob patterned key from a string with a separator.
func NewGlobKey(s, separator string) (radix.Key, error) {
	return NewGlobSliceKey(strings.Split(s, separator))
}

// NewGlobSliceKey creates a glob patterned key from a string slice.
func NewGlobSliceKey(v []string) (radix.Key, error) {
	var key radix.Key
	for _, s := range v {
		label, err := NewLabel(s)
		if err != nil {
			return nil, err
		}
		key = append(key, label)
	}
	return key, nil
}

type globLabel struct {
	glob glob.Glob
	text string
}

// Literal implements `radix.Label` interface.
func (p *globLabel) Literal() bool {
	return p.glob == nil
}

// String implements `radix.Label` interface.
func (p *globLabel) String() string {
	return p.text
}

// Match implements `radix.Label` interface.
func (p *globLabel) Match(s string) bool {
	if p.glob != nil {
		return p.glob.Match(s)
	}

	return p.text == s
}

// NewLabel creates an empty label from either a glob pattern text or a string literal.
func NewLabel(s string) (radix.Label, error) {
	p := new(globLabel)
	p.text = s
	if s != "" && glob.QuoteMeta(s) != s {
		g, err := glob.Compile(s)
		if err != nil {
			return nil, err
		}
		p.glob = g
	}

	return p, nil
}

// NewLiteralLabel creates an empty literal label.
func NewLiteralLabel(s string) (radix.Label, error) {
	return &globLabel{
		text: s,
	}, nil
}
