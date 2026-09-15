package config_test

import (
	"strings"
	"testing"

	"textanalyzerasync/internal/config"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
		check   func(*testing.T, config.Config)
	}{
		{
			name:    "workers zero",
			args:    []string{"-path", ".", "-workers", "0"},
			wantErr: "-workers",
		},
		{
			name:    "workers negative",
			args:    []string{"-path", ".", "-workers", "-1"},
			wantErr: "-workers",
		},
		{
			name:    "min-size greater than max-size",
			args:    []string{"-path", ".", "-min-size", "10", "-max-size", "1"},
			wantErr: "-min-size",
		},
		{
			name:    "negative min-size",
			args:    []string{"-path", ".", "-min-size", "-1"},
			wantErr: "-min-size",
		},
		{
			name:    "negative max-size",
			args:    []string{"-path", ".", "-max-size", "-5"},
			wantErr: "-max-size",
		},
		{
			name:    "empty ext",
			args:    []string{"-path", ".", "-ext", ""},
			wantErr: "-ext",
		},
		{
			name: "defaults and cpu profile",
			args: []string{"-path", "docs", "-cpuprofile", "cpu.out"},
			check: func(t *testing.T, cfg config.Config) {
				t.Helper()
				if cfg.Path != "docs" || cfg.CPUProfile != "cpu.out" {
					t.Fatalf("unexpected config: %+v", cfg)
				}
				if cfg.Workers < 1 || !cfg.ParallelAnalyzers {
					t.Fatalf("unexpected defaults: %+v", cfg)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.Parse(tc.args)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.check != nil {
				tc.check(t, cfg)
			}
		})
	}
}
