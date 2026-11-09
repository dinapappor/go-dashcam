package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/dinapappor/go-dashcam/camera/v4l2"
	"github.com/dinapappor/go-dashcam/config"
	"github.com/dinapappor/go-dashcam/overlay"
	"github.com/dinapappor/go-dashcam/recording"
	"github.com/dinapappor/go-dashcam/video"
	"github.com/dinapappor/go-dashcam/video/v4l2m2m"
	"image"
	"image/jpeg"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	cpuProfile := flag.String("cpuprofile", "", "write a CPU profile to this file")
	configFile := flag.String("config", "dashcam_config.json", "Config file to use")
	flag.Parse()

	if *cpuProfile != "" {
		stopProfile, err := startCPUProfile(*cpuProfile)
		if err != nil {
			log.Printf("Unable to start CPU profile: %v", err)
			return
		}
		defer func() {
			if err := stopProfile(); err != nil {
				log.Printf("Unable to close CPU profile: %v", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	dashcamConfig, err := config.ParseConfig(*configFile)

	if err != nil {
		log.Printf("Could not open config file %v", err)
		return
	}

	overlayConfig := overlay.Config{Title: dashcamConfig.DashcamTitle}

	dashcamTitleImageGray, dashcamTitleImageAlpha := overlay.RenderTitle(overlayConfig)

	segmentDuration, err := dashcamConfig.GetSegmentDuration()

	if err != nil {
		log.Printf("Could not get segment duration: %v", err)
		return
	}

	log.Printf("duration: %s", segmentDuration)

	log.Printf("%+v\n", dashcamConfig)

	// open the video device
	device, err := v4l2.Open(dashcamConfig.VideoConfig.VideoSource)

	if err != nil {
		log.Printf("%s: %s", dashcamConfig.VideoConfig.VideoSource, err)
		return
	}

	defer func() {
		if err := device.Close(); err != nil {
			log.Printf("Unable to close device: %v", err)
		}
	}()

	deviceCapability, err := v4l2.QueryCapability(device.Fd())

	if err != nil {
		log.Printf("Unable to query for device Capabilities, err:%s", err)
		return
	}

	log.Printf("Device Capability struct:%+v", deviceCapability)

	driverName, err := v4l2.ConvertByteToString(deviceCapability.Driver[:])

	if err != nil {
		log.Printf("Could not get driver name, err: %s", err)
		return
	}

	cameraName, err := v4l2.ConvertByteToString(deviceCapability.Card[:])

	if err != nil {
		log.Printf("Could not get camera name, err: %s", err)
		return
	}

	busInfo, err := v4l2.ConvertByteToString(deviceCapability.BusInfo[:])

	if err != nil {
		log.Printf("Could not get bus info, err: %s", err)
		return
	}

	fmt.Printf("Video device: %s (%s), using %s driver on %s\n", cameraName, dashcamConfig.VideoConfig.VideoSource, driverName, busInfo)

	capabilities := v4l2.SelectCapabilities(deviceCapability)

	// Check for video capture flag
	if v4l2.HasCapability(capabilities, v4l2.VideoCapture) {
		log.Printf("We have video capture capabilities")

	} else {
		log.Printf("We do not have video capture capabilities")
		return
	}

	// Check for streaming capabilities.
	// This means buffers that we queue and requeue later
	if v4l2.HasCapability(capabilities, v4l2.StreamingFlag) {
		log.Printf("We have streaming capabilities")
	} else {
		log.Printf("We do not have streaming capabilities")
		return
	}

	deviceFormat, err := v4l2.QueryVideoCaptureFormat(device.Fd())

	if err != nil {
		log.Printf("Unable to query video format: %s", err)
		return
	}

	log.Printf("%+v", deviceFormat.PixFormat)

	// Get the four CC
	fourCC := v4l2.DecodeFourCC(deviceFormat.PixFormat.PixelFormat)

	log.Printf("Pixformat: %s", fourCC)

	// To hold which format we request from camera
	var requestedFourCC uint32
	switch dashcamConfig.VideoConfig.PixelFormat {
	case "MJPG":
		requestedFourCC = v4l2.PixFmtMJPEG
	case "YU12":
		requestedFourCC = v4l2.PixFmtYUV420
	}

	requestedFormat, err := v4l2.SetVideoCaptureFormat(
		device.Fd(),
		dashcamConfig.VideoConfig.Width,
		dashcamConfig.VideoConfig.Height,
		requestedFourCC,
	)

	if err != nil {
		log.Printf("Unable to set format: %s", err)
		return
	}

	log.Printf("We got format: %s, width:%d, height: %d",
		v4l2.DecodeFourCC(requestedFormat.PixFormat.PixelFormat),
		requestedFormat.PixFormat.Width,
		requestedFormat.PixFormat.Height)

	log.Printf("%+v", requestedFormat)
	streamParams, err := v4l2.QueryVideoCaptureStreamParams(device.Fd())
	if err != nil {
		log.Printf("Unable to query for stream params, got: %s", err)
		return
	}

	log.Printf("%v", streamParams.Capture.TimePerFrame)

	if v4l2.HasCapability(streamParams.Capture.Capability, v4l2.CapTimePerFrame) {
		log.Printf("We have TimePerFrame capability")
	} else {
		log.Printf("We do not have TimePerFrame capability")
		return
	}

	requestedParams, err := v4l2.SetVideoCaptureStreamParams(device.Fd(), 1, dashcamConfig.VideoConfig.FrameRate)

	if err != nil {
		log.Printf("Unable to set stream params, got %s", err)
		return
	}

	log.Printf("%+v", requestedParams.Capture.TimePerFrame)

	// see if we have hardware mode (v4l2m2m). This is /dev/video11 on the pi's
	hardware := dashcamConfig.VideoConfig.Encoder == "v4l2m2m"

	err = device.SetupBuffers(4)

	if err != nil {
		log.Printf("Unable to setup buffers: %v", err)
		return
	}

	if hardware {
		err := device.ExportBuffers()

		if err != nil {
			log.Printf("Could not setup export buffers: %v", err)
			return
		}
	}

	mode := recording.Continuous

	if dashcamConfig.VideoConfig.Segmented {
		mode = recording.Segmented
	}

	recordingConfig := recording.Config{
		Mode:            mode,
		SegmentDuration: segmentDuration,
		OutputDirectory: "videos",
		FrameRate:       dashcamConfig.VideoConfig.FrameRate,
		StreamCopy:      hardware,
	}

	outputArgs, err := recording.OutputArgs(recordingConfig, time.Now())
	if err != nil {
		log.Printf("Outputargs: %v", err)
		return
	}

	inputPixelFormat := video.YUV422

	if dashcamConfig.VideoConfig.PixelFormat == "YU12" {
		inputPixelFormat = video.YUV420
	}

	inputConfig := video.InputConfig{
		Width:       dashcamConfig.VideoConfig.Width,
		Height:      dashcamConfig.VideoConfig.Height,
		FrameRate:   dashcamConfig.VideoConfig.FrameRate,
		PixelFormat: inputPixelFormat,
	}

	var inputArgs []string

	if hardware {
		inputArgs, err = video.H264InputArgs(dashcamConfig.VideoConfig.FrameRate)
	} else {
		inputArgs, err = video.InputArgs(inputConfig)
	}

	if err != nil {
		log.Printf("InputArgs: %v", err)
		return
	}

	// Find a way to  make the pi use h264_v4l2m2m on pi3/pi4/pizero2w
	if hardware {
		outputArgs = append([]string{"-c:v", "copy"}, outputArgs...)
	} else {
		outputArgs = append([]string{"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p"}, outputArgs...)
	}
	//outputArgs = append([]string{"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p"}, outputArgs...)
	encoder, err := video.NewEncoder(inputArgs, outputArgs)

	if err != nil {
		log.Printf("Unable to start encoder: %v", err)
		return
	}

	monitorResult := make(chan error, 1)

	go func() {
		err := recording.Monitor(ctx, recordingConfig.OutputDirectory, 10*time.Second)
		if err != nil {
			// stop is from the context
			stop()
		}
		monitorResult <- err
	}()

	defer func() {
		stop()
		if err := <-monitorResult; err != nil {
			log.Printf("Recording monitor failed: %v", err)
		}
		if err = encoder.Close(); err != nil {
			log.Printf("FFmpeg did not finish correctly: %v", err)
		}
	}()

	var hwEncoder *v4l2m2m.Encoder
	if hardware {
		hwEncoder, err = v4l2m2m.Open(v4l2m2m.Config{
			Device:           dashcamConfig.VideoConfig.EncoderDevice,
			Width:            dashcamConfig.VideoConfig.Width,
			Height:           dashcamConfig.VideoConfig.Height,
			Stride:           requestedFormat.PixFormat.Bytesperline,
			FrameRate:        dashcamConfig.VideoConfig.FrameRate,
			Bitrate:          dashcamConfig.VideoConfig.Bitrate,
			KeyframeInterval: int32(segmentDuration.Seconds() * float64(dashcamConfig.VideoConfig.FrameRate)),
			InputSlots:       uint32(len(device.Buffers())),
			OutputBuffers:    4,
		})
		if err != nil {
			log.Printf("Unable to open hardware encoder: %v", err)
			return
		}
		defer func() {
			if err := hwEncoder.Close(); err != nil {
				log.Printf("Hardware encoder close: %v", err)
			}
		}()
	}

	err = device.StreamOn()
	if err != nil {
		log.Printf("Unable to start stream: %v", err)
		return
	}

	if hardware {
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

		stride := int(requestedFormat.PixFormat.Bytesperline)
		width := int(requestedFormat.PixFormat.Width)
		height := int(requestedFormat.PixFormat.Height)
		const rw = v4l2.DmaBufSyncRead | v4l2.DmaBufSyncWrite

		for frameNumber := 0; frameNumber < 900 && ctx.Err() == nil; {
			err := pump()
			if err != nil {
				log.Printf("Hardware encoder: %v", err)
				return
			}
			if inFlightCount == len(held) {
				time.Sleep(time.Millisecond)
				continue
			}

			frame, err := device.NextFrame(ctx)
			if err != nil {
				log.Printf("Unable to get frame: %v", err)
				return
			}
			if uint32(len(frame.Data)) < requestedFormat.PixFormat.Sizeimage {
				err = device.Requeue(frame)
				if err != nil {
					log.Printf("Could not requeue the buffer: %v", err)
					return
				}
				continue
			}

			exportFd, err := device.ExportFd(frame)
			if err != nil {
				log.Printf("No export fd: %v", err)
				return
			}
			err = v4l2.SyncBuffer(exportFd, v4l2.DmaBufSyncStart|rw)
			if err != nil {
				log.Printf("Sync start: %v", err)
				return
			}
			overlay.BlendTitle(frame.Data[:stride*height], stride, width, height, dashcamTitleImageGray, dashcamTitleImageAlpha, 0, 0)
			err = v4l2.SyncBuffer(exportFd, v4l2.DmaBufSyncEnd|rw)
			if err != nil {
				log.Printf("Sync end: %v", err)
				return
			}

			index := frame.Index()
			err = hwEncoder.QueueInput(index, exportFd, uint32(len(device.Buffers()[index])), hwEncoder.SizeImage(), frame.Timestamp())
			if err != nil {
				log.Printf("Queue to encoder: %v", err)
				return
			}
			held[index] = frame
			inFlight[index] = true
			inFlightCount++
			frameNumber++
		}

		deadline := time.Now().Add(2 * time.Second)
		lastOutputs := time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) {
			err := pump()
			if err != nil {
				log.Printf("Hardware encoder drain: %v", err)
				return
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
			log.Printf("Hardware encoder still held %d frames at shutdown", inFlightCount)
		}
		return
	}

	// Used for the decoded image
	var rawFrame []byte

	for frameNumber := 0; frameNumber < 900 && ctx.Err() == nil; {
		// Get frames
		frame, err := device.NextFrame(ctx)

		if errors.Is(err, context.Canceled) {
			log.Printf("Shutting down")
			return
		}

		if err != nil {
			log.Printf("Unable to get frame. Got %+v", err)
			return
		}

		var ycbcr *image.YCbCr
		// decode the image so we can draw title overlay over it
		if dashcamConfig.VideoConfig.PixelFormat == "YU12" {
			width := int(requestedFormat.PixFormat.Width)
			height := int(requestedFormat.PixFormat.Height)
			yStride := int(requestedFormat.PixFormat.Bytesperline)
			cStride := yStride / 2
			ySize := yStride * height
			cSize := cStride * ((height + 1) / 2)

			if len(frame.Data) < ySize+2*cSize {
				log.Printf("YU12 frame too short: got %d bytes, want %d", len(frame.Data), ySize+2*cSize)
				err = device.Requeue(frame)
				if err != nil {
					log.Printf("Could not requeue the buffer: %v", err)
					return
				}

				continue
			}

			ycbcr = &image.YCbCr{
				Y:              frame.Data[:ySize],
				Cb:             frame.Data[ySize : ySize+cSize],
				Cr:             frame.Data[ySize+cSize : ySize+2*cSize],
				YStride:        yStride,
				CStride:        cStride,
				SubsampleRatio: image.YCbCrSubsampleRatio420,
				Rect:           image.Rect(0, 0, width, height),
			}
		} else {
			// Assume mjpeg for now
			decoded, err := jpeg.Decode(bytes.NewReader(frame.Data))

			if err != nil {
				log.Printf("Unable to decode jpeg: %v", err)
				err = device.Requeue(frame)
				if err != nil {
					log.Printf("Could not requeue the buffer: %v", err)
					return
				}

				continue
			}

			var ok bool

			ycbcr, ok = decoded.(*image.YCbCr)

			if !ok {
				log.Printf("Unexpexted image type %T", decoded)

				// Requeue the buffer again
				err = device.Requeue(frame)
				if err != nil {
					log.Printf("Could not requeue the buffer: %v", err)
					return
				}

				continue
			}
		}

		err = video.ValidateSubsampleRatio(ycbcr, inputConfig.PixelFormat)

		if err != nil {
			log.Print(err)
			return
		}

		overlay.BlendTitle(
			ycbcr.Y,
			ycbcr.YStride,
			ycbcr.Rect.Dx(),
			ycbcr.Rect.Dy(),
			dashcamTitleImageGray,
			dashcamTitleImageAlpha,
			0,
			0,
		)

		rawFrame, err = video.PackFrame(rawFrame, ycbcr)
		if err != nil {
			log.Printf("Could not pack frame: %v", err)
			return
		}

		err = encoder.WriteFrame(rawFrame)
		if err != nil {
			log.Printf("Could not write data to encoder: %v", err)
			return
		}
		err = device.Requeue(frame)
		if err != nil {
			log.Printf("Could not requeue the buffer: %v", err)
			return
		}
		frameNumber++
	}
}
