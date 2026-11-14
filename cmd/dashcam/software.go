package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"sync/atomic"

	"github.com/dinapappor/go-dashcam/camera/v4l2"
	"github.com/dinapappor/go-dashcam/overlay"
	"github.com/dinapappor/go-dashcam/video"
)

func recordSoftware(ctx context.Context, device *v4l2.Device, encoder *video.Encoder, format v4l2.V4L2Format, cameraPixelFormat string, inputPixelFormat video.PixelFormat, titleGray *image.Gray, titleAlpha *image.Alpha, bar *atomic.Pointer[barImage]) error {
	var rawFrame []byte

	for ctx.Err() == nil {
		frame, err := device.NextFrame(ctx)
		if errors.Is(err, context.Canceled) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("unable to get frame: %w", err)
		}

		ycbcr, err := frameImage(frame, format, cameraPixelFormat)
		if err != nil {
			log.Printf("Skipping frame: %v", err)
			err = device.Requeue(frame)
			if err != nil {
				return fmt.Errorf("could not requeue the buffer: %w", err)
			}
			continue
		}

		err = video.ValidateSubsampleRatio(ycbcr, inputPixelFormat)
		if err != nil {
			return err
		}

		// Dashcam title
		overlay.BlendOverlay(ycbcr.Y, ycbcr.YStride, ycbcr.Rect.Dx(), ycbcr.Rect.Dy(), titleGray, titleAlpha, 0, 0)

		// Load bottom bar from bar atomic pointer
		b := bar.Load()
		overlay.BlendOverlay(ycbcr.Y, ycbcr.YStride, ycbcr.Rect.Dx(), ycbcr.Rect.Dy(), b.gray, b.alpha, 0, ycbcr.Rect.Dy()-b.gray.Bounds().Dy())

		rawFrame, err = video.PackFrame(rawFrame, ycbcr)
		if err != nil {
			return fmt.Errorf("could not pack frame: %w", err)
		}

		err = encoder.WriteFrame(rawFrame)
		if err != nil {
			return fmt.Errorf("could not write data to encoder: %w", err)
		}

		err = device.Requeue(frame)
		if err != nil {
			return fmt.Errorf("could not requeue the buffer: %w", err)
		}
	}
	return nil
}

func frameImage(frame v4l2.Frame, format v4l2.V4L2Format, cameraPixelFormat string) (*image.YCbCr, error) {
	if cameraPixelFormat == "YU12" {
		return wrapYU12(frame.Data, format)
	}

	decoded, err := jpeg.Decode(bytes.NewReader(frame.Data))
	if err != nil {
		return nil, fmt.Errorf("unable to decode jpeg: %w", err)
	}
	ycbcr, ok := decoded.(*image.YCbCr)
	if !ok {
		return nil, fmt.Errorf("unexpected image type %T", decoded)
	}
	return ycbcr, nil
}

func wrapYU12(data []byte, format v4l2.V4L2Format) (*image.YCbCr, error) {
	width := int(format.PixFormat.Width)
	height := int(format.PixFormat.Height)
	yStride := int(format.PixFormat.Bytesperline)

	if yStride < width {
		return nil, fmt.Errorf("YU12 stride %d is smaller than width %d", yStride, width)
	}

	if yStride%2 != 0 {
		return nil, fmt.Errorf("YU12 stride %d is odd", yStride)
	}

	cStride := yStride / 2
	ySize := yStride * height
	cSize := cStride * ((height + 1) / 2)

	if len(data) < ySize+2*cSize {
		return nil, fmt.Errorf("YU12 frame too short: got %d bytes, want %d", len(data), ySize+2*cSize)
	}

	return &image.YCbCr{
		Y:              data[:ySize],
		Cb:             data[ySize : ySize+cSize],
		Cr:             data[ySize+cSize : ySize+2*cSize],
		YStride:        yStride,
		CStride:        cStride,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, width, height),
	}, nil
}
