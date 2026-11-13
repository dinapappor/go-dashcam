package gps

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestOpenI2CMissingPath(t *testing.T) {
	f, err := OpenI2C(filepath.Join(t.TempDir(), "missing"), 0x10)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got error %v, want os.ErrNotExist", err)
	}
	if f != nil {
		t.Error("expected nil file")
	}
}

func TestOpenI2CNotI2CDevice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-i2c")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}

	f, err := OpenI2C(path, 0x10)
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("got error %v, want ENOTTY", err)
	}
	if f != nil {
		t.Error("expected nil file")
	}
}

func TestOpenI2CDevice(t *testing.T) {
	path := os.Getenv("DASHCAM_I2C_DEVICE")
	if path == "" {
		t.Skip("set DASHCAM_I2C_DEVICE=/dev/i2c-1 to test a real GPS at 0x10")
	}

	f, err := OpenI2C(path, 0x10)
	if err != nil {
		t.Fatalf("OpenI2C: %v", err)
	}
	defer f.Close()

	buf := make([]byte, 255)
	n, err := f.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != len(buf) {
		t.Errorf("read %d bytes, want %d", n, len(buf))
	}
}
