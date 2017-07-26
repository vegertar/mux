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
}

// StringLabel is the label based on literal string.
type StringLabel string

// Match implements the `Label` interface.
func (self StringLabel) Match(other string) bool {
	return string(self) == other
}

// String implements the `Label` interface.
func (self StringLabel) String() string {
	return string(self)
}

// GlobLabel is the label supposed to be matched by asterisk (*).
type GlobLabel struct {
	s        string
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

func NewGlobLabel(s string) *GlobLabel {
	return &GlobLabel{s: s, parts: strings.Split(s, glob)}
}

// Key is the key to insert, delete, and search a tree.
type Key []Label

// String returns the string representation, which equals to StringWith("").
func (self Key) String() string {
	return strings.Join(self.Strings(), "")
}

// StringWith returns a string joined with the given separator.
func (self Key) StringWith(separator string) string {
	return strings.Join(self.Strings(), separator)
}

// Strings returns string slice representation.
func (self Key) Strings() []string {
	s := make([]string, 0, len(self))
	for _, t := range self {
		s = append(s, t.String())
	}
	return s
}

// NewCharKey creates a general character sequenced key.
func NewCharKey(s string) Key {
	return NewStringKey(s, "")
}

// NewStringKey creates a literal string key from a string with a separator.
func NewStringKey(s, separator string) Key {
	return NewStringSliceKey(strings.Split(s, separator))
}

// NewStringKey creates a literal key from a string slice.
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
