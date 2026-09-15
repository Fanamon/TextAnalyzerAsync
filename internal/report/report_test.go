package report_test

import (
	"bytes"
	"strings"
	"testing"

	"textanalyzerasync/internal/analyzer"
	"textanalyzerasync/internal/report"
)

func TestTopN(t *testing.T) {
	freq := map[string]int{"b": 1, "a": 3, "c": 3}

	if report.TopN(freq, 0) != nil {
		t.Fatal("n=0 should return nil")
	}

	got := report.TopN(freq, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Value != "a" || got[0].Count != 3 {
		t.Fatalf("first = %+v, want a:3", got[0])
	}
	if got[1].Value != "c" || got[1].Count != 3 {
		t.Fatalf("second = %+v, want c:3", got[1])
	}
}

func TestKeepResult(t *testing.T) {
	if !report.KeepResult(analyzer.Metrics{Words: 10}, 10) {
		t.Fatal("want keep")
	}
	if report.KeepResult(analyzer.Metrics{Words: 9}, 10) {
		t.Fatal("want drop")
	}
}

func TestPrintFileStats(t *testing.T) {
	var buf bytes.Buffer
	report.PrintFileStats(&buf, analyzer.Metrics{Path: "a.txt", Words: 2, Lines: 1, Chars: 4})
	if !strings.Contains(buf.String(), "Файл: a.txt") {
		t.Fatalf("unexpected %q", buf.String())
	}
}
