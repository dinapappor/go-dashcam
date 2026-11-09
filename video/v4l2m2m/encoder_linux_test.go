package v4l2m2m

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"
)

func validConfig(device string) Config {
	return Config{
		Device:           device,
		Width:            1920,
		Height:           1080,
		Stride:           1920,
		FrameRate:        30,
		Bitrate:          4000000,
		KeyframeInterval: 30,
		InputSlots:       4,
		OutputBuffers:    4,
	}
}

func ioc(dir, typ, nr, size uintptr) uintptr {
	return dir<<30 | size<<16 | typ<<8 | nr
}

func TestStructLayout(t *testing.T) {
	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"capability size", unsafe.Sizeof(capability{}), 104},
		{"plane pix format size", unsafe.Sizeof(planePixFormat{}), 20},
		{"pix format mplane size", unsafe.Sizeof(pixFormatMplane{}), 192},
		{"num planes offset", unsafe.Offsetof(pixFormatMplane{}.NumPlanes), 180},
		{"format size", unsafe.Sizeof(format{}), 208},
		{"format pix offset", unsafe.Offsetof(format{}.Pix), 8},
		{"request buffers size", unsafe.Sizeof(requestBuffers{}), 20},
		{"plane size", unsafe.Sizeof(plane{}), 64},
		{"plane m offset", unsafe.Offsetof(plane{}.M), 8},
		{"plane data offset", unsafe.Offsetof(plane{}.DataOffset), 16},
		{"buffer size", unsafe.Sizeof(buffer{}), 88},
		{"buffer timestamp offset", unsafe.Offsetof(buffer{}.Timestamp), 24},
		{"buffer sequence offset", unsafe.Offsetof(buffer{}.Sequence), 56},
		{"buffer memory offset", unsafe.Offsetof(buffer{}.Memory), 60},
		{"buffer planes offset", unsafe.Offsetof(buffer{}.Planes), 64},
		{"buffer length offset", unsafe.Offsetof(buffer{}.Length), 72},
		{"control size", unsafe.Sizeof(control{}), 8},
		{"stream parm size", unsafe.Sizeof(outputStreamParm{}), 204},
		{"time per frame offset", unsafe.Offsetof(outputStreamParm{}.Numerator), 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %d, want %d", tt.got, tt.want)
			}
		})
	}
}

func TestIoctlRequests(t *testing.T) {
	const read, write = 2, 1
	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"QUERYCAP", vidiocQueryCap, ioc(read, 'V', 0, unsafe.Sizeof(capability{}))},
		{"S_FMT", vidiocSFmt, ioc(read|write, 'V', 5, unsafe.Sizeof(format{}))},
		{"REQBUFS", vidiocReqBufs, ioc(read|write, 'V', 8, unsafe.Sizeof(requestBuffers{}))},
		{"QUERYBUF", vidiocQueryBuf, ioc(read|write, 'V', 9, unsafe.Sizeof(buffer{}))},
		{"QBUF", vidiocQBuf, ioc(read|write, 'V', 15, unsafe.Sizeof(buffer{}))},
		{"DQBUF", vidiocDQBuf, ioc(read|write, 'V', 17, unsafe.Sizeof(buffer{}))},
		{"STREAMON", vidiocStreamOn, ioc(write, 'V', 18, 4)},
		{"STREAMOFF", vidiocStreamOff, ioc(write, 'V', 19, 4)},
		{"S_CTRL", vidiocSCtrl, ioc(read|write, 'V', 28, unsafe.Sizeof(control{}))},
		{"S_PARM", vidiocSParm, ioc(read|write, 'V', 22, unsafe.Sizeof(outputStreamParm{}))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %#x, want %#x", tt.got, tt.want)
			}
		})
	}
}

func TestOpenRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
	}{
		{"zero width", func(c *Config) { c.Width = 0 }},
		{"zero height", func(c *Config) { c.Height = 0 }},
		{"stride below width", func(c *Config) { c.Stride = c.Width - 1 }},
		{"zero frame rate", func(c *Config) { c.FrameRate = 0 }},
		{"zero bitrate", func(c *Config) { c.Bitrate = 0 }},
		{"negative bitrate", func(c *Config) { c.Bitrate = -1 }},
		{"negative keyframe interval", func(c *Config) { c.KeyframeInterval = -1 }},
		{"zero input slots", func(c *Config) { c.InputSlots = 0 }},
		{"zero output buffers", func(c *Config) { c.OutputBuffers = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig(filepath.Join(t.TempDir(), "never-opened"))
			tt.modify(&cfg)
			encoder, err := Open(cfg)
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, os.ErrNotExist) {
				t.Errorf("config should be rejected before opening the device, got %v", err)
			}
			if encoder != nil {
				t.Error("expected nil encoder")
			}
		})
	}
}

func TestOpenAcceptsZeroKeyframeInterval(t *testing.T) {
	cfg := validConfig(filepath.Join(t.TempDir(), "missing"))
	cfg.KeyframeInterval = 0
	_, err := Open(cfg)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected config to pass validation and fail at open, got %v", err)
	}
}

