package v4l2

import (
	"os"
	"errors"
	"testing"
	"unsafe"
	"syscall"
)

func TestV4L2CapabilitySize(t *testing.T) {
	got := unsafe.Sizeof(V4L2Capability{})

	if got != 104 {
		t.Errorf("V4L2Capability size = %d, want 104", got)
	}
}

// These are important because linux kernel expects them at certain
// byteoffsets
func TestV4L2CapabilityVersionOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2Capability{}.Version)

	if got != 80 {
		t.Errorf("VersionOffset = %d, want 80", got)
	}
}

func TestV4L2CapabilityCapabilitiesOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2Capability{}.Capabilities)

	if got != 84 {
		t.Errorf("CapabilitiesOffset = %d, want 84", got)
	}
}

func TestV4L2CapabilityDeviceCapsOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2Capability{}.DeviceCaps)

	if got != 88 {
		t.Errorf("DeviceCapOffset = %d, want 88", got)
	}
}

func TestV4L2CapabilityReservedOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2Capability{}.Reserved)

	if got != 92 {
		t.Errorf("ReservedOffset = %d, want 92", got)
	}
}

func TestSelectCapabilitiesUsesDeviceCaps(t *testing.T) {
        caps := V4L2Capability{
                Capabilities: CapDeviceCaps,
                DeviceCaps:   VideoCapture,
        }

        got := SelectCapabilities(caps)

        if got != VideoCapture {
                t.Errorf("got %#x, want %#x", got, VideoCapture)
        }
}

func TestSelectCapabilitiesUsesCapabilities(t *testing.T) {
        caps := V4L2Capability{
                Capabilities: VideoCapture,
                DeviceCaps:   0,
        }

        got := SelectCapabilities(caps)

        if got != VideoCapture {
                t.Errorf("got %#x, want %#x", got, VideoCapture)
        }
}

// test the vidicQueryCap const that is defined in main
func TestVIDIOCQueryCap(t *testing.T) {
	got := VidiocQueryCap

	if got != 0x80685600 {
		t.Errorf("vidicQueryCap = %#x, want x80685600", got)
	}
}

// Function tests
func TestConvertByteToStrings(t *testing.T) {
	got, err := ConvertByteToString([]byte{'u', 'v', 'c', 0, 0})

	if err != nil {
		t.Fatalf("Test failed, %s", err)
	}

	if got != "uvc" {
		t.Errorf("Expected uvc, got %s", got)
	}
}

func TestConvertByteToStringsMissingTerminator(t *testing.T) {
	_, err := ConvertByteToString([]byte{'u', 'v', 'c'})

	if err == nil {
		t.Fatalf("function should catch missing separator but it didnt")
	}

}

// Also test that it finds 0 correctly
func TestConvertBytesToStringReturnsEmptyStringWithout(t *testing.T) {
	got, err := ConvertByteToString([]byte{0})

	if err != nil {
		t.Fatalf("This should not happen, reason: %s", err)
	}
	if got != "" {
		t.Errorf("String was not empty, got %s", got)
	}
}

func TestHasCapabilityPresent(t *testing.T) {
	got := HasCapability(VideoCapture, VideoCapture)

	if !got {
		t.Errorf("Expected video capture capability to be present")
	}
}

func TestHasCapabilityAbsent(t *testing.T) {
	got := HasCapability(0, VideoCapture)

	if got {
		t.Errorf("Expected video capture capability to be absent")
	}
}

func TestHasCapabilityUnrelatedFlag(t *testing.T) {
	got := HasCapability(0x2, VideoCapture)

	if got {
		t.Errorf("Test hould fail if we gut an unrelated flag")
	}
}

func TestHasStreamingCapabilityPresent(t *testing.T) {
	got := HasCapability(StreamingFlag, StreamingFlag)

	if !got {
		t.Errorf("Expected streaming capability to be present")
	}
}

func TestHasCaptureCapabilityAbsent(t *testing.T) {
	got := HasCapability(VideoCapture, StreamingFlag)

	if got {
		t.Errorf("Expected streaming capability to be present")
	}
}


func TestV4L2PixFormatSize(t *testing.T) {
	got := unsafe.Sizeof(V4L2PixFormat{})

	if got != 48 {
		t.Errorf("V4L2PixFormat size = %d, want 48", got)
	}
}

func TestV4L2PixFormatEncodingOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2PixFormat{}.Encoding)

	if got != 36 {
		t.Errorf("V4L2PixFormat.Encoding offset = %d, want 36", got)
	}
}

