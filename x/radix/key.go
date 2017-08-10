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
	// Wildcards returns if this label matches multiple continued tokens.
	Wildcards() bool
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

// Wildcards implements the `Label` interface.
func (s StringLabel) Wildcards() bool {
	return false
}

// globLabel is the label supposed to be matched by asterisk (*).
type globLabel struct {
	s         string
	parts     []string
	wildcards bool
}

// String implements the `Label` interface.
func (p *globLabel) String() string {
	return p.s
}

// Match implements the `Label` interface.
func (p *globLabel) Match(subj string) bool {
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
		subj = subj[idx + len(p.parts[i]):]
	}

	// Reached the last section. Requires special handling.
	return trailingGlob || strings.HasSuffix(subj, p.parts[end])
}

// Literal implements the `Label` interface.
func (p *globLabel) Literal() bool {
	return false
}

// Wildcards implements the `Label` interface.
func (p *globLabel) Wildcards() bool {
	return p.wildcards
}

// newGlobLabel creates a glob patterned label which can use '*' to match any string.
func newGlobLabel(s string) *globLabel {
	return &globLabel{
		s: s,
		parts: strings.Split(s, glob),
		wildcards: len(s) > 1 && strings.Trim(s, glob) == "",
	}
}

// Key is the key to insert, delete, and search a tree.
type Key []Label

func (k Key) split(f func(Label) bool) []Key {
	if len(k) == 0 {
		return nil
	}

	out := make([]Key, 0, 2)
	last := -1

	for i, label := range k {
		if f(label) {
			out = append(out, k[last + 1:i])
			last = i
		}
	}
	if last >= 0 && last < len(k) - 1 {
		out = append(out, k[last + 1:])
	}

	return out
}

func (k Key) matchExactly(x Key) bool {
	n := len(k)
	if n != len(x) {
		return false
	}

	for i := 0; i < n; i++ {
		if !k[i].Match(x[i].String()) {
			return false
		}
	}
	return true
}

// Match returns if key matches another one.
func (k Key) Match(x Key) bool {
	if len(k) == 0 {
		return len(x) == 0
	}

	leadingGlob := k[0].Wildcards()
	trailingGlob := k[len(k) - 1].Wildcards()

	parts := k.split(func(l Label) bool {
		return l.Wildcards()
	})

	if len(parts) == 0 {
		return true
	}

	if len(parts) == 1 {
		return parts[0].matchExactly(x)
	}

	end := len(parts) - 1

	// Go over the leading parts and ensure they match.
	for i := 0; i < end; i++ {
		idx := -1
		y := parts[i]
		n := len(y)
		for j := 0; j + n <= len(x); j++ {
			if y.matchExactly(x[j:j + n]) {
				idx = j
				break
			}
		}

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

		// Trim evaluated label from x as we loop over the pattern.
		x = x[idx + n:]
	}

	// Reached the last section. Requires special handling.
	if trailingGlob {
		return true
	}
	if y := parts[end]; len(x) >= len(y) {
		return y.matchExactly(x[len(x) - len(y):])
	}
	return false
}

// Is returns if key equals to given strings.
func (k Key) Is(x ...string) bool {
	if len(k) == len(x) {
		for i, label := range k {
			if label.String() != x[i] {
				return false
			}
		}
		return true
	}
	return false
}

// Wildcards returns if key contains wildcarded labels only.
func (k Key) Wildcards() bool {
	if len(k) > 0 {
		for _, label := range k {
			if !label.Wildcards() {
				return false
			}
		}
		return true
	}

	return false
}

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
		v    []string
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
		var label Label
		if strings.Index(s, glob) != -1 {
			label = newGlobLabel(s)
		} else {
			label = StringLabel(s)
		}
		labels = append(labels, label)
	}
	return labels
}
