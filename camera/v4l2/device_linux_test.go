package v4l2

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
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

func TestSetupBuffersNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	d, err := Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}
	defer d.Close()

	err = d.SetupBuffers(4)
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("SetupBuffers should fail with ENOTTY, got %v", err)
	}

	if len(d.Buffers()) != 0 {
		t.Errorf("Buffers should be empty after failed setup, got %d", len(d.Buffers()))
	}
}

func TestSetupBuffersClosedDevice(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	d, err := Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("Unable to close device: %v", err)
	}

	err = d.SetupBuffers(4)
	if !errors.Is(err, syscall.EBADF) {
		t.Errorf("SetupBuffers should fail with EBADF, got %v", err)
	}
}

func TestNextFrameNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	d, err := Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}
	defer d.Close()

	frame, err := d.NextFrame(context.Background())
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("NextFrame should fail with ENOTTY, got %v", err)
	}

	if frame.Data != nil {
		t.Errorf("Frame data should be nil after failure, got %d bytes", len(frame.Data))
	}
}

func TestNextFrameClosedDevice(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	d, err := Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("Unable to close device: %v", err)
	}

	_, err = d.NextFrame(context.Background())
	if !errors.Is(err, syscall.EBADF) {
		t.Errorf("NextFrame should fail with EBADF, got %v", err)
	}
}

func TestRequeueNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	d, err := Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}
	defer d.Close()

	err = d.Requeue(Frame{})
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("Requeue should fail with ENOTTY, got %v", err)
	}
}

func TestRequeueClosedDevice(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	d, err := Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("Unable to close device: %v", err)
	}

	err = d.Requeue(Frame{})
	if !errors.Is(err, syscall.EBADF) {
		t.Errorf("Requeue should fail with EBADF, got %v", err)
	}
}

func TestCloseStreamingNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	d, err := Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}
	d.streaming = true

	err = d.Close()
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("Close should report ENOTTY from StreamOff, got %v", err)
	}

	err = d.Close()
	if !errors.Is(err, os.ErrClosed) {
		t.Errorf("File should be closed even when StreamOff fails, got %v", err)
	}
}

func TestExportFdKnownIndex(t *testing.T) {
	d := &Device{exports: []int{10, 11}}
	frame := Frame{buffer: V4L2Buffer{Index: 1}}

	fd, err := d.ExportFd(frame)
	if err != nil {
		t.Fatalf("ExportFd should succeed, got %v", err)
	}

	if fd != 11 {
		t.Errorf("ExportFd should return 11, got %d", fd)
	}
}

func TestExportFdMissingExport(t *testing.T) {
	d := &Device{exports: []int{10, 11}}
	frame := Frame{buffer: V4L2Buffer{Index: 5}}

	fd, err := d.ExportFd(frame)
	if err == nil {
		t.Fatalf("ExportFd should fail for a buffer without an export")
	}

	if fd != -1 {
		t.Errorf("ExportFd should return -1 on failure, got %d", fd)
	}
}

func TestExportBuffersNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	d, err := Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}
	defer d.Close()

	d.buffers = make([][]byte, 2)

	err = d.ExportBuffers()
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("ExportBuffers should fail with ENOTTY, got %v", err)
	}

	if len(d.exports) != 0 {
		t.Errorf("No exports should be stored after failure, got %d", len(d.exports))
	}
}

func TestExportBuffersNoBuffers(t *testing.T) {
	d := &Device{}

	err := d.ExportBuffers()
	if err != nil {
		t.Errorf("ExportBuffers with no buffers should succeed, got %v", err)
	}
}
