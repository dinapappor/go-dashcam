package v4l2m2m

import "syscall"

type capability struct {
	Driver       [16]byte
	Card         [32]byte
	BusInfo      [32]byte
	Version      uint32
	Capabilities uint32
	DeviceCaps   uint32
	Reserved     [3]uint32
}

type planePixFormat struct {
	SizeImage    uint32
	BytesPerLine uint32
	Reserved     [6]uint16
}

type pixFormatMplane struct {
	Width        uint32
	Height       uint32
	PixelFormat  uint32
	Field        uint32
	Colorspace   uint32
	PlaneFmt     [8]planePixFormat
	NumPlanes    uint8
	Flags        uint8
	YCbCrEnc     uint8
	Quantization uint8
	XferFunc     uint8
	Reserved     [7]uint8
}

type format struct {
	Type uint32
	_    uint32
	Pix  pixFormatMplane
	_    [8]byte
}

type requestBuffers struct {
	Count        uint32
	Type         uint32
	Memory       uint32
	Capabilities uint32
	Flags        uint8
	Reserved     [3]uint8
}

type plane struct {
	BytesUsed  uint32
	Length     uint32
	M          uint64
	DataOffset uint32
	Reserved   [11]uint32
}

type timecode struct {
	Type     uint32
	Flags    uint32
	Frames   uint8
	Seconds  uint8
	Minutes  uint8
	Hours    uint8
	UserBits [4]uint8
}

type buffer struct {
	Index     uint32
	Type      uint32
	BytesUsed uint32
	Flags     uint32
	Field     uint32
	Timestamp syscall.Timeval
	Timecode  timecode
	Sequence  uint32
	Memory    uint32
	Planes    *plane
	Length    uint32
	Reserved2 uint32
	RequestFd uint32
}

type control struct {
	ID    uint32
	Value int32
}

type outputStreamParm struct {
	Type        uint32
	Capability  uint32
	OutputMode  uint32
	Numerator   uint32
	Denominator uint32
	Reserved    [46]uint32
}
