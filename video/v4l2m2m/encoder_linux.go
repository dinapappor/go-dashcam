package v4l2m2m

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

var ErrBufferError = errors.New("driver flagged buffer as damaged")

type Config struct {
	Device           string
	Width            uint32
	Height           uint32
	Stride           uint32
	FrameRate        uint32
	Bitrate          int32
	KeyframeInterval int32
	InputSlots       uint32
	OutputBuffers    uint32
}

type Encoder struct {
	fd         int
	inputSlots uint32
	sizeImage  uint32
	outputs    [][]byte
	inputOn    bool
	captureOn  bool
}

func validateConfig(cfg Config) error {
	if cfg.Width == 0 || cfg.Height == 0 {
		return fmt.Errorf("invalid size %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Stride < cfg.Width {
		return fmt.Errorf("stride %d is smaller than width %d", cfg.Stride, cfg.Width)
	}
	if cfg.FrameRate == 0 {
		return errors.New("frame rate must be positive")
	}
	if cfg.Bitrate <= 0 {
		return errors.New("bitrate must be positive")
	}
	if cfg.KeyframeInterval < 0 {
		return errors.New("keyframe interval must not be negative")
	}
	if cfg.InputSlots == 0 || cfg.OutputBuffers == 0 {
		return errors.New("input slots and output buffers must be positive")
	}
	return nil
}

func Open(cfg Config) (*Encoder, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid encoder config: %w", err)
	}

	fd, err := syscall.Open(cfg.Device, syscall.O_RDWR|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open encoder %s: %w", cfg.Device, err)
	}

	e := &Encoder{fd: fd}
	if err := e.setup(cfg); err != nil {
		return nil, errors.Join(err, e.Close())
	}
	return e, nil
}

