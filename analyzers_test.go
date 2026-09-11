package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestAnalyzerNames(t *testing.T) {
	cases := []struct {
		analyzer Analyzer
		want     string
	}{
		{WordCountAnalyzer{}, "words"},
		{LineCountAnalyzer{}, "lines"},
		{CharCountAnalyzer{}, "chars"},
		{MostFrequentWordsAnalyzer{}, "top-words"},
	}

	for _, tc := range cases {
		if got := tc.analyzer.Name(); got != tc.want {
			t.Errorf("%T.Name() = %q, want %q", tc.analyzer, got, tc.want)
		}
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
			got := WordCountAnalyzer{}.Analyze(tc.content)
			if got.Name != "words" {
				t.Errorf("Name = %q, want words", got.Name)
			}
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
			got := LineCountAnalyzer{}.Analyze(tc.content)
			if got.Err != nil {
				t.Fatalf("unexpected err: %v", got.Err)
			}
			if got.Name != "lines" {
				t.Errorf("Name = %q, want lines", got.Name)
			}
			if got.Lines != tc.want {
				t.Errorf("Lines = %d, want %d", got.Lines, tc.want)
			}
		})
	}
}

func TestCharCountAnalyzer(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{name: "empty", content: "", want: 0},
		{name: "ascii", content: "hello", want: 5},
		{name: "cyrillic runes", content: "привет", want: 6},
		{name: "space and newline", content: "a b\n", want: 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CharCountAnalyzer{}.Analyze(tc.content)
			if got.Name != "chars" {
				t.Errorf("Name = %q, want chars", got.Name)
			}
			if got.Chars != tc.want {
				t.Errorf("Chars = %d, want %d", got.Chars, tc.want)
			}
		})
	}
}

func TestMostFrequentWordsAnalyzer(t *testing.T) {
	got := MostFrequentWordsAnalyzer{}.Analyze("The cat and the dog")
	if got.Name != "top-words" {
		t.Errorf("Name = %q, want top-words", got.Name)
	}
	if got.WordFreq["the"] != 2 {
		t.Errorf("the = %d, want 2", got.WordFreq["the"])
	}
	if got.WordFreq["cat"] != 1 {
		t.Errorf("cat = %d, want 1", got.WordFreq["cat"])
	}
	if _, ok := got.WordFreq["The"]; ok {
		t.Fatal("expected lowercase keys only")
	}

	empty := MostFrequentWordsAnalyzer{}.Analyze("")
	if len(empty.WordFreq) != 0 {
		t.Errorf("empty content freq = %v, want empty map", empty.WordFreq)
	}
}

func TestLineCountAnalyzerLongLine(t *testing.T) {
	content := strings.Repeat("a", bufio.MaxScanTokenSize+1)
	got := LineCountAnalyzer{}.Analyze(content)
	if got.Err == nil {
		t.Fatal("expected error for a line longer than the scanner limit")
	}
	if got.Name != "lines" {
		t.Errorf("Name = %q, want lines", got.Name)
	}
}
