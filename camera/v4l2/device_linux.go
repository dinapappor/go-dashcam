package v4l2

import (
	"fmt"
	"os"
	"syscall"
)

type Device struct {
	file *os.File
}

func Open(path string) (*Device, error) {
	f, err := os.OpenFile(
		path,
		os.O_RDWR|syscall.O_NONBLOCK,
		0,
	)

	if err != nil {
		return nil, fmt.Errorf("Unable to open device %s: %w", path, err)
	}
	return &Device{file: f}, nil
}

// This is a method
func (d *Device) Close() error {
	return d.file.Close()
}

func (d *Device) Fd() uintptr {
	return d.file.Fd()
}
