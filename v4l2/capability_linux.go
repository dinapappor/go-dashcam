package v4l2

import (
	"bytes"
	"fmt"
	"log"
)

func SelectCapabilities(caps V4L2Capability) uint32 {
	// Use DeviceCaps if its flag is set, otherwise use Capabilities.
	if caps.Capabilities&CapDeviceCaps != 0 {
		return caps.DeviceCaps
	} else {
		return caps.Capabilities
	}
}

func ConvertByteToString(data []byte) (string, error) {
	// Find the first 0 in data and returns it's position
	end := bytes.IndexByte(data, 0)

	if end == -1 {
		return "", fmt.Errorf("No zero terminator found!")
	}

	// we already know it was good, so we just return the answer
	return string(data[:end]), nil
}

func HasCapability(capabilities uint32, flag uint32) bool {
	return  capabilities&flag != 0
}

func DecodeFourCC(format uint32) string {

	first := byte(format & 0xFF)
	second := byte((format >> 8) & 0xFF)
	third := byte((format >> 16) & 0xFF)
	fourth := byte((format >> 24) & 0xFF)

	return string([]byte{first, second, third, fourth})
}

func ValidateFormat(requested V4L2Format, actual V4L2Format) bool {
	// requested is what we requested
	// actual is what kernel returned to us
	errors := 0

	if actual.PixFormat.PixelFormat != requested.PixFormat.PixelFormat {
		log.Printf("We requested format %s but got %s.",
			DecodeFourCC(requested.PixFormat.PixelFormat),
			DecodeFourCC(actual.PixFormat.PixelFormat))
		errors++
	}

	if actual.PixFormat.Width != requested.PixFormat.Width {
		log.Printf("We requested width %d but got %d.",
			requested.PixFormat.Width,
			actual.PixFormat.Width)
		errors++
	}

	if actual.PixFormat.Height != requested.PixFormat.Height {
		log.Printf("We requested height %d but got %d.",
			requested.PixFormat.Height,
			actual.PixFormat.Height)
		errors++
	}

	if errors > 0 {
		return false
	}
	return true
}

func ValidateFrameInterval(orig V4L2Fract, actual V4L2Fract) bool {
	errors := 0

	left := uint64(orig.Numerator) * uint64(actual.Denominator)
	right := uint64(actual.Numerator) * uint64(orig.Denominator)

	if left != right {
		log.Printf("left and right does not match")
		errors++
	}
	if orig.Numerator == 0 {
		log.Printf("Requested numerator must be non-zero")
		errors++
	}

	if actual.Numerator == 0 {
		log.Printf("Actual numerator is zero")
		errors++
	}

	if orig.Denominator == 0 {
		log.Printf("Requested numerator must be non-zero")
		errors++
	}

	if actual.Denominator == 0 {
		log.Printf("Actual numerator is zero")
		errors++
	}


	if errors > 0 {
		log.Printf("FrameInterval validation failed")
		return false
	}

	return true
}

func FrameBytes(buffers [][]byte, index uint32, bytesUsed uint32) ([]byte, error) {
	if uint64(index) >= uint64(len(buffers)) {
		return nil, fmt.Errorf("buffer index out of range")
	}

	if uint64(bytesUsed) > uint64(len(buffers[index])) {
		return nil, fmt.Errorf("BytesUsed exceeds size")
	}

	return buffers[index][:bytesUsed], nil
}

