package analyzer

import (
	"strings"
	"sync"
)

type Document struct {
	Path    string
	Content string

	once  sync.Once
	words []string
}

func (d *Document) Words() []string {
	d.once.Do(func() {
		d.words = strings.Fields(d.Content)
	})
	return d.words
}
