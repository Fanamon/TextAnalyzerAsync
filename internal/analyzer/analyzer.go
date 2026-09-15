package analyzer

import "context"

type Analyzer interface {
	Analyze(ctx context.Context, doc *Document) (Result, error)
	Name() string
}

type Metrics struct {
	Path     string
	Words    int
	Lines    int
	Chars    int
	WordFreq map[string]int
}

type Result struct {
	Name  string
	Merge func(*Metrics)
}

func List(topWords int) []Analyzer {
	list := []Analyzer{
		WordCountAnalyzer{},
		CharCountAnalyzer{},
		LineCountAnalyzer{},
	}
	if topWords > 0 {
		list = append(list, MostFrequentWordsAnalyzer{})
	}
	return list
}
