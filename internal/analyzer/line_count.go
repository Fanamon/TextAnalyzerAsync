package analyzer

import (
	"context"
	"strings"
)

type LineCountAnalyzer struct{}

func (LineCountAnalyzer) Name() string { return "lines" }

func (a LineCountAnalyzer) Analyze(_ context.Context, doc *Document) (Result, error) {
	content := doc.Content
	lines := strings.Count(content, "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		lines++
	}

	return Result{
		Name: a.Name(),
		Merge: func(m *Metrics) {
			m.Lines += lines
		},
	}, nil
}
