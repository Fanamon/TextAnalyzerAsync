package analyzer

import "context"

type WordCountAnalyzer struct{}

func (WordCountAnalyzer) Name() string { return "words" }

func (a WordCountAnalyzer) Analyze(_ context.Context, doc *Document) (Result, error) {
	n := len(doc.Words())
	return Result{
		Name: a.Name(),
		Merge: func(m *Metrics) {
			m.Words += n
		},
	}, nil
}
