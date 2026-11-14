package main

import (
	"context"
	"flag"
	"github.com/dinapappor/go-dashcam/config"
	"github.com/dinapappor/go-dashcam/overlay"
	"github.com/dinapappor/go-dashcam/recording"
	"github.com/dinapappor/go-dashcam/video"
	"github.com/dinapappor/go-dashcam/video/v4l2m2m"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
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

	titleOverlayConfig := overlay.Config{
		Text:        dashcamConfig.DashcamTitle,
		FrameHeight: int(dashcamConfig.VideoConfig.Height),
	}

	dashcamTitleImageGray, dashcamTitleImageAlpha, err := overlay.RenderOverlay(titleOverlayConfig)

	if err != nil {
		log.Printf("Unable to create dashcam title: %v", err)
		return
	}

	bottomBarOverlayConfig := overlay.Config{
		Text:        "No GPS fix",
		FrameHeight: int(dashcamConfig.VideoConfig.Height),
		Width:       int(dashcamConfig.VideoConfig.Width),
	}

	bottomBarImageGray, bottomBarImageAlpha, err := overlay.RenderOverlay(bottomBarOverlayConfig)

	if err != nil {
		log.Printf("Unable to create bottom bar: %v", err)
		return
	}

	var bottomBar atomic.Pointer[barImage]

	bottomBar.Store(
		&barImage{
			gray:  bottomBarImageGray,
			alpha: bottomBarImageAlpha,
		},
	)

	// gps loop
	go func() {
		// hardcoded for now, should be read from config later
		err := readGPS("/dev/i2c-1", 0x10, &bottomBar, bottomBarOverlayConfig)

		if err != nil {
			log.Printf("GPS stopped: %v", err)
		}
	}()

	segmentDuration, err := dashcamConfig.GetSegmentDuration()

	if err != nil {
		log.Printf("Could not get segment duration: %v", err)
		return
	}

	// see if we have hardware mode (v4l2m2m). This is /dev/video11 on the pi's
	hardware := dashcamConfig.VideoConfig.Encoder == "v4l2m2m"

	// open the video device
	device, requestedFormat, err := openCamera(dashcamConfig.VideoConfig, hardware)

	if err != nil {
		log.Printf("Unable to open camera: %v", err)
		return
	}

	defer func() {
		if err := device.Close(); err != nil {
			log.Printf("Unable to close device: %v", err)
		}
	}()

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

	if hardware {
		outputArgs = append([]string{"-c:v", "copy", "-bsf:v", "h264_metadata=video_full_range_flag=1"}, outputArgs...)
	} else {
		outputArgs = append([]string{"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p"}, outputArgs...)
	}

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
			&bottomBar,
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
			&bottomBar,
		)
	}

	if err != nil {
		log.Printf("Recording failed: %v", err)
	}
}
