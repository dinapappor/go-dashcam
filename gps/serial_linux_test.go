package gps

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"
)

func TestOpenSerialMissingPath(t *testing.T) {
	f, err := OpenSerial(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got error %v, want os.ErrNotExist", err)
	}
	if f != nil {
		t.Error("expected nil file")
	}
}

func TestOpenSerialNotTerminal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-serial")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}

	f, err := OpenSerial(path)
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("got error %v, want ENOTTY", err)
	}
	if f != nil {
		t.Error("expected nil file")
	}
}

func TestOpenSerialDevice(t *testing.T) {
	path := os.Getenv("DASHCAM_SERIAL_DEVICE")
	if path == "" {
		t.Skip("set DASHCAM_SERIAL_DEVICE=/dev/ttyACM0 to test a real serial GPS")
	}

	f, err := OpenSerial(path)
	if err != nil {
		t.Fatalf("OpenSerial: %v", err)
	}
	defer f.Close()

	var term syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&term)))
	if errno != 0 {
		t.Fatalf("TCGETS: %v", errno)
	}

	flags := []struct {
		name  string
		value uint32
		flag  uint32
	}{
		{"ICRNL", term.Iflag, syscall.ICRNL},
		{"INLCR", term.Iflag, syscall.INLCR},
		{"IGNCR", term.Iflag, syscall.IGNCR},
		{"IXON", term.Iflag, syscall.IXON},
		{"OPOST", term.Oflag, syscall.OPOST},
		{"ICANON", term.Lflag, syscall.ICANON},
		{"ECHO", term.Lflag, syscall.ECHO},
		{"ECHONL", term.Lflag, syscall.ECHONL},
		{"ISIG", term.Lflag, syscall.ISIG},
		{"IEXTEN", term.Lflag, syscall.IEXTEN},
	}
	for _, tt := range flags {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value&tt.flag != 0 {
				t.Errorf("%s still set", tt.name)
			}
		})
	}

	var got []byte
	buf := make([]byte, 256)
	for len(got) < 4096 {
		n, err := f.Read(buf)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		got = append(got, buf[:n]...)
	}

	sentences := 0
	err = ReadSentences(bytes.NewReader(got), func(Sentence) { sentences++ })
	if err != nil {
		t.Fatalf("ReadSentences: %v", err)
	}
	if sentences == 0 {
		t.Error("no RMC or GGA sentences parsed from raw serial data")
	}
}
