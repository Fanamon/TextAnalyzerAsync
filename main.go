package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type ParseResult struct {
	Path    string
	Ext     string
	Workers int
}

type FileStats struct {
	Path  string
	Words int
	Lines int
	Chars int
}

func main() {
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

	for i := 0; i < cfg.Workers; i++ {
		waitGroup.Add(1)
		go fileProcessor(filePaths, results, &waitGroup)
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
}

func parseFlags() (ParseResult, error) {
	path := flag.String("path", "", "путь к директории с текстовыми файлами или к одному файлу")
	ext := flag.String("ext", ".txt", "расширение файлов для анализа")
	workers := flag.Int("workers", runtime.NumCPU(), "количество рабочих горутин")

	flag.Parse()
	if *path == "" {
		return ParseResult{}, fmt.Errorf("нужен флаг -path")
	}

	return ParseResult{Path: *path, Ext: *ext, Workers: *workers}, nil
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
	words := len(strings.Fields(content))

	scanner := bufio.NewScanner(strings.NewReader(content))
	lines := 0
	for scanner.Scan() {
		lines++
	}

	if err := scanner.Err(); err != nil {
		return FileStats{}, err
	}

	chars := 0
	for range content {
		chars++
	}

	return FileStats{Path: path, Lines: lines, Words: words, Chars: chars}, nil
}

func printFileStats(stats FileStats) {
	fmt.Printf("Файл: %s\n", stats.Path)
	fmt.Printf("Количество слов: %d\n", stats.Words)
	fmt.Printf("Количество строк: %d\n", stats.Lines)
	fmt.Printf("Количество символов: %d\n", stats.Chars)
	fmt.Println()
}

func fileProcessor(filePaths <-chan string, results chan<- FileStats, wg *sync.WaitGroup) {
	defer wg.Done()

	for path := range filePaths {
		stats, err := processFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка при обработке файла %s: %v\n", path, err)
			continue
		}

		results <- stats
	}
}
