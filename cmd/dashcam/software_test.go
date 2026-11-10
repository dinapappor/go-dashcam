package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"

	"github.com/dinapappor/go-dashcam/camera/v4l2"
)

// Helper function
func yu12Format(width, height, stride uint32) v4l2.V4L2Format {
	var format v4l2.V4L2Format

	format.PixFormat.Width = width
	format.PixFormat.Height = height
	format.PixFormat.Bytesperline = stride

	return format
}

func TestWrapYU12Planes(t *testing.T) {
	data := make([]byte, 12)
	for i := range data {
		data[i] = byte(i)
	}
	img, err := wrapYU12(data, yu12Format(4, 2, 4))

	if err != nil {
		t.Fatalf("wrapYU12 didn't succeed, got: %v", err)
	}

	if img.Y[0] != 0 {
		t.Errorf("Y should start at byte 0, got %d", img.Y[0])
	}

	if img.Cb[0] != 8 {
		t.Errorf("Cb should start at byte 8, got %d", img.Cb[0])
	}

	if img.Cr[0] != 10 {
		t.Errorf("Cr should start at byte 10, got %d", img.Cr[0])
	}
}

func TestWrapYU12ShortBuffer(t *testing.T) {
	data := make([]byte, 11)
	for i := range data {
		data[i] = byte(i)
	}
	img, err := wrapYU12(data, yu12Format(4, 2, 4))

	if err == nil {
		t.Fatalf("wrapYU12 shouldn't succeed with short buffer")
	}

	if img != nil {
		t.Fatalf("We should get a nil image when buffer is too short")
	}
}

func TestWrapYU12NoCopy(t *testing.T) {
	data := make([]byte, 12)

	img, err := wrapYU12(data, yu12Format(4, 2, 4))
	if err != nil {
		t.Fatalf("wrapYU12 should succeed, got: %v", err)
	}

	data[0] = 99

	if img.Y[0] != 99 {
		t.Errorf("Y should share memory with data, got %d", img.Y[0])
	}
}

func TestWrapYU12PaddedStride(t *testing.T) {
	data := make([]byte, 18)
	for i := range data {
		data[i] = byte(i)
	}

	img, err := wrapYU12(data, yu12Format(4, 2, 6))
	if err != nil {
		t.Fatalf("wrapYU12 should succeed, got: %v", err)
	}

	if img.YStride != 6 {
		t.Errorf("YStride should be 6, got %d", img.YStride)
	}

	if img.Rect.Dx() != 4 {
		t.Errorf("width should be 4, got %d", img.Rect.Dx())
	}

	if img.Cb[0] != 12 {
		t.Errorf("Cb should start at byte 12, got %d", img.Cb[0])
	}

	if img.Cr[0] != 15 {
		t.Errorf("Cr should start at byte 15, got %d", img.Cr[0])
	}
}

func TestWrapYU12TrailingBytes(t *testing.T) {
	data := make([]byte, 16)

	img, err := wrapYU12(data, yu12Format(4, 2, 4))
	if err != nil {
		t.Fatalf("wrapYU12 should accept extra trailing bytes, got: %v", err)
	}

	if len(img.Cr) != 2 {
		t.Errorf("Cr should be 2 bytes, got %d", len(img.Cr))
	}
}

func TestWrapYU12OddStride(t *testing.T) {
	data := make([]byte, 64)

	img, err := wrapYU12(data, yu12Format(5, 2, 5))
	if err == nil {
		t.Fatalf("wrapYU12 should fail for an odd stride")
	}

	if img != nil {
		t.Errorf("expected nil image for an odd stride")
	}
}

func TestWrapYU12StrideSmallerThanWidth(t *testing.T) {
	tests := []struct {
		name   string
		stride uint32
	}{
		{name: "zero stride", stride: 0},
		{name: "stride below width", stride: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, 64)

			img, err := wrapYU12(data, yu12Format(4, 2, tt.stride))
			if err == nil {
				t.Fatalf("wrapYU12 should fail for stride %d", tt.stride)
			}

			if img != nil {
				t.Errorf("expected nil image for stride %d", tt.stride)
			}
		})
	}
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()

	var buffer bytes.Buffer
	err := jpeg.Encode(&buffer, img, nil)
	if err != nil {
		t.Fatalf("Unable to encode test JPEG: %v", err)
	}
	return buffer.Bytes()
}

func TestFrameImageYU12(t *testing.T) {
	data := make([]byte, 12)
	frame := v4l2.Frame{Data: data}

	img, err := frameImage(frame, yu12Format(4, 2, 4), "YU12")
	if err != nil {
		t.Fatalf("frameImage should succeed for YU12, got: %v", err)
	}

	data[0] = 99

	if img.Y[0] != 99 {
		t.Errorf("YU12 frame should be wrapped without copying, got %d", img.Y[0])
	}
}

func TestFrameImageYU12ShortBuffer(t *testing.T) {
	frame := v4l2.Frame{Data: make([]byte, 11)}

	img, err := frameImage(frame, yu12Format(4, 2, 4), "YU12")
	if err == nil {
		t.Fatalf("frameImage should fail for a short YU12 buffer")
	}

	if img != nil {
		t.Errorf("expected nil image for a short YU12 buffer")
	}
}

func TestFrameImageJPEG(t *testing.T) {
	source := image.NewYCbCr(image.Rect(0, 0, 16, 8), image.YCbCrSubsampleRatio420)
	frame := v4l2.Frame{Data: encodeJPEG(t, source)}

	img, err := frameImage(frame, v4l2.V4L2Format{}, "MJPG")
	if err != nil {
		t.Fatalf("frameImage should decode a valid JPEG, got: %v", err)
	}

	if img.Rect.Dx() != 16 || img.Rect.Dy() != 8 {
		t.Errorf("decoded size should be 16x8, got %dx%d", img.Rect.Dx(), img.Rect.Dy())
	}
}

func TestFrameImageInvalidJPEG(t *testing.T) {
	frame := v4l2.Frame{Data: []byte("not a jpeg")}

	img, err := frameImage(frame, v4l2.V4L2Format{}, "MJPG")
	if err == nil {
		t.Fatalf("frameImage should fail for invalid JPEG data")
	}

	if img != nil {
		t.Errorf("expected nil image for invalid JPEG data")
	}
}

func TestFrameImageGrayJPEG(t *testing.T) {
	source := image.NewGray(image.Rect(0, 0, 16, 8))
	frame := v4l2.Frame{Data: encodeJPEG(t, source)}

	img, err := frameImage(frame, v4l2.V4L2Format{}, "MJPG")
	if err == nil {
		t.Fatalf("frameImage should reject a JPEG that is not YCbCr")
	}

	if img != nil {
		t.Errorf("expected nil image for a greyscale JPEG")
	}
}
