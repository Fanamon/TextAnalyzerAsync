package main

import (
	"bufio"
	"strings"
)

type Analyzer interface {
	Analyze(content string) AnalysisResult
	Name() string
}

type WordCountAnalyzer struct{}

type CharCountAnalyzer struct{}

type LineCountAnalyzer struct{}

type MostFrequentWordsAnalyzer struct{}

type AnalysisResult struct {
	Name     string
	Words    int
	Lines    int
	Chars    int
	WordFreq map[string]int
	Err      error
}

func (WordCountAnalyzer) Name() string         { return "words" }
func (CharCountAnalyzer) Name() string         { return "chars" }
func (LineCountAnalyzer) Name() string         { return "lines" }
func (MostFrequentWordsAnalyzer) Name() string { return "top-words" }

func (WordCountAnalyzer) Analyze(content string) AnalysisResult {
	wordsList := strings.Fields(content)

	return AnalysisResult{
		Name:  "words",
		Words: len(wordsList),
	}
}

func (CharCountAnalyzer) Analyze(content string) AnalysisResult {
	chars := 0
	for range content {
		chars++
	}

	return AnalysisResult{
		Name:  "chars",
		Chars: chars,
	}
}

func (LineCountAnalyzer) Analyze(content string) AnalysisResult {
	scanner := bufio.NewScanner(strings.NewReader(content))
	lines := 0
	for scanner.Scan() {
		lines++
	}

	if err := scanner.Err(); err != nil {
		return AnalysisResult{Name: "lines", Err: err}
	}

	return AnalysisResult{
		Name:  "lines",
		Lines: lines,
	}
}

func (MostFrequentWordsAnalyzer) Analyze(content string) AnalysisResult {
	wordsList := strings.Fields(content)
	freq := make(map[string]int)
	for _, w := range wordsList {
		freq[strings.ToLower(w)]++
	}

	return AnalysisResult{
		Name:     "top-words",
		WordFreq: freq,
	}
}