func (e *Encoder) setup(cfg Config) error {
	var caps capability
	if err := ioctl(e.fd, vidiocQueryCap, unsafe.Pointer(&caps)); err != nil {
		return fmt.Errorf("query encoder capability: %w", err)
	}
	selected := caps.Capabilities
	if caps.Capabilities&capDeviceCaps != 0 {
		selected = caps.DeviceCaps
	}
	if selected&capVideoM2MMplane == 0 || selected&capStreaming == 0 {
		return fmt.Errorf("device is not a multiplanar streaming memory-to-memory device (caps %#x)", selected)
	}

	if err := e.setControl(cidBitrate, cfg.Bitrate); err != nil {
		return err
	}
	if err := e.setControl(cidRepeatSeqHeader, 1); err != nil {
		return err
	}
	if cfg.KeyframeInterval > 0 {
		if err := e.setControl(cidH264IPeriod, cfg.KeyframeInterval); err != nil {
			return err
		}
	}

	raw := format{Type: bufTypeOutputMplane}
	raw.Pix.Width = cfg.Width
	raw.Pix.Height = cfg.Height
	raw.Pix.PixelFormat = pixFmtYUV420
	raw.Pix.Field = fieldNone
	raw.Pix.NumPlanes = 1
	raw.Pix.PlaneFmt[0].BytesPerLine = cfg.Stride
	if err := ioctl(e.fd, vidiocSFmt, unsafe.Pointer(&raw)); err != nil {
		return fmt.Errorf("set encoder input format: %w", err)
	}
	if err := validateRawFormat(cfg, raw.Pix); err != nil {
		return err
	}
	e.sizeImage = raw.Pix.PlaneFmt[0].SizeImage

	compressed := format{Type: bufTypeCaptureMplane}
	compressed.Pix.Width = cfg.Width
	compressed.Pix.Height = cfg.Height
	compressed.Pix.PixelFormat = pixFmtH264
	compressed.Pix.NumPlanes = 1
	compressed.Pix.PlaneFmt[0].SizeImage = 512 << 10
	if err := ioctl(e.fd, vidiocSFmt, unsafe.Pointer(&compressed)); err != nil {
		return fmt.Errorf("set encoder H264 format: %w", err)
	}
	if compressed.Pix.PixelFormat != pixFmtH264 {
		return fmt.Errorf("encoder substituted output format %#x", compressed.Pix.PixelFormat)
	}

	rate := outputStreamParm{Type: bufTypeOutputMplane, Numerator: 1, Denominator: cfg.FrameRate}
	if err := ioctl(e.fd, vidiocSParm, unsafe.Pointer(&rate)); err != nil {
		return fmt.Errorf("set encoder frame rate: %w", err)
	}

	inputRequest := requestBuffers{Count: cfg.InputSlots, Type: bufTypeOutputMplane, Memory: memoryDmabuf}
	if err := ioctl(e.fd, vidiocReqBufs, unsafe.Pointer(&inputRequest)); err != nil {
		return fmt.Errorf("request encoder DMABUF input slots: %w", err)
	}
	if inputRequest.Count < cfg.InputSlots {
		return fmt.Errorf("encoder gave %d input slots, need %d", inputRequest.Count, cfg.InputSlots)
	}
	e.inputSlots = cfg.InputSlots

	outputRequest := requestBuffers{Count: cfg.OutputBuffers, Type: bufTypeCaptureMplane, Memory: memoryMmap}
	if err := ioctl(e.fd, vidiocReqBufs, unsafe.Pointer(&outputRequest)); err != nil {
		return fmt.Errorf("request encoder output buffers: %w", err)
	}
	if outputRequest.Count == 0 {
		return errors.New("encoder gave no output buffers")
	}

	for index := uint32(0); index < outputRequest.Count; index++ {
		var p plane
		b := buffer{Index: index, Type: bufTypeCaptureMplane, Memory: memoryMmap, Length: 1, Planes: &p}
		if err := ioctl(e.fd, vidiocQueryBuf, unsafe.Pointer(&b)); err != nil {
			return fmt.Errorf("query encoder output buffer %d: %w", index, err)
		}
		data, err := syscall.Mmap(e.fd, int64(uint32(p.M)), int(p.Length), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			return fmt.Errorf("map encoder output buffer %d: %w", index, err)
		}
		e.outputs = append(e.outputs, data)
		if err := ioctl(e.fd, vidiocQBuf, unsafe.Pointer(&b)); err != nil {
			return fmt.Errorf("queue encoder output buffer %d: %w", index, err)
		}
	}

	if err := e.streamOn(bufTypeOutputMplane); err != nil {
		return fmt.Errorf("start encoder input: %w", err)
	}
	e.inputOn = true
	if err := e.streamOn(bufTypeCaptureMplane); err != nil {
		return fmt.Errorf("start encoder output: %w", err)
	}
	e.captureOn = true
	return nil
}

func validateRawFormat(cfg Config, got pixFormatMplane) error {
	if got.PixelFormat != pixFmtYUV420 {
		return fmt.Errorf("encoder substituted input format %#x", got.PixelFormat)
	}
	if got.NumPlanes != 1 {
		return fmt.Errorf("encoder wants %d input planes, need 1", got.NumPlanes)
	}
	if got.Width != cfg.Width || got.Height != cfg.Height {
		return fmt.Errorf("encoder input size %dx%d, need %dx%d", got.Width, got.Height, cfg.Width, cfg.Height)
	}
	if got.PlaneFmt[0].BytesPerLine != cfg.Stride {
		return fmt.Errorf("encoder input stride %d, need %d", got.PlaneFmt[0].BytesPerLine, cfg.Stride)
	}
	return nil
}

func (e *Encoder) setControl(id uint32, value int32) error {
	c := control{ID: id, Value: value}
	if err := ioctl(e.fd, vidiocSCtrl, unsafe.Pointer(&c)); err != nil {
		return fmt.Errorf("set encoder control %#x: %w", id, err)
	}
	return nil
}

func (e *Encoder) streamOn(bufType uint32) error {
	return ioctl(e.fd, vidiocStreamOn, unsafe.Pointer(&bufType))
}

func (e *Encoder) streamOff(bufType uint32) error {
	return ioctl(e.fd, vidiocStreamOff, unsafe.Pointer(&bufType))
}

func (e *Encoder) SizeImage() uint32 {
	return e.sizeImage
}

