package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"os/signal"
	"time"
	"github.com/dinapappor/go-dashcam/camera/v4l2"
	"github.com/dinapappor/go-dashcam/config"
	"github.com/dinapappor/go-dashcam/overlay"
	"github.com/dinapappor/go-dashcam/recording"
	"github.com/dinapappor/go-dashcam/video"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	dashcamConfig, err := config.ParseConfig("/home/andreas/dashcam_config.json")


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

	requestedFormat, err := v4l2.SetVideoCaptureFormat(
		device.Fd(),
		dashcamConfig.VideoConfig.Width,
		dashcamConfig.VideoConfig.Height,
		v4l2.PixFmtMJPEG,
	)

	if err != nil {
		log.Printf("Unable to set format: %s", err)
		return
	}

	log.Printf("We got format: %s, width:%d, height: %d",
		v4l2.DecodeFourCC(requestedFormat.PixFormat.PixelFormat),
		requestedFormat.PixFormat.Width,
		requestedFormat.PixFormat.Height)

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

	requestedParams, err := v4l2.SetVideoCaptureStreamParams(device.Fd(), 1, dashcamConfig.VideoConfig.FramesPerSecond)

	if err != nil {
		log.Printf("Unable to set stream params, got %s", err)
		return
	}


	log.Printf("%+v", requestedParams.Capture.TimePerFrame)

	err = device.SetupBuffers(4)

	if err != nil {
		log.Printf("Unable to setup buffers: %v", err)
		return
	}

	mode := recording.Continuous

	if dashcamConfig.VideoConfig.Segmented {
		mode = recording.Segmented
	}

	recordingConfig := recording.Config{
		Mode:            mode,
		SegmentDuration: segmentDuration,
		OutputDirectory: "videos",
		FrameRate:       dashcamConfig.VideoConfig.FramesPerSecond,
	}

	outputArgs, err := recording.OutputArgs(recordingConfig, time.Now())
	if err != nil {
		log.Printf("Outputargs: %v", err)
		return
	}

	outputArgs = append([]string{"-c:v", "libx264", "-pix_fmt", "yuv420p"}, outputArgs...)
	encoder, err := video.NewEncoder(
		[]string{"-f", "rawvideo", "-pix_fmt", "yuvj422p", "-video_size", "1280x720", "-framerate", fmt.Sprint(dashcamConfig.VideoConfig.FramesPerSecond)},
		outputArgs,
	)

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

	err = device.StreamOn()
	if err != nil {
		log.Printf("Unable to start stream: %v", err)
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

		// decode the image so we can draw title overlay over it
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

		ycbcr, ok := decoded.(*image.YCbCr)

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

		// Assembled needed stuff in one step before send off
		// to ffmpeg
		rawFrame = append(rawFrame[:0], ycbcr.Y...)
		rawFrame = append(rawFrame, ycbcr.Cb...)
		rawFrame = append(rawFrame, ycbcr.Cr...)

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
