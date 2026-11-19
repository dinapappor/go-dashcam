package gps

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func OpenSerial(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, fmt.Errorf("Could not open serial device %s: %w", path, err)
	}

	var t syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&t)))
	if errno != 0 {
		f.Close()
		return nil, fmt.Errorf("Could not read serial settings on %s: %w", path, errno)
	}

	t.Iflag &^= syscall.ICRNL | syscall.INLCR | syscall.IGNCR | syscall.IXON
	t.Oflag &^= syscall.OPOST
	t.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ECHONL | syscall.ISIG | syscall.IEXTEN

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TCSETS, uintptr(unsafe.Pointer(&t)))
	if errno != 0 {
		f.Close()
		return nil, fmt.Errorf("Could not set raw mode on %s: %w", path, errno)
	}
	return f, nil
}