func TestV4L2PixFormatQuantizationOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2PixFormat{}.Quantization)

	if got != 40 {
		t.Errorf("V4L2PixFormat.Quantization offset = %d, want 40", got)
	}
}

func TestV4L2FormatSize(t *testing.T) {
	got := unsafe.Sizeof(V4L2Format{})

	if got != 208 {
		t.Errorf("V4L2Format size = %d, want 208", got)
	}
}

func TestV4L2FormatPixformatOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2Format{}.PixFormat)

	if got != 8 {
		t.Errorf("V4L2Format.PixFormat offset = %d, want 8", got)
	}
}

func TestV4L2FormatAlignment(t *testing.T) {
	got := unsafe.Alignof(V4L2Format{})

	if got != 8 {
		t.Errorf("V4L2Format alignment = %d, want 8", got)
	}
}

func TestDecodeFourCCMJPG(t *testing.T) {
	format:= DecodeFourCC(0x47504A4D)

	if format != "MJPG" {
		t.Errorf("Expected format to be MJPG, got:%s", format)
	}
}

func TestDecodeFourCCYUYV(t *testing.T) {
	format:= DecodeFourCC(0x56595559)

	if format != "YUYV" {
		t.Errorf("Expected format to be YUYV, got:%s", format)
	}
}

func TestDecodeFourCCPreservesBytes(t *testing.T) {
	format:= DecodeFourCC(0x80434241)

	if format != "ABC\x80" {
		t.Errorf("Expected %q, got:%q", "ABC\x80", format)
	}
}


func TestVIDIOCGFMT(t *testing.T) {
	got := VidiocGFMT

	if got != 0xC0D05604 {
		t.Errorf("VidiocGFMT = %#x, want 0xC0D05604", got)
	}
}

func TestVIDIOCSFMT(t *testing.T) {
	got := VidiocSFMT

	if got != 0xC0D05605 {
		t.Errorf("VidiocSFMT = %#x, want 0xC0D05605", got)
	}
}

func TestValidateFormatCorrect(t *testing.T) {
	var orig V4L2Format
	orig.PixFormat.PixelFormat = PixFmtMJPEG
	orig.PixFormat.Width = 1280
	orig.PixFormat.Height = 720

	var actual V4L2Format
	actual.PixFormat.PixelFormat = PixFmtMJPEG
	actual.PixFormat.Width = 1280
	actual.PixFormat.Height = 720

	if ! ValidateFormat(orig, actual) {
		t.Errorf("Structs should match")
	}
}

func TestValidateFormatPixelFormatIncorrect(t *testing.T) {
	var orig V4L2Format
	orig.PixFormat.PixelFormat = PixFmtMJPEG
	orig.PixFormat.Width = 1280
	orig.PixFormat.Height = 720

	var actual V4L2Format
	actual.PixFormat.PixelFormat = PixFmtYUYV
	actual.PixFormat.Width = 1280
	actual.PixFormat.Height = 720

	if ValidateFormat(orig, actual) {
		t.Errorf("We should get mismatch on PixelFormat")
	}
}

func TestValidateFormatHeightIncorrect(t *testing.T) {
	var orig V4L2Format
	orig.PixFormat.PixelFormat = PixFmtMJPEG
	orig.PixFormat.Width = 1280
	orig.PixFormat.Height = 720

	var actual V4L2Format
	actual.PixFormat.PixelFormat = PixFmtMJPEG
	actual.PixFormat.Width = 1280
	actual.PixFormat.Height = 721

	if ValidateFormat(orig, actual) {
		t.Errorf("Structs should not match")
	}
}

func TestValidateFormatWidthIncorrect(t *testing.T) {
	var orig V4L2Format
	orig.PixFormat.PixelFormat = PixFmtMJPEG
	orig.PixFormat.Width = 1280
	orig.PixFormat.Height = 720

	var actual V4L2Format
	actual.PixFormat.PixelFormat = PixFmtMJPEG
	actual.PixFormat.Width = 1281
	actual.PixFormat.Height = 720

	if ValidateFormat(orig, actual) {
		t.Errorf("Structs should not match")
	}
}

func TestV4L2FractSize(t *testing.T) {
	got := unsafe.Sizeof(V4L2Fract{})

	if got != 8 {
		t.Errorf("V4L2Fract size = %d, want 8", got)
	}
}

func TestV4L2CaptureParmSize(t *testing.T) {
	got := unsafe.Sizeof(V4L2CaptureParm{})

	if got != 40 {
		t.Errorf("V4L2CaptureParm size = %d, want 40", got)
	}
}

func TestV4L2CaptureParmTimePerFrameOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2CaptureParm{}.TimePerFrame)

	// Two uint32 (4 bytes each is before this field
	// so we want 8 bytes
	if got != 8 {
		t.Errorf("VersionOffset = %d, want 8", got)
	}
}

func TestV4L2StreamParmSize(t *testing.T) {
	got := unsafe.Sizeof(V4L2StreamParm{})

	if got != 204 {
		t.Errorf("V4L2StreamParm size = %d, want 204", got)
	}
}

func TestV4L2StreamParmCaptureOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2StreamParm{}.Capture)

	if got != 4 {
		t.Errorf("CaptureOffset = %d, want 4", got)
	}
}

func TestV4L2StreamParmAlignment(t *testing.T) {
	got := unsafe.Alignof(V4L2StreamParm{})

	if got != 4 {
		t.Errorf("V4L2StreamParm alignment = %d, want 4", got)
	}
}


func TestVIDIOCGPARM(t *testing.T) {
	got := VidiocGPARM

	if got != 0xC0CC5615 {
		t.Errorf("VidiocGPARM = %#x, want 0xC0CC5615", got)
	}
}

func TestValidateFrameIntervalCorrect(t *testing.T ) {
	var orig V4L2Fract
	orig.Numerator = 1
	orig.Denominator = 30

	var actual V4L2Fract
	actual.Numerator = 1
	actual.Denominator = 30

	if ! ValidateFrameInterval(orig, actual) {
		t.Errorf("These should match")
	}
}

func TestValidateFrameIntervalDoubleFraction(t *testing.T) {
	var orig V4L2Fract
	orig.Numerator = 1
	orig.Denominator = 30

	var actual V4L2Fract
	actual.Numerator = 2
	actual.Denominator = 60

	if ! ValidateFrameInterval(orig, actual) {
		t.Errorf("These should match")
	}
}

func TestValidateFrameIntervalOrigNumeratorZero(t *testing.T) {

	var orig V4L2Fract
	orig.Numerator = 0
	orig.Denominator = 30

	var actual V4L2Fract
	actual.Numerator = 1
	actual.Denominator = 30

	if ValidateFrameInterval(orig, actual) {
		t.Errorf("A zero numerator should faild")
	}
}

func TestValidateFrameIntervalOrigDenominatorZero(t *testing.T) {

	var orig V4L2Fract
	orig.Numerator = 1
	orig.Denominator = 0

	var actual V4L2Fract
	actual.Numerator = 1
	actual.Denominator = 30

	if ValidateFrameInterval(orig, actual) {
		t.Errorf("A zero denominator should faild")
	}
}

func TestValidateFrameIntervalActualNumeratorZero(t *testing.T) {

	var orig V4L2Fract
	orig.Numerator = 1
	orig.Denominator = 30

	var actual V4L2Fract
	actual.Numerator = 0
	actual.Denominator = 30

	if ValidateFrameInterval(orig, actual) {
		t.Errorf("A zero numerator should fail")
	}
}

func TestValidateFrameIntervalActualDenominatorZero(t *testing.T) {

	var orig V4L2Fract
	orig.Numerator = 1
	orig.Denominator = 30

	var actual V4L2Fract
	actual.Numerator = 1
	actual.Denominator = 0

	if ValidateFrameInterval(orig, actual) {
		t.Errorf("A zero denominator should fail")
	}
}

func TestValidateFrameIntervalDifferentRate(t *testing.T) {

	var orig V4L2Fract
	orig.Numerator = 1
	orig.Denominator = 30

	var actual V4L2Fract
	actual.Numerator = 1
	actual.Denominator = 25

	if ValidateFrameInterval(orig, actual) {
		t.Errorf("Different frame intervals should be rejected.")
	}
}

func TestValidateFrameIntervalOverflow(t *testing.T) {

	var orig V4L2Fract
	orig.Numerator = 65536
	orig.Denominator = 1

	var actual V4L2Fract
	actual.Numerator = 65536
	actual.Denominator = 65537

	if ValidateFrameInterval(orig, actual) {
		t.Errorf("Different frame intervals should be rejected.")
	}
}

func TestV4L2RequestBuffersSize(t *testing.T) {
	got := unsafe.Sizeof(V4L2RequestBuffers{})

	if got != 20 {
		t.Errorf("V4L2RequestBuffers size = %d, want 20", got)
	}
}

