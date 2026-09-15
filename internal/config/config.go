package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

type Config struct {
	Path              string
	Ext               string
	Workers           int
	TopWords          int
	MinSize           int64
	MaxSize           int64 // 0 — без верхней границы
	MinWords          int
	ParallelAnalyzers bool
	CPUProfile        string
	MemProfile        string
}

func Parse(args []string) (Config, error) {
	fs := flag.NewFlagSet("text-analyzer", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	path := fs.String("path", "", "путь к директории с текстовыми файлами или к одному файлу")
	ext := fs.String("ext", ".txt", "расширение файлов для анализа")
	workers := fs.Int("workers", runtime.NumCPU(), "количество рабочих горутин")
	topWords := fs.Int("top-words", 0, "N самых частых слов во всех файлах")
	minSize := fs.Int64("min-size", 0, "минимальный размер файла для анализа, байты")
	maxSize := fs.Int64("max-size", 0, "максимальный размер файла в байтах, 0 — без ограничения")
	minWords := fs.Int("min-words", 0, "показывать только файлы с не меньшим числом слов")
	cpuProfile := fs.String("cpuprofile", "", "записать CPU-профиль pprof в файл")
	memProfile := fs.String("memprofile", "", "записать профиль памяти pprof в файл")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if *path == "" {
		return Config{}, fmt.Errorf("нужен флаг -path")
	}
	if *ext == "" {
		return Config{}, fmt.Errorf("флаг -ext не должен быть пустым")
	}
	if *workers < 1 {
		return Config{}, fmt.Errorf("флаг -workers должен быть >= 1")
	}
	if *minSize < 0 || *maxSize < 0 {
		return Config{}, fmt.Errorf("флаги -min-size и -max-size не должны быть отрицательными")
	}
	if *maxSize > 0 && *minSize > *maxSize {
		return Config{}, fmt.Errorf("флаг -min-size не должен быть больше -max-size")
	}

	return Config{
		Path:              *path,
		Ext:               *ext,
		Workers:           *workers,
		TopWords:          *topWords,
		MinSize:           *minSize,
		MaxSize:           *maxSize,
		MinWords:          *minWords,
		ParallelAnalyzers: true,
		CPUProfile:        *cpuProfile,
		MemProfile:        *memProfile,
	}, nil
}
