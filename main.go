package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type ParseResult struct {
	Path     string
	Ext      string
	Workers  int
	TopWords int
	MinSize  int64
	MaxSize  int64
	MinWords int
}

type FileStats struct {
	Path           string
	Words          int
	Lines          int
	Chars          int
	WordsFrequency map[string]int
}

type FileResults struct {
	TopWords map[string]int
	Mutex    sync.Mutex
}

type WordCount struct {
	Word  string
	Count int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg, err := parseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	files, err := collectFiles(cfg.Path, cfg.Ext, cfg.MinSize, cfg.MaxSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	filePaths := make(chan string, len(files))
	for _, path := range files {
		filePaths <- path
	}

	close(filePaths)
	var waitGroup sync.WaitGroup
	results := make(chan FileStats)
	done := make(chan struct{})
	fileResults := FileResults{TopWords: make(map[string]int)}

	for i := 0; i < cfg.Workers; i++ {
		waitGroup.Add(1)
		go fileProcessor(filePaths, results, &waitGroup, &fileResults, ctx)
	}

	go func() {
		waitGroup.Wait()
		close(results)
		close(done)
	}()

	filtered := make(chan FileStats)
	go func() {
		defer close(filtered)

		for stats := range results {
			if keepResult(stats, cfg.MinWords) {
				filtered <- stats
			}
		}
	}()

	for stats := range filtered {
		printFileStats(stats)
	}

	<-done

	printTopWords(fileResults.TopWords, cfg.TopWords)
}

func parseFlags() (ParseResult, error) {
	path := flag.String("path", "", "путь к директории с текстовыми файлами или к одному файлу")
	ext := flag.String("ext", ".txt", "расширение файлов для анализа")
	workers := flag.Int("workers", runtime.NumCPU(), "количество рабочих горутин")
	topWords := flag.Int("top-words", 0, "N самых частых слов во всех файлах")
	minSize := flag.Int64("min-size", 0, "минимальный размер файла для анализа")
	maxSize := flag.Int64("max-size", 0, "максимальный размер файла для анализа")
	minWords := flag.Int("min-words", 0, "показывать только файлы с не меньшим числом слов")

	flag.Parse()
	if *path == "" {
		return ParseResult{}, fmt.Errorf("нужен флаг -path")
	}

	return ParseResult{
		Path: *path, Ext: *ext, Workers: *workers, TopWords: *topWords, MinSize: *minSize, MaxSize: *maxSize,
		MinWords: *minWords,
	}, nil
}

func collectFiles(root, ext string, minSize, maxSize int64) ([]string, error) {
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			return nil
		}

		if strings.EqualFold(filepath.Ext(path), ext) {
			info, err := dirEntry.Info()
			if err != nil {
				return err
			}

			size := info.Size()
			if size < minSize || (maxSize > 0 && size > maxSize) {
				return nil
			}

			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func processFile(path string) (FileStats, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileStats{}, err
	}

	content := string(data)
	analyzers := []Analyzer{
		WordCountAnalyzer{},
		CharCountAnalyzer{},
		LineCountAnalyzer{},
		MostFrequentWordsAnalyzer{},
	}

	parts := make([]AnalysisResult, len(analyzers))
	var inner sync.WaitGroup
	for i, a := range analyzers {
		inner.Add(1)

		go func(index int, analyzer Analyzer) {
			defer inner.Done()

			parts[index] = analyzer.Analyze(content)
		}(i, a)
	}

	inner.Wait()

	return mergeFileAnalysis(path, parts)
}

func mergeFileAnalysis(path string, analysisResults []AnalysisResult) (FileStats, error) {
	stats := FileStats{
		Path:           path,
		WordsFrequency: make(map[string]int),
	}

	for _, analysisResult := range analysisResults {
		if analysisResult.Err != nil {
			return FileStats{}, fmt.Errorf("%s: %w", analysisResult.Name, analysisResult.Err)
		}

		stats.Words += analysisResult.Words
		stats.Lines += analysisResult.Lines
		stats.Chars += analysisResult.Chars
		for partWord, partWordFreq := range analysisResult.WordFreq {
			stats.WordsFrequency[partWord] += partWordFreq
		}
	}

	return stats, nil
}

func printFileStats(stats FileStats) {
	fmt.Printf("Файл: %s\n", stats.Path)
	fmt.Printf("Количество слов: %d\n", stats.Words)
	fmt.Printf("Количество строк: %d\n", stats.Lines)
	fmt.Printf("Количество символов: %d\n", stats.Chars)
	fmt.Println()
}

func fileProcessor(filePaths <-chan string, results chan<- FileStats, wg *sync.WaitGroup,
	fileResults *FileResults, ctx context.Context,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case path, ok := <-filePaths:
			if !ok {
				return
			}

			stats, err := processFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ошибка при обработке файла %s: %v\n", path, err)
				continue
			}

			fileResults.Mutex.Lock()
			for word, count := range stats.WordsFrequency {
				fileResults.TopWords[word] += count
			}
			fileResults.Mutex.Unlock()

			results <- stats
		}
	}
}

func keepResult(stats FileStats, minWords int) bool {
	return stats.Words >= minWords
}

func rankTopWords(wordCounts map[string]int, n int) []WordCount {
	if n <= 0 {
		return nil
	}

	sortedWords := make([]WordCount, 0, len(wordCounts))
	for word, cnt := range wordCounts {
		sortedWords = append(sortedWords, WordCount{Word: word, Count: cnt})
	}

	sort.Slice(sortedWords, func(i, j int) bool {
		if sortedWords[i].Count == sortedWords[j].Count {
			return sortedWords[i].Word < sortedWords[j].Word
		}
		return sortedWords[i].Count > sortedWords[j].Count
	})

	if n > len(sortedWords) {
		n = len(sortedWords)
	}
	return sortedWords[:n]
}

func printTopWords(wordCounts map[string]int, count int) {
	for _, item := range rankTopWords(wordCounts, count) {
		fmt.Printf("%s: %d\n", item.Word, item.Count)
	}
}
