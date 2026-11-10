package main

import (
	"fmt"
	"github.com/dinapappor/go-dashcam/camera/v4l2"
	"github.com/dinapappor/go-dashcam/config"
	"log"
)

func configureCamera(device *v4l2.Device, cfg config.VideoConfig, export bool) (v4l2.V4L2Format, error) {
	deviceCapability, err := v4l2.QueryCapability(device.Fd())

	if err != nil {
		return v4l2.V4L2Format{}, fmt.Errorf("Unable to query for device Capabilities, err:%w", err)
	}

	driverName, err := v4l2.ConvertByteToString(deviceCapability.Driver[:])

	if err != nil {
		return v4l2.V4L2Format{}, fmt.Errorf("Could not get driver name, err: %w", err)
	}

	cameraName, err := v4l2.ConvertByteToString(deviceCapability.Card[:])

	if err != nil {
		return v4l2.V4L2Format{}, fmt.Errorf("Could not get camera name, err: %w", err)
	}

	busInfo, err := v4l2.ConvertByteToString(deviceCapability.BusInfo[:])

	if err != nil {
		return v4l2.V4L2Format{}, fmt.Errorf("Could not get bus info, err: %w", err)
	}

	fmt.Printf("Video device: %s (%s), using %s driver on %s\n", cameraName, cfg.VideoSource, driverName, busInfo)

	capabilities := v4l2.SelectCapabilities(deviceCapability)

	// Check for video capture flag
	if !v4l2.HasCapability(capabilities, v4l2.VideoCapture) {
		return v4l2.V4L2Format{}, fmt.Errorf("We do not have video capture capabilities")
	}

	// Check for streaming capabilities.
	// This means buffers that we queue and requeue later
	if !v4l2.HasCapability(capabilities, v4l2.StreamingFlag) {
		return v4l2.V4L2Format{}, fmt.Errorf("We do not have streaming capabilities")
	}

	// To hold which format we request from camera
	var requestedFourCC uint32
	switch cfg.PixelFormat {
	case "MJPG":
		requestedFourCC = v4l2.PixFmtMJPEG
	case "YU12":
		requestedFourCC = v4l2.PixFmtYUV420
	}

	requestedFormat, err := v4l2.SetVideoCaptureFormat(
		device.Fd(),
		cfg.Width,
		cfg.Height,
		requestedFourCC,
	)

	if err != nil {
		return v4l2.V4L2Format{}, fmt.Errorf("Unable to set format: %w", err)
	}

	log.Printf("We got format: %s, width:%d, height: %d",
		v4l2.DecodeFourCC(requestedFormat.PixFormat.PixelFormat),
		requestedFormat.PixFormat.Width,
		requestedFormat.PixFormat.Height)

	streamParams, err := v4l2.QueryVideoCaptureStreamParams(device.Fd())
	if err != nil {
		return v4l2.V4L2Format{}, fmt.Errorf("Unable to query for stream params, got: %w", err)
	}

	if !v4l2.HasCapability(streamParams.Capture.Capability, v4l2.CapTimePerFrame) {
		return v4l2.V4L2Format{}, fmt.Errorf("We do not have TimePerFrame capability")
	}

	_, err = v4l2.SetVideoCaptureStreamParams(device.Fd(), 1, cfg.FrameRate)

	if err != nil {
		return v4l2.V4L2Format{}, fmt.Errorf("Unable to set stream params, got %w", err)
	}
	err = device.SetupBuffers(4)

	if err != nil {
		return v4l2.V4L2Format{}, fmt.Errorf("Unable to setup buffers: %w", err)
	}

	if export {
		err := device.ExportBuffers()
		if err != nil {
			return v4l2.V4L2Format{}, fmt.Errorf("Could not setup export buffers: %w", err)
		}
	}
	return requestedFormat, nil
}

func openCamera(cfg config.VideoConfig, export bool) (*v4l2.Device, v4l2.V4L2Format, error) {
	device, err := v4l2.Open(cfg.VideoSource)

	if err != nil {
		return nil, v4l2.V4L2Format{}, fmt.Errorf("Could not open %s: %w", cfg.VideoSource, err)
	}
	format, err := configureCamera(device, cfg, export)

	if err != nil {
		device.Close()
		return nil, v4l2.V4L2Format{}, fmt.Errorf("Could not configure camera: %w", err)
	}

	return device, format, nil
}
