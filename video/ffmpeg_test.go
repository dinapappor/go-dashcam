package video

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
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

func TestEncoderWriteReturnsLength(t *testing.T) {
	installFakeFFmpeg(t, "cat > /dev/null")

	data := make([]byte, 1024)

	encoder, err := NewEncoder([]string{"-f", "mjpeg"}, []string{"out.mp4"})
	if err != nil {
		t.Fatalf("Unable to start fake ffmpeg: %v", err)
	}

	n, err := encoder.Write(data)
	if err != nil {
		t.Errorf("Write should succeed, got %v", err)
	}
	encoder.Close()

	if n != len(data) {
		t.Errorf("Bytes written should be equal to length of data send. Got:%d wanted %d", n, len(data))
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

func TestEncoderWriteErrorAfterExit(t *testing.T) {
	installFakeFFmpeg(t, "exit 0")

	encoder, err := NewEncoder(nil, nil)
	if err != nil {
		t.Fatalf("Unable to start fake ffmpeg: %v", err)
	}
	defer encoder.Close()

	data := make([]byte, 64*1024)
	var n int
	for i := 0; i < 100; i++ {
		n, err = encoder.Write(data)
		if err != nil {
			break
		}
	}

	if !errors.Is(err, syscall.EPIPE) {
		t.Errorf("WriteFrame should fail with EPIPE after ffmpeg exits, got %v", err)
	}

	if n != 0 {
		t.Errorf("Expted written bytes to be zero, got %d", n)
	}
}

func TestEncoderWriteStalled(t *testing.T) {
	installFakeFFmpeg(t, "trap '' INT\nexec sleep 60")

	encoder, err := NewEncoder(nil, nil)
	if err != nil {
		t.Fatalf("Unable to start fake ffmpeg: %v", err)
	}
	encoder.writeTimeout = 100 * time.Millisecond
	encoder.interruptTimeout = 100 * time.Millisecond
	encoder.killTimeout = 300 * time.Millisecond

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

func TestNewEncoderShutdownDefaults(t *testing.T) {
	installFakeFFmpeg(t, "cat > /dev/null")

	encoder, err := NewEncoder(nil, nil)
	if err != nil {
		t.Fatalf("Unable to start fake ffmpeg: %v", err)
	}
	defer encoder.Close()

	if encoder.interruptTimeout != 2*time.Second {
		t.Errorf("interruptTimeout should default to 2s, got %v", encoder.interruptTimeout)
	}

	if encoder.killTimeout != 5*time.Second {
		t.Errorf("killTimeout should default to 5s, got %v", encoder.killTimeout)
	}
}

func TestEncoderCloseShutdown(t *testing.T) {
	tests := []struct {
		name         string
		script       string
		wantErr      bool
		wantExitCode int
		wantSignal   syscall.Signal
	}{
		{
			name:    "exits on closed pipe before interrupt",
			script:  "cat > /dev/null\nexit 0",
			wantErr: false,
		},
		{
			name:    "ignores closed pipe, exits 255 on interrupt",
			script:  "trap 'exit 255' INT\ncat > /dev/null\nwhile true; do sleep 0.05; done",
			wantErr: false,
		},
		{
			name:         "ignores closed pipe, exits 1 on interrupt",
			script:       "trap 'exit 1' INT\ncat > /dev/null\nwhile true; do sleep 0.05; done",
			wantErr:      true,
			wantExitCode: 1,
		},
		{
			name:         "exits 255 on its own without interrupt",
			script:       "cat > /dev/null\nexit 255",
			wantErr:      true,
			wantExitCode: 255,
		},
		{
			name:       "ignores interrupt, gets killed",
			script:     "trap '' INT\ncat > /dev/null\nexec sleep 60",
			wantErr:    true,
			wantSignal: syscall.SIGKILL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installFakeFFmpeg(t, tt.script)

			encoder, err := NewEncoder(nil, nil)
			if err != nil {
				t.Fatalf("Unable to start fake ffmpeg: %v", err)
			}
			encoder.interruptTimeout = 100 * time.Millisecond
			encoder.killTimeout = 1 * time.Second

			start := time.Now()
			err = encoder.Close()
			elapsed := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Close error = %v, wantErr %t", err, tt.wantErr)
			}

			if tt.wantSignal == 0 && elapsed >= encoder.killTimeout {
				t.Errorf("Close took %v, should finish before killTimeout %v", elapsed, encoder.killTimeout)
			}

			if !tt.wantErr {
				return
			}

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("Close should return an exec.ExitError, got %v", err)
			}

			if tt.wantSignal != 0 {
				status, ok := exitErr.Sys().(syscall.WaitStatus)
				if !ok || !status.Signaled() || status.Signal() != tt.wantSignal {
					t.Errorf("ffmpeg should be stopped by %v, got %v", tt.wantSignal, err)
				}
				return
			}

			if exitErr.ExitCode() != tt.wantExitCode {
				t.Errorf("Exit code should be %d, got %d", tt.wantExitCode, exitErr.ExitCode())
			}
		})
	}
}