func TestOpenMissingDevice(t *testing.T) {
	encoder, err := Open(validConfig(filepath.Join(t.TempDir(), "missing")))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
	if encoder != nil {
		t.Error("expected nil encoder")
	}
}

func TestOpenNotV4L2Device(t *testing.T) {
	path := filepath.Join(t.TempDir(), "regular")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	encoder, err := Open(validConfig(path))
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("expected ENOTTY, got %v", err)
	}
	if encoder != nil {
		t.Error("expected nil encoder")
	}
}

func newFileEncoder(t *testing.T) *Encoder {
	t.Helper()
	path := filepath.Join(t.TempDir(), "regular")
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_CLOEXEC, 0600)
	if err != nil {
		t.Fatal(err)
	}
	e := &Encoder{fd: fd, inputSlots: 4}
	t.Cleanup(func() {
		if e.fd != -1 {
			e.Close()
		}
	})
	return e
}

func TestCloseTwice(t *testing.T) {
	e := newFileEncoder(t)
	if err := e.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := e.Close(); !errors.Is(err, os.ErrClosed) {
		t.Errorf("expected os.ErrClosed, got %v", err)
	}
}

func TestCloseStreamingNotV4L2Device(t *testing.T) {
	e := newFileEncoder(t)
	e.inputOn = true
	e.captureOn = true
	err := e.Close()
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("expected ENOTTY from stream off, got %v", err)
	}
	if e.fd != -1 {
		t.Error("descriptor should be closed even when stream off fails")
	}
}

func TestQueueInputSlotOutOfRange(t *testing.T) {
	e := newFileEncoder(t)
	err := e.QueueInput(4, 10, 100, 100, syscall.Timeval{})
	if err == nil || errors.Is(err, syscall.ENOTTY) {
		t.Errorf("expected range error before ioctl, got %v", err)
	}
}

func TestQueueInputBytesUsedTooLarge(t *testing.T) {
	e := newFileEncoder(t)
	err := e.QueueInput(0, 10, 100, 101, syscall.Timeval{})
	if err == nil || errors.Is(err, syscall.ENOTTY) {
		t.Errorf("expected size error before ioctl, got %v", err)
	}
}

func TestQueueInputNotV4L2Device(t *testing.T) {
	e := newFileEncoder(t)
	err := e.QueueInput(0, 10, 100, 100, syscall.Timeval{})
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("expected ENOTTY, got %v", err)
	}
}

func TestDequeueInputNotV4L2Device(t *testing.T) {
	e := newFileEncoder(t)
	_, ok, err := e.DequeueInput()
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("expected ENOTTY, got %v", err)
	}
	if ok {
		t.Error("expected ok to be false")
	}
}

func TestDequeueOutputNotV4L2Device(t *testing.T) {
	e := newFileEncoder(t)
	var out bytes.Buffer
	n, ok, err := e.DequeueOutput(&out)
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("expected ENOTTY, got %v", err)
	}
	if ok || n != 0 || out.Len() != 0 {
		t.Errorf("expected nothing written, got ok=%v n=%d len=%d", ok, n, out.Len())
	}
}

func TestDequeueClosedEncoder(t *testing.T) {
	e := newFileEncoder(t)
	fd := e.fd
	if err := syscall.Close(fd); err != nil {
		t.Fatal(err)
	}
	_, _, err := e.DequeueInput()
	e.fd = -1
	if !errors.Is(err, syscall.EBADF) {
		t.Errorf("expected EBADF, got %v", err)
	}
}

func TestContainsPicture(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"idr slice", []byte{0, 0, 1, 0x65, 0x88}, true},
		{"non idr slice", []byte{0, 0, 1, 0x41, 0x9a}, true},
		{"four byte start code", []byte{0, 0, 0, 1, 0x65}, true},
		{"slice after headers", []byte{0, 0, 0, 1, 0x67, 0x42, 0, 0, 0, 1, 0x68, 0xce, 0, 0, 1, 0x65}, true},
		{"sps only", []byte{0, 0, 0, 1, 0x67, 0x42}, false},
		{"pps only", []byte{0, 0, 0, 1, 0x68, 0xce}, false},
		{"start code without type", []byte{0, 0, 1}, false},
		{"no start code", []byte{0x65, 0x41, 0x65}, false},
		{"empty", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsPicture(tt.data); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenHardwareEncoder(t *testing.T) {
	device := os.Getenv("DASHCAM_ENCODER_DEVICE")
	if device == "" {
		t.Skip("set DASHCAM_ENCODER_DEVICE (for example /dev/video11) to run")
	}
	encoder, err := Open(validConfig(device))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if encoder.SizeImage() < 1920*1080*3/2 {
		t.Errorf("size image %d too small for 1080p YU12", encoder.SizeImage())
	}
	if err := encoder.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}
