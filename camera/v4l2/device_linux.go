package v4l2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

type Device struct {
	file    *os.File
	buffers [][]byte
}

type Frame struct {
	buffer V4L2Buffer
	Data   []byte
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

func (d *Device) NextFrame(ctx context.Context) (Frame, error) {

	for {
		completedBuffer, err := DequeueBuffer(d.Fd())

		if errors.Is(err, syscall.EAGAIN) {
			select {
				case <- ctx.Done():
					return Frame{}, ctx.Err()
				case <- time.After(10 * time.Millisecond):
				}
				continue
		}

		if err != nil {
			return Frame{}, fmt.Errorf("Not able to dequeue: %w", err)
		}

		if completedBuffer.Flags&BufFlagError != 0 {
			err = QueueBuffer(d.Fd(), completedBuffer)
			if err != nil {
				return Frame{}, fmt.Errorf("Could not reqeueue buffer for damaged frame: %w", err)
			}
			continue

		}
		frame, err := FrameBytes(
			d.Buffers(),
			completedBuffer.Index,
			completedBuffer.BytesUsed,
		)

		if err != nil {
			requeueErr := QueueBuffer(d.Fd(), completedBuffer)
			if requeueErr != nil {
				return Frame{}, errors.Join(err, fmt.Errorf("Unable to requeue buffer after failed FrameBytes: %w", requeueErr))
			}
			return Frame{}, fmt.Errorf("Unable to read buffers, got %w", err)
		}
		return Frame{buffer: completedBuffer, Data: frame}, nil
	}
}

func (d *Device) Requeue(frame Frame) error {
	err := QueueBuffer(d.Fd(), frame.buffer)

	if err != nil {
		return fmt.Errorf("Could not requeue: %w", err)
	}
	return nil
}
