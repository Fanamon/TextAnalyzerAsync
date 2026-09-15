package pipeline

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"textanalyzerasync/internal/analyzer"
	"textanalyzerasync/internal/config"
)

type stubErrAnalyzer struct{}

func (stubErrAnalyzer) Name() string { return "lines" }

func (stubErrAnalyzer) Analyze(context.Context, *analyzer.Document) (analyzer.Result, error) {
	return analyzer.Result{Name: "lines"}, errors.New("scanner failed")
}

func TestRunAnalyzersKeepsPartialResults(t *testing.T) {
	doc := &analyzer.Document{Content: "hello"}
	parts, err := runAnalyzers(context.Background(), doc, []analyzer.Analyzer{
		analyzer.WordCountAnalyzer{},
		stubErrAnalyzer{},
	}, true)
	if err == nil {
		t.Fatal("expected analyzer error")
	}
	if !strings.Contains(err.Error(), "lines") {
		t.Fatalf("error %v", err)
	}

	got := mergeMetrics("a.txt", parts)
	if got.Path != "a.txt" || got.Words != 1 {
		t.Fatalf("partial stats should be kept, got %+v", got)
	}
}

func TestFilterMinWords(t *testing.T) {
	p := &Pipeline{cfg: config.Config{MinWords: 2}}
	in := make(chan analyzer.Metrics, 2)
	out := make(chan analyzer.Metrics, 2)

	go p.filter(context.Background(), in, out)

	in <- analyzer.Metrics{Path: "keep.txt", Words: 3}
	in <- analyzer.Metrics{Path: "skip.txt", Words: 1}
	close(in)

	var got []string
	for stats := range out {
		got = append(got, stats.Path)
	}
	if len(got) != 1 || got[0] != "keep.txt" {
		t.Fatalf("filtered = %v, want [keep.txt]", got)
	}
}

func TestWorkerDoesNotTakeQueuedJobsAfterCancel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	jobs := make(chan string, 8)
	for i := 0; i < 8; i++ {
		jobs <- path
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &Pipeline{
		cfg:       config.Config{ParallelAnalyzers: false},
		analyzers: analyzer.List(0),
		words:     NewWordFrequencyCollector(),
		failed:    new(atomic.Int64),
		errOut:    io.Discard,
	}
	results := make(chan analyzer.Metrics, 8)

	done := make(chan struct{})
	go func() {
		p.worker(ctx, jobs, results)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not exit after cancel")
	}

	if n := len(results); n != 0 {
		t.Fatalf("processed %d queued jobs after cancel, want 0", n)
	}
}
