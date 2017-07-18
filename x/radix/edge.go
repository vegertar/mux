package radix

import (
	"sort"
)

// edge is used to represent an edge node
type edge struct {
	label Label
	node  *node
}

type edges []edge

func (e edges) Len() int {
	return len(e)
}

func (e edges) Less(i, j int) bool {
	return e[i].label.Compare(e[j].label.Bytes()) < 0
}

func (e edges) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e edges) Sort() {
	sort.Sort(e)
}

func (e edges) Search(l Label) int {
	num := len(e)
	b := l.Bytes()

	idx := sort.Search(num, func(i int) bool {
		return e[i].label.Compare(b) >= 0
	})
	if idx < num && e[idx].label.Compare(b) == 0 {
		return idx
	}
	return -1
}
