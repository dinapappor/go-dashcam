package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/dinapappor/go-dashcam/camera/v4l2"
	"github.com/dinapappor/go-dashcam/config"
	"github.com/dinapappor/go-dashcam/overlay"
	"github.com/dinapappor/go-dashcam/recording"
	"github.com/dinapappor/go-dashcam/video"
	"github.com/dinapappor/go-dashcam/video/v4l2m2m"
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

	err = os.MkdirAll(dashcamConfig.OutputDirectory, 0755)

	if err != nil {
		log.Printf("Could not create output directory: %v", err)
		return
	}

	overlayConfig := overlay.Config{Title: dashcamConfig.DashcamTitle}

	dashcamTitleImageGray, dashcamTitleImageAlpha := overlay.RenderTitle(overlayConfig)

	segmentDuration, err := dashcamConfig.GetSegmentDuration()

	if err != nil {
		log.Printf("Could not get segment duration: %v", err)
		return
	}

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
	if !v4l2.HasCapability(capabilities, v4l2.VideoCapture) {
		log.Printf("We do not have video capture capabilities")
		return
	}

	// Check for streaming capabilities.
	// This means buffers that we queue and requeue later
	if ! v4l2.HasCapability(capabilities, v4l2.StreamingFlag) {
		log.Printf("We do not have streaming capabilities")
		return
	}

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

	streamParams, err := v4l2.QueryVideoCaptureStreamParams(device.Fd())
	if err != nil {
		log.Printf("Unable to query for stream params, got: %s", err)
		return
	}


	if !v4l2.HasCapability(streamParams.Capture.Capability, v4l2.CapTimePerFrame) {
		log.Printf("We do not have TimePerFrame capability")
		return
	}

	_, err = v4l2.SetVideoCaptureStreamParams(device.Fd(), 1, dashcamConfig.VideoConfig.FrameRate)

	if err != nil {
		log.Printf("Unable to set stream params, got %s", err)
		return
	}

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
		OutputDirectory: dashcamConfig.OutputDirectory,
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
		err = recordHardware(
			ctx,
			device,
			hwEncoder,
			encoder,
			requestedFormat,
			dashcamTitleImageGray,
			dashcamTitleImageAlpha,
		)
	} else {
		err = recordSoftware(
			ctx,
			device,
			encoder,
			requestedFormat,
			dashcamConfig.VideoConfig.PixelFormat,
			inputConfig.PixelFormat,
			dashcamTitleImageGray,
			dashcamTitleImageAlpha,
		)
	}

	if err != nil {
		log.Printf("Recording failed: %v", err)
	}
}
