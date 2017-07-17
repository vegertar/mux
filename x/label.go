package x

import (
	"github.com/gobwas/glob"
)

// Label is a payload of handles and middlewares.
// Also, a Label carries a text label, the current node and a down stream node.
type Label struct {
	glob.Glob
	Handler    []interface{}
	Middleware []interface{}
	Node       Node
	Down       Node

	text string
}

// NewLabel creates an empty label from either a glob pattern text or a literal string.
func NewLabel(s string) (*Label, error) {
	p := new(Label)
	p.text = s
	if s != "" && glob.QuoteMeta(s) != s {
		g, err := glob.Compile(s)
		if err != nil {
			return nil, err
		}
		p.Glob = g
	}

	return p, nil
}

// NewLiteralLabel returns an empty literal label.
func NewLiteralLabel(s string) (*Label, error) {
	return &Label{
		text: s,
	}, nil
}

// String returns the text label value.
func (p *Label) String() string {
	return p.text
}

// Match checks if the label matches given text.
// For a literal label, only matching by text value and no pattern checking applied.
func (p *Label) Match(s string) bool {
	if p.Glob != nil {
		return p.Glob.Match(s)
	}

	return p.text == s
}

// Root returns up to the toppest root node.
func (p *Label) Root() Node {
	var node Node

	for label := p; label != nil && label.Node != nil; {
		node = label.Node
		label = node.Up()
	}

	return node
}