func TestV4L2RequestBuffersFlagsOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2RequestBuffers{}.Flags)

	if got != 16 {
		t.Errorf("FlagsOffset = %d, want 16", got)
	}
}

func TestV4L2TimecodeSize(t *testing.T) {
	got := unsafe.Sizeof(V4L2Timecode{})

	if got != 16 {
		t.Errorf("V4L2Timecode size = %d, want 16", got)
	}
}

func TestTimeValSize(t *testing.T) {
	got := unsafe.Sizeof(syscall.Timeval{})

	if got != 16 {
		t.Errorf("syscall.Timeval size didn't match, got %d, expected 16", got)
	}
}

func TestV4L2BufferTimestampOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2Buffer{}.TimeStamp)

	if got != 24 {
		t.Errorf("Expected V4L2Buffer{}.TimeStamp offset to be 24, got %d", got)
	}
}

func TestV4L2BufferMemorySize(t *testing.T) {
	got := unsafe.Sizeof(V4L2BufferMemory{})

	if got != 8 {
		t.Errorf("Expected V4L2BufferMemory size to be 8, got %d", got)
	}
}

func TestV4L2BufferMOffset(t *testing.T) {
	got := unsafe.Offsetof(V4L2Buffer{}.M)

	if got != 64 {
		t.Errorf("Expected V4L2Buffer{}.TimeStamp offset to be 64, got %d", got)
	}
}

func TestV4L2BufferSize(t *testing.T) {
	got := unsafe.Sizeof(V4L2Buffer{})

	if got != 88 {
		t.Errorf("Expected V4l2Buffer to be 88 bytes, got %d bytes", got)
	}
}

func TestVIDIOCQueryBuf(t *testing.T) {
	got := VidiocQueryBuf

	if got != 0xC0585609 {
		t.Errorf("Expected to be 0xC0585609, got %+v", got)
	}
}

func TestVIDIOCQBuf(t *testing.T) {
	got := VidiocQBuf

	if got != 0xC058560F {
		t.Errorf("Expected vidiocQbuf to be 0xC058560F, got %+v", got)
	}
}

func TestVidiocStreamOn(t *testing.T) {
	got := VidiocStreamOn

	if got != 0x40045612 {
		t.Errorf("Expected VidiocStreamOn to be 0x40045612, got %+v", got)
	}
}

func TestVidiocStreamOff(t *testing.T) {
	got := VidiocStreamOff

	if got != 0x40045613 {
		t.Errorf("Expected VidiocStreamOff to be 0x40045613, got %+v", got)
	}
}

func TestVidiocDQBuf(t *testing.T) {
	got := VidiocDQBuf

	if got != 0xc0585611 {
		t.Errorf("Expected VidiocDQBuf to be 0xc0585611, got %+v", got)
	}
}

func TestFrameBytesValid(t *testing.T) {
	buffers := [][]byte{
		{10, 20, 30},
		{40, 50, 60},
	}

	got, err := FrameBytes(buffers, 1, 2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("Expected 2 bytes, got %d", len(got))
	}

	if got[0] != 40 || got[1] != 50 {
		t.Errorf("Expected [40 50], got %v", got)
	}
}

func TestFrameBytesInvalidIndex(t *testing.T) {
	buffers := [][]byte{
		{10, 20, 30},
		{40, 50, 60},
	}

	got, err := FrameBytes(buffers, 2, 1)

	if err == nil {
		t.Fatal("Expected an error for invalid index")
	}

	if got != nil {
		t.Errorf("Expected nil bytes, got %v", got)
	}
}

func TestFrameBytesTooLarge(t *testing.T) {
	buffers := [][]byte{
		{10, 20, 30},
		{40, 50, 60},
	}

	got, err := FrameBytes(buffers, 0, 4)

	if err == nil {
		t.Fatal("Expected an error for oversized frame")
	}

	if got != nil {
		t.Errorf("Expected nil bytes, got %v", got)
	}
}

func TestFrameExactBufferLength(t *testing.T) {
	buffers := [][]byte{
		{10, 20, 30},
		{40, 50, 60},
	}

	got, err := FrameBytes(buffers, 1, 3)


	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("Expected length 3, got %d", len(got))
	}

	if got[0] != 40 || got[1] != 50 || got[2] != 60 {
		t.Errorf("Expected [40 50 60], got %v", got)
	}
}

func TestFrameBytesZeroLength(t *testing.T) {
	buffers := [][]byte{
		{10, 20, 30},
		{40, 50, 60},
	}

	got, err := FrameBytes(buffers, 0, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("Expected 0 bytes, got %d", len(got))
	}
}

func TestQueryCapabilityNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	_, err = QueryCapability(file.Fd())

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestQueryCapabilityInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	_, err = QueryCapability(fd)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestQueryVideoCaptureFormatNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	_, err = QueryVideoCaptureFormat(file.Fd())

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestQueryVideoCaptureFormatInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	_, err = QueryVideoCaptureFormat(fd)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestSetVideoCaptureFormatNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	_, err = SetVideoCaptureFormat(file.Fd(), 1280, 720, PixFmtMJPEG)

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestSetVideoCaptureFormatInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	_, err = SetVideoCaptureFormat(fd, 1280, 720, PixFmtMJPEG)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestQueryVideoCaptureStreamParamsNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	_, err = QueryVideoCaptureStreamParams(file.Fd())

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestQueryVideoCaptureStreamParamsInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	_, err = QueryVideoCaptureStreamParams(fd)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestSetVideoCaptureStreamParamsNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	_, err = SetVideoCaptureStreamParams(file.Fd(), 1, 30)

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestSetVideoCaptureStreamParamsInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	_, err = SetVideoCaptureStreamParams(fd, 1, 30)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestRequestMmapBuffersNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	_, err = RequestMmapBuffers(file.Fd(), 4)

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestRequestMmapBuffersInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	_, err = RequestMmapBuffers(fd, 4)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestQueryBufferNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	_, err = QueryBuffer(file.Fd(), 1)

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestQueryBufferInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	_, err = QueryBuffer(fd, 1)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestMapBufferInvalidFD(t *testing.T) {
        file, err := os.CreateTemp(t.TempDir(), "fake")
        if err != nil {
                t.Fatalf("Unable to open temporary file: %v", err)
        }

        fd := file.Fd()
        file.Close()

        var buffer V4L2Buffer
        buffer.Length = 4096

        _, err = MapBuffer(fd, buffer)

        if !errors.Is(err, syscall.EBADF) {
                t.Fatalf("Expected EBADF, got %v", err)
        }
}

func TestMapBufferZeroLength(t *testing.T) {
        file, err := os.CreateTemp(t.TempDir(), "fake")
        if err != nil {
                t.Fatalf("Unable to open temporary file: %v", err)
        }
        defer file.Close()

        var buffer V4L2Buffer

        _, err = MapBuffer(file.Fd(), buffer)

        if !errors.Is(err, syscall.EINVAL) {
                t.Fatalf("Expected EINVAL, got %v", err)
        }
}

func TestMapBufferSuccess(t *testing.T) {
        file, err := os.CreateTemp(t.TempDir(), "fake")
        if err != nil {
                t.Fatalf("Unable to open temporary file: %v", err)
        }
        defer file.Close()

        if err := file.Truncate(4096); err != nil {
                t.Fatalf("Unable to resize temporary file: %v", err)
        }

        var buffer V4L2Buffer
        buffer.Length = 4096

        data, err := MapBuffer(file.Fd(), buffer)
        if err != nil {
                t.Fatalf("Expected success, got %v", err)
        }
        defer syscall.Munmap(data)

        if len(data) != 4096 {
                t.Fatalf("Expected 4096 bytes, got %d", len(data))
        }
}


func TestQueueBufferNotV4L2Device(t *testing.T) {
	testBuf := V4L2Buffer{}
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	err = QueueBuffer(file.Fd(), testBuf)

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestQueueBufferInvalidFD(t *testing.T) {
	testBuf := V4L2Buffer{}
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	err = QueueBuffer(fd, testBuf)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestDequeueBufferNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	_, err = DequeueBuffer(file.Fd())

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestDequeueBufferInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	_, err = DequeueBuffer(fd)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestStreamOnNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	err = StreamOn(file.Fd())

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestStreamOnInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	err = StreamOn(fd)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}

func TestStreamOffNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	defer file.Close()

	err = StreamOff(file.Fd())

	if ! errors.Is(err, syscall.ENOTTY) {
		t.Fatalf("This should fail with inappropriate ioctl")
	}
}

func TestStreamOffInvalidFD(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")

	if err != nil {
		t.Fatalf("Unable to open temporary file")
	}

	fd := file.Fd()
	file.Close()

	err = StreamOff(fd)

	if ! errors.Is(err, syscall.EBADF) {
		t.Fatalf("This should fail with bad file descriptor")
	}
}
