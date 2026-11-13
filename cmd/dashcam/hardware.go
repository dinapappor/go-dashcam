package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log"
	"time"

	"github.com/dinapappor/go-dashcam/camera/v4l2"
	"github.com/dinapappor/go-dashcam/overlay"
	"github.com/dinapappor/go-dashcam/video"
	"github.com/dinapappor/go-dashcam/video/v4l2m2m"
)

func recordHardware(ctx context.Context, device *v4l2.Device, hwEncoder *v4l2m2m.Encoder, encoder *video.Encoder, format v4l2.V4L2Format, titleGray *image.Gray, titleAlpha *image.Alpha, barGray *image.Gray, barAlpha *image.Alpha) error {
	held := make([]v4l2.Frame, len(device.Buffers()))
	inFlight := make([]bool, len(device.Buffers()))
	inFlightCount := 0

	pump := func() error {
		for {
			slot, ok, err := hwEncoder.DequeueInput()
			if !ok && err == nil {
				break
			}
			if !ok {
				return err
			}
			if int(slot) >= len(inFlight) || !inFlight[slot] {
				return fmt.Errorf("encoder returned slot %d that was not in flight", slot)
			}
			if err != nil {
				log.Printf("Encoder input: %v", err)
			}
			inFlight[slot] = false
			inFlightCount--
			err = device.Requeue(held[slot])
			if err != nil {
				return err
			}
		}
		for {
			_, ok, err := hwEncoder.DequeueOutput(encoder)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
		}
		return nil
	}

	stride := int(format.PixFormat.Bytesperline)
	width := int(format.PixFormat.Width)
	height := int(format.PixFormat.Height)
	const rw = v4l2.DmaBufSyncRead | v4l2.DmaBufSyncWrite

	for ctx.Err() == nil {
		err := pump()
		if err != nil {
			return fmt.Errorf("hardware encoder: %w", err)
		}
		if inFlightCount == len(held) {
			time.Sleep(time.Millisecond)
			continue
		}

		frame, err := device.NextFrame(ctx)
		if errors.Is(err, context.Canceled) {
			break
		}
		if err != nil {
			return fmt.Errorf("unable to get frame: %w", err)
		}
		if uint32(len(frame.Data)) < format.PixFormat.Sizeimage {
			err = device.Requeue(frame)
			if err != nil {
				return fmt.Errorf("could not requeue the buffer: %w", err)
			}
			continue
		}

		exportFd, err := device.ExportFd(frame)
		if err != nil {
			return fmt.Errorf("no export fd: %w", err)
		}
		err = v4l2.SyncBuffer(exportFd, v4l2.DmaBufSyncStart|rw)
		if err != nil {
			return fmt.Errorf("sync start: %w", err)
		}
		overlay.BlendOverlay(frame.Data[:stride*height], stride, width, height, titleGray, titleAlpha, 0, 0)

		overlay.BlendOverlay(frame.Data[:stride*height], stride, width, height, barGray, barAlpha, 0, height-barGray.Bounds().Dy())

		err = v4l2.SyncBuffer(exportFd, v4l2.DmaBufSyncEnd|rw)
		if err != nil {
			return fmt.Errorf("sync end: %w", err)
		}

		index := frame.Index()
		err = hwEncoder.QueueInput(index, exportFd, uint32(len(device.Buffers()[index])), hwEncoder.SizeImage(), frame.Timestamp())
		if err != nil {
			return fmt.Errorf("queue to encoder: %w", err)
		}
		held[index] = frame
		inFlight[index] = true
		inFlightCount++
	}

	deadline := time.Now().Add(2 * time.Second)
	lastOutputs := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		err := pump()
		if err != nil {
			return fmt.Errorf("hardware encoder drain: %w", err)
		}
		if inFlightCount == 0 && time.Now().After(lastOutputs) {
			break
		}
		if inFlightCount > 0 {
			lastOutputs = time.Now().Add(200 * time.Millisecond)
		}
		time.Sleep(time.Millisecond)
	}
	if inFlightCount > 0 {
		return fmt.Errorf("hardware encoder still held %d frames at shutdown", inFlightCount)
	}
	return nil
}
