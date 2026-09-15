package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"textanalyzerasync/internal/config"
)

func startProfiles(cfg config.Config) (func(), error) {
	var stopCPU func()

	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			return nil, fmt.Errorf("cpu profile: %w", err)
		}

		if err := pprof.StartCPUProfile(f); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("cpu profile: %w", err)
		}

		stopCPU = func() {
			pprof.StopCPUProfile()
			_ = f.Close()
		}
	}

	return func() {
		if stopCPU != nil {
			stopCPU()
		}

		if cfg.MemProfile == "" {
			return
		}

		f, err := os.Create(cfg.MemProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mem profile: %v\n", err)
			return
		}

		defer f.Close()
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "mem profile: %v\n", err)
		}
	}, nil
}
