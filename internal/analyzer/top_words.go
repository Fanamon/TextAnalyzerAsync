package analyzer

import (
	"context"
	"strings"
)

type MostFrequentWordsAnalyzer struct{}

func (MostFrequentWordsAnalyzer) Name() string { return "top-words" }

func (a MostFrequentWordsAnalyzer) Analyze(_ context.Context, doc *Document) (Result, error) {
	freq := make(map[string]int)
	for _, w := range doc.Words() {
		freq[strings.ToLower(w)]++
	}

	return Result{
		Name: a.Name(),
		Merge: func(m *Metrics) {
			if m.WordFreq == nil {
				m.WordFreq = make(map[string]int, len(freq))
			}
			for word, count := range freq {
				m.WordFreq[word] += count
			}
		},
	}, nil
}
