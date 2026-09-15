package report

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"textanalyzerasync/internal/analyzer"
)

type Counted[T cmp.Ordered] struct {
	Value T
	Count int
}

func TopN[T cmp.Ordered](freq map[T]int, n int) []Counted[T] {
	if n <= 0 {
		return nil
	}

	out := make([]Counted[T], 0, len(freq))
	for value, count := range freq {
		out = append(out, Counted[T]{Value: value, Count: count})
	}

	slices.SortFunc(out, func(a, b Counted[T]) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return cmp.Compare(a.Value, b.Value)
	})

	if n > len(out) {
		n = len(out)
	}
	return out[:n]
}

func KeepResult(stats analyzer.Metrics, minWords int) bool {
	return stats.Words >= minWords
}

func PrintFileStats(w io.Writer, stats analyzer.Metrics) {
	fmt.Fprintf(w, "Файл: %s\n", stats.Path)
	fmt.Fprintf(w, "Количество слов: %d\n", stats.Words)
	fmt.Fprintf(w, "Количество строк: %d\n", stats.Lines)
	fmt.Fprintf(w, "Количество символов: %d\n", stats.Chars)
	fmt.Fprintln(w)
}

func PrintTopWords(w io.Writer, wordCounts map[string]int, n int) {
	for _, item := range TopN(wordCounts, n) {
		fmt.Fprintf(w, "%s: %d\n", item.Value, item.Count)
	}
}
