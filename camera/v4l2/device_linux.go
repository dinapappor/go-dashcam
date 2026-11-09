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
	fd        int
	buffers   [][]byte
	streaming bool
	exports   []int
}

type Frame struct {
	buffer V4L2Buffer
	Data   []byte
}

func Open(path string) (*Device, error) {
	f, err := openDevice(path)

	if err != nil {
		return nil, fmt.Errorf("Unable to open device %s: %w", path, err)
	}
	return &Device{fd: f}, nil
}

// This is a method
func (d *Device) Close() error {

	if d.fd == -1 {
		return os.ErrClosed
	}

	// Store all errors
	var closeErr error

	if d.streaming {
		err := d.StreamOff()
		if err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("Unable to shut down stream: %w", err))
		}
	}
	for _, eb := range d.exports {
		err := closeDevice(eb)
		if err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close export %d: %w", eb, err))
		}
	}
	d.exports = nil
	// Munmap all the buffers
	for index := range d.buffers {
		// In case we already closed the buffer
		if d.buffers[index] == nil {
			continue
		}
		if err := unmapDevice(d.buffers[index]); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("unmap buffer %d: %w", index, err))
		}
		d.buffers[index] = nil
	}

	err := closeDevice(d.fd)
	d.fd = -1
	return errors.Join(closeErr, err)
}

func (d *Device) Fd() uintptr {
	return uintptr(d.fd)
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

func (d *Device) ExportBuffers() error {
	for index := range d.buffers {
		exportFd, err := ExportBuffer(d.Fd(), uint32(index))

		if err != nil {
			return fmt.Errorf("Could not export buffer %d: %w", index, err)
		}

		d.exports = append(d.exports, exportFd)
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
			case <-ctx.Done():
				return Frame{}, ctx.Err()
			case <-time.After(10 * time.Millisecond):
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

func (d *Device) StreamOn() error {
	// StreamOn below is from v4l2, not this method
	err := StreamOn(d.Fd())

	if err != nil {
		return fmt.Errorf("Unable to set stream on: %w", err)
	}
	d.streaming = true
	return nil
}

func (d *Device) StreamOff() error {
	// StreamOn below is from v4l2, not this method
	err := StreamOff(d.Fd())

	if err != nil {
		return fmt.Errorf("Unable to set stream off: %w", err)
	}
	d.streaming = false
	return nil
}

func (d *Device) ExportFd(frame Frame) (int, error) {
	index := int(frame.buffer.Index)

	if index >= len(d.exports) {
		return -1, fmt.Errorf("buffer %d has no export", index)
	}

	return d.exports[index], nil
}

func (f Frame) Index() uint32 {
	return f.buffer.Index
}

func (f Frame) Sequence() uint32 {
	return f.buffer.Sequence
}

func (f Frame) Timestamp() syscall.Timeval {
	return f.buffer.TimeStamp
}