func (e *Encoder) QueueInput(slot uint32, dmabufFd int, length uint32, bytesUsed uint32, timestamp syscall.Timeval) error {
	if slot >= e.inputSlots {
		return fmt.Errorf("input slot %d out of range (have %d)", slot, e.inputSlots)
	}
	if bytesUsed > length {
		return fmt.Errorf("bytes used %d exceeds buffer length %d", bytesUsed, length)
	}
	p := plane{BytesUsed: bytesUsed, Length: length, M: uint64(uint32(int32(dmabufFd)))}
	b := buffer{
		Index:     slot,
		Type:      bufTypeOutputMplane,
		Memory:    memoryDmabuf,
		Field:     fieldNone,
		Timestamp: timestamp,
		Length:    1,
		Planes:    &p,
	}
	if err := ioctl(e.fd, vidiocQBuf, unsafe.Pointer(&b)); err != nil {
		return fmt.Errorf("queue encoder input slot %d: %w", slot, err)
	}
	return nil
}

func (e *Encoder) DequeueInput() (uint32, bool, error) {
	var p plane
	b := buffer{Type: bufTypeOutputMplane, Memory: memoryDmabuf, Length: 1, Planes: &p}
	err := ioctl(e.fd, vidiocDQBuf, unsafe.Pointer(&b))
	if errors.Is(err, syscall.EAGAIN) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("dequeue encoder input: %w", err)
	}
	if b.Flags&bufFlagError != 0 {
		return b.Index, true, fmt.Errorf("encoder input slot %d: %w", b.Index, ErrBufferError)
	}
	return b.Index, true, nil
}

func (e *Encoder) DequeueOutput(w io.Writer) (int, bool, error) {
	var p plane
	b := buffer{Type: bufTypeCaptureMplane, Memory: memoryMmap, Length: 1, Planes: &p}
	err := ioctl(e.fd, vidiocDQBuf, unsafe.Pointer(&b))
	if errors.Is(err, syscall.EAGAIN) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("dequeue encoder output: %w", err)
	}
	if int(b.Index) >= len(e.outputs) {
		return 0, true, fmt.Errorf("encoder returned unknown output buffer %d", b.Index)
	}

	var written int
	var resultErr error
	data := e.outputs[b.Index]
	switch {
	case b.Flags&bufFlagError != 0:
		resultErr = fmt.Errorf("encoder output buffer %d: %w", b.Index, ErrBufferError)
	case int(p.BytesUsed) > len(data) || p.DataOffset > p.BytesUsed:
		resultErr = fmt.Errorf("encoder output buffer %d has invalid size %d offset %d", b.Index, p.BytesUsed, p.DataOffset)
	default:
		written, resultErr = w.Write(data[p.DataOffset:p.BytesUsed])
		if resultErr != nil {
			resultErr = fmt.Errorf("write encoded data: %w", resultErr)
		}
	}

	p.BytesUsed = 0
	p.DataOffset = 0
	b.Flags = 0
	if err := ioctl(e.fd, vidiocQBuf, unsafe.Pointer(&b)); err != nil {
		resultErr = errors.Join(resultErr, fmt.Errorf("requeue encoder output buffer %d: %w", b.Index, err))
	}
	return written, true, resultErr
}

func (e *Encoder) Close() error {
	if e.fd == -1 {
		return os.ErrClosed
	}
	var closeErr error
	if e.inputOn {
		if err := e.streamOff(bufTypeOutputMplane); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("stop encoder input: %w", err))
		}
		e.inputOn = false
	}
	if e.captureOn {
		if err := e.streamOff(bufTypeCaptureMplane); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("stop encoder output: %w", err))
		}
		e.captureOn = false
	}
	for index, data := range e.outputs {
		if err := syscall.Munmap(data); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("unmap encoder output %d: %w", index, err))
		}
	}
	e.outputs = nil
	if err := syscall.Close(e.fd); err != nil {
		closeErr = errors.Join(closeErr, fmt.Errorf("close encoder: %w", err))
	}
	e.fd = -1
	return closeErr
}

func ContainsPicture(data []byte) bool {
	for i := 0; i+3 < len(data); i++ {
		if data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 {
			nalType := data[i+3] & 31
			if nalType == 1 || nalType == 5 {
				return true
			}
		}
	}
	return false
}
