package analyzer_test

import (
	"bufio"
	"context"
	"strings"
	"testing"

	"textanalyzerasync/internal/analyzer"
)

func doc(content string) *analyzer.Document {
	return &analyzer.Document{Content: content}
}

func mustAnalyze(t *testing.T, a analyzer.Analyzer, content string) analyzer.Metrics {
	t.Helper()
	got, err := a.Analyze(context.Background(), doc(content))
	if err != nil {
		t.Fatal(err)
	}
	var m analyzer.Metrics
	if got.Merge != nil {
		got.Merge(&m)
	}
	if got.Name != a.Name() {
		t.Fatalf("Result.Name = %q, Name() = %q", got.Name, a.Name())
	}
	return m
}

func TestAnalyzerNames(t *testing.T) {
	cases := []analyzer.Analyzer{
		analyzer.WordCountAnalyzer{},
		analyzer.LineCountAnalyzer{},
		analyzer.CharCountAnalyzer{},
		analyzer.MostFrequentWordsAnalyzer{},
	}
	wants := []string{"words", "lines", "chars", "top-words"}
	for i, a := range cases {
		if got := a.Name(); got != wants[i] {
			t.Errorf("%T.Name() = %q, want %q", a, got, wants[i])
		}
		_ = mustAnalyze(t, a, "x")
	}
}

func TestWordCountAnalyzer(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{name: "two words", content: "hello world", want: 2},
		{name: "empty", content: "", want: 0},
		{name: "only spaces", content: "   \t\n", want: 0},
		{name: "extra spaces", content: "  foo   bar  baz ", want: 3},
		{name: "cyrillic", content: "привет мир", want: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustAnalyze(t, analyzer.WordCountAnalyzer{}, tc.content)
			if got.Words != tc.want {
				t.Errorf("Words = %d, want %d", got.Words, tc.want)
			}
		})
	}
}

func TestLineCountAnalyzer(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{name: "empty", content: "", want: 0},
		{name: "no newline", content: "hello", want: 1},
		{name: "two lines", content: "hello\nworld", want: 2},
		{name: "trailing newline", content: "hello\nworld\n", want: 2},
		{name: "windows newline", content: "a\r\nb\r\n", want: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustAnalyze(t, analyzer.LineCountAnalyzer{}, tc.content)
			if got.Lines != tc.want {
				t.Errorf("Lines = %d, want %d", got.Lines, tc.want)
			}
		})
	}
}

func TestCharCountAnalyzer(t *testing.T) {
	got := mustAnalyze(t, analyzer.CharCountAnalyzer{}, "привет")
	if got.Chars != 6 {
		t.Errorf("Chars = %d, want 6", got.Chars)
	}
}

func TestMostFrequentWordsAnalyzer(t *testing.T) {
	got := mustAnalyze(t, analyzer.MostFrequentWordsAnalyzer{}, "The cat and the dog")
	if got.WordFreq["the"] != 2 {
		t.Errorf("the = %d, want 2", got.WordFreq["the"])
	}
	if _, ok := got.WordFreq["The"]; ok {
		t.Fatal("expected lowercase keys only")
	}
}

func TestLineCountAnalyzerLongLine(t *testing.T) {
	content := strings.Repeat("a", bufio.MaxScanTokenSize+1)
	got := mustAnalyze(t, analyzer.LineCountAnalyzer{}, content)
	if got.Lines != 1 {
		t.Errorf("Lines = %d, want 1", got.Lines)
	}
}

func TestDocumentWordsShared(t *testing.T) {
	d := doc("hello world hello")
	first := d.Words()
	second := d.Words()
	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("Words() = %v / %v", first, second)
	}
}

func TestList(t *testing.T) {
	if len(analyzer.List(0)) != 3 {
		t.Fatalf("without top-words: want 3")
	}
	if n := analyzer.List(5); n[len(n)-1].Name() != "top-words" {
		t.Fatal("with top-words last analyzer should be top-words")
	}
}
