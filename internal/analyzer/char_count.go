package analyzer

import (
	"context"
	"unicode/utf8"
)

type CharCountAnalyzer struct{}

func (CharCountAnalyzer) Name() string { return "chars" }

func (a CharCountAnalyzer) Analyze(_ context.Context, doc *Document) (Result, error) {
	n := utf8.RuneCountInString(doc.Content)
	return Result{
		Name: a.Name(),
		Merge: func(m *Metrics) {
			m.Chars += n
		},
	}, nil
}
