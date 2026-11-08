package v4l2

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestOpenDeviceMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")

	fd, err := openDevice(path)

	if fd != -1 {
		t.Errorf("expected descriptor -1, got %d", fd)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestCloseDeviceInvalidDescriptor(t *testing.T) {
	err := closeDevice(-1)

	if !errors.Is(err, syscall.EBADF) {
		t.Errorf("expected syscall.EBADF, got %v", err)
	}
}

func TestOpenCloseDevice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	fd, err := openDevice(path)
	if err != nil {
		t.Fatalf("open existing file: %v", err)
	}
	if fd < 0 {
		t.Fatalf("expected a nonnegative descriptor, got %d", fd)
	}

	if err := closeDevice(fd); err != nil {
		t.Errorf("close opened descriptor: %v", err)
	}
}
