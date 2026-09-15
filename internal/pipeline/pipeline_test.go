package pipeline_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"textanalyzerasync/internal/analyzer"
	"textanalyzerasync/internal/config"
	"textanalyzerasync/internal/pipeline"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProcessFile(t *testing.T) {
	path := writeFile(t, t.TempDir(), "sample.txt", "hello world\nпривет")

	got, err := pipeline.ProcessFile(context.Background(), path, analyzer.List(1), true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Words != 3 || got.Lines != 2 || got.Chars != 18 {
		t.Fatalf("stats = %+v", got)
	}
	if got.WordFreq["hello"] != 1 || got.WordFreq["привет"] != 1 {
		t.Errorf("freq = %v", got.WordFreq)
	}
}

func TestProcessFileMissing(t *testing.T) {
	_, err := pipeline.ProcessFile(context.Background(), filepath.Join(t.TempDir(), "missing.txt"), analyzer.List(1), true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWordFrequencyCollector(t *testing.T) {
	got := pipeline.NewWordFrequencyCollector()
	got.Add(map[string]int{"a": 2})
	got.Add(map[string]int{"a": 3, "b": 1})
	snap := got.Snapshot()
	if snap["a"] != 5 || snap["b"] != 1 {
		t.Fatalf("snapshot = %v", snap)
	}
	snap["a"] = 0
	if got.Snapshot()["a"] != 5 {
		t.Fatal("Snapshot should copy")
	}
}

func TestRunPipeline(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "keep.txt", "hello world hello")
	writeFile(t, dir, "skip.txt", "x")

	cfg := config.Config{
		Path:              dir,
		Ext:               ".txt",
		Workers:           2,
		TopWords:          1,
		MinWords:          2,
		ParallelAnalyzers: true,
	}
	var buf bytes.Buffer
	if err := pipeline.Run(context.Background(), cfg, &buf, io.Discard); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "keep.txt") {
		t.Fatalf("expected keep.txt: %q", out)
	}
	if strings.Contains(out, "skip.txt") {
		t.Fatalf("skip.txt should be filtered: %q", out)
	}
	if !strings.Contains(out, "hello: 2") {
		t.Fatalf("expected top word hello: 2, got %q", out)
	}
}

func TestRunPipelineMissingPath(t *testing.T) {
	cfg := config.Config{
		Path:    filepath.Join(t.TempDir(), "nope"),
		Ext:     ".txt",
		Workers: 1,
	}
	if err := pipeline.Run(context.Background(), cfg, io.Discard, io.Discard); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := config.Config{
		Path:    t.TempDir(),
		Ext:     ".txt",
		Workers: 1,
	}
	done := make(chan error, 1)
	go func() {
		done <- pipeline.Run(ctx, cfg, io.Discard, io.Discard)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancel error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}

func TestProcessFileTestdata(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("testdata not found")
	}

	cfg := config.Config{
		Path:              dir,
		Ext:               ".txt",
		Workers:           2,
		TopWords:          3,
		ParallelAnalyzers: true,
	}
	var buf bytes.Buffer
	if err := pipeline.Run(context.Background(), cfg, &buf, io.Discard); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output for testdata")
	}
}

func benchDir(b *testing.B) string {
	b.Helper()
	dir := b.TempDir()
	content := strings.Repeat("word ", 200) + "\n" + strings.Repeat("other ", 200)
	for i := 0; i < 40; i++ {
		path := filepath.Join(dir, fmt.Sprintf("%02d.txt", i))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	return dir
}

func BenchmarkSequential(b *testing.B) {
	dir := benchDir(b)
	cfg := config.Config{Path: dir, Ext: ".txt", Workers: 1, ParallelAnalyzers: false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pipeline.Run(context.Background(), cfg, io.Discard, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallel(b *testing.B) {
	dir := benchDir(b)
	cfg := config.Config{Path: dir, Ext: ".txt", Workers: runtime.NumCPU(), ParallelAnalyzers: true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pipeline.Run(context.Background(), cfg, io.Discard, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallelSequentialAnalyzers(b *testing.B) {
	dir := benchDir(b)
	cfg := config.Config{Path: dir, Ext: ".txt", Workers: runtime.NumCPU(), ParallelAnalyzers: false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pipeline.Run(context.Background(), cfg, io.Discard, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}
