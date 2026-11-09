package v4l2

import (
	"syscall"
)
type V4L2Capability struct {
	Driver       [16]byte  // Driver name, stored as NUL-terminated bytes.
	Card         [32]byte  // Human-readable device name, stored as NUL-terminated bytes.
	BusInfo      [32]byte  // Device location on the bus, stored as NUL-terminated bytes.
	Version      uint32    // Packed V4L2 subsystem version: major, minor, and patch.
	Capabilities uint32    // Capability flags across the physical device's V4L2 nodes.
	DeviceCaps   uint32    // Flags for this node; valid when Capabilities includes V4L2_CAP_DEVICE_CAPS.
	Reserved     [3]uint32 // Reserved for future extensions; required for the kernel ABI.
}


type V4L2PixFormat struct {
	Width        uint32
	Height       uint32
	PixelFormat  uint32
	Field        uint32
	Bytesperline uint32
	Sizeimage    uint32
	Colorspace   uint32
	Priv         uint32
	Flags        uint32
	Encoding     uint32
	Quantization uint32
	Xferfunc     uint32
}

type V4L2Format struct {
	_                [0]uint64 // To align with kernel struct
	Type             uint32
	AlignmentPadding uint32
	PixFormat        V4L2PixFormat
	Remaining        [152]byte
}

type V4L2Fract struct {
	Numerator uint32
	Denominator uint32
}

type V4L2CaptureParm struct {
	Capability   uint32
	CaptureMode  uint32
	TimePerFrame V4L2Fract
	ExtendedMode uint32
	ReadBuffers  uint32
	Reserved     [4]uint32
}

type V4L2StreamParm struct {
	Type      uint32
	Capture   V4L2CaptureParm
	Remaining [160]byte
}

type V4L2RequestBuffers struct {
	Count        uint32
	Type         uint32
	Memory       uint32
	Capabilities uint32
	Flags        uint8
	Reserved     [3]uint8
}

type V4L2Timecode struct {
	Type     uint32
	Flags    uint32
	Frames   uint8
	Seconds  uint8
	Minutes  uint8
	Hours    uint8
	Userbits [4]uint8
}

type V4L2Buffer struct {
	Index      uint32
	Type       uint32
	BytesUsed  uint32
	Flags      uint32
	Field      uint32
	TimeStamp  syscall.Timeval
	Timecode   V4L2Timecode
	Sequence   uint32
	Memory     uint32
	M          V4L2BufferMemory
	Length     uint32
	Reserved2  uint32
	Reserved   uint32
}

type V4L2BufferMemory struct {
	_          [0]uint64
	Offset     uint32
	Remaining [4]byte
}

// needed for DMA-BUF on the PIs
type V4L2ExportBuffer struct {
	Type     uint32
	Index    uint32
	Plane    uint32
	Flags    uint32
	Fd       int32
	Reserved [11]uint32
}

type DmaBufSync struct {
	Flags uint64
}
