package main

import (
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCPUProfileFlushesOnStop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.pprof")
	stop, err := startCPUProfile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil || len(data) == 0 {
		t.Fatalf("expected a complete, nonempty profile: bytes=%d, err=%v", len(data), err)
	}
}

func TestCPUProfileMissingDirectory(t *testing.T) {
	stop, err := startCPUProfile(filepath.Join(t.TempDir(), "missing", "cpu.pprof"))
	if !errors.Is(err, os.ErrNotExist) || stop != nil {
		t.Fatalf("expected missing-directory error and nil cleanup, got %v", err)
	}
}

func TestCPUProfilePreservesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.pprof")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	stop, err := startCPUProfile(path)
	if !errors.Is(err, os.ErrExist) || stop != nil {
		t.Fatalf("expected existing-file error and nil cleanup, got %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing file changed: %q, err=%v", data, err)
	}
}
