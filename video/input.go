package video

import (
	"fmt"
	"image"
)

type PixelFormat int

const (
	YUV422 PixelFormat = iota // yuvj422p, for mjpeg mode
	YUV420 // yuv420p - for when we go YUV mode
)


type InputConfig struct {
	Width       uint32
	Height      uint32
	FrameRate   uint32
	PixelFormat PixelFormat
}

func InputArgs(cfg InputConfig) ([]string, error) {

	if cfg.Width == 0 {
		return nil, fmt.Errorf("Width must be a positive number")
	}

	if cfg.Height == 0 {
		return nil, fmt.Errorf("Height must be a positive number")
	}

	if cfg.FrameRate == 0 {
		return nil, fmt.Errorf("FrameRate must be a positive number")
	}

	inputArgs := []string{"-f", "rawvideo", "-pix_fmt"}

	switch cfg.PixelFormat {
		case YUV422:
			inputArgs = append(inputArgs, "yuvj422p")
		case YUV420:
			inputArgs = append(inputArgs, "yuv420p")
		default:
			return nil, fmt.Errorf("Unknown pixelformat")
	}

	inputArgs = append(
		inputArgs,
		"-video_size",
		fmt.Sprintf("%dx%d", cfg.Width, cfg.Height),
		"-framerate",
		fmt.Sprint(cfg.FrameRate),
	)

	return inputArgs, nil
}

func ValidateSubsampleRatio(ycbcr *image.YCbCr, expected PixelFormat) error {
	// validate that inputs are okay
	// we'd rather end life gracefully
	// instead of panicing and die

	switch expected {
		case YUV422:
			if ycbcr.SubsampleRatio != image.YCbCrSubsampleRatio422 {
				return fmt.Errorf("Expected 422, got %T", ycbcr.SubsampleRatio)
			}
			return nil
		case YUV420:
			if ycbcr.SubsampleRatio != image.YCbCrSubsampleRatio420 {
				return fmt.Errorf("Expected 420, got %T", ycbcr.SubsampleRatio)
			}
			return nil
		default:
			return fmt.Errorf("non-supported pixelformat")
	}
}

// I do not understand this function - YET!
func appendPlane(dst []byte, src []byte, stride int, width int, height int) []byte {

	for row := 0; row < height; row++ {
		start := row * stride
		dst = append(dst, src[start:start+width]...)
	}

	return dst
}

func PackFrame(dst []byte, frame *image.YCbCr) ([]byte, error) {

	width := frame.Rect.Dx()
	height := frame.Rect.Dy()
	dst = appendPlane(dst[:0], frame.Y, frame.YStride, width, height)

	chromaWidth := (width + 1) / 2

	chromaHeight := height

	switch frame.SubsampleRatio {
		case image.YCbCrSubsampleRatio422:
			// Colour has the same number of rows as brightness.
		case image.YCbCrSubsampleRatio420:
			chromaHeight = (height + 1) / 2
		default:
			return nil, fmt.Errorf("unsupported subsample ratio: %v", frame.SubsampleRatio)
	}

	dst = appendPlane(dst, frame.Cb, frame.CStride, chromaWidth, chromaHeight)
	dst = appendPlane(dst, frame.Cr, frame.CStride, chromaWidth, chromaHeight)

	return dst, nil

}
