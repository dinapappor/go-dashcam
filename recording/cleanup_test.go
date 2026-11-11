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

func TestCleanupDeletesOldest(t *testing.T) {
	dir := t.TempDir()

	paths := []string{
		filepath.Join(dir, "dashcam_20261106-090000.mp4"),
		filepath.Join(dir, "dashcam_20261106-100000.mp4"),
		filepath.Join(dir, "dashcam_20261106-110000.mp4"),
	}
	for _, path := range paths {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatalf("create recording: %v", err)
		}
	}

	calls := 0
	freeSpace = func(path string) (uint64, error) {
		calls++
		if calls == 1 {
			return 0, nil
		}
		return MinSpace, nil
	}
	t.Cleanup(func() { freeSpace = FreeSpace })

	if err := Cleanup(dir); err != nil {
		t.Fatalf("Cleanup should succeed, got %v", err)
	}

	if _, err := os.Stat(paths[0]); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("oldest recording should be deleted, got %v", err)
	}

	// paths[1:] is everything but the first that starts at [0]
	for _, path := range paths[1:] {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s should still exist, got %v", path, err)
		}
	}
}

func TestCleanupKeepsNewestAndUnrelated(t *testing.T) {
	dir := t.TempDir()

	paths := []string{
		filepath.Join(dir, "dashcam_20261106-090000.mp4"),
		filepath.Join(dir, "dashcam_20261106-100000.mp4"),
		filepath.Join(dir, "dashcam_20261106-110000.mp4"),
		filepath.Join(dir, "notes.txt"),
	}
	for _, path := range paths {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatalf("create recording: %v", err)
		}
	}

	freeSpace = func(path string) (uint64, error) {
		return 0, nil
	}
	t.Cleanup(func() { freeSpace = FreeSpace })

	if err := Cleanup(dir); err == nil {
		t.Fatalf("Cleanup should fail with only one recording left")
	}

	for _, path := range paths[:2] {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s should be deleted, got %v", path, err)
		}
	}

	for _, path := range paths[2:] {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s should still exist, got %v", path, err)
		}
	}
}

func TestCleanupFreeSpaceError(t *testing.T) {
	diskGone := errors.New("disk gone")
	freeSpace = func(path string) (uint64, error) {
		return 0, diskGone
	}
	t.Cleanup(func() { freeSpace = FreeSpace })

	err := Cleanup(t.TempDir())
	if !errors.Is(err, diskGone) {
		t.Errorf("Cleanup should pass the free space error through, got %v", err)
	}
}
