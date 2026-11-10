package video

import (
	"fmt"
	"slices"
	"testing"
)

func TestInputArgsSuccessYUV422(t *testing.T) {
	i := InputConfig{Width: 1280, Height: 720, FrameRate: 30, PixelFormat: YUV422}

	inputArgs, err := InputArgs(i)

	if err != nil {
		t.Fatalf("Unable to compile input args: %v", err)
	}

	fmt.Printf("%v", inputArgs)

	expected := []string{
		"-f",
		"rawvideo",
		"-pix_fmt",
		"yuvj422p",
		"-video_size",
		"1280x720",
		"-framerate",
		"30",
	}

	if !slices.Equal(inputArgs, expected) {
		t.Fatalf("TestInputArgsSuccessYUV422, inputArgs is incorrect, got %v", inputArgs)
	}
}

func TestInputArgsSuccessYUV420(t *testing.T) {
	i := InputConfig{Width: 1920, Height: 1080, FrameRate: 24, PixelFormat: YUV420}

	inputArgs, err := InputArgs(i)

	if err != nil {
		t.Fatalf("Unable to compile input args: %v", err)
	}

	fmt.Printf("%v", inputArgs)

	expected := []string{
		"-f",
		"rawvideo",
		"-pix_fmt",
		"yuvj420p",
		"-video_size",
		"1920x1080",
		"-framerate",
		"24",
	}

	if !slices.Equal(inputArgs, expected) {
		t.Fatalf("TestInputArgsSuccessYUV420, inputArgs is incorrect, got %v", inputArgs)
	}
}

func TestInputConfigWidthZero(t *testing.T) {
	i := InputConfig{Width: 0, Height: 1080, FrameRate: 24, PixelFormat: YUV420}

	inputArgs, err := InputArgs(i)

	if err == nil {
		t.Fatalf("Expected an error due to having zero width")
	}

	if inputArgs != nil {
		t.Fatalf("Expected inputArgs to be nil if width was zero")
	}
}

func TestInputConfigHeightZero(t *testing.T) {
	i := InputConfig{Width: 1920, Height: 0, FrameRate: 24, PixelFormat: YUV420}

	inputArgs, err := InputArgs(i)

	if err == nil {
		t.Fatalf("Expected an error due to having zero height")
	}

	if inputArgs != nil {
		t.Fatalf("Expected inputArgs to be nil if height was zero")
	}
}

func TestInputConfigFrameRateZero(t *testing.T) {
	i := InputConfig{Width: 1920, Height: 1080, FrameRate: 0, PixelFormat: YUV420}

	inputArgs, err := InputArgs(i)

	if err == nil {
		t.Fatalf("Expected an error fail due to having zero FrameRate")
	}

	if inputArgs != nil {
		t.Fatalf("Expected inputArgs to be nil if FrameRate was zero")
	}
}

func TestInputConfigPixelFormatIncorect(t *testing.T) {
	i := InputConfig{Width: 1920, Height: 1080, FrameRate: 24, PixelFormat: 42}

	inputArgs, err := InputArgs(i)

	if err == nil {
		t.Fatalf("Expected an error due to having non-existing PixelFormat")
	}

	if inputArgs != nil {
		t.Fatalf("Expected inputArgs to be nil if PixelFormat is incorrect")
	}
}

func TestH264InputArgsFrameRateZero(t *testing.T) {
	res, err := H264InputArgs(0)
	if err == nil {
		t.Fatalf("A framerate of zero should fail")
	}
	if res != nil {
		t.Fatalf("The resulting string slice should be nil")
	}
}

func TestH264InputArgsSuccess(t *testing.T) {
	res, err := H264InputArgs(30)
	if err != nil {
		t.Fatalf("We do not expect an error for a correct framerate, got: %v", err)
	}
	want := []string{"-f", "h264", "-framerate", "30"}
	if !slices.Equal(want, res) {
		t.Fatalf("The resulting string slice differs. Got %v, want %v", res, want)
	}
}
