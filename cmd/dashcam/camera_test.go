package main

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/dinapappor/go-dashcam/camera/v4l2"
	"github.com/dinapappor/go-dashcam/config"
)

func TestOpenCameraMissingPath(t *testing.T) {
	cfg := config.VideoConfig{VideoSource: filepath.Join(t.TempDir(), "missing")}

	device, _, err := openCamera(cfg, false)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("openCamera should fail with ErrNotExist, got %v", err)
	}

	if device != nil {
		t.Errorf("expected nil device for a missing path")
	}
}

func TestConfigureCameraNotV4L2Device(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "fake")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %v", err)
	}
	file.Close()

	device, err := v4l2.Open(file.Name())
	if err != nil {
		t.Fatalf("Unable to open temporary file: %v", err)
	}
	defer device.Close()

	_, err = configureCamera(device, config.VideoConfig{}, false)
	if !errors.Is(err, syscall.ENOTTY) {
		t.Errorf("configureCamera should fail with ENOTTY, got %v", err)
	}
}
