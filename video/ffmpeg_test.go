package video

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func installFakeFFmpeg(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" + body + "\n"
	err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte(script), 0o755)
	if err != nil {
		t.Fatalf("Unable to write fake ffmpeg: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestNewEncoderMissingFFmpeg(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	encoder, err := NewEncoder(nil, nil)
	if !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("NewEncoder should fail with exec.ErrNotFound, got %v", err)
	}

	if encoder != nil {
		t.Errorf("Encoder should be nil on failure, got %v", encoder)
	}
}

func TestEncoderWriteAndClose(t *testing.T) {
	installFakeFFmpeg(t, "cat > /dev/null")

	encoder, err := NewEncoder([]string{"-f", "mjpeg"}, []string{"out.mp4"})
	if err != nil {
		t.Fatalf("Unable to start fake ffmpeg: %v", err)
	}

	err = encoder.WriteFrame(make([]byte, 1024))
	if err != nil {
		t.Errorf("WriteFrame should succeed, got %v", err)
	}

	err = encoder.Close()
	if err != nil {
		t.Errorf("Close should succeed, got %v", err)
	}
}

func TestEncoderCloseReportsExitStatus(t *testing.T) {
	installFakeFFmpeg(t, "cat > /dev/null\nexit 3")

	encoder, err := NewEncoder(nil, nil)
	if err != nil {
		t.Fatalf("Unable to start fake ffmpeg: %v", err)
	}

	err = encoder.Close()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Close should return an exec.ExitError, got %v", err)
	}

	if exitErr.ExitCode() != 3 {
		t.Errorf("Exit code should be 3, got %d", exitErr.ExitCode())
	}
}

func TestEncoderWriteAfterExit(t *testing.T) {
	installFakeFFmpeg(t, "exit 0")

	encoder, err := NewEncoder(nil, nil)
	if err != nil {
		t.Fatalf("Unable to start fake ffmpeg: %v", err)
	}
	defer encoder.Close()

	data := make([]byte, 64*1024)
	for i := 0; i < 100; i++ {
		err = encoder.WriteFrame(data)
		if err != nil {
			break
		}
	}

	if !errors.Is(err, syscall.EPIPE) {
		t.Errorf("WriteFrame should fail with EPIPE after ffmpeg exits, got %v", err)
	}
}

func TestEncoderWriteStalled(t *testing.T) {
	installFakeFFmpeg(t, "exec sleep 60")

	encoder, err := NewEncoder(nil, nil)
	if err != nil {
		t.Fatalf("Unable to start fake ffmpeg: %v", err)
	}

	err = encoder.WriteFrame(make([]byte, 1024*1024))
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Errorf("WriteFrame should hit the write deadline, got %v", err)
	}

	err = encoder.Close()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Close should return an exec.ExitError, got %v", err)
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Errorf("Stalled ffmpeg should be killed with SIGKILL, got %v", err)
	}
}
