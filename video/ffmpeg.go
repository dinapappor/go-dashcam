package video

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Encoder struct {
	pipe *os.File
	cmd  *exec.Cmd
}

func NewEncoder(inputArgs []string, outputArgs[]string) (*Encoder, error) {
	encoder := Encoder{}
	ffmpeg, err :=  exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("Could not lookup ffmpeg path: %w", err)
	}

	args := []string{"-hide_banner"}
	args = append(args, inputArgs...)
	args = append(args, "-i", "pipe:0")
	args = append(args, outputArgs...)
	encoder.cmd = exec.Command(
		ffmpeg,
		args...,
	)
	encoder.cmd.Stderr = os.Stderr

	encoder.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	read, write, err := os.Pipe()

	if err != nil {
		return nil, fmt.Errorf("Could not create pipe for ffmpeg: %w", err)
	}

	// Connect ffmpeg to the stdin pipe
	encoder.cmd.Stdin = read

	// This is what we write to
	encoder.pipe = write

	err = encoder.cmd.Start()
	if err != nil {
		read.Close()
		encoder.pipe.Close()
		return nil, fmt.Errorf("Unable to start encoder: %w", err)
	}

	read.Close()

	return &encoder, nil
}

func (e *Encoder) Close() error {
	var closeErr error
	err := e.pipe.Close()

	if err != nil {
		closeErr = errors.Join(closeErr, err)
	}

	// Forcefully kill ffmpeg if e.cmd.Wait() doesn't
	// return in time
	timer := time.AfterFunc(5*time.Second, func() {
		_ = e.cmd.Process.Kill()
	})

	// This happens only when timer ended
	defer timer.Stop()

	if waitErr := e.cmd.Wait(); waitErr != nil {
		closeErr = errors.Join(closeErr, waitErr)
	}

	return closeErr

}

func (e *Encoder) WriteFrame(data []byte) error {
	// if we fail to return within 2 seconds that
	// probably means ffmpeg stalled
	err := e.pipe.SetWriteDeadline(time.Now().Add(10 * time.Second))

	if err != nil {
		return fmt.Errorf("Unable to set write deadline: %w", err)
	}

	written, err := e.pipe.Write(data)

	if err != nil {
		return fmt.Errorf("Unable to write data to ffmpeg: %w", err)
	}

	if written != len(data) {
		return fmt.Errorf("Incomplete frame write: %d (written) of %d (intended)", written, len(data))
	}
	return nil
}

func (e *Encoder) Write(data []byte) (int, error) {
	err := e.WriteFrame(data)

	if err != nil {
		return 0, fmt.Errorf("Could not Write data with WriteFrame: %w", err)
	}
	return len(data), nil
}
