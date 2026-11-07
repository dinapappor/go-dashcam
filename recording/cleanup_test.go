package recording

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")

	err := Cleanup(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestCleanupEnoughSpacePreservesRecordings(t *testing.T) {
	dir := t.TempDir()
	free, err := FreeSpace(dir)
	if err != nil {
		t.Fatalf("check test filesystem space: %v", err)
	}
	if free < MinSpace {
		t.Skip("test requires at least MinSpace available on the test filesystem")
	}

	// Both files must survive when cleanup has no reason to delete anything.
	paths := []string{
		filepath.Join(dir, "dashcam_20261106-090000.mp4"),
		filepath.Join(dir, "dashcam_20261106-100000.mp4"),
	}
	for _, path := range paths {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatalf("create recording: %v", err)
		}
	}

	if err := Cleanup(dir); err != nil {
		t.Fatalf("cleanup with enough space: %v", err)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("recording should remain at %s: %v", path, err)
		}
	}
}
