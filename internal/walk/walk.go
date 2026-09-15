package walk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// Options задаёт фильтр обхода. MaxSize == 0 означает «без верхней границы».
type Options struct {
	Ext     string
	MinSize int64
	MaxSize int64
}

// Walk рекурсивно обходит root и вызывает emit для каждого подходящего файла.
// emit вызывается на лету, без предварительного сбора всего списка в память.
func Walk(ctx context.Context, root string, opts Options, emit func(path string) error) error {
	ext := opts.Ext
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	return filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if dirEntry.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ext) {
			return nil
		}

		info, err := dirEntry.Info()
		if err != nil {
			return err
		}

		size := info.Size()
		if size < opts.MinSize || (opts.MaxSize > 0 && size > opts.MaxSize) {
			return nil
		}

		return emit(path)
	})
}

// Collect — удобная обёртка для тестов: возвращает слайс путей.
func Collect(ctx context.Context, root string, opts Options) ([]string, error) {
	var files []string
	err := Walk(ctx, root, opts, func(path string) error {
		files = append(files, path)
		return nil
	})
	return files, err
}
