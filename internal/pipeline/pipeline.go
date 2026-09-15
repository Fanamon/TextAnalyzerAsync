package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"textanalyzerasync/internal/analyzer"
	"textanalyzerasync/internal/config"
	"textanalyzerasync/internal/report"
	"textanalyzerasync/internal/walk"

	"golang.org/x/sync/errgroup"
)

type WordFrequencyCollector struct {
	mu       sync.Mutex
	topWords map[string]int
}

func NewWordFrequencyCollector() *WordFrequencyCollector {
	return &WordFrequencyCollector{topWords: make(map[string]int)}
}

func (c *WordFrequencyCollector) Add(frequency map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for word, count := range frequency {
		c.topWords[word] += count
	}
}

func (c *WordFrequencyCollector) Snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make(map[string]int, len(c.topWords))
	for word, count := range c.topWords {
		out[word] = count
	}
	return out
}

type Pipeline struct {
	cfg       config.Config
	analyzers []analyzer.Analyzer
	words     *WordFrequencyCollector
	failed    *atomic.Int64
	errOut    io.Writer
}

func Run(ctx context.Context, cfg config.Config, out, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	p := &Pipeline{
		cfg:       cfg,
		analyzers: analyzer.List(cfg.TopWords),
		words:     NewWordFrequencyCollector(),
		failed:    new(atomic.Int64),
		errOut:    errOut,
	}

	filePaths := make(chan string, cfg.Workers)
	results := make(chan analyzer.Metrics)
	filtered := make(chan analyzer.Metrics)
	done := make(chan struct{})

	runCtx := ctx
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer close(filePaths)
		return walk.Walk(ctx, cfg.Path, walk.Options{
			Ext:     cfg.Ext,
			MinSize: cfg.MinSize,
			MaxSize: cfg.MaxSize,
		}, func(path string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case filePaths <- path:
				return nil
			}
		})
	})

	g.Go(func() error {
		defer close(done)
		defer close(results)

		var workers errgroup.Group
		for i := 0; i < cfg.Workers; i++ {
			workers.Go(func() error {
				p.worker(ctx, filePaths, results)
				return nil
			})
		}
		return workers.Wait()
	})

	g.Go(func() error {
		p.filter(ctx, results, filtered)
		return nil
	})

	for stats := range filtered {
		report.PrintFileStats(out, stats)
	}

	<-done

	if err := g.Wait(); err != nil && !errors.Is(err, runCtx.Err()) && !errors.Is(err, context.Canceled) {
		return err
	}

	report.PrintTopWords(out, p.words.Snapshot(), cfg.TopWords)

	if err := runCtx.Err(); err != nil {
		return fmt.Errorf("обработка прервана: %w", err)
	}
	if n := p.failed.Load(); n > 0 {
		return fmt.Errorf("не удалось обработать файлов: %d", n)
	}

	return nil
}

func (p *Pipeline) filter(ctx context.Context, in <-chan analyzer.Metrics, out chan<- analyzer.Metrics) {
	defer close(out)

	for stats := range in {
		if !report.KeepResult(stats, p.cfg.MinWords) {
			continue
		}

		select {
		case out <- stats:
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pipeline) worker(ctx context.Context, filePaths <-chan string, results chan<- analyzer.Metrics) {
	for {
		if ctx.Err() != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case path, ok := <-filePaths:
			if !ok {
				return
			}

			stats, err := ProcessFile(ctx, path, p.analyzers, p.cfg.ParallelAnalyzers)
			if err != nil {
				fmt.Fprintf(p.errOut, "ошибка при обработке файла %s: %v\n", path, err)
				p.failed.Add(1)
				if stats.Path == "" {
					continue
				}
			}

			p.words.Add(stats.WordFreq)

			select {
			case results <- stats:
			case <-ctx.Done():
				return
			}
		}
	}
}

func ProcessFile(ctx context.Context, path string, analyzers []analyzer.Analyzer, parallel bool) (analyzer.Metrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return analyzer.Metrics{}, err
	}

	doc := &analyzer.Document{Path: path, Content: string(data)}
	parts, err := runAnalyzers(ctx, doc, analyzers, parallel)
	return mergeMetrics(path, parts), err
}

func runAnalyzers(ctx context.Context, doc *analyzer.Document, analyzers []analyzer.Analyzer, parallel bool) ([]analyzer.Result, error) {
	parts := make([]analyzer.Result, len(analyzers))
	if !parallel {
		var errs []error
		for i, a := range analyzers {
			result, err := a.Analyze(ctx, doc)
			parts[i] = result
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", a.Name(), err))
			}
		}
		return parts, errors.Join(errs...)
	}

	g := new(errgroup.Group)
	for i, a := range analyzers {
		g.Go(func() error {
			result, err := a.Analyze(ctx, doc)
			parts[i] = result
			if err != nil {
				return fmt.Errorf("%s: %w", a.Name(), err)
			}
			return nil
		})
	}

	return parts, g.Wait()
}

func mergeMetrics(path string, parts []analyzer.Result) analyzer.Metrics {
	stats := analyzer.Metrics{
		Path:     path,
		WordFreq: make(map[string]int),
	}
	for _, part := range parts {
		if part.Merge != nil {
			part.Merge(&stats)
		}
	}
	return stats
}
