package radix

import (
	"strings"
)

const glob = "*"

// Label is the minimum comparing unit in the tree.
type Label interface {
	// Match returns true if label matches the given string.
	Match(other string) bool
	// String returns the string representation.
	String() string
	// Literal returns if this label contains string literal only.
	Literal() bool
}

// StringLabel is the label based on string literal.
type StringLabel string

// Match implements the `Label` interface.
func (s StringLabel) Match(other string) bool {
	return string(s) == other
}

// String implements the `Label` interface.
func (s StringLabel) String() string {
	return string(s)
}

// Literal implements the `Label` interface.
func (s StringLabel) Literal() bool {
	return true
}

// GlobLabel is the label supposed to be matched by asterisk (*).
type GlobLabel struct {
	s     string
	parts []string
}

// String implements the `Label` interface.
func (p *GlobLabel) String() string {
	return p.s
}

// Match implements the `Label` interface.
func (p *GlobLabel) Match(subj string) bool {
	if p.s == glob {
		return true
	}

	if len(p.parts) == 1 {
		return p.s == subj
	}

	leadingGlob := strings.HasPrefix(p.s, glob)
	trailingGlob := strings.HasSuffix(p.s, glob)
	end := len(p.parts) - 1

	// Go over the leading parts and ensure they match.
	for i := 0; i < end; i++ {
		idx := strings.Index(subj, p.parts[i])

		switch i {
		case 0:
			// Check the first section. Requires special handling.
			if !leadingGlob && idx != 0 {
				return false
			}
		default:
			// Check that the middle parts match.
			if idx < 0 {
				return false
			}
		}

		// Trim evaluated text from subj as we loop over the pattern.
		subj = subj[idx+len(p.parts[i]):]
	}

	// Reached the last section. Requires special handling.
	return trailingGlob || strings.HasSuffix(subj, p.parts[end])
}

// Literal implements the `Label` interface.
func (p *GlobLabel) Literal() bool {
	return false
}

// NewGlobLabel creates a glob patterned label which can use '*' to match any string.
func NewGlobLabel(s string) *GlobLabel {
	return &GlobLabel{s: s, parts: strings.Split(s, glob)}
}

// Key is the key to insert, delete, and search a tree.
type Key []Label

// StringWith returns a string joined with the given separator.
func (k Key) StringWith(separator string) string {
	return strings.Join(k.Strings(), separator)
}

// Strings returns string slice representation.
func (k Key) Strings() []string {
	if len(k) == 0 {
		return nil
	}

	s := make([]string, 0, len(k))
	for _, t := range k {
		s = append(s, t.String())
	}
	return s
}

// NewCharKey creates a general character sequenced key.
func NewCharKey(s string) Key {
	var (
		last = -1
		v []string
	)
	for i := range s {
		if last != -1 {
			v = append(v, s[last:i])
		}
		last = i
	}
	if last != -1 {
		v = append(v, s[last:])
	}
	return NewStringSliceKey(v)
}

// NewStringKey creates a string literal key with a separator.
func NewStringKey(s, separator string) Key {
	return NewStringSliceKey(strings.Split(s, separator))
}

// NewStringSliceKey creates a key from a string literal slice.
func NewStringSliceKey(v []string) Key {
	labels := make([]Label, 0, len(v))
	for _, s := range v {
		labels = append(labels, StringLabel(s))
	}
	return labels
}

// NewGlobKey creates a glob patterned key from a string with a separator.
func NewGlobKey(s, separator string) Key {
	return NewGlobSliceKey(strings.Split(s, separator))
}

// NewGlobSliceKey creates a glob patterned key from a string slice.
func NewGlobSliceKey(v []string) Key {
	labels := make([]Label, 0, len(v))
	for _, s := range v {
		labels = append(labels, NewGlobLabel(s))
	}
	return labels
}
