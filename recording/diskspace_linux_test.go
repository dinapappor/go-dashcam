package recording

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFreeSpace(t *testing.T) {

	free, err := FreeSpace(t.TempDir())

	if err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	if free <= 0 {
		t.Fatalf("Free bytes should be greater than zero on actual path")
	}
}

func TestFreeSpaceNonExistantPath(t *testing.T) {
	free, err := FreeSpace(filepath.Join(t.TempDir(), "poop"))

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("We should not pass this with fake path")
	}

	if free != 0 {
		t.Fatalf("Returned bytes should be 0 on fake path")
	}
}

func TestListRecordings(t *testing.T) {
	dir := t.TempDir()

	// Create fake files
	err := os.WriteFile(filepath.Join(dir, "dashcam_test.mp4"), nil, 0600)

	if err != nil {
		t.Fatalf("Could not create test file (dashcam_test.mp4)")
	}

	err = os.WriteFile(filepath.Join(dir, "other.mp4"), nil, 0600)

	if err != nil {
		t.Fatalf("Could not create testfile (other.mp4)")
	}

	err = os.WriteFile(filepath.Join(dir, "dashcam_test.txt"), nil, 0600)

	if err != nil {
		t.Fatalf("Could not create testfile (dashcam_test.txt)")
	}

	err = os.Mkdir(filepath.Join(dir, "dashcam_folder.mp4"), 0755)

	if err != nil {
		t.Fatalf("Could not create test directory named mp4")
	}

	files, err := ListRecordings(dir)

	if err != nil {
		t.Fatalf("Could not list contents of %s: %v", dir, err)
	}

	// We should only return one real recording as per our test case
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d files", len(files))
	}

	if files[0] != filepath.Join(dir, "dashcam_test.mp4") {
		t.Fatalf("Expected dashcam_test.mp4, got %s", files[0])
	}
}

func TestMissingDirectory(t *testing.T) {
	files, err := ListRecordings(filepath.Join(t.TempDir(), "non-existent"))

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("We should not be able to list a non-existant directory: %v", err)
	}

	if len(files) != 0 {
		t.Fatalf("A failed directory should contain zero files, got %d", len(files))
	}
}
func TestExistingEmptyDirectory(t *testing.T) {

	files, err := ListRecordings(t.TempDir())

	if err != nil {
		t.Fatalf("We should not fail listing an empty directory")
	}

	if len(files) != 0 {
		t.Fatalf("An empty directory should return 0 files, got %d files", len(files))
	}
}

func TestOldestFirst(t *testing.T) {
	dir := t.TempDir()

	//dashcam_20261106-090000.mp4
	//dashcam_20261106-100000.mp4
	err := os.WriteFile(filepath.Join(dir, "dashcam_20261106-100000.mp4"), nil, 0600)

	if err != nil {
		t.Fatalf("Could not create testfile dashcam_20261106-100000.mp4: %v", err)
	}

	err = os.WriteFile(filepath.Join(dir, "dashcam_20261106-090000.mp4"), nil, 0600)

	if err != nil {
		t.Fatalf("Could not create testfile dashcam_20261106-090000.mp4: %v", err)
	}

	files, err := ListRecordings(dir)

	if err != nil {
		t.Fatalf("Could not list test directory: %s", dir)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}
	if files[0] != filepath.Join(dir, "dashcam_20261106-090000.mp4") {
		t.Fatalf("Expected: %s, got: %s", filepath.Join(dir, "dashcam_20261106-090000.mp4"), files[0])
	}
	if files[1] != filepath.Join(dir, "dashcam_20261106-100000.mp4") {
		t.Fatalf("Expected: %s, got: %s", filepath.Join(dir, "dashcam_20261106-100000.mp4"), files[1])
	}
}
