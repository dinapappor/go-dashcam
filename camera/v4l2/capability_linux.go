package v4l2

import (
	"bytes"
	"fmt"
	"log"
	"syscall"
	"unsafe"
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


func QueryCapability(fd uintptr)  (V4L2Capability, error) {
	var deviceCapability V4L2Capability

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, VidiocQueryCap, uintptr(unsafe.Pointer(&deviceCapability)))
	if errno != 0 {
		// Return Empty Struct because V4L2Capability is needed as
		// return type
		return V4L2Capability{}, fmt.Errorf("Unable to query device capabilities: %w", errno)
	}

	return deviceCapability, nil
}

func QueryVideoCaptureFormat(fd uintptr) (V4L2Format, error) {
	var deviceFormat V4L2Format
	deviceFormat.Type = BufTypeVideoCapture

	// query the ioctl for v4l2 device now
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, VidiocGFMT, uintptr(unsafe.Pointer(&deviceFormat)))
	if errno != 0 {
		return V4L2Format{}, fmt.Errorf("Unable to query for device format: %w", errno)
	}
	return deviceFormat, nil
}

func SetVideoCaptureFormat(fd uintptr, width uint32, height uint32, fourcc uint32) (V4L2Format, error) {
	var requestedFormat V4L2Format
	requestedFormat.Type = BufTypeVideoCapture
	requestedFormat.PixFormat.Width = width
	requestedFormat.PixFormat.Height = height
	requestedFormat.PixFormat.PixelFormat = fourcc

	origRequestedFormat := requestedFormat

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, VidiocSFMT, uintptr(unsafe.Pointer(&requestedFormat)))
	if errno !=0 {
		return V4L2Format{}, fmt.Errorf("Unable to set format. Got %w", errno)
	}

	if ! ValidateFormat(origRequestedFormat, requestedFormat) {
		return V4L2Format{}, fmt.Errorf("Requested format and returned format did not match")
	}

	return requestedFormat, nil
}

func QueryVideoCaptureStreamParams(fd uintptr) (V4L2StreamParm, error) {
	var streamParams V4L2StreamParm
	streamParams.Type = BufTypeVideoCapture

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, VidiocGPARM, uintptr(unsafe.Pointer(&streamParams)))

	if errno != 0 {
		return V4L2StreamParm{}, fmt.Errorf("Unable to query streamparams, got %w", errno)
	}
	return streamParams, nil
}

func SetVideoCaptureStreamParams(fd uintptr, numerator uint32, denominator uint32) (V4L2StreamParm, error) {
	var requestedParams V4L2StreamParm
	requestedParams.Type = BufTypeVideoCapture
	requestedParams.Capture.TimePerFrame.Numerator = numerator
	requestedParams.Capture.TimePerFrame.Denominator = denominator

	origParams := requestedParams

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, VidiocSPARM, uintptr(unsafe.Pointer(&requestedParams)))

	if errno != 0 {
		return V4L2StreamParm{}, fmt.Errorf("Unable to set stream params, got %w", errno)
	}

	if ! ValidateFrameInterval(origParams.Capture.TimePerFrame, requestedParams.Capture.TimePerFrame) {
		return V4L2StreamParm{}, fmt.Errorf("TimePerFrame validation failed")
	}
	return requestedParams, nil
}

func RequestMmapBuffers(fd uintptr, count uint32) (V4L2RequestBuffers, error) {
	var requestBuffers V4L2RequestBuffers

	requestBuffers.Count = count
	requestBuffers.Type = BufTypeVideoCapture
	requestBuffers.Memory = MemoryMmap

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, VidiocReqBufs, uintptr(unsafe.Pointer(&requestBuffers)))

	if errno != 0 {
		return V4L2RequestBuffers{}, fmt.Errorf("Unable to allocate request buffers, got %w", errno)
	}

	if requestBuffers.Count == 0 {
		return V4L2RequestBuffers{}, fmt.Errorf("Kernel gave back 0 request buffers")
	}

	return requestBuffers, nil
}

func QueryBuffer(fd uintptr, index uint32) (V4L2Buffer, error) {
	var buffer V4L2Buffer
	buffer.Type = BufTypeVideoCapture
	buffer.Index = index
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, VidiocQueryBuf, uintptr(unsafe.Pointer(&buffer)))

	if errno != 0 {
		return V4L2Buffer{}, fmt.Errorf("Unable to Query buffer, got %w", errno)
	}
	return buffer, nil
}

func MapBuffer(fd uintptr, buffer V4L2Buffer) ([]byte, error) {
	data, err := syscall.Mmap(
		int(fd),
		int64(buffer.M.Offset),
		int(buffer.Length),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)

	if err != nil {
		return nil, fmt.Errorf("Could not map buffer, got %w", err)
	}
	return data, nil
}

func QueueBuffer(fd uintptr, buffer V4L2Buffer) error {
	buffer.Memory = MemoryMmap
	buffer.Flags = 0

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		VidiocQBuf,
		uintptr(unsafe.Pointer(&buffer)),
	)

	if errno != 0 {
		return fmt.Errorf("Could not queue buffer %d: %w", buffer.Index, errno)
	}
	return nil
}

func DequeueBuffer(fd uintptr) (V4L2Buffer, error) {
	completedBuffer := V4L2Buffer{}
	completedBuffer.Type = BufTypeVideoCapture
	completedBuffer.Memory = MemoryMmap

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, VidiocDQBuf, uintptr(unsafe.Pointer(&completedBuffer)))

	if errno != 0 {
		return V4L2Buffer{}, fmt.Errorf("Errno: %w", errno)
	}

	return completedBuffer, nil
}
