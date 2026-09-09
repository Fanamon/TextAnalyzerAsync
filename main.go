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

	files, err := collectFiles(cfg.Path, cfg.Ext)
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

	for stats := range results {
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

	flag.Parse()
	if *path == "" {
		return ParseResult{}, fmt.Errorf("нужен флаг -path")
	}

	return ParseResult{Path: *path, Ext: *ext, Workers: *workers, TopWords: *topWords}, nil
}

func collectFiles(root, ext string) ([]string, error) {
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

func printTopWords(wordCounts map[string]int, count int) {
	if count <= 0 {
		return
	}

	sortedWords := make([]WordCount, 0, len(wordCounts))
	for word, cnt := range wordCounts {
		sortedWords = append(sortedWords, WordCount{Word: word, Count: cnt})
	}

	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].Count > sortedWords[j].Count
	})

	for i := 0; i < count && i < len(sortedWords); i++ {
		fmt.Printf("%s: %d\n", sortedWords[i].Word, sortedWords[i].Count)
	}
}
