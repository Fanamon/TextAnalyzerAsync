package main

import (
	"context"
	"os"
	"os/signal"
	"textanalyzerasync/internal/config"
	"textanalyzerasync/internal/pipeline"
)

func run() error {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	stopProfile, err := startProfiles(cfg)
	if err != nil {
		return err
	}

	defer stopProfile()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return pipeline.Run(ctx, cfg, os.Stdout, os.Stderr)
}
