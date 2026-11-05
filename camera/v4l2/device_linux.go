package v4l2

import (
	"fmt"
	"os"
	"syscall"
	"errors"
)

type Device struct {
	file    *os.File
	buffers [][]byte
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

	// Store all errors
	var unmapErr error
	// Munmap all the buffers
	for index := range d.buffers {
		// In case we already closed the buffer
		if d.buffers[index] == nil {
			continue
		}
		if err := syscall.Munmap(d.buffers[index]); err != nil {
			unmapErr = errors.Join(unmapErr, fmt.Errorf("unmap buffer %d: %w", index, err))
		}
		d.buffers[index] = nil
	}
	return errors.Join(unmapErr, d.file.Close())
}

func (d *Device) Fd() uintptr {
	return d.file.Fd()
}

func (d *Device) SetupBuffers(count uint32) error {
	requestBuffers, err := RequestMmapBuffers(d.Fd(), count)

	if err != nil {
		return fmt.Errorf("Unable to request buffers, got %w", err)
	}

	// Add buffers to the Device Struct
	d.buffers = make([][]byte, requestBuffers.Count)

	for index := uint32(0); index < requestBuffers.Count; index++ {
		buffer, err := QueryBuffer(d.Fd(), index)

		if err != nil {
			return fmt.Errorf("Could not query buffer %d: %w", index, err)
		}

		data, err := MapBuffer(d.Fd(), buffer)

		if err != nil {
			return fmt.Errorf("Could not Map buffer %d: %w", index, err)
		}

		d.buffers[index] = data

		err = QueueBuffer(d.Fd(), buffer)

		if err != nil {
			return fmt.Errorf("Could not Queue buffer %d: %w", index, err)
		}
	}
	return nil
}

func (d *Device) Buffers() [][]byte {
	return d.buffers
}
