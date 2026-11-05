package v4l2

import (
	"errors"
	"path/filepath"
	"os"
	"testing"
)

func TestOpenMissingPath(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("This should fail on non_existent_file, got %v", err)
	}

	if d != nil {
		t.Errorf("device should be nil, got %v", d)
	}
}

func TestOpenAndClose(t *testing.T) {
	// Set up file for testing with
	fileName := filepath.Join(t.TempDir(), "fake")

	f, err := os.Create(fileName)

	if err != nil {
		t.Fatalf("Could not create %s: %v", fileName, err)
	}

	f.Close()

	d, err := Open(fileName)

	if err != nil {
		t.Fatalf("This should not happen: %v", err)
	}

	err = d.Close()
	if err != nil {
		t.Errorf("We should be able to close a file: %v", err)
	}
}

func TestCloseTwice(t *testing.T) {
	// Set up file for testing with
	fileName := filepath.Join(t.TempDir(), "fake")

	f, err := os.Create(fileName)

	if err != nil {
		t.Fatalf("Could not create %s: %v", fileName, err)
	}

	f.Close()

	d, err := Open(fileName)

	if err != nil {
		t.Fatalf("This should not happen: %v", err)
	}

	err = d.Close()
	if err != nil {
		t.Fatalf("We should be able to close a file: %v", err)
	}

	err = d.Close()

	if !errors.Is(err, os.ErrClosed) {
		t.Errorf("We should not be able to close a file twice:%v", err)
	}
}
