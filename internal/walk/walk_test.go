package walk_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"textanalyzerasync/internal/walk"
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

func TestCollect(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one")
	writeFile(t, dir, "b.TXT", "two")
	writeFile(t, dir, "skip.md", "nope")
	writeFile(t, dir, filepath.Join("nested", "c.txt"), "three")

	files, err := walk.Collect(context.Background(), dir, walk.Options{Ext: ".txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("got %d files, want 3: %v", len(files), files)
	}
}

func TestCollectMinSize(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "small.txt", "aa")
	writeFile(t, dir, "big.txt", string(bytes.Repeat([]byte("a"), 1000)))

	files, err := walk.Collect(context.Background(), dir, walk.Options{Ext: ".txt", MinSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || !strings.HasSuffix(files[0], "big.txt") {
		t.Fatalf("got %v", files)
	}
}

func TestCollectMaxSize(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "small.txt", "aa")
	writeFile(t, dir, "big.txt", string(bytes.Repeat([]byte("a"), 1000)))

	files, err := walk.Collect(context.Background(), dir, walk.Options{Ext: "txt", MaxSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || !strings.HasSuffix(files[0], "small.txt") {
		t.Fatalf("got %v", files)
	}
}

func TestCollectSingleFile(t *testing.T) {
	path := writeFile(t, t.TempDir(), "only.txt", "hello")
	files, err := walk.Collect(context.Background(), path, walk.Options{Ext: ".txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != path {
		t.Fatalf("got %v", files)
	}
}

func TestCollectMissing(t *testing.T) {
	_, err := walk.Collect(context.Background(), filepath.Join(t.TempDir(), "nope"), walk.Options{Ext: ".txt"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCollectCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := walk.Collect(ctx, t.TempDir(), walk.Options{Ext: ".txt"})
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestCollectTestdata(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("testdata not found")
	}
	files, err := walk.Collect(context.Background(), dir, walk.Options{Ext: ".txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("testdata: got %d files, want 2: %v", len(files), files)
	}
}
