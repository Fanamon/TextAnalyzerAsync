package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
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

func writeSizedFile(t *testing.T, dir, name string, size int) string {
	t.Helper()
	return writeFile(t, dir, name, string(bytes.Repeat([]byte("a"), size)))
}

func TestCollectFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one")
	writeFile(t, dir, "b.TXT", "two")
	writeFile(t, dir, "skip.md", "nope")
	writeFile(t, dir, filepath.Join("nested", "c.txt"), "three")

	files, err := collectFiles(dir, ".txt", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("got %d files, want 3: %v", len(files), files)
	}
}

func TestCollectFilesExtWithoutDot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one")

	files, err := collectFiles(dir, "txt", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
}

func TestCollectFilesMinSize(t *testing.T) {
	dir := t.TempDir()
	writeSizedFile(t, dir, "small.txt", 10)
	writeSizedFile(t, dir, "big.txt", 1000)

	files, err := collectFiles(dir, ".txt", 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if !strings.HasSuffix(files[0], "big.txt") {
		t.Fatalf("unexpected file %q", files[0])
	}
}

func TestCollectFilesMaxSize(t *testing.T) {
	dir := t.TempDir()
	writeSizedFile(t, dir, "small.txt", 10)
	writeSizedFile(t, dir, "big.txt", 1000)

	files, err := collectFiles(dir, ".txt", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if !strings.HasSuffix(files[0], "small.txt") {
		t.Fatalf("unexpected file %q", files[0])
	}
}

func TestCollectFilesSingleFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "only.txt", "hello")

	files, err := collectFiles(path, ".txt", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != path {
		t.Fatalf("got %v, want [%s]", files, path)
	}
}

func TestCollectFilesMissingPath(t *testing.T) {
	_, err := collectFiles(filepath.Join(t.TempDir(), "nope"), ".txt", 0, 0)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestCollectFilesEmptyDir(t *testing.T) {
	files, err := collectFiles(t.TempDir(), ".txt", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("got %v, want empty", files)
	}
}

func TestCollectFilesTestdata(t *testing.T) {
	files, err := collectFiles("testdata", ".txt", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("testdata: got %d files, want 2: %v", len(files), files)
	}
}

func TestProcessFile(t *testing.T) {
	path := writeFile(t, t.TempDir(), "sample.txt", "hello world\nпривет")

	got, err := processFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != path {
		t.Errorf("Path = %q, want %q", got.Path, path)
	}
	if got.Words != 3 {
		t.Errorf("Words = %d, want 3", got.Words)
	}
	if got.Lines != 2 {
		t.Errorf("Lines = %d, want 2", got.Lines)
	}
	if got.Chars != 18 {
		t.Errorf("Chars = %d, want 18", got.Chars)
	}
	if got.WordsFrequency["hello"] != 1 || got.WordsFrequency["привет"] != 1 {
		t.Errorf("freq = %v", got.WordsFrequency)
	}
}

func TestProcessFileMissing(t *testing.T) {
	_, err := processFile(filepath.Join(t.TempDir(), "missing.txt"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMergeFileAnalysis(t *testing.T) {
	got, err := mergeFileAnalysis("a.txt", []AnalysisResult{
		{Name: "words", Words: 4},
		{Name: "lines", Lines: 2},
		{Name: "chars", Chars: 10},
		{Name: "top-words", WordFreq: map[string]int{"go": 2, "ok": 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "a.txt" || got.Words != 4 || got.Lines != 2 || got.Chars != 10 {
		t.Fatalf("stats = %+v", got)
	}
	if got.WordsFrequency["go"] != 2 || got.WordsFrequency["ok"] != 1 {
		t.Errorf("freq = %v", got.WordsFrequency)
	}
}

func TestMergeFileAnalysisError(t *testing.T) {
	_, err := mergeFileAnalysis("a.txt", []AnalysisResult{
		{Name: "words", Words: 1},
		{Name: "lines", Err: errors.New("scanner failed")},
	})
	if err == nil {
		t.Fatal("expected merge error")
	}
	if !strings.Contains(err.Error(), "lines") {
		t.Fatalf("error should mention analyzer name, got %v", err)
	}
}

func TestKeepResult(t *testing.T) {
	cases := []struct {
		words    int
		minWords int
		want     bool
	}{
		{words: 0, minWords: 0, want: true},
		{words: 10, minWords: 10, want: true},
		{words: 9, minWords: 10, want: false},
		{words: 1001, minWords: 1000, want: true},
	}

	for _, tc := range cases {
		got := keepResult(FileStats{Words: tc.words}, tc.minWords)
		if got != tc.want {
			t.Errorf("keepResult(words=%d, min=%d) = %v, want %v", tc.words, tc.minWords, got, tc.want)
		}
	}
}

func TestRankTopWords(t *testing.T) {
	freq := map[string]int{"b": 1, "a": 3, "c": 3}

	if rankTopWords(freq, 0) != nil {
		t.Fatal("n=0 should return nil")
	}

	got := rankTopWords(freq, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Word != "a" || got[0].Count != 3 {
		t.Fatalf("first = %+v, want a:3", got[0])
	}
	if got[1].Word != "c" || got[1].Count != 3 {
		t.Fatalf("second = %+v, want c:3 (tie broken by word)", got[1])
	}

	all := rankTopWords(freq, 100)
	if len(all) != 3 {
		t.Fatalf("n larger than map: got %d, want 3", len(all))
	}
}

func TestFileProcessor(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.txt", "foo bar foo")

	filePaths := make(chan string, 1)
	filePaths <- path
	close(filePaths)

	results := make(chan FileStats)
	fileResults := FileResults{TopWords: make(map[string]int)}
	var wg sync.WaitGroup
	wg.Add(1)
	go fileProcessor(filePaths, results, &wg, &fileResults, context.Background())
	go func() {
		wg.Wait()
		close(results)
	}()

	var got []FileStats
	for stats := range results {
		got = append(got, stats)
	}
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1", len(got))
	}
	if got[0].Words != 3 {
		t.Errorf("Words = %d, want 3", got[0].Words)
	}
	if fileResults.TopWords["foo"] != 2 || fileResults.TopWords["bar"] != 1 {
		t.Errorf("top words = %v", fileResults.TopWords)
	}
}

func TestFileProcessorCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	filePaths := make(chan string)
	results := make(chan FileStats)
	fileResults := FileResults{TopWords: make(map[string]int)}
	var wg sync.WaitGroup
	wg.Add(1)
	go fileProcessor(filePaths, results, &wg, &fileResults, ctx)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("fileProcessor did not stop after cancel")
	}
}

func benchFiles(b *testing.B) []string {
	b.Helper()
	dir := b.TempDir()
	content := strings.Repeat("word ", 200) + "\n" + strings.Repeat("other ", 200)
	files := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		path := filepath.Join(dir, fmt.Sprintf("%02d.txt", i))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
		files = append(files, path)
	}
	return files
}

func processSequential(files []string) error {
	for _, path := range files {
		if _, err := processFile(path); err != nil {
			return err
		}
	}
	return nil
}

func processParallel(files []string, workers int) error {
	if workers < 1 {
		workers = 1
	}

	filePaths := make(chan string, len(files))
	for _, path := range files {
		filePaths <- path
	}
	close(filePaths)

	results := make(chan FileStats)
	fileResults := FileResults{TopWords: make(map[string]int)}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go fileProcessor(filePaths, results, &wg, &fileResults, context.Background())
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	for range results {
	}
	return nil
}

func BenchmarkSequential(b *testing.B) {
	files := benchFiles(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := processSequential(files); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallel(b *testing.B) {
	files := benchFiles(b)
	workers := runtime.NumCPU()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := processParallel(files, workers); err != nil {
			b.Fatal(err)
		}
	}
}
